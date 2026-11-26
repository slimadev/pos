package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

var (
	ErrCaixaJaAberto = errors.New("o caixa já está aberto")
	ErrSemCaixa      = errors.New("o caixa não está aberto")
)

// CaixaService abre e fecha o turno de caixa: na abertura conta-se o fundo
// de troco; no fecho, conta-se o que há por método e compara-se com o que
// o sistema diz que entrou desde a abertura (vendas + dívidas recebidas).
type CaixaService struct {
	rest         *repository.RestauranteRepository
	relatorios   *RelatorioService
	utilizadores *repository.UtilizadorRepository
	agora        func() time.Time
}

func NewCaixaService(rest *repository.RestauranteRepository, relatorios *RelatorioService, utilizadores *repository.UtilizadorRepository) *CaixaService {
	return &CaixaService{rest: rest, relatorios: relatorios, utilizadores: utilizadores, agora: time.Now}
}

type LinhaConferencia struct {
	Metodo   domain.MetodoPagamento
	Esperado float64 // no dinheiro inclui o fundo inicial
	Contado  float64
	// Diferenca = Contado - Esperado: negativo é falta, positivo é sobra.
	Diferenca float64
}

type EstadoCaixa struct {
	Aberto    bool
	Sessao    *domain.SessaoCaixa
	AbertoPor string
	// Recebido desde a abertura, por método.
	Recebido FechoCaixa
	// DinheiroEsperado = fundo inicial + dinheiro recebido.
	DinheiroEsperado float64
}

type FechoSessao struct {
	Sessao     domain.SessaoCaixa
	AbertoPor  string
	FechadoPor string
	Recebido   FechoCaixa
	Linhas     []LinhaConferencia
	// DiferencaTotal soma as diferenças de todos os métodos.
	DiferencaTotal float64
}

func (s *CaixaService) nome(ctx context.Context, id int64) string {
	u, err := s.utilizadores.Obter(ctx, id)
	if err != nil {
		return "?"
	}
	return u.Nome
}

// Atual diz se o caixa está aberto e quanto entrou até agora.
func (s *CaixaService) Atual(ctx context.Context) (EstadoCaixa, error) {
	sessao, ok, err := s.rest.SessaoAberta(ctx)
	if err != nil || !ok {
		return EstadoCaixa{Recebido: FechoCaixa{PorMetodo: []LinhaFechoMetodo{}}}, err
	}
	rec, err := s.relatorios.RecebidoNoPeriodo(ctx, sessao.AbertaEm, s.agora().Add(time.Second))
	if err != nil {
		return EstadoCaixa{}, err
	}
	est := EstadoCaixa{Aberto: true, Sessao: &sessao, AbertoPor: s.nome(ctx, sessao.UtilizadorAberturaID), Recebido: rec,
		DinheiroEsperado: sessao.FundoInicial}
	for _, l := range rec.PorMetodo {
		if l.Metodo == domain.MetodoDinheiro {
			est.DinheiroEsperado += l.Total
		}
	}
	return est, nil
}

// Abrir começa o turno com o fundo de troco que está na gaveta.
func (s *CaixaService) Abrir(ctx context.Context, utilizadorID int64, fundo float64) (domain.SessaoCaixa, error) {
	if fundo < 0 || math.IsNaN(fundo) {
		return domain.SessaoCaixa{}, fmt.Errorf("o fundo de caixa não pode ser negativo")
	}
	if _, ok, err := s.rest.SessaoAberta(ctx); err != nil {
		return domain.SessaoCaixa{}, err
	} else if ok {
		return domain.SessaoCaixa{}, ErrCaixaJaAberto
	}
	if _, err := s.utilizadores.Obter(ctx, utilizadorID); err != nil {
		return domain.SessaoCaixa{}, err
	}
	return s.rest.AbrirSessao(ctx, utilizadorID, fundo)
}

