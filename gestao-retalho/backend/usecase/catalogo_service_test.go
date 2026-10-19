package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func TestCatalogoPDV_UnidadesStockCompostosEOrdem(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	stockRepo := repository.NewStockRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	produtos := usecase.NewProdutoService(produtoRepo)
	armazens := usecase.NewArmazemService(armazemRepo)
	stock := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), loteRepo)
	vendas := usecase.NewVendaService(repository.NewVendaRepository(db, stockRepo, loteRepo), produtoRepo, armazemRepo, fichaRepo, repository.NewClienteRepository(db))
	fichas := usecase.NewFichaTecnicaService(fichaRepo, produtoRepo, stockRepo, loteRepo)
	catalogo := usecase.NewCatalogoService(produtoRepo, stockRepo, fichaRepo, repository.NewRelatorioRepository(db))

	armazem := criarArmazem(t, ctx, armazens, "Loja")
	outro := criarArmazem(t, ctx, armazens, "Depósito")

	codigo := "5601234567890"
	parafuso, unidades, err := produtos.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: "Parafuso 4mm", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoVenda: 2, IsPadraoVenda: true, IsPadraoCompra: true},
			{NomeUnidade: "caixa 100", QtdEmUnidadeBase: 100, PrecoVenda: 180, CodigoBarras: &codigo},
		},
	})
	if err != nil {
		t.Fatalf("criar parafuso: %v", err)
	}
	martelo, uMartelo := criarProdutoComPreco(t, ctx, produtos, "Martelo", 450)
	velho, _ := criarProdutoComPreco(t, ctx, produtos, "Artigo descontinuado", 10)
	if err := produtos.Desativar(ctx, velho.ID); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	kit, _ := criarComposto(t, ctx, produtos, "Kit fixação", 300)
	semReceita, _ := criarComposto(t, ctx, produtos, "Kit sem receita", 100)
	if _, err := fichas.DefinirReceita(ctx, kit.ID, []usecase.IngredienteInput{{ProdutoIngredienteID: parafuso.ID, Quantidade: 50}}); err != nil {
		t.Fatalf("receita: %v", err)
	}

	for _, e := range []struct {
		id, arm int64
		q       float64
	}{{parafuso.ID, armazem.ID, 230}, {martelo.ID, armazem.ID, 5}, {martelo.ID, outro.ID, 40}} {
		if _, err := stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: e.id, ArmazemDestinoID: e.arm, Quantidade: e.q, CustoUnitario: 1, UtilizadorID: 1}); err != nil {
			t.Fatalf("entrada: %v", err)
		}
	}
	if _, err := vendas.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: martelo.ID, UnidadeProdutoID: uMartelo.ID, Quantidade: 2}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 900}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("vender: %v", err)
	}

	itens, err := catalogo.CatalogoPDV(ctx, armazem.ID)
	if err != nil {
		t.Fatalf("catálogo: %v", err)
	}
	porID := map[int64]usecase.ItemCatalogo{}
	for _, i := range itens {
		porID[i.Produto.ID] = i
	}
	if _, ok := porID[velho.ID]; ok {
		t.Error("produto inativo não pode aparecer no PDV")
	}
	if itens[0].Produto.ID != martelo.ID {
		t.Errorf("primeiro do catálogo = %q, want o Martelo (o único vendido)", itens[0].Produto.Nome)
	}
	p := porID[parafuso.ID]
	if len(p.Unidades) != 2 || p.UnidadePadraoID != unidades[0].ID || p.Disponivel != 230 {
		t.Errorf("parafuso = %+v, want 2 unidades, padrão a unidade e 230 em stock", p)
	}
	if m := porID[martelo.ID]; m.Disponivel != 3 || m.Vendido30Dias != 2 {
		t.Errorf("martelo = disponível %v vendido %v, want 3 (só deste armazém) e 2", m.Disponivel, m.Vendido30Dias)
	}
	if k := porID[kit.ID]; k.Disponivel != 4 { // 230 / 50 = 4,6
		t.Errorf("kit disponível = %v, want 4", k.Disponivel)
	}
	if k := porID[semReceita.ID]; k.Disponivel != 0 {
		t.Errorf("composto sem receita disponível = %v, want 0", k.Disponivel)
	}
}
