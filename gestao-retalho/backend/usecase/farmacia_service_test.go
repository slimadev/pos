package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestFarmaciaServices(t *testing.T) (*usecase.ProdutoService, *usecase.ArmazemService, *usecase.StockService, *usecase.VendaService, *usecase.FarmaciaService) {
	t.Helper()
	db := newTestDB(t)

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	vendaRepo := repository.NewVendaRepository(db, stockRepo, loteRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	receitaRepo := repository.NewReceitaMedicaRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo)
	vendaSvc := usecase.NewVendaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo)
	farmaciaSvc := usecase.NewFarmaciaService(loteRepo, produtoRepo, armazemRepo, receitaRepo)

	return produtoSvc, armazemSvc, stockSvc, vendaSvc, farmaciaSvc
}

func criarProdutoLoteControlado(t *testing.T, ctx context.Context, svc *usecase.ProdutoService, nome string, precoVenda float64) (domain.Produto, domain.UnidadeProduto) {
	t.Helper()
	p, unidades, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: nome, TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "caixa", LoteControlado: true},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "caixa", QtdEmUnidadeBase: 1, PrecoVenda: precoVenda, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto lote-controlado %q: %v", nome, err)
	}
	return p, unidades[0]
}

func TestRegistarEntradaComLote_ExigidaParaProdutoLoteControlado(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	paracetamol, _ := criarProdutoLoteControlado(t, ctx, produtoSvc, "Paracetamol 500mg", 55)

	// StockService.RegistarEntrada (o caminho genérico) tem de rejeitar —
	// deixaria stock_atual e lotes_produto divergentes.
	_, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: paracetamol.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 30, UtilizadorID: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "controla lotes") {
		t.Fatalf("esperava erro a mencionar 'controla lotes', obteve %v", err)
	}

	lote, movimento, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "L001",
		DataValidade: time.Now().AddDate(1, 0, 0), Quantidade: 10, CustoUnitario: 30, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("registar entrada com lote: %v", err)
	}
	if lote.Quantidade != 10 {
		t.Errorf("quantidade do lote = %v, want 10", lote.Quantidade)
	}
	if movimento.Tipo != domain.TipoMovimentoEntrada {
		t.Errorf("tipo do movimento = %v, want entrada", movimento.Tipo)
	}

	stock, err := stockSvc.ObterStock(ctx, paracetamol.ID, armazem.ID)
	if err != nil {
		t.Fatalf("obter stock: %v", err)
	}
	if stock.Quantidade != 10 {
		t.Errorf("stock_atual = %v, want 10 (a entrada com lote também grava o movimento normal)", stock.Quantidade)
	}
}

func TestRegistarEntradaComLote_RejeitaProdutoNaoLoteControlado(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, _, _, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	produto, _ := criarProdutoComPreco(t, ctx, produtoSvc, "Água 500ml", 60)

	_, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: produto.ID, ArmazemID: armazem.ID, NumeroLote: "L001",
		DataValidade: time.Now().AddDate(1, 0, 0), Quantidade: 10, CustoUnitario: 20, UtilizadorID: 1,
	})
	if err == nil || !strings.Contains(err.Error(), "não controla lotes") {
		t.Fatalf("esperava erro a mencionar 'não controla lotes', obteve %v", err)
	}
}

func TestVendaLoteControlado_ConsomePorFEFO(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	paracetamol, unidade := criarProdutoLoteControlado(t, ctx, produtoSvc, "Paracetamol 500mg", 55)

	// lote que vence mais cedo entra DEPOIS mas tem de ser consumido PRIMEIRO.
	if _, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "VENCE-TARDE",
		DataValidade: time.Now().AddDate(1, 0, 0), Quantidade: 20, CustoUnitario: 30, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada lote tardio: %v", err)
	}
	if _, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "VENCE-CEDO",
		DataValidade: time.Now().AddDate(0, 1, 0), Quantidade: 5, CustoUnitario: 25, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada lote cedo: %v", err)
	}

	resultado, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: paracetamol.ID, UnidadeProdutoID: unidade.ID, Quantidade: 3}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 165}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("vender: %v", err)
	}
	// custo do lote consumido (VENCE-CEDO, a 25) — não o mais recente/caro.
	if resultado.Itens[0].CustoUnitarioNoMomento != 25 {
		t.Errorf("custo unitário no momento = %v, want 25 (custo do lote FEFO, não do mais recente)", resultado.Itens[0].CustoUnitarioNoMomento)
	}

	lotes, err := farmaciaSvc.ListarLotes(ctx, paracetamol.ID, armazem.ID)
	if err != nil {
		t.Fatalf("listar lotes: %v", err)
	}
	var cedo, tarde domain.Lote
	for _, l := range lotes {
		switch l.NumeroLote {
		case "VENCE-CEDO":
			cedo = l
		case "VENCE-TARDE":
			tarde = l
		}
	}
	if cedo.Quantidade != 2 { // 5 - 3
		t.Errorf("lote VENCE-CEDO = %v, want 2", cedo.Quantidade)
	}
	if tarde.Quantidade != 20 {
		t.Errorf("lote VENCE-TARDE = %v, want continuar 20 (não devia ter sido tocado)", tarde.Quantidade)
	}

	stock, _ := stockSvc.ObterStock(ctx, paracetamol.ID, armazem.ID)
	if stock.Quantidade != 22 { // 25 - 3
		t.Errorf("stock_atual = %v, want 22", stock.Quantidade)
	}
}