// Fechar fecha o turno com a contagem de cada método (o dinheiro na
// gaveta, o total das M-Pesa, do cartão…) e devolve a conferência.
func (s *CaixaService) Fechar(ctx context.Context, utilizadorID int64, contagem map[domain.MetodoPagamento]float64, observacoes string) (FechoSessao, error) {
	sessao, ok, err := s.rest.SessaoAberta(ctx)
	if err != nil {
		return FechoSessao{}, err
	}
	if !ok {
		return FechoSessao{}, ErrSemCaixa
	}
	limpa := map[domain.MetodoPagamento]float64{}
	for m, v := range contagem {
		if v < 0 || math.IsNaN(v) {
			return FechoSessao{}, fmt.Errorf("a contagem não pode ser negativa")
		}
		limpa[m] = v
	}
	var obs *string
	if o := strings.TrimSpace(observacoes); o != "" {
		obs = &o
	}
	fechada, err := s.rest.FecharSessao(ctx, sessao.ID, utilizadorID, limpa, obs)
	if err != nil {
		return FechoSessao{}, err
	}
	return s.conferencia(ctx, fechada)
}

// ObterFecho refaz a conferência de uma sessão já fechada (para reimprimir).
func (s *CaixaService) ObterFecho(ctx context.Context, id int64) (FechoSessao, error) {
	sessao, err := s.rest.ObterSessao(ctx, id)
	if err != nil {
		return FechoSessao{}, err
	}
	if sessao.FechadaEm == nil {
		return FechoSessao{}, fmt.Errorf("esta sessão de caixa ainda está aberta")
	}
	return s.conferencia(ctx, sessao)
}

func (s *CaixaService) Historico(ctx context.Context, limite int64) ([]domain.SessaoCaixa, error) {
	if limite <= 0 {
		limite = 30
	}
	return s.rest.ListarSessoes(ctx, limite)
}

var ordemMetodos = map[domain.MetodoPagamento]int{
	domain.MetodoDinheiro: 0, domain.MetodoMpesa: 1, domain.MetodoEmola: 2, domain.MetodoCartao: 3, domain.MetodoTransferencia: 4,
}

func (s *CaixaService) conferencia(ctx context.Context, sessao domain.SessaoCaixa) (FechoSessao, error) {
	// O fim inclui o próprio instante do fecho (o filtro é [inicio, fim)).
	rec, err := s.relatorios.RecebidoNoPeriodo(ctx, sessao.AbertaEm, sessao.FechadaEm.Add(time.Millisecond))
	if err != nil {
		return FechoSessao{}, err
	}
	f := FechoSessao{Sessao: sessao, Recebido: rec, AbertoPor: s.nome(ctx, sessao.UtilizadorAberturaID)}
	if sessao.UtilizadorFechoID != nil {
		f.FechadoPor = s.nome(ctx, *sessao.UtilizadorFechoID)
	}
	esperado := map[domain.MetodoPagamento]float64{domain.MetodoDinheiro: sessao.FundoInicial}
	for _, l := range rec.PorMetodo {
		esperado[l.Metodo] += l.Total
	}
	for m := range sessao.Contagem {
		if _, ok := esperado[m]; !ok {
			esperado[m] = 0
		}
	}
	for m, e := range esperado {
		c := sessao.Contagem[m]
		f.Linhas = append(f.Linhas, LinhaConferencia{Metodo: m, Esperado: e, Contado: c, Diferenca: c - e})
		f.DiferencaTotal += c - e
	}
	sort.Slice(f.Linhas, func(i, j int) bool {
		oi, ok := ordemMetodos[f.Linhas[i].Metodo]
		if !ok {
			oi = 99
		}
		oj, ok := ordemMetodos[f.Linhas[j].Metodo]
		if !ok {
			oj = 99
		}
		if oi != oj {
			return oi < oj
		}
		return f.Linhas[i].Metodo < f.Linhas[j].Metodo
	})
	return f, nil
}
