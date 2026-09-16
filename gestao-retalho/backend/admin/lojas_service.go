package admin

import (
	"context"
	"fmt"
	"os"
	"strings"

	"gestao-retalho/backend/repository"
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
