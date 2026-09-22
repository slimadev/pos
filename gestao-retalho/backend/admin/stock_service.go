package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// StockService gere stock (entrada, transferência, perda, consumo interno,
// consulta e movimentos) da única loja que este cmd/admin serve — dados de
// negócio continuam inteiramente dentro da base da loja (SDD 8.5), este
// serviço só resolve a ligação (AbrirLoja) e delega no mesmo StockService
// que o cmd/app usaria.
type StockService struct {
	lojas *repository.LojaRepository
}

func NewStockService(lojas *repository.LojaRepository) *StockService {
	return &StockService{lojas: lojas}
}

func (s *StockService) ListarPorArmazem(ctx context.Context, armazemID int64) ([]domain.StockItem, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarStockPorArmazem(armazemID)
}

func (s *StockService) ListarMovimentos(ctx context.Context, produtoID int64) ([]domain.MovimentoStock, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarMovimentosStock(produtoID)
}

func (s *StockService) RegistarEntrada(ctx context.Context, p usecase.EntradaParams) (domain.MovimentoStock, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	defer fechar()

	return app.RegistarEntradaStock(p)
}

func (s *StockService) Transferir(ctx context.Context, p usecase.TransferenciaParams) (domain.MovimentoStock, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	defer fechar()

	return app.TransferirStock(p)
}

func (s *StockService) RegistarPerda(ctx context.Context, p usecase.SaidaParams) (domain.MovimentoStock, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	defer fechar()

	return app.RegistarPerdaStock(p)
}

func (s *StockService) RegistarConsumoInterno(ctx context.Context, p usecase.SaidaParams) (domain.MovimentoStock, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	defer fechar()

	return app.RegistarConsumoInternoStock(p)
}

// RegistarEntradaComLote é o único caminho de entrada para produtos
// LoteControlado — RegistarEntrada rejeita-os (ver StockService no
// backend/usecase, que StockService.RegistarEntrada delega para cá).
func (s *StockService) RegistarEntradaComLote(ctx context.Context, p usecase.EntradaComLoteParams) (api.EntradaComLoteResponse, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return api.EntradaComLoteResponse{}, err
	}
	defer fechar()

	return app.RegistarEntradaComLote(p)
}

func (s *StockService) ListarLotes(ctx context.Context, produtoID, armazemID int64) ([]domain.Lote, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarLotes(produtoID, armazemID)
}

func (s *StockService) abrirLoja(ctx context.Context) (*api.App, func(), error) {
	return abrirLojaUnica(ctx, s.lojas)
}
