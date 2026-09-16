package admin

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"

	"gestao-retalho/backend/repository"
)

// ErrCredenciaisInvalidas cobre tanto "utilizador não existe" como
// "password errada" com a mesma mensagem — distinguir os dois ajudaria
// alguém a adivinhar o username certo por tentativa e erro.
var ErrCredenciaisInvalidas = errors.New("utilizador ou password incorretos")

// AuthService é o único login de todo o sistema (decisão pós-Fase 10, a
// pedido explícito): um único administrador, aqui, nunca por loja. O
// primeiro arranque do admin.db está sempre vazio — PrecisaConfigurar diz
// ao frontend para mostrar o ecrã de criação em vez do de login.
type AuthService struct {
	repo *repository.AdministradorRepository
}

func NewAuthService(repo *repository.AdministradorRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) PrecisaConfigurar(ctx context.Context) (bool, error) {
	existe, err := s.repo.Existe(ctx)
	if err != nil {
		return false, err
	}
	return !existe, nil
}

// CriarAdministrador só é aceite quando ainda não existe nenhum — "só deve
// existir um" não é só convenção de UI, é imposto aqui.
func (s *AuthService) CriarAdministrador(ctx context.Context, nome, username, password string) (repository.Administrador, error) {
	if nome == "" || username == "" {
		return repository.Administrador{}, fmt.Errorf("nome e username são obrigatórios")
	}
	if len(password) < 6 {
		return repository.Administrador{}, fmt.Errorf("password deve ter pelo menos 6 caracteres")
	}

	existe, err := s.repo.Existe(ctx)
	if err != nil {
		return repository.Administrador{}, err
	}
	if existe {
		return repository.Administrador{}, fmt.Errorf("já existe um administrador configurado")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return repository.Administrador{}, fmt.Errorf("gerar hash da password: %w", err)
	}
	return s.repo.Criar(ctx, nome, username, string(hash))
}

func (s *AuthService) Login(ctx context.Context, username, password string) (repository.Administrador, error) {
	if username == "" || password == "" {
		return repository.Administrador{}, ErrCredenciaisInvalidas
	}

	a, err := s.repo.ObterPorUsername(ctx, username)
	if errors.Is(err, sql.ErrNoRows) {
		return repository.Administrador{}, ErrCredenciaisInvalidas
	}
	if err != nil {
		return repository.Administrador{}, err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(a.PasswordHash), []byte(password)); err != nil {
		return repository.Administrador{}, ErrCredenciaisInvalidas
	}

	return a, nil
}
