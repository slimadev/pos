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
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo, loteRepo)
	vendaSvc := usecase.NewVendaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, repository.NewClienteRepository(db))
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

func entradaLote(t *testing.T, ctx context.Context, svc *usecase.FarmaciaService, produtoID, armazemID int64, numero string, validade time.Time, qtd float64) domain.Lote {
	t.Helper()
	lote, _, err := svc.RegistarEntradaComLote(ctx, usecase.EntradaComLoteParams{
		ProdutoID: produtoID, ArmazemID: armazemID, NumeroLote: numero,
		DataValidade: validade, Quantidade: qtd, CustoUnitario: 30, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("entrada lote %s: %v", numero, err)
	}
	return lote
}

func TestSaidasGenericas_RejeitamProdutoLoteControlado(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, farmaciaSvc := newTestFarmaciaServices(t)
	principal := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	deposito := criarArmazem(t, ctx, armazemSvc, "Depósito")
	p, _ := criarProdutoLoteControlado(t, ctx, produtoSvc, "Amoxicilina", 120)
	entradaLote(t, ctx, farmaciaSvc, p.ID, principal.ID, "A1", time.Now().AddDate(1, 0, 0), 10)

	saida := usecase.SaidaParams{ProdutoID: p.ID, ArmazemOrigemID: principal.ID, Quantidade: 1, Motivo: "x", UtilizadorID: 1}
	if _, err := stockSvc.RegistarPerda(ctx, saida); !errors.Is(err, domain.ErrLoteObrigatorio) {
		t.Errorf("perda genérica: esperava ErrLoteObrigatorio, obteve %v", err)
	}
	if _, err := stockSvc.RegistarConsumoInterno(ctx, saida); !errors.Is(err, domain.ErrLoteObrigatorio) {
		t.Errorf("consumo genérico: esperava ErrLoteObrigatorio, obteve %v", err)
	}
	_, err := stockSvc.TransferirStock(ctx, usecase.TransferenciaParams{
		ProdutoID: p.ID, ArmazemOrigemID: principal.ID, ArmazemDestinoID: deposito.ID, Quantidade: 1, UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrLoteObrigatorio) {
		t.Errorf("transferência genérica: esperava ErrLoteObrigatorio, obteve %v", err)
	}
}

func TestPerdaEConsumoDeLote_BaixamLoteEStock(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	p, _ := criarProdutoLoteControlado(t, ctx, produtoSvc, "Ibuprofeno", 80)
	lote := entradaLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "I1", time.Now().AddDate(1, 0, 0), 10)

	if _, err := farmaciaSvc.RegistarPerdaDeLote(ctx, usecase.SaidaLoteParams{LoteID: lote.ID, Quantidade: 3, Motivo: "partido", UtilizadorID: 1}); err != nil {
		t.Fatalf("perda de lote: %v", err)
	}
	if _, err := farmaciaSvc.RegistarConsumoInternoDeLote(ctx, usecase.SaidaLoteParams{LoteID: lote.ID, Quantidade: 2, Motivo: "uso", UtilizadorID: 1}); err != nil {
		t.Fatalf("consumo de lote: %v", err)
	}
	if _, err := farmaciaSvc.RegistarPerdaDeLote(ctx, usecase.SaidaLoteParams{LoteID: lote.ID, Quantidade: 6, Motivo: "x", UtilizadorID: 1}); !errors.Is(err, domain.ErrStockInsuficiente) {
		t.Errorf("esperava ErrStockInsuficiente ao tirar mais do que o lote tem, obteve %v", err)
	}

	lotes, _ := farmaciaSvc.ListarLotes(ctx, p.ID, armazem.ID)
	if len(lotes) != 1 || lotes[0].Quantidade != 5 {
		t.Fatalf("lotes = %+v, want I1 com 5", lotes)
	}
	stock, _ := stockSvc.ObterStock(ctx, p.ID, armazem.ID)
	if stock.Quantidade != 5 {
		t.Errorf("stock_atual = %v, want 5 (igual à soma dos lotes)", stock.Quantidade)
	}
}

func TestTransferirLote_MantemNumeroEValidadeNoDestino(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, farmaciaSvc := newTestFarmaciaServices(t)
	principal := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	deposito := criarArmazem(t, ctx, armazemSvc, "Depósito")
	p, _ := criarProdutoLoteControlado(t, ctx, produtoSvc, "Omeprazol", 90)
	validade := time.Now().AddDate(1, 0, 0).UTC().Truncate(time.Second)
	lote := entradaLote(t, ctx, farmaciaSvc, p.ID, principal.ID, "O1", validade, 10)

	for _, qtd := range []float64{4, 2} {
		if _, err := farmaciaSvc.TransferirLote(ctx, usecase.TransferenciaLoteParams{
			LoteID: lote.ID, ArmazemDestinoID: deposito.ID, Quantidade: qtd, UtilizadorID: 1,
		}); err != nil {
			t.Fatalf("transferir %v: %v", qtd, err)
		}
	}

	destino, _ := farmaciaSvc.ListarLotes(ctx, p.ID, deposito.ID)
	if len(destino) != 1 || destino[0].NumeroLote != "O1" || destino[0].Quantidade != 6 || !destino[0].DataValidade.Equal(validade) {
		t.Fatalf("lotes no destino = %+v, want um único O1 com 6 e a mesma validade", destino)
	}
	origem, _ := farmaciaSvc.ListarLotes(ctx, p.ID, principal.ID)
	if len(origem) != 1 || origem[0].Quantidade != 4 {
		t.Fatalf("lotes na origem = %+v, want O1 com 4", origem)
	}
	stockDestino, _ := stockSvc.ObterStock(ctx, p.ID, deposito.ID)
	stockOrigem, _ := stockSvc.ObterStock(ctx, p.ID, principal.ID)
	if stockDestino.Quantidade != 6 || stockOrigem.Quantidade != 4 {
		t.Errorf("stock_atual origem/destino = %v/%v, want 4/6", stockOrigem.Quantidade, stockDestino.Quantidade)
	}

	if _, err := farmaciaSvc.TransferirLote(ctx, usecase.TransferenciaLoteParams{
		LoteID: lote.ID, ArmazemDestinoID: principal.ID, Quantidade: 1, UtilizadorID: 1,
	}); !errors.Is(err, domain.ErrArmazemIgual) {
		t.Errorf("esperava ErrArmazemIgual, obteve %v", err)
	}
}

func venderLote(ctx context.Context, svc *usecase.VendaService, armazemID int64, unidade domain.UnidadeProduto, qtd float64) error {
	_, err := svc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazemID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: unidade.ProdutoID, UnidadeProdutoID: unidade.ID, Quantidade: qtd}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: qtd * unidade.PrecoVenda}},
		UtilizadorID: 1,
	})
	return err
}

