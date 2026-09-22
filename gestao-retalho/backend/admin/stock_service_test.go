package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestStockService_ListarPorArmazemRejeitaSemLojaConfigurada(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewStockService(repo)

	_, err := svc.ListarPorArmazem(ctx, 1)
	if err == nil {
		t.Fatal("esperava erro sem nenhuma loja registada")
	}
}

func TestStockService_EntradaTransferenciaPerdaEConsumo(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewStockService(repo)

	loja := novaLojaTeste(t, "loja-stock-admin.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	produto, _, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: "Cimento 50kg", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "saco"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "saco", QtdEmUnidadeBase: 1, PrecoCusto: 400, PrecoVenda: 620, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}

	deposito, err := loja.Armazens.Criar(ctx, domain.Armazem{Nome: "Depósito", Tipo: domain.TipoArmazemDeposito})
	if err != nil {
		t.Fatalf("criar segundo armazém: %v", err)
	}

	// Entrada no armazém principal.
	if _, err := svc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 50, CustoUnitario: 400,
	}); err != nil {
		t.Fatalf("registar entrada: %v", err)
	}

	stockPrincipal, err := svc.ListarPorArmazem(ctx, loja.Armazem.ID)
	if err != nil {
		t.Fatalf("listar stock: %v", err)
	}
	if len(stockPrincipal) != 1 || stockPrincipal[0].Quantidade != 50 {
		t.Fatalf("stock principal = %+v, want 1 item com 50", stockPrincipal)
	}

	// Transferência para o depósito.
	if _, err := svc.Transferir(ctx, usecase.TransferenciaParams{
		ProdutoID: produto.ID, ArmazemOrigemID: loja.Armazem.ID, ArmazemDestinoID: deposito.ID, Quantidade: 20,
	}); err != nil {
		t.Fatalf("transferir: %v", err)
	}

	stockDeposito, err := svc.ListarPorArmazem(ctx, deposito.ID)
	if err != nil {
		t.Fatalf("listar stock depósito: %v", err)
	}
	if len(stockDeposito) != 1 || stockDeposito[0].Quantidade != 20 {
		t.Fatalf("stock depósito = %+v, want 1 item com 20", stockDeposito)
	}

	// Perda no armazém principal (restam 30, tira 5).
	if _, err := svc.RegistarPerda(ctx, usecase.SaidaParams{
		ProdutoID: produto.ID, ArmazemOrigemID: loja.Armazem.ID, Quantidade: 5, Motivo: "quebra",
	}); err != nil {
		t.Fatalf("registar perda: %v", err)
	}

	// Consumo interno (restam 25, tira 2).
	if _, err := svc.RegistarConsumoInterno(ctx, usecase.SaidaParams{
		ProdutoID: produto.ID, ArmazemOrigemID: loja.Armazem.ID, Quantidade: 2, Motivo: "uso interno",
	}); err != nil {
		t.Fatalf("registar consumo interno: %v", err)
	}

	stockFinal, err := svc.ListarPorArmazem(ctx, loja.Armazem.ID)
	if err != nil {
		t.Fatalf("listar stock final: %v", err)
	}
	if len(stockFinal) != 1 || stockFinal[0].Quantidade != 23 {
		t.Fatalf("stock final = %+v, want 1 item com 23 (50-20-5-2)", stockFinal)
	}

	movimentos, err := svc.ListarMovimentos(ctx, produto.ID)
	if err != nil {
		t.Fatalf("listar movimentos: %v", err)
	}
	if len(movimentos) != 4 {
		t.Fatalf("movimentos = %d, want 4 (entrada, transferência, perda, consumo)", len(movimentos))
	}
}
