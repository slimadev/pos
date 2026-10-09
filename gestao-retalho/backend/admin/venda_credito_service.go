package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

// A venda a crédito só existe aqui, atrás do login do administrador: o
// api.App do PDV não tem nenhum binding para ela, e chama-se diretamente o
// VendaService da loja.

type ItemCatalogo struct {
	Produto  domain.Produto
	Unidades []domain.UnidadeProduto
	// Stock é em unidade base, no armazém pedido; 0 para compostos (o
	// stock é dos ingredientes, validado na venda).
	Stock float64
}

type ResultadoVendaCredito struct {
	Venda  usecase.VendaCompleta
	Alerta usecase.AlertaLimiteCredito
}

// CatalogoVenda devolve, numa só ligação, os produtos ativos com as
// unidades de venda e o stock no armazém.
func (s *VendasService) CatalogoVenda(ctx context.Context, armazemID int64) ([]ItemCatalogo, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]ItemCatalogo, error) {
		produtos, err := a.ListarProdutos()
		if err != nil {
			return nil, err
		}
		stock, err := a.ListarStockPorArmazem(armazemID)
		if err != nil {
			return nil, err
		}
		stockDe := make(map[int64]float64, len(stock))
		for _, it := range stock {
			stockDe[it.ProdutoID] = it.Quantidade
		}

		out := make([]ItemCatalogo, 0, len(produtos))
		for _, p := range produtos {
			if !p.Ativo {
				continue
			}
			unidades, err := a.ListarUnidadesProduto(p.ID)
			if err != nil {
				return nil, err
			}
			if len(unidades) == 0 {
				continue
			}
			out = append(out, ItemCatalogo{Produto: p, Unidades: unidades, Stock: stockDe[p.ID]})
		}
		return out, nil
	})
}

// RegistarVendaACredito regista a venda e devolve também o aviso de limite
// de crédito — que nunca bloqueia, como no fecho de uma conta em dívida.
func (s *VendasService) RegistarVendaACredito(ctx context.Context, p usecase.VendaDiretaParams) (ResultadoVendaCredito, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (ResultadoVendaCredito, error) {
		p.UtilizadorID = api.UtilizadorAdminID
		venda, err := a.VendaService.RegistarVendaACredito(ctx, p)
		if err != nil {
			return ResultadoVendaCredito{}, err
		}
		alerta, err := a.DividaService.VerificarLimiteCredito(ctx, *p.ClienteID, 0)
		if err != nil {
			// A venda já está gravada; o aviso é só informativo.
			return ResultadoVendaCredito{Venda: venda}, nil
		}
		return ResultadoVendaCredito{Venda: venda, Alerta: alerta}, nil
	})
}