func quantidadeDoLote(t *testing.T, ctx context.Context, svc *usecase.FarmaciaService, produtoID, armazemID int64, numero string) float64 {
	t.Helper()
	lotes, err := svc.ListarLotes(ctx, produtoID, armazemID)
	if err != nil {
		t.Fatalf("listar lotes: %v", err)
	}
	for _, l := range lotes {
		if l.NumeroLote == numero {
			return l.Quantidade
		}
	}
	return 0
}

func TestVendaLoteControlado_SaltaLotesVencidos(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, _, vendaSvc, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	p, unidade := criarProdutoLoteControlado(t, ctx, produtoSvc, "Xarope", 150)
	entradaLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "VENCIDO", time.Now().AddDate(0, 0, -1), 5)
	entradaLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "VALIDO", time.Now().AddDate(1, 0, 0), 5)

	if err := venderLote(ctx, vendaSvc, armazem.ID, unidade, 3); err != nil {
		t.Fatalf("vender: %v", err)
	}
	if q := quantidadeDoLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "VENCIDO"); q != 5 {
		t.Errorf("lote VENCIDO = %v, want 5 (nunca pode sair numa venda)", q)
	}
	if q := quantidadeDoLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "VALIDO"); q != 2 {
		t.Errorf("lote VALIDO = %v, want 2", q)
	}

	// Restam 2 válidos e 5 vencidos: pedir 4 tem de falhar e explicar porquê.
	err := venderLote(ctx, vendaSvc, armazem.ID, unidade, 4)
	if !errors.Is(err, domain.ErrStockInsuficiente) || !strings.Contains(err.Error(), "vencidos") {
		t.Fatalf("esperava stock insuficiente a mencionar lotes vencidos, obteve %v", err)
	}
}

