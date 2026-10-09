package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// newTestServices devolve os três services de Fase 1 já ligados a uma base
// de teste (ver newTestDB em testutil_test.go).
func newTestServices(t *testing.T) (*usecase.ProdutoService, *usecase.ArmazemService, *usecase.StockService) {
	t.Helper()

	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)

	return usecase.NewProdutoService(produtoRepo),
		usecase.NewArmazemService(armazemRepo),
		usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo, repository.NewLoteRepository(db, stockRepo))
}

func criarProdutoSimples(t *testing.T, ctx context.Context, svc *usecase.ProdutoService, nome string) domain.Produto {
	t.Helper()
	p, _, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{NegocioID: 1, Nome: nome, TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoCusto: 10, PrecoVenda: 20, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	return p
}

func criarArmazem(t *testing.T, ctx context.Context, svc *usecase.ArmazemService, nome string) domain.Armazem {
	t.Helper()
	a, err := svc.Criar(ctx, domain.Armazem{NegocioID: 1, Nome: nome, Tipo: domain.TipoArmazemPrincipal})
	if err != nil {
		t.Fatalf("criar armazém: %v", err)
	}
	return a
}

func TestRegistarEntrada_AumentaStockEAtualizaCustoMedio(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	p := criarProdutoSimples(t, ctx, produtoSvc, "Cimento 50kg")
	a := criarArmazem(t, ctx, armazemSvc, "Armazém Principal")

	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: p.ID, ArmazemDestinoID: a.ID, Quantidade: 10, CustoUnitario: 500, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("primeira entrada: %v", err)
	}

	stock, err := stockSvc.ObterStock(ctx, p.ID, a.ID)
	if err != nil {
		t.Fatalf("obter stock: %v", err)
	}
	if stock.Quantidade != 10 {
		t.Errorf("quantidade = %v, want 10", stock.Quantidade)
	}
	if stock.CustoMedioPonderado != 500 {
		t.Errorf("custo médio = %v, want 500", stock.CustoMedioPonderado)
	}

	// segunda entrada a preço diferente exercita a média ponderada.
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: p.ID, ArmazemDestinoID: a.ID, Quantidade: 10, CustoUnitario: 700, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("segunda entrada: %v", err)
	}

	stock, _ = stockSvc.ObterStock(ctx, p.ID, a.ID)
	if stock.Quantidade != 20 {
		t.Errorf("quantidade = %v, want 20", stock.Quantidade)
	}
	const wantCusto = (10*500.0 + 10*700.0) / 20.0 // 600
	if stock.CustoMedioPonderado != wantCusto {
		t.Errorf("custo médio = %v, want %v", stock.CustoMedioPonderado, wantCusto)
	}
}

