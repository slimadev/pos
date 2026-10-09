package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestClientesService_SaldoLimiteEPagamento(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewClientesService(repo)

	loja := novaLojaTeste(t, "loja-clientes-admin.db", "Bar Central", domain.TipoNegocioBottlestore)
	if _, err := repo.Criar(ctx, "Bar Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}
	limite := 500.0
	cliente, err := svc.Criar(ctx, domain.Cliente{Nome: "Joana Sitoe", LimiteCredito: &limite})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}

	produto, unidades, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Cerveja", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "garrafa"},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "garrafa", QtdEmUnidadeBase: 1, PrecoCusto: 40, PrecoVenda: 600, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 10, CustoUnitario: 40, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	conta, err := loja.Contas.AbrirConta(ctx, "Mesa 3", &cliente.ID, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := loja.Contas.AdicionarItem(ctx, usecase.AdicionarItemContaParams{ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidades[0].ID, Quantidade: 1, UtilizadorID: 1}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}
	if _, _, err := loja.Contas.FecharConta(ctx, conta.ID, usecase.FecharContaParams{ArmazemID: loja.Armazem.ID, UtilizadorID: 1}); err != nil {
		t.Fatalf("fechar conta em dívida: %v", err)
	}

	lista, err := svc.Listar(ctx)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	if len(lista) != 1 {
		t.Fatalf("lista = %d clientes, want só a Joana (o anónimo fica de fora)", len(lista))
	}
	if lista[0].TotalEmAberto != 600 || lista[0].DividasAbertas != 1 || !lista[0].LimiteExcedido {
		t.Fatalf("resumo = %+v, want 600 em aberto, 1 dívida, limite excedido", lista[0])
	}

	ficha, err := svc.ObterFicha(ctx, cliente.ID)
	if err != nil || len(ficha.Dividas) != 1 || len(ficha.Vendas) != 1 {
		t.Fatalf("ficha = %+v, err %v; want 1 dívida e 1 venda", ficha, err)
	}
	if _, err := svc.RegistarPagamentoDivida(ctx, usecase.RegistarPagamentoDividaParams{DividaID: ficha.Dividas[0].ID, Valor: 250, Metodo: domain.MetodoMpesa}); err != nil {
		t.Fatalf("pagar: %v", err)
	}
	lista, _ = svc.Listar(ctx)
	if lista[0].TotalEmAberto != 350 || lista[0].LimiteExcedido {
		t.Fatalf("depois de pagar 250: %+v, want 350 em aberto e dentro do limite", lista[0])
	}
	pagamentos, err := svc.ListarPagamentosDivida(ctx, ficha.Dividas[0].ID)
	if err != nil || len(pagamentos) != 1 {
		t.Fatalf("pagamentos = %d, err %v; want 1", len(pagamentos), err)
	}
}
