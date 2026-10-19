package usecase

import (
	"context"
	"math"
	"sort"
	"strings"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// ItemCatalogo é um produto tal como o PDV o mostra e vende.
type ItemCatalogo struct {
	Produto         domain.Produto
	Unidades        []domain.UnidadeProduto
	UnidadePadraoID int64
	// Disponivel é o stock no armazém (unidade base); para compostos, quantas
	// unidades dá para preparar com o stock dos ingredientes.
	Disponivel float64
	// Vendido30Dias ordena a grelha: os mais vendidos ficam à mão.
	Vendido30Dias float64
}

// CatalogoService monta o catálogo do PDV com um número fixo de consultas
// (produtos, unidades, stock, vendas recentes e uma por receita de
// composto) — pedir unidades e stock produto a produto tornava o arranque
// do PDV lento numa loja com centenas de artigos.
type CatalogoService struct {
	produtos   *repository.ProdutoRepository
	stock      *repository.StockRepository
	fichas     *repository.FichaTecnicaRepository
	relatorios *repository.RelatorioRepository
}

func NewCatalogoService(
	produtos *repository.ProdutoRepository,
	stock *repository.StockRepository,
	fichas *repository.FichaTecnicaRepository,
	relatorios *repository.RelatorioRepository,
) *CatalogoService {
	return &CatalogoService{produtos: produtos, stock: stock, fichas: fichas, relatorios: relatorios}
}

func (s *CatalogoService) CatalogoPDV(ctx context.Context, armazemID int64) ([]ItemCatalogo, error) {
	produtos, err := s.produtos.ListarAtivos(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	unidades, err := s.produtos.ListarUnidadesDoNegocio(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	unidadesDe := make(map[int64][]domain.UnidadeProduto)
	for _, u := range unidades {
		unidadesDe[u.ProdutoID] = append(unidadesDe[u.ProdutoID], u)
	}

	stockItens, err := s.stock.ListarPorArmazem(ctx, armazemID)
	if err != nil {
		return nil, err
	}
	stock := make(map[int64]float64, len(stockItens))
	for _, it := range stockItens {
		stock[it.ProdutoID] = it.Quantidade
	}

	agora := time.Now()
	vendas, err := s.relatorios.VendasPorProduto(ctx, repository.PeriodoFiltro{
		NegocioID: domain.NegocioID,
		Desde:     agora.AddDate(0, 0, -30),
		Ate:       agora.Add(time.Minute),
	})
	if err != nil {
		return nil, err
	}
	vendido := make(map[int64]float64, len(vendas))
	for _, v := range vendas {
		vendido[v.ProdutoID] = v.QuantidadeTotal
	}

	out := make([]ItemCatalogo, 0, len(produtos))
	for _, p := range produtos {
		us := unidadesDe[p.ID]
		if len(us) == 0 {
			continue
		}
		padrao := us[0]
		for _, u := range us {
			if u.IsPadraoVenda {
				padrao = u
				break
			}
		}

		disponivel := stock[p.ID]
		if p.TipoProduto == domain.TipoProdutoComposto {
			disponivel, err = s.disponivelComposto(ctx, p.ID, stock)
			if err != nil {
				return nil, err
			}
		}
		out = append(out, ItemCatalogo{
			Produto:         p,
			Unidades:        us,
			UnidadePadraoID: padrao.ID,
			Disponivel:      disponivel,
			Vendido30Dias:   vendido[p.ID],
		})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Vendido30Dias != out[j].Vendido30Dias {
			return out[i].Vendido30Dias > out[j].Vendido30Dias
		}
		return strings.ToLower(out[i].Produto.Nome) < strings.ToLower(out[j].Produto.Nome)
	})
	return out, nil
}

// disponivelComposto é o ingrediente mais restritivo, arredondado para
// baixo; sem receita, nada se pode vender.
func (s *CatalogoService) disponivelComposto(ctx context.Context, produtoID int64, stock map[int64]float64) (float64, error) {
	ingredientes, err := s.fichas.ListarIngredientes(ctx, produtoID)
	if err != nil {
		return 0, err
	}
	if len(ingredientes) == 0 {
		return 0, nil
	}
	possivel := math.Inf(1)
	for _, ing := range ingredientes {
		if q := stock[ing.ProdutoIngredienteID] / ing.ConsumoReal(); q < possivel {
			possivel = q
		}
	}
	if math.IsInf(possivel, 1) || possivel < 0 {
		return 0, nil
	}
	return math.Floor(possivel), nil
}
