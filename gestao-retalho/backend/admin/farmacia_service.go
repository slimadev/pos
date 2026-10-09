package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// FarmaciaService serve o ecrã "Farmácia" do cmd/admin: lotes a vencer e
// receitas médicas (opcionais) associadas às vendas do dia. Tal como os
// outros serviços de admin, só abre a ligação à loja e delega no
// FarmaciaService/VendaService que o cmd/app já usa.
type FarmaciaService struct {
	lojas *repository.LojaRepository
}

func NewFarmaciaService(lojas *repository.LojaRepository) *FarmaciaService {
	return &FarmaciaService{lojas: lojas}
}

func (s *FarmaciaService) ListarLotesAVencer(ctx context.Context, dentroDeDias int) ([]domain.Lote, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarLotesAVencer(dentroDeDias)
}

func (s *FarmaciaService) ListarVendasDoDia(ctx context.Context) ([]domain.Venda, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarVendasDoDia()
}

func (s *FarmaciaService) RegistarReceitaMedica(ctx context.Context, p usecase.ReceitaMedicaInput) (domain.ReceitaMedica, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.ReceitaMedica{}, err
	}
	defer fechar()

	return app.RegistarReceitaMedica(p)
}

func (s *FarmaciaService) ListarReceitasPorVenda(ctx context.Context, vendaID int64) ([]domain.ReceitaMedica, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarReceitasPorVenda(vendaID)
}

func (s *FarmaciaService) abrirLoja(ctx context.Context) (*api.App, func(), error) {
	return abrirLojaUnica(ctx, s.lojas)
}
