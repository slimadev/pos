package admin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// LojasService é o CRUD de "Lojas" da tela `1c` (SDD, Fase 9) — gere só o
// registo (nome + caminho do ficheiro .db); os dados de negócio continuam
// inteiramente dentro da base de cada loja, nunca copiados para admin.db.
type LojasService struct {
	repo *repository.LojaRepository
}

func NewLojasService(repo *repository.LojaRepository) *LojasService {
	return &LojasService{repo: repo}
}

func (s *LojasService) Listar(ctx context.Context) ([]repository.Loja, error) {
	return s.repo.Listar(ctx)
}

// Criar valida que o caminho aponta para um ficheiro existente e já
// configurado (wizard concluído) antes de o registar — evita o erro comum
// de escrever mal o caminho e só descobrir no dashboard, com um cartão
// "offline" sem explicação.
func (s *LojasService) Criar(ctx context.Context, nome, caminhoDB string) (repository.Loja, error) {
	nome = strings.TrimSpace(nome)
	caminhoDB = strings.TrimSpace(caminhoDB)
	if nome == "" {
		return repository.Loja{}, fmt.Errorf("nome da loja é obrigatório")
	}
	if caminhoDB == "" {
		return repository.Loja{}, fmt.Errorf("caminho do ficheiro .db é obrigatório")
	}
	if _, err := os.Stat(caminhoDB); err != nil {
		return repository.Loja{}, fmt.Errorf("ficheiro não encontrado em %s", caminhoDB)
	}

	app, fechar, err := AbrirLoja(caminhoDB)
	if err != nil {
		return repository.Loja{}, fmt.Errorf("não foi possível ligar à base: %w", err)
	}
	defer fechar()

	configurada, err := app.EstaConfigurado()
	if err != nil {
		return repository.Loja{}, fmt.Errorf("base em %s não parece ser uma loja válida: %w", caminhoDB, err)
	}
	if !configurada {
		return repository.Loja{}, fmt.Errorf("a loja em %s ainda não concluiu o assistente de configuração", caminhoDB)
	}

	return s.repo.Criar(ctx, nome, caminhoDB)
}

// senhaAuditoriaPlaceholder preenche a validação de FinalizarWizard (exige
// password >= 6 caracteres) para o utilizador semente de uma loja criada
// por aqui — nunca serve para autenticar nada (o único login de todo o
// sistema é o do cmd/admin, ver AuthService); é só atribuição em
// utilizador_id para auditoria de vendas/movimentos dentro dessa loja.
const senhaAuditoriaPlaceholder = "auditoria-sem-login"

// CriarLojaWizardParams junta o que o primeiro arranque do cmd/admin
// precisa para criar uma loja nova de raiz (Fase 10 addendum) — tipo de
// negócio, nome e logótipo, mais os dados do administrador central que
// ficam gravados dentro da loja só para atribuição de auditoria.
type CriarLojaWizardParams struct {
	Nome          string
	TipoNegocio   domain.TipoNegocio
	Logo          string
	AdminNome     string
	AdminUsername string
}

// CriarComWizard cria o ficheiro .db de uma loja nova (em vez de esperar
// que já exista, como Criar), corre o mesmo FinalizarWizard que o cmd/app
// correria manualmente, e só depois regista o resultado — dados de negócio
// continuam inteiramente dentro da base da loja, admin.db só fica com o
// essencial para lá chegar (ver comentário de repository.Loja).
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

func (s *LojasService) Atualizar(ctx context.Context, id int64, nome, caminhoDB string) error {
	nome = strings.TrimSpace(nome)
	caminhoDB = strings.TrimSpace(caminhoDB)
	if nome == "" {
		return fmt.Errorf("nome da loja é obrigatório")
	}
	if caminhoDB == "" {
		return fmt.Errorf("caminho do ficheiro .db é obrigatório")
	}
	return s.repo.Atualizar(ctx, id, nome, caminhoDB)
}

func (s *LojasService) Ativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtiva(ctx, id, true)
}

func (s *LojasService) Desativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtiva(ctx, id, false)
}
