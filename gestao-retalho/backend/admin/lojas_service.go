package admin

import (
	"context"
	"fmt"
	"strings"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// LojasService resolve e cria a única loja que este cmd/admin serve
// (decisão pós-Fase 10, a pedido explícito: um administrador, uma loja, que
// pode ter vários armazéns internamente — nunca um registo de várias lojas
// separadas). Dados de negócio continuam inteiramente dentro da base da
// loja, nunca copiados para admin.db.
type LojasService struct {
	repo *repository.LojaRepository
}

func NewLojasService(repo *repository.LojaRepository) *LojasService {
	return &LojasService{repo: repo}
}

// senhaAuditoriaPlaceholder preenche a validação de FinalizarWizard (exige
// password >= 6 caracteres) para o utilizador semente da loja — nunca serve
// para autenticar nada (o único login de todo o sistema é o do cmd/admin,
// ver AuthService); é só atribuição em utilizador_id para auditoria de
// vendas/movimentos dentro dessa loja.
const senhaAuditoriaPlaceholder = "auditoria-sem-login"

// CriarLojaWizardParams junta o que o primeiro arranque do cmd/admin
// precisa para criar a única loja de raiz (Fase 10 addendum) — tipo de
// negócio, nome e logótipo, mais os dados do administrador central que
// ficam gravados dentro da loja só para atribuição de auditoria.
type CriarLojaWizardParams struct {
	Nome          string
	TipoNegocio   domain.TipoNegocio
	Logo          string
	AdminNome     string
	AdminUsername string
}

// CriarComWizard cria o ficheiro .db da loja (em vez de esperar que já
// exista), corre o mesmo FinalizarWizard que o cmd/app correria
// manualmente, e só depois regista o resultado em admin.db.
func (s *LojasService) CriarComWizard(ctx context.Context, p CriarLojaWizardParams) (repository.Loja, error) {
	nome := strings.TrimSpace(p.Nome)
	if nome == "" {
		return repository.Loja{}, fmt.Errorf("nome do negócio é obrigatório")
	}
	if p.TipoNegocio == "" {
		return repository.Loja{}, fmt.Errorf("tipo de negócio é obrigatório")
	}

	caminho, err := config.ResolveNovaLojaDBPath(nome)
	if err != nil {
		return repository.Loja{}, fmt.Errorf("resolver caminho da loja: %w", err)
	}

	app, fechar, err := AbrirLoja(string(caminho))
	if err != nil {
		return repository.Loja{}, fmt.Errorf("criar base da loja: %w", err)
	}
	defer fechar()

	if err := app.FinalizarWizard(usecase.FinalizarWizardParams{
		NegocioNome:          nome,
		TipoNegocio:          p.TipoNegocio,
		ArmazemPrincipalNome: "Principal",
		Logo:                 p.Logo,
		AdminNome:            p.AdminNome,
		AdminUsername:        p.AdminUsername,
		AdminPassword:        senhaAuditoriaPlaceholder,
	}); err != nil {
		return repository.Loja{}, fmt.Errorf("configurar loja: %w", err)
	}

	return s.repo.Criar(ctx, nome, string(caminho))
}

// LojaAtual identifica a única loja que este cmd/admin serve — usada para
// colorir a app inteira (sidebar, botões) com a cor do tipo de negócio em
// vez do azul neutro genérico, e para mostrar o nome real em vez de
// "Administração".
type LojaAtual struct {
	Nome        string
	TipoNegocio string
	Tema        string
	Logo        string // data URL, "" = sem logótipo
}

func (s *LojasService) ObterAtual(ctx context.Context) (LojaAtual, error) {
	app, fechar, err := abrirLojaUnica(ctx, s.repo)
	if err != nil {
		return LojaAtual{}, err
	}
	defer fechar()

	estado, err := app.ObterEstadoConfig()
	if err != nil {
		return LojaAtual{}, err
	}
	return LojaAtual{
		Nome:        estado.Negocio.Nome,
		TipoNegocio: string(estado.Negocio.TipoNegocio),
		Tema:        string(estado.Tema),
		Logo:        estado.Logo,
	}, nil
}

// AtualizarIdentidade muda o nome e o logótipo da loja (dentro da base da
// loja) e o nome no registo do admin, e devolve a loja já atualizada.
func (s *LojasService) AtualizarIdentidade(ctx context.Context, nome, logo string) (LojaAtual, error) {
	loja, existe, err := s.repo.ObterUnica(ctx)
	if err != nil {
		return LojaAtual{}, err
	}
	if !existe {
		return LojaAtual{}, fmt.Errorf("nenhuma loja configurada")
	}
	app, fechar, err := abrirLojaUnica(ctx, s.repo)
	if err != nil {
		return LojaAtual{}, err
	}
	negocio, err := app.AtualizarIdentidadeLoja(nome, logo)
	fechar()
	if err != nil {
		return LojaAtual{}, err
	}
	if err := s.repo.AtualizarNome(ctx, loja.ID, negocio.Nome); err != nil {
		return LojaAtual{}, err
	}
	return s.ObterAtual(ctx)
}
