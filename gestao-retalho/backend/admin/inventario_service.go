package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// InventarioService serve o ecrã "Inventário" do cmd/admin: iniciar uma
// contagem por armazém, registar o que foi contado e fechar (gerando os
// ajustes). Só resolve a ligação à loja e delega no StockService que o
// cmd/app já usa.
type InventarioService struct {
	lojas *repository.LojaRepository
}

func NewInventarioService(lojas *repository.LojaRepository) *InventarioService {
	return &InventarioService{lojas: lojas}
}

func (s *InventarioService) Listar(ctx context.Context) ([]domain.Inventario, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarInventarios()
}

func (s *InventarioService) Iniciar(ctx context.Context, armazemID int64) (api.IniciarInventarioResponse, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return api.IniciarInventarioResponse{}, err
	}
	defer fechar()

	return app.IniciarInventario(armazemID)
}

func (s *InventarioService) ListarItens(ctx context.Context, inventarioID int64) ([]domain.InventarioItem, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarInventarioItens(inventarioID)
}

func (s *InventarioService) RegistarContagem(ctx context.Context, itemID int64, quantidadeContada float64) (domain.InventarioItem, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.InventarioItem{}, err
	}
	defer fechar()

	return app.RegistarContagemInventario(itemID, quantidadeContada)
}

func (s *InventarioService) Fechar(ctx context.Context, inventarioID int64) ([]domain.MovimentoStock, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.FecharInventario(inventarioID)
}

func (s *InventarioService) abrirLoja(ctx context.Context) (*api.App, func(), error) {
	return abrirLojaUnica(ctx, s.lojas)
}
