package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

type servicosCredito struct {
	produtos *usecase.ProdutoService
	armazens *usecase.ArmazemService
	stock    *usecase.StockService
	vendas   *usecase.VendaService
	clientes *usecase.ClienteService
	dividas  *usecase.DividaService
}

func newTestCreditoServices(t *testing.T) servicosCredito {
	t.Helper()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	clienteRepo := repository.NewClienteRepository(db)
	return servicosCredito{
		produtos: usecase.NewProdutoService(produtoRepo),
		armazens: usecase.NewArmazemService(armazemRepo),
		stock:    usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), loteRepo),
		vendas: usecase.NewVendaService(repository.NewVendaRepository(db, stockRepo, loteRepo), produtoRepo, armazemRepo,
			repository.NewFichaTecnicaRepository(db), clienteRepo),
		clientes: usecase.NewClienteService(clienteRepo),
		dividas:  usecase.NewDividaService(repository.NewDividaRepository(db), clienteRepo),
	}
}

// prepararCredito cria um armazém com 10 sacos de cimento a 500 e uma cliente.
func prepararCredito(t *testing.T, ctx context.Context, s servicosCredito) (domain.Armazem, domain.UnidadeProduto, domain.Cliente) {
	t.Helper()
	armazem := criarArmazem(t, ctx, s.armazens, "Loja")
	produto, unidade := criarProdutoComPreco(t, ctx, s.produtos, "Cimento", 500)
	if _, err := s.stock.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 350, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	cliente, err := s.clientes.Criar(ctx, domain.Cliente{Nome: "Construções Muianga"})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}
	return armazem, unidade, cliente
}

func TestVendaACredito_SemPagamentoFicaTudoEmDivida(t *testing.T) {
	ctx := context.Background()
	s := newTestCreditoServices(t)
	armazem, unidade, cliente := prepararCredito(t, ctx, s)

	venda, err := s.vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		ClienteID:    &cliente.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: unidade.ProdutoID, UnidadeProdutoID: unidade.ID, Quantidade: 4}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("venda a crédito: %v", err)
	}
	if venda.Venda.Status != domain.StatusVendaDivida || venda.Venda.Total != 2000 {
		t.Fatalf("venda = %+v, want 2000 em dívida", venda.Venda)
	}
	dividas, _ := s.dividas.ListarPorCliente(ctx, cliente.ID)
	if len(dividas) != 1 || dividas[0].ValorEmAberto != 2000 || dividas[0].Status != domain.StatusDividaAberta {
		t.Fatalf("dívidas = %+v, want uma de 2000 em aberto", dividas)
	}
	stock, _ := s.stock.ObterStock(ctx, unidade.ProdutoID, armazem.ID)
	if stock.Quantidade != 6 {
		t.Errorf("stock = %v, want 6 (o cliente levou 4)", stock.Quantidade)
	}
}

func TestVendaACredito_EntradaParcialFicaParcial(t *testing.T) {
	ctx := context.Background()
	s := newTestCreditoServices(t)
	armazem, unidade, cliente := prepararCredito(t, ctx, s)

	venda, err := s.vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		ClienteID:    &cliente.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: unidade.ProdutoID, UnidadeProdutoID: unidade.ID, Quantidade: 2}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoMpesa, Valor: 300}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("venda a crédito: %v", err)
	}
	if venda.Venda.Status != domain.StatusVendaParcial || len(venda.Pagamentos) != 1 {
		t.Fatalf("venda = %+v, want parcial com 1 pagamento", venda)
	}
	dividas, _ := s.dividas.ListarPorCliente(ctx, cliente.ID)
	if len(dividas) != 1 || dividas[0].ValorPago != 300 || dividas[0].ValorEmAberto != 700 || dividas[0].Status != domain.StatusDividaParcial {
		t.Fatalf("dívidas = %+v, want 300 pago e 700 em aberto", dividas)
	}

	if _, err := s.vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		ClienteID:    &cliente.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: unidade.ProdutoID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 600}},
		UtilizadorID: 1,
	}); err == nil {
		t.Fatal("esperava erro: entrada (600) maior que o total (500)")
	}
}

func TestVendaACredito_RecusaClienteAnonimoOuInativo(t *testing.T) {
	ctx := context.Background()
	s := newTestCreditoServices(t)
	armazem, unidade, cliente := prepararCredito(t, ctx, s)
	itens := []usecase.ItemVendaDiretaParams{{ProdutoID: unidade.ProdutoID, UnidadeProdutoID: unidade.ID, Quantidade: 1}}

	if _, err := s.vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{ArmazemID: armazem.ID, Itens: itens, UtilizadorID: 1}); !errors.Is(err, domain.ErrClienteAnonimoNaoPodeDivida) {
		t.Errorf("sem cliente: esperava ErrClienteAnonimoNaoPodeDivida, obteve %v", err)
	}
	anonimo := domain.ClienteAnonimoID
	if _, err := s.vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{ArmazemID: armazem.ID, ClienteID: &anonimo, Itens: itens, UtilizadorID: 1}); !errors.Is(err, domain.ErrClienteAnonimoNaoPodeDivida) {
		t.Errorf("anónimo: esperava ErrClienteAnonimoNaoPodeDivida, obteve %v", err)
	}

	if err := s.clientes.Desativar(ctx, cliente.ID); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	if _, err := s.vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{ArmazemID: armazem.ID, ClienteID: &cliente.ID, Itens: itens, UtilizadorID: 1}); err == nil {
		t.Error("esperava erro ao vender a crédito a um cliente inativo")
	}
	stock, _ := s.stock.ObterStock(ctx, unidade.ProdutoID, armazem.ID)
	if stock.Quantidade != 10 {
		t.Errorf("stock = %v, want 10 (nenhuma venda recusada pode mexer no stock)", stock.Quantidade)
	}
}
