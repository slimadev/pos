package usecase_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

type servicosFichaLotes struct {
	produtos *usecase.ProdutoService
	armazens *usecase.ArmazemService
	stock    *usecase.StockService
	vendas   *usecase.VendaService
	fichas   *usecase.FichaTecnicaService
	farmacia *usecase.FarmaciaService
}

func newTestFichaLotes(t *testing.T) servicosFichaLotes {
	t.Helper()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	return servicosFichaLotes{
		produtos: usecase.NewProdutoService(produtoRepo),
		armazens: usecase.NewArmazemService(armazemRepo),
		stock:    usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), loteRepo),
		vendas: usecase.NewVendaService(repository.NewVendaRepository(db, stockRepo, loteRepo), produtoRepo, armazemRepo,
			fichaRepo, repository.NewClienteRepository(db)),
		fichas:   usecase.NewFichaTecnicaService(fichaRepo, produtoRepo, stockRepo, loteRepo),
		farmacia: usecase.NewFarmaciaService(loteRepo, produtoRepo, armazemRepo, repository.NewReceitaMedicaRepository(db)),
	}
}

// Um ingrediente com lotes (ex: um xarope com validade) tem de sair pelos
// lotes quando o composto é vendido ou o semi-elaborado é produzido.
func TestFichaTecnica_IngredienteComLotesSaiPelosLotes(t *testing.T) {
	ctx := context.Background()
	s := newTestFichaLotes(t)
	armazem := criarArmazem(t, ctx, s.armazens, "Bar")

	xarope, _ := criarProdutoLoteControlado(t, ctx, s.produtos, "Xarope de groselha", 0)
	entradaLote(t, ctx, s.farmacia, xarope.ID, armazem.ID, "VENCIDO", time.Now().AddDate(0, 0, -2), 5)
	entradaLote(t, ctx, s.farmacia, xarope.ID, armazem.ID, "BOM", time.Now().AddDate(0, 6, 0), 10)

	cocktail, unidade := criarComposto(t, ctx, s.produtos, "Cocktail", 250)
	if _, err := s.fichas.DefinirReceita(ctx, cocktail.ID, []usecase.IngredienteInput{{ProdutoIngredienteID: xarope.ID, Quantidade: 2}}); err != nil {
		t.Fatalf("receita do cocktail: %v", err)
	}
	if _, err := s.vendas.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: cocktail.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 250}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("vender cocktail: %v", err)
	}
	if q := quantidadeDoLote(t, ctx, s.farmacia, xarope.ID, armazem.ID, "BOM"); q != 8 {
		t.Errorf("lote BOM depois da venda = %v, want 8 (o vencido nunca é usado)", q)
	}
	if q := quantidadeDoLote(t, ctx, s.farmacia, xarope.ID, armazem.ID, "VENCIDO"); q != 5 {
		t.Errorf("lote VENCIDO = %v, want 5", q)
	}

	base, _, err := s.produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Base de cocktail", TipoProduto: domain.TipoProdutoSemiElaborado, UnidadeBase: "L"},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "L", QtdEmUnidadeBase: 1, PrecoVenda: 1, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar semi-elaborado: %v", err)
	}
	if _, err := s.fichas.DefinirReceita(ctx, base.ID, []usecase.IngredienteInput{{ProdutoIngredienteID: xarope.ID, Quantidade: 3}}); err != nil {
		t.Fatalf("receita da base: %v", err)
	}
	if _, err := s.fichas.RegistarProducao(ctx, usecase.ProducaoParams{ProdutoSemiElaboradoID: base.ID, ArmazemID: armazem.ID, Quantidade: 2, UtilizadorID: 1}); err != nil {
		t.Fatalf("produzir: %v", err)
	}
	if q := quantidadeDoLote(t, ctx, s.farmacia, xarope.ID, armazem.ID, "BOM"); q != 2 {
		t.Errorf("lote BOM depois da produção = %v, want 2 (8 - 2×3)", q)
	}
	stock, _ := s.stock.ObterStock(ctx, xarope.ID, armazem.ID)
	if stock.Quantidade != 7 {
		t.Errorf("stock_atual do xarope = %v, want 7 (igual à soma dos lotes: 5 vencidos + 2)", stock.Quantidade)
	}
}

func TestRegistarProducao_RecusaSemiElaboradoComLotes(t *testing.T) {
	ctx := context.Background()
	s := newTestFichaLotes(t)
	armazem := criarArmazem(t, ctx, s.armazens, "Cozinha")
	farinha := criarIngrediente(t, ctx, s.produtos, "Farinha", "kg", 50)
	if _, err := s.stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: farinha.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	massa, _, err := s.produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Massa fresca", TipoProduto: domain.TipoProdutoSemiElaborado, UnidadeBase: "kg", LoteControlado: true},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "kg", QtdEmUnidadeBase: 1, PrecoVenda: 1, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar massa: %v", err)
	}
	if _, err := s.fichas.DefinirReceita(ctx, massa.ID, []usecase.IngredienteInput{{ProdutoIngredienteID: farinha.ID, Quantidade: 1}}); err != nil {
		t.Fatalf("receita: %v", err)
	}
	_, err = s.fichas.RegistarProducao(ctx, usecase.ProducaoParams{ProdutoSemiElaboradoID: massa.ID, ArmazemID: armazem.ID, Quantidade: 2, UtilizadorID: 1})
	if err == nil || !strings.Contains(err.Error(), "controla lotes") {
		t.Fatalf("esperava recusa a mencionar lotes, obteve %v", err)
	}
	stock, _ := s.stock.ObterStock(ctx, farinha.ID, armazem.ID)
	if stock.Quantidade != 10 {
		t.Errorf("farinha = %v, want 10 (a produção recusada não pode consumir nada)", stock.Quantidade)
	}
}
