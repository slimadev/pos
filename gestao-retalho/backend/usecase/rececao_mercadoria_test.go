package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestRegistarRececao_ConverteUnidadeDeCompraEAtualizaCustoMedio(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	a := criarArmazem(t, ctx, armazemSvc, "Principal")

	parafuso, unidades, err := produtoSvc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: "Parafuso 4x40", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoVenda: 3, IsPadraoVenda: true},
			{NomeUnidade: "caixa 100", QtdEmUnidadeBase: 100, PrecoCusto: 200, PrecoVenda: 250, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar parafuso: %v", err)
	}
	caixa := unidades[1]
	cimento := criarProdutoSimples(t, ctx, produtoSvc, "Cimento 50kg")
	cimentoUnidades, _ := produtoSvc.ListarUnidades(ctx, cimento.ID)

	// Já havia 100 parafusos a 1,00 MT cada.
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: parafuso.ID, ArmazemDestinoID: a.ID, Quantidade: 100, CustoUnitario: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada inicial: %v", err)
	}

	movimentos, err := stockSvc.RegistarRececao(ctx, usecase.RececaoParams{
		ArmazemID:  a.ID,
		Referencia: "Fatura 123",
		Itens: []usecase.ItemRececao{
			{ProdutoID: parafuso.ID, UnidadeProdutoID: caixa.ID, Quantidade: 3, CustoUnitario: 200},
			{ProdutoID: cimento.ID, UnidadeProdutoID: cimentoUnidades[0].ID, Quantidade: 10, CustoUnitario: 500},
		},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("receção: %v", err)
	}
	if len(movimentos) != 2 {
		t.Fatalf("movimentos = %d, want 2", len(movimentos))
	}
	if movimentos[0].Motivo == nil || *movimentos[0].Motivo != "Receção de mercadoria: Fatura 123" {
		t.Errorf("motivo = %v, want com a referência da fatura", movimentos[0].Motivo)
	}

	// 3 caixas de 100 = 300 unidades a 2,00 MT; com as 100 a 1,00 MT a
	// média fica (100×1 + 300×2) / 400 = 1,75.
	stock, _ := stockSvc.ObterStock(ctx, parafuso.ID, a.ID)
	if stock.Quantidade != 400 || stock.CustoMedioPonderado != 1.75 {
		t.Errorf("parafuso = %v a %v, want 400 a 1,75", stock.Quantidade, stock.CustoMedioPonderado)
	}
	stock, _ = stockSvc.ObterStock(ctx, cimento.ID, a.ID)
	if stock.Quantidade != 10 || stock.CustoMedioPonderado != 500 {
		t.Errorf("cimento = %v a %v, want 10 a 500", stock.Quantidade, stock.CustoMedioPonderado)
	}
}

func TestRegistarRececao_LinhaInvalidaNaoDeixaEntrarNenhuma(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	a := criarArmazem(t, ctx, armazemSvc, "Principal")
	cimento := criarProdutoSimples(t, ctx, produtoSvc, "Cimento 50kg")
	cimentoUnidades, _ := produtoSvc.ListarUnidades(ctx, cimento.ID)
	areia := criarProdutoSimples(t, ctx, produtoSvc, "Areia")

	// A unidade do cimento não pertence à areia.
	_, err := stockSvc.RegistarRececao(ctx, usecase.RececaoParams{
		ArmazemID: a.ID,
		Itens: []usecase.ItemRececao{
			{ProdutoID: cimento.ID, UnidadeProdutoID: cimentoUnidades[0].ID, Quantidade: 10, CustoUnitario: 500},
			{ProdutoID: areia.ID, UnidadeProdutoID: cimentoUnidades[0].ID, Quantidade: 5, CustoUnitario: 100},
		},
		UtilizadorID: 1,
	})
	if err == nil {
		t.Fatal("esperava erro com unidade de outro produto")
	}
	stock, _ := stockSvc.ObterStock(ctx, cimento.ID, a.ID)
	if stock.Quantidade != 0 {
		t.Errorf("cimento = %v, want 0 (a receção falhou toda)", stock.Quantidade)
	}

	if _, err := stockSvc.RegistarRececao(ctx, usecase.RececaoParams{ArmazemID: a.ID, UtilizadorID: 1}); err == nil {
		t.Error("esperava erro sem nenhuma linha")
	}
}

func TestRegistarRececao_RecusaProdutoComLotes(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	a := criarArmazem(t, ctx, armazemSvc, "Principal")
	p, unidades, err := produtoSvc.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Paracetamol", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "comprimido", LoteControlado: true},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "caixa 20", QtdEmUnidadeBase: 20, PrecoVenda: 100, IsPadraoVenda: true}},
	})
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	if _, err := stockSvc.RegistarRececao(ctx, usecase.RececaoParams{
		ArmazemID:    a.ID,
		Itens:        []usecase.ItemRececao{{ProdutoID: p.ID, UnidadeProdutoID: unidades[0].ID, Quantidade: 2, CustoUnitario: 60}},
		UtilizadorID: 1,
	}); err == nil {
		t.Fatal("esperava erro para produto com lotes")
	}
}
