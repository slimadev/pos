package usecase

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

type ArmazemService struct {
	repo *repository.ArmazemRepository
}

func NewArmazemService(repo *repository.ArmazemRepository) *ArmazemService {
	return &ArmazemService{repo: repo}
}

func (s *ArmazemService) Listar(ctx context.Context, negocioID int64) ([]domain.Armazem, error) {
	return s.repo.Listar(ctx, negocioID)
}

func (s *ArmazemService) Obter(ctx context.Context, id int64) (domain.Armazem, error) {
	return s.repo.Obter(ctx, id)
}

func (s *ArmazemService) Criar(ctx context.Context, a domain.Armazem) (domain.Armazem, error) {
	if a.Nome == "" {
		return domain.Armazem{}, fmt.Errorf("nome do armazém é obrigatório")
	}
	// Sempre a mesma loja (SDD 2.1) — forçado aqui, não confiado a cada
	// chamador, para nunca haver um armazém órfão por esquecimento.
	a.NegocioID = domain.NegocioID
	a.Ativo = true
	return s.repo.Criar(ctx, a)
}

func (s *ArmazemService) Atualizar(ctx context.Context, a domain.Armazem) error {
	if a.Nome == "" {
		return fmt.Errorf("nome do armazém é obrigatório")
	}
	return s.repo.Atualizar(ctx, a)
}

func (s *ArmazemService) Ativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtivo(ctx, id, true)
}

func (s *ArmazemService) Desativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtivo(ctx, id, false)
}
