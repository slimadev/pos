package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// UtilizadoresService serve o ecrã "Utilizadores" do cmd/admin: os
// operadores do PDV e os seus PIN. Chama o OperadorService da loja
// diretamente — o api.App do PDV só sabe listar os ativos e validar o PIN,
// nunca criar operadores nem mudar PINs.
type UtilizadoresService struct {
	lojas *repository.LojaRepository
}

func NewUtilizadoresService(lojas *repository.LojaRepository) *UtilizadoresService {
	return &UtilizadoresService{lojas: lojas}
}

func (s *UtilizadoresService) Listar(ctx context.Context) ([]usecase.Operador, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]usecase.Operador, error) { return a.OperadorService.Listar(ctx) })
}

func (s *UtilizadoresService) Criar(ctx context.Context, nome, pin string) (usecase.Operador, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (usecase.Operador, error) { return a.OperadorService.Criar(ctx, nome, pin) })
}

func (s *UtilizadoresService) Renomear(ctx context.Context, id int64, nome string) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.OperadorService.Renomear(ctx, id, nome) })
	return err
}

func (s *UtilizadoresService) RedefinirPIN(ctx context.Context, id int64, pin string) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) { return struct{}{}, a.OperadorService.RedefinirPIN(ctx, id, pin) })
	return err
}

func (s *UtilizadoresService) DefinirAtivo(ctx context.Context, id int64, ativo bool) error {
	_, err := comLoja(ctx, s.lojas, func(a *api.App) (struct{}, error) {
		return struct{}{}, a.OperadorService.DefinirAtivo(ctx, id, ativo)
	})
	return err
}