func TestTransferirStock_AtomicoEFalhaSemStock(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	p := criarProdutoSimples(t, ctx, produtoSvc, "Cimento 50kg")
	origem := criarArmazem(t, ctx, armazemSvc, "Armazém A")
	destino := criarArmazem(t, ctx, armazemSvc, "Armazém B")

	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: p.ID, ArmazemDestinoID: origem.ID, Quantidade: 5, CustoUnitario: 100, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada inicial: %v", err)
	}

	_, err := stockSvc.TransferirStock(ctx, usecase.TransferenciaParams{
		ProdutoID: p.ID, ArmazemOrigemID: origem.ID, ArmazemDestinoID: destino.ID, Quantidade: 10, UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrStockInsuficiente) {
		t.Fatalf("esperava ErrStockInsuficiente, obteve %v", err)
	}

	// a transferência falhada não pode ter mexido em nada (atomicidade).
	stockOrigem, _ := stockSvc.ObterStock(ctx, p.ID, origem.ID)
	if stockOrigem.Quantidade != 5 {
		t.Errorf("origem quantidade = %v, want 5 (transferência falhada não devia alterar stock)", stockOrigem.Quantidade)
	}

	if _, err := stockSvc.TransferirStock(ctx, usecase.TransferenciaParams{
		ProdutoID: p.ID, ArmazemOrigemID: origem.ID, ArmazemDestinoID: destino.ID, Quantidade: 3, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("transferência válida: %v", err)
	}

	stockOrigem, _ = stockSvc.ObterStock(ctx, p.ID, origem.ID)
	stockDestino, _ := stockSvc.ObterStock(ctx, p.ID, destino.ID)
	if stockOrigem.Quantidade != 2 {
		t.Errorf("origem = %v, want 2", stockOrigem.Quantidade)
	}
	if stockDestino.Quantidade != 3 {
		t.Errorf("destino = %v, want 3", stockDestino.Quantidade)
	}
	if stockDestino.CustoMedioPonderado != 100 {
		t.Errorf("destino custo médio = %v, want 100 (herdado da origem)", stockDestino.CustoMedioPonderado)
	}
}

func TestRegistarPerda_ExigeMotivo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	p := criarProdutoSimples(t, ctx, produtoSvc, "Cerveja 330ml")
	a := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: p.ID, ArmazemDestinoID: a.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada inicial: %v", err)
	}

	_, err := stockSvc.RegistarPerda(ctx, usecase.SaidaParams{
		ProdutoID: p.ID, ArmazemOrigemID: a.ID, Quantidade: 1, UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrMotivoObrigatorio) {
		t.Fatalf("esperava ErrMotivoObrigatorio, obteve %v", err)
	}

	if _, err := stockSvc.RegistarPerda(ctx, usecase.SaidaParams{
		ProdutoID: p.ID, ArmazemOrigemID: a.ID, Quantidade: 1, Motivo: "quebra", UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("perda com motivo: %v", err)
	}

	stock, _ := stockSvc.ObterStock(ctx, p.ID, a.ID)
	if stock.Quantidade != 9 {
		t.Errorf("quantidade = %v, want 9", stock.Quantidade)
	}
}

func TestInventario_GeraAjustesCorretosEBloqueiaFechoDuplo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	faltou := criarProdutoSimples(t, ctx, produtoSvc, "Arroz 5kg")
	sobrou := criarProdutoSimples(t, ctx, produtoSvc, "Óleo 1L")
	a := criarArmazem(t, ctx, armazemSvc, "Armazém Principal")

	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: faltou.ID, ArmazemDestinoID: a.ID, Quantidade: 10, CustoUnitario: 300, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada arroz: %v", err)
	}
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: sobrou.ID, ArmazemDestinoID: a.ID, Quantidade: 5, CustoUnitario: 150, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada óleo: %v", err)
	}

	inv, itens, err := stockSvc.IniciarInventario(ctx, 1, a.ID, 1)
	if err != nil {
		t.Fatalf("iniciar inventário: %v", err)
	}
	if len(itens) != 2 {
		t.Fatalf("esperava 2 itens no inventário, obteve %d", len(itens))
	}

	for _, item := range itens {
		var contado float64
		switch item.ProdutoID {
		case faltou.ID:
			contado = 8 // menos 2 do que o sistema esperava
		case sobrou.ID:
			contado = 6 // mais 1 do que o sistema esperava
		}
		if _, err := stockSvc.RegistarContagem(ctx, item.ID, contado); err != nil {
			t.Fatalf("registar contagem item %d: %v", item.ID, err)
		}
	}

	movimentos, err := stockSvc.FecharInventario(ctx, inv.ID, 1)
	if err != nil {
		t.Fatalf("fechar inventário: %v", err)
	}
	if len(movimentos) != 2 {
		t.Fatalf("esperava 2 movimentos de ajuste, obteve %d", len(movimentos))
	}

	for _, m := range movimentos {
		if m.Tipo != domain.TipoMovimentoAjusteInventario {
			t.Errorf("tipo = %v, want ajuste_inventario", m.Tipo)
		}
		switch m.ProdutoID {
		case faltou.ID:
			if m.QuantidadeUnidadeBase != 2 {
				t.Errorf("ajuste arroz = %v, want 2", m.QuantidadeUnidadeBase)
			}
			if m.ArmazemOrigemID == nil || *m.ArmazemOrigemID != a.ID || m.ArmazemDestinoID != nil {
				t.Errorf("ajuste negativo devia sair do armazém (só origem preenchida), got origem=%v destino=%v", m.ArmazemOrigemID, m.ArmazemDestinoID)
			}
		case sobrou.ID:
			if m.QuantidadeUnidadeBase != 1 {
				t.Errorf("ajuste óleo = %v, want 1", m.QuantidadeUnidadeBase)
			}
			if m.ArmazemDestinoID == nil || *m.ArmazemDestinoID != a.ID || m.ArmazemOrigemID != nil {
				t.Errorf("ajuste positivo devia entrar no armazém (só destino preenchido), got origem=%v destino=%v", m.ArmazemOrigemID, m.ArmazemDestinoID)
			}
		}
	}

	stockFaltou, _ := stockSvc.ObterStock(ctx, faltou.ID, a.ID)
	if stockFaltou.Quantidade != 8 {
		t.Errorf("stock final arroz = %v, want 8", stockFaltou.Quantidade)
	}
	stockSobrou, _ := stockSvc.ObterStock(ctx, sobrou.ID, a.ID)
	if stockSobrou.Quantidade != 6 {
		t.Errorf("stock final óleo = %v, want 6", stockSobrou.Quantidade)
	}

	if _, err := stockSvc.FecharInventario(ctx, inv.ID, 1); !errors.Is(err, domain.ErrInventarioFechado) {
		t.Fatalf("esperava ErrInventarioFechado ao fechar de novo, obteve %v", err)
	}
}

func TestRegistarEntrada_BloqueiaProdutoComposto(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc := newTestServices(t)
	a := criarArmazem(t, ctx, armazemSvc, "Armazém Principal")

	composto, _, err := produtoSvc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{NegocioID: 1, Nome: "Caipirinha de Vodka", TipoProduto: domain.TipoProdutoComposto, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoVenda: 450, IsPadraoVenda: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto composto: %v", err)
	}
	if composto.ControlaStock {
		t.Fatalf("produto composto não devia controlar stock")
	}

	_, err = stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: composto.ID, ArmazemDestinoID: a.ID, Quantidade: 1, CustoUnitario: 1, UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrProdutoComposto) {
		t.Fatalf("esperava ErrProdutoComposto, obteve %v", err)
	}
}