func TestVendaLoteControlado_AtravessaVariosLotes(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	paracetamol, unidade := criarProdutoLoteControlado(t, ctx, produtoSvc, "Paracetamol 500mg", 55)

	if _, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "CEDO",
		DataValidade: time.Now().AddDate(0, 1, 0), Quantidade: 2, CustoUnitario: 25, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada lote cedo: %v", err)
	}
	if _, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "TARDE",
		DataValidade: time.Now().AddDate(1, 0, 0), Quantidade: 10, CustoUnitario: 30, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada lote tarde: %v", err)
	}

	// pede 5 — o lote CEDO só tem 2, tem de completar com 3 do lote TARDE.
	resultado, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: paracetamol.ID, UnidadeProdutoID: unidade.ID, Quantidade: 5}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 275}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("vender: %v", err)
	}
	// custo = (2*25 + 3*30)/5 = (50+90)/5 = 28
	if resultado.Itens[0].CustoUnitarioNoMomento != 28 {
		t.Errorf("custo unitário no momento = %v, want 28 (média dos dois lotes tocados)", resultado.Itens[0].CustoUnitarioNoMomento)
	}

	movimentos, err := stockSvc.ListarMovimentos(ctx, paracetamol.ID)
	if err != nil {
		t.Fatalf("listar movimentos: %v", err)
	}
	vendasCount := 0
	for _, m := range movimentos {
		if m.Tipo == domain.TipoMovimentoVenda {
			vendasCount++
		}
	}
	if vendasCount != 2 {
		t.Errorf("esperava 2 movimentos de venda (um por lote tocado), obteve %d", vendasCount)
	}

	lotes, _ := farmaciaSvc.ListarLotes(ctx, paracetamol.ID, armazem.ID)
	for _, l := range lotes {
		if l.NumeroLote == "CEDO" && l.Quantidade != 0 {
			t.Errorf("lote CEDO = %v, want 0 (esgotado)", l.Quantidade)
		}
		if l.NumeroLote == "TARDE" && l.Quantidade != 7 { // 10-3
			t.Errorf("lote TARDE = %v, want 7", l.Quantidade)
		}
	}
}

func TestVendaLoteControlado_BloqueiaQuandoLotesEsgotados(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, _, vendaSvc, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	paracetamol, unidade := criarProdutoLoteControlado(t, ctx, produtoSvc, "Paracetamol 500mg", 55)

	if _, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "L001",
		DataValidade: time.Now().AddDate(1, 0, 0), Quantidade: 2, CustoUnitario: 25, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	_, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: paracetamol.ID, UnidadeProdutoID: unidade.ID, Quantidade: 5}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 275}},
		UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrStockInsuficiente) {
		t.Fatalf("esperava ErrStockInsuficiente, obteve %v", err)
	}
}

func TestListarLotesAVencer_RespeitaJanelaDeDias(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, _, _, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	paracetamol, _ := criarProdutoLoteControlado(t, ctx, produtoSvc, "Paracetamol 500mg", 55)

	if _, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "VENCE-EM-5-DIAS",
		DataValidade: time.Now().AddDate(0, 0, 5), Quantidade: 10, CustoUnitario: 25, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	if _, _, err := farmaciaSvc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: paracetamol.ID, ArmazemID: armazem.ID, NumeroLote: "VENCE-EM-90-DIAS",
		DataValidade: time.Now().AddDate(0, 0, 90), Quantidade: 10, CustoUnitario: 25, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	aVencer, err := farmaciaSvc.ListarLotesAVencer(ctx, domain.NegocioID, 30)
	if err != nil {
		t.Fatalf("listar lotes a vencer: %v", err)
	}
	if len(aVencer) != 1 || aVencer[0].NumeroLote != "VENCE-EM-5-DIAS" {
		t.Fatalf("esperava só o lote a vencer em 5 dias dentro da janela de 30, obteve %+v", aVencer)
	}
}

func TestRegistarReceitaMedica(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Amoxicilina 500mg", 90)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	resultado, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 90}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("vender: %v", err)
	}

	receita, err := farmaciaSvc.RegistarReceitaMedica(ctx, usecase.ReceitaMedicaInput{
		VendaID: resultado.Venda.ID, ClienteID: domain.ClienteAnonimoID,
		Medico: "Dr. Sitoe", NumeroReceita: "REC-2026-001", ValidadoPor: 1,
	})
	if err != nil {
		t.Fatalf("registar receita médica: %v", err)
	}

	receitas, err := farmaciaSvc.ListarReceitasPorVenda(ctx, resultado.Venda.ID)
	if err != nil {
		t.Fatalf("listar receitas: %v", err)
	}
	if len(receitas) != 1 || receitas[0].ID != receita.ID {
		t.Fatalf("esperava encontrar a receita criada, obteve %+v", receitas)
	}
	if receitas[0].NumeroReceita != "REC-2026-001" {
		t.Errorf("número da receita = %q, want %q", receitas[0].NumeroReceita, "REC-2026-001")
	}
}
