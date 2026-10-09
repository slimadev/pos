package admin

import (
	"context"
	"fmt"
	"time"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// RelatoriosService serve o ecrã "Relatórios" do cmd/admin: os mesmos
// relatórios que o cmd/app já calcula (RelatorioService), cada um numa
// ligação curta à loja, e a exportação Excel/PDF.
type RelatoriosService struct {
	lojas *repository.LojaRepository
}

func NewRelatoriosService(lojas *repository.LojaRepository) *RelatoriosService {
	return &RelatoriosService{lojas: lojas}
}

func comLoja[T any](ctx context.Context, lojas *repository.LojaRepository, fn func(*api.App) (T, error)) (T, error) {
	app, fechar, err := abrirLojaUnica(ctx, lojas)
	if err != nil {
		var vazio T
		return vazio, err
	}
	defer fechar()
	return fn(app)
}

func (s *RelatoriosService) VendasPorPeriodo(ctx context.Context, f api.FiltroRelatorioRequest) (api.RelatorioVendasResponse, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (api.RelatorioVendasResponse, error) { return a.RelatorioVendasPorPeriodo(f) })
}

func (s *RelatoriosService) LucroPorPeriodo(ctx context.Context, f api.FiltroRelatorioRequest) (usecase.RelatorioLucro, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (usecase.RelatorioLucro, error) { return a.RelatorioLucroPorPeriodo(f) })
}

func (s *RelatoriosService) VendasPorProduto(ctx context.Context, f api.FiltroRelatorioRequest) ([]repository.LinhaVendasPorProduto, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]repository.LinhaVendasPorProduto, error) { return a.RelatorioVendasPorProduto(f) })
}

func (s *RelatoriosService) ConsumoTeoricoVsReal(ctx context.Context, f api.FiltroRelatorioRequest) ([]usecase.LinhaConsumoIngrediente, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]usecase.LinhaConsumoIngrediente, error) { return a.RelatorioConsumoTeoricoVsReal(f) })
}

func (s *RelatoriosService) VendasPorMetodoPagamento(ctx context.Context, f api.FiltroRelatorioRequest) ([]repository.LinhaVendasPorMetodo, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]repository.LinhaVendasPorMetodo, error) {
		return a.RelatorioVendasPorMetodoPagamento(f)
	})
}

func (s *RelatoriosService) StockAtual(ctx context.Context, armazemID *int64) (api.RelatorioStockAtualResponse, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (api.RelatorioStockAtualResponse, error) { return a.RelatorioStockAtual(armazemID) })
}

func (s *RelatoriosService) MovimentosPorPeriodo(ctx context.Context, f api.FiltroRelatorioRequest) ([]repository.LinhaMovimento, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]repository.LinhaMovimento, error) { return a.RelatorioMovimentosPorPeriodo(f) })
}

func (s *RelatoriosService) PerdasConsumoInterno(ctx context.Context, f api.FiltroRelatorioRequest) (api.RelatorioPerdasConsumoInternoResponse, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (api.RelatorioPerdasConsumoInternoResponse, error) {
		return a.RelatorioPerdasConsumoInterno(f)
	})
}

// ContasPorPeriodo recebe datas "AAAA-MM-DD", ambas inclusive.
func (s *RelatoriosService) ContasPorPeriodo(ctx context.Context, desde, ate string) (api.RelatorioContasResponse, error) {
	inicio, err := diaLocal(desde)
	if err != nil {
		return api.RelatorioContasResponse{}, err
	}
	fim, err := diaLocal(ate)
	if err != nil {
		return api.RelatorioContasResponse{}, err
	}
	return comLoja(ctx, s.lojas, func(a *api.App) (api.RelatorioContasResponse, error) {
		return a.RelatorioContasPorPeriodo(inicio, fim.AddDate(0, 0, 1))
	})
}

func (s *RelatoriosService) DividasEmAberto(ctx context.Context) ([]usecase.LinhaDividaRelatorio, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]usecase.LinhaDividaRelatorio, error) { return a.RelatorioDividasEmAberto() })
}

// FechoCaixaDiario recebe o dia como "AAAA-MM-DD".
func (s *RelatoriosService) FechoCaixaDiario(ctx context.Context, dia string) (usecase.FechoCaixa, error) {
	d, err := diaLocal(dia)
	if err != nil {
		return usecase.FechoCaixa{}, err
	}
	return comLoja(ctx, s.lojas, func(a *api.App) (usecase.FechoCaixa, error) { return a.RelatorioFechoCaixaDiario(d) })
}

// diaLocal lê "AAAA-MM-DD" como a meia-noite desse dia no relógio da loja.
// As datas chegam como texto, e não como time.Time, porque um parâmetro
// time.Time num binding faz o Wails tipar todas as datas dos modelos TS
// como uma classe Time em vez de string.
func diaLocal(dia string) (time.Time, error) {
	d, err := time.ParseInLocation("2006-01-02", dia, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("data inválida: %q", dia)
	}
	return d, nil
}

// Exportar grava a tabela já formatada pelo ecrã ("xlsx" ou "pdf") e
// devolve o caminho; não precisa da loja.
func (s *RelatoriosService) Exportar(t export.Tabela, formato string) (string, error) {
	return export.GravarTabela(t, formato)
}
