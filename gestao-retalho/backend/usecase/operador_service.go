package usecase

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// Operador é o que sai para os ecrãs: nunca o hash do PIN.
type Operador struct {
	ID    int64
	Nome  string
	Ativo bool
	// Gerente autoriza o que o garçom não pode fazer sozinho: anular itens
	// já enviados para a cozinha, transferir mesas, reabrir o caixa.
	Gerente bool
	// Caixa: fecha contas, vende direto, abre/fecha o caixa e mexe nas
	// mesas dos outros — como o gerente, no restaurante.
	Caixa bool
}

func operadorDe(u domain.Utilizador) Operador {
	return Operador{ID: u.ID, Nome: u.Nome, Ativo: u.Ativo, Gerente: u.Role == domain.RoleGerente, Caixa: u.Caixa}
}

// eOperador: operadores do PDV e gerentes (que também trabalham no PDV).
func eOperador(u domain.Utilizador) bool {
	return u.Role == domain.RoleOperadorPOS || u.Role == domain.RoleGerente
}

var (
	ErrPINFormato      = errors.New("o PIN tem de ter 4 a 6 algarismos")
	ErrPINErrado       = errors.New("PIN errado")
	ErrNaoEOperador    = errors.New("utilizador não é um operador do PDV")
	ErrOperadorInativo = errors.New("operador desativado")
)

var pinValido = regexp.MustCompile(`^[0-9]{4,6}$`)

const (
	maxTentativasPIN = 5
	bloqueioPIN      = 30 * time.Second
)

// OperadorService gere quem trabalha no PDV (role operador_pos) e valida o
// PIN com que cada um se identifica ao abrir o PDV — não é um login
// completo, só o suficiente para cada venda/movimento ficar com quem a fez.
type OperadorService struct {
	repo *repository.UtilizadorRepository

	mu         sync.Mutex
	falhas     map[int64]int
	bloqueados map[int64]time.Time
	agora      func() time.Time
}

func NewOperadorService(repo *repository.UtilizadorRepository) *OperadorService {
	return &OperadorService{repo: repo, falhas: map[int64]int{}, bloqueados: map[int64]time.Time{}, agora: time.Now}
}

func (s *OperadorService) Listar(ctx context.Context) ([]Operador, error) {
	todos, err := s.repo.Listar(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Operador, 0, len(todos))
	for _, u := range todos {
		if eOperador(u) {
			out = append(out, operadorDe(u))
		}
	}
	return out, nil
}

func (s *OperadorService) ListarAtivos(ctx context.Context) ([]Operador, error) {
	todos, err := s.Listar(ctx)
	if err != nil {
		return nil, err
	}
	ativos := make([]Operador, 0, len(todos))
	for _, o := range todos {
		if o.Ativo {
			ativos = append(ativos, o)
		}
	}
	return ativos, nil
}

func (s *OperadorService) Criar(ctx context.Context, nome, pin string) (Operador, error) {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return Operador{}, fmt.Errorf("o nome do operador é obrigatório")
	}
	hash, err := hashPIN(pin)
	if err != nil {
		return Operador{}, err
	}
	username, err := s.usernameLivre(ctx, nome)
	if err != nil {
		return Operador{}, err
	}
	negocio := domain.NegocioID
	u, err := s.repo.Criar(ctx, domain.Utilizador{
		NegocioID:    &negocio,
		Nome:         nome,
		Username:     username,
		PasswordHash: hash,
		Role:         domain.RoleOperadorPOS,
	})
	if err != nil {
		return Operador{}, err
	}
	return Operador{ID: u.ID, Nome: u.Nome, Ativo: u.Ativo}, nil
}

func (s *OperadorService) Renomear(ctx context.Context, id int64, nome string) error {
	nome = strings.TrimSpace(nome)
	if nome == "" {
		return fmt.Errorf("o nome do operador é obrigatório")
	}
	if _, err := s.obterOperador(ctx, id); err != nil {
		return err
	}
	return s.repo.AtualizarNome(ctx, id, nome)
}

func (s *OperadorService) RedefinirPIN(ctx context.Context, id int64, pin string) error {
	hash, err := hashPIN(pin)
	if err != nil {
		return err
	}
	if _, err := s.obterOperador(ctx, id); err != nil {
		return err
	}
	if err := s.repo.AtualizarPassword(ctx, id, hash); err != nil {
		return err
	}
	s.mu.Lock()
	delete(s.falhas, id)
	delete(s.bloqueados, id)
	s.mu.Unlock()
	return nil
}

func (s *OperadorService) DefinirAtivo(ctx context.Context, id int64, ativo bool) error {
	if _, err := s.obterOperador(ctx, id); err != nil {
		return err
	}
	return s.repo.DefinirAtivo(ctx, id, ativo)
}