func TestVendaLoteControlado_ValidadeDeHojeAindaVende(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, _, vendaSvc, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	p, unidade := criarProdutoLoteControlado(t, ctx, produtoSvc, "Soro", 40)
	hoje := time.Now()
	entradaLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "HOJE", time.Date(hoje.Year(), hoje.Month(), hoje.Day(), 0, 0, 0, 0, time.UTC), 2)

	if err := venderLote(ctx, vendaSvc, armazem.ID, unidade, 1); err != nil {
		t.Fatalf("lote que vence hoje ainda devia poder ser vendido: %v", err)
	}
}

func TestFecharInventario_FaltaEmProdutoComLotesBaixaPorFEFO(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	p, _ := criarProdutoLoteControlado(t, ctx, produtoSvc, "Vitamina C", 60)
	entradaLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "CEDO", time.Now().AddDate(0, 0, -3), 3)
	entradaLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "TARDE", time.Now().AddDate(1, 0, 0), 5)

	inv, itens, err := stockSvc.IniciarInventario(ctx, domain.NegocioID, armazem.ID, 1)
	if err != nil {
		t.Fatalf("iniciar inventário: %v", err)
	}
	for _, item := range itens {
		if item.ProdutoID == p.ID {
			if _, err := stockSvc.RegistarContagem(ctx, item.ID, 6); err != nil {
				t.Fatalf("contagem: %v", err)
			}
		}
	}
	if _, err := stockSvc.FecharInventario(ctx, inv.ID, 1); err != nil {
		t.Fatalf("fechar inventário: %v", err)
	}

	if q := quantidadeDoLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "CEDO"); q != 1 {
		t.Errorf("lote CEDO = %v, want 1 (a falta de 2 sai primeiro do que vence primeiro, mesmo vencido)", q)
	}
	if q := quantidadeDoLote(t, ctx, farmaciaSvc, p.ID, armazem.ID, "TARDE"); q != 5 {
		t.Errorf("lote TARDE = %v, want 5", q)
	}
	stock, _ := stockSvc.ObterStock(ctx, p.ID, armazem.ID)
	if stock.Quantidade != 6 {
		t.Errorf("stock_atual = %v, want 6 (igual à contagem e à soma dos lotes)", stock.Quantidade)
	}
}

func TestFecharInventario_SobraEmProdutoComLotesBloqueiaSemAjustarNada(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, farmaciaSvc := newTestFarmaciaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Farmácia")
	comLotes, _ := criarProdutoLoteControlado(t, ctx, produtoSvc, "Aspirina", 30)
	entradaLote(t, ctx, farmaciaSvc, comLotes.ID, armazem.ID, "A1", time.Now().AddDate(1, 0, 0), 5)
	semLotes := criarProdutoSimples(t, ctx, produtoSvc, "Algodão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: semLotes.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 5, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada sem lotes: %v", err)
	}

	inv, itens, err := stockSvc.IniciarInventario(ctx, domain.NegocioID, armazem.ID, 1)
	if err != nil {
		t.Fatalf("iniciar inventário: %v", err)
	}
	for _, item := range itens {
		contado := 7.0 // sobra de 2 em Aspirina
		if item.ProdutoID == semLotes.ID {
			contado = 8 // falta de 2 em Algodão — não pode ser aplicada se o fecho falhar
		}
		if _, err := stockSvc.RegistarContagem(ctx, item.ID, contado); err != nil {
			t.Fatalf("contagem: %v", err)
		}
	}

	_, err = stockSvc.FecharInventario(ctx, inv.ID, 1)
	if err == nil || !strings.Contains(err.Error(), "Entrada c/ lote") {
		t.Fatalf("esperava bloqueio a pedir Entrada c/ lote, obteve %v", err)
	}
	algodao, _ := stockSvc.ObterStock(ctx, semLotes.ID, armazem.ID)
	if algodao.Quantidade != 10 {
		t.Errorf("Algodão = %v, want 10 (nenhum ajuste pode ter sido aplicado)", algodao.Quantidade)
	}
	if q := quantidadeDoLote(t, ctx, farmaciaSvc, comLotes.ID, armazem.ID, "A1"); q != 5 {
		t.Errorf("lote A1 = %v, want 5", q)
	}
}
