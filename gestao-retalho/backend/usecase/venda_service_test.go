package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestVendaServices(t *testing.T) (*usecase.ProdutoService, *usecase.ArmazemService, *usecase.StockService, *usecase.VendaService) {
	t.Helper()
	db := newTestDB(t)

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	vendaRepo := repository.NewVendaRepository(db, stockRepo, loteRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo)
	vendaSvc := usecase.NewVendaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo)

	return produtoSvc, armazemSvc, stockSvc, vendaSvc
}

// criarProdutoComPreco cria um produto simples com uma única unidade de
// venda/compra a um preço/custo conhecido, devolvendo produto e unidade.
func criarProdutoComPreco(t *testing.T, ctx context.Context, svc *usecase.ProdutoService, nome string, precoVenda float64) (domain.Produto, domain.UnidadeProduto) {
	t.Helper()
	p, unidades, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: nome, TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoCusto: precoVenda / 2, PrecoVenda: precoVenda, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	return p, unidades[0]
}

func TestRegistarVendaDireta_CalculaSubtotalDescontoTotal(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc := newTestVendaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Cimento 50kg", 620)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 20, CustoUnitario: 480, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada inicial: %v", err)
	}

	// 3 sacos a 620 MT, 50 MT de desconto na linha, mais 20 MT de desconto
	// global -> subtotal 1860, desconto 70, total 1790.
	resultado, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID: armazem.ID,
		Itens: []usecase.ItemVendaDiretaParams{
			{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 3, Desconto: 50},
		},
		DescontoTotal: 20,
		Pagamentos:    []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 1790}},
		UtilizadorID:  1,
	})
	if err != nil {
		t.Fatalf("registar venda: %v", err)
	}

	if resultado.Venda.Subtotal != 1860 {
		t.Errorf("subtotal = %v, want 1860", resultado.Venda.Subtotal)
	}
	if resultado.Venda.DescontoTotal != 70 {
		t.Errorf("desconto total = %v, want 70", resultado.Venda.DescontoTotal)
	}
	if resultado.Venda.Total != 1790 {
		t.Errorf("total = %v, want 1790", resultado.Venda.Total)
	}
	if resultado.Venda.Status != domain.StatusVendaPaga {
		t.Errorf("status = %v, want paga", resultado.Venda.Status)
	}
	if resultado.Venda.ClienteID != domain.ClienteAnonimoID {
		t.Errorf("cliente = %v, want anónimo (%d)", resultado.Venda.ClienteID, domain.ClienteAnonimoID)
	}

	if len(resultado.Itens) != 1 || resultado.Itens[0].Subtotal != 1810 { // 3*620 - 50
		t.Errorf("item subtotal = %+v, want 1810", resultado.Itens)
	}

	stockFinal, err := stockSvc.ObterStock(ctx, produto.ID, armazem.ID)
	if err != nil {
		t.Fatalf("obter stock final: %v", err)
	}
	if stockFinal.Quantidade != 17 {
		t.Errorf("stock final = %v, want 17 (20-3)", stockFinal.Quantidade)
	}
}

func TestRegistarVendaDireta_CustoUnitarioNoMomentoCorreto(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc := newTestVendaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Lâmpada LED 9W", 130)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 90, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	resultado, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 2}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoCartao, Valor: 260}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("registar venda: %v", err)
	}

	if resultado.Itens[0].CustoUnitarioNoMomento != 90 {
		t.Errorf("custo unitário no momento = %v, want 90", resultado.Itens[0].CustoUnitarioNoMomento)
	}

	// mudar o custo médio (nova entrada mais cara) não pode alterar o que
	// já foi gravado na venda anterior — lucro real fica correto (SDD 3.2).
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 150, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("segunda entrada: %v", err)
	}
	completa, err := vendaSvc.ObterVendaCompleta(ctx, resultado.Venda.ID)
	if err != nil {
		t.Fatalf("obter venda completa: %v", err)
	}
	if completa.Itens[0].CustoUnitarioNoMomento != 90 {
		t.Errorf("custo unitário no momento mudou depois da venda: %v, want continuar 90", completa.Itens[0].CustoUnitarioNoMomento)
	}
}

func TestRegistarVendaDireta_BloqueiaSemStockSuficiente(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc := newTestVendaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Tubo PVC 50mm", 410)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 3, CustoUnitario: 300, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	_, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 5}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 2050}},
		UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrStockInsuficiente) {
		t.Fatalf("esperava ErrStockInsuficiente, obteve %v", err)
	}

	// nada deve ter mudado — nem stock, nem venda criada.
	stockFinal, _ := stockSvc.ObterStock(ctx, produto.ID, armazem.ID)
	if stockFinal.Quantidade != 3 {
		t.Errorf("stock = %v, want continuar 3 (venda bloqueada não deve mexer em nada)", stockFinal.Quantidade)
	}
}

func TestRegistarVendaDireta_BloqueiaPagamentoIncompleto(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc := newTestVendaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Disjuntor 16A", 340)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 200, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	_, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID: armazem.ID,
		Itens:     []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 2}},
		// total é 680, só paga 500 -> deve ser bloqueado.
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 500}},
		UtilizadorID: 1,
	})
	if err == nil {
		t.Fatalf("esperava erro de pagamento incompleto, obteve nil")
	}

	stockFinal, _ := stockSvc.ObterStock(ctx, produto.ID, armazem.ID)
	if stockFinal.Quantidade != 10 {
		t.Errorf("stock = %v, want continuar 10", stockFinal.Quantidade)
	}
}

// TestRegistarVendaDireta_BloqueiaProdutoCompostoSemFichaTecnica cobre o
// caso em que ainda não Fase 5 completa: um composto sem receita definida
// não pode ser vendido (nada para expandir). A venda de um composto COM
// receita está coberta em ficha_tecnica_service_test.go.
func TestRegistarVendaDireta_BloqueiaProdutoCompostoSemFichaTecnica(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, _, vendaSvc := newTestVendaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")

	composto, unidades, err := produtoSvc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: "Caipirinha de Vodka", TipoProduto: domain.TipoProdutoComposto, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoVenda: 450, IsPadraoVenda: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto composto: %v", err)
	}

	_, err = vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: composto.ID, UnidadeProdutoID: unidades[0].ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 450}},
		UtilizadorID: 1,
	})
	if err == nil {
		t.Fatalf("esperava erro (composto sem ficha técnica definida), obteve nil")
	}
}