// Autenticar valida o PIN do operador escolhido no PDV. Depois de
// maxTentativasPIN falhas seguidas, recusa durante bloqueioPIN — um PIN de 4
// algarismos só tem 10 000 combinações.
func (s *OperadorService) Autenticar(ctx context.Context, id int64, pin string) (Operador, error) {
	s.mu.Lock()
	ate, bloqueado := s.bloqueados[id]
	if bloqueado && s.agora().Before(ate) {
		s.mu.Unlock()
		segundos := int(ate.Sub(s.agora()).Seconds()) + 1
		return Operador{}, fmt.Errorf("demasiadas tentativas erradas — tente de novo daqui a %d s", segundos)
	}
	s.mu.Unlock()

	u, err := s.obterOperador(ctx, id)
	if err != nil {
		return Operador{}, err
	}
	if !u.Ativo {
		return Operador{}, ErrOperadorInativo
	}
	if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pin)) != nil {
		s.mu.Lock()
		s.falhas[id]++
		if s.falhas[id] >= maxTentativasPIN {
			s.bloqueados[id] = s.agora().Add(bloqueioPIN)
			s.falhas[id] = 0
		}
		s.mu.Unlock()
		return Operador{}, ErrPINErrado
	}

	s.mu.Lock()
	delete(s.falhas, id)
	delete(s.bloqueados, id)
	s.mu.Unlock()
	return operadorDe(u), nil
}

// DefinirGerente promove um operador a gerente ou volta a pô-lo como
// operador simples.
func (s *OperadorService) DefinirGerente(ctx context.Context, id int64, gerente bool) error {
	if _, err := s.obterOperador(ctx, id); err != nil {
		return err
	}
	role := domain.RoleOperadorPOS
	if gerente {
		role = domain.RoleGerente
	}
	return s.repo.AtualizarRole(ctx, id, role)
}

// DefinirCaixa marca (ou desmarca) o operador como caixa.
func (s *OperadorService) DefinirCaixa(ctx context.Context, id int64, caixa bool) error {
	if _, err := s.obterOperador(ctx, id); err != nil {
		return err
	}
	return s.repo.DefinirCaixa(ctx, id, caixa)
}

// ErrSemAutorizacao: o PIN não é de nenhum gerente nem caixa ativo.
var ErrSemAutorizacao = errors.New("PIN de gerente ou caixa inválido")

// AutorizarGerente procura o gerente (ou caixa) ativo dono deste PIN — é o
// "chame o gerente" do restaurante: ele digita o PIN no aparelho do garçom. As falhas
// contam para um bloqueio comum (id 0), para não se adivinhar o PIN.
func (s *OperadorService) AutorizarGerente(ctx context.Context, pin string) (Operador, error) {
	s.mu.Lock()
	ate, bloqueado := s.bloqueados[0]
	if bloqueado && s.agora().Before(ate) {
		s.mu.Unlock()
		segundos := int(ate.Sub(s.agora()).Seconds()) + 1
		return Operador{}, fmt.Errorf("demasiadas tentativas erradas — tente de novo daqui a %d s", segundos)
	}
	s.mu.Unlock()

	todos, err := s.repo.Listar(ctx)
	if err != nil {
		return Operador{}, err
	}
	for _, u := range todos {
		if u.PodeGerirMesas() && u.Role != domain.RoleAdmin && u.Ativo && bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(pin)) == nil {
			s.mu.Lock()
			delete(s.falhas, 0)
			s.mu.Unlock()
			return operadorDe(u), nil
		}
	}
	s.mu.Lock()
	s.falhas[0]++
	if s.falhas[0] >= maxTentativasPIN {
		s.bloqueados[0] = s.agora().Add(bloqueioPIN)
		s.falhas[0] = 0
	}
	s.mu.Unlock()
	return Operador{}, ErrSemAutorizacao
}

func (s *OperadorService) obterOperador(ctx context.Context, id int64) (domain.Utilizador, error) {
	u, err := s.repo.Obter(ctx, id)
	if err != nil {
		return domain.Utilizador{}, err
	}
	if !eOperador(u) {
		return domain.Utilizador{}, ErrNaoEOperador
	}
	return u, nil
}

// usernameLivre gera um username único a partir do nome — a coluna é
// NOT NULL UNIQUE, mas o operador nunca o usa (escolhe o nome no PDV).
func (s *OperadorService) usernameLivre(ctx context.Context, nome string) (string, error) {
	base := "op-" + slug(nome)
	for i := 1; ; i++ {
		candidato := base
		if i > 1 {
			candidato = fmt.Sprintf("%s-%d", base, i)
		}
		_, err := s.repo.ObterPorUsername(ctx, candidato)
		if errors.Is(err, sql.ErrNoRows) {
			return candidato, nil
		}
		if err != nil {
			return "", err
		}
	}
}

var naoAlfanumerico = regexp.MustCompile(`[^a-z0-9]+`)

func slug(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i", "ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c").Replace(s)
	s = strings.Trim(naoAlfanumerico.ReplaceAllString(s, "-"), "-")
	if s == "" {
		return "operador"
	}
	return s
}

func hashPIN(pin string) (string, error) {
	if !pinValido.MatchString(pin) {
		return "", ErrPINFormato
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("gerar hash do PIN: %w", err)
	}
	return string(hash), nil
}
