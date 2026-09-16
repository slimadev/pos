package usecase_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestDividaServices(t *testing.T) (*usecase.ProdutoService, *usecase.ArmazemService, *usecase.StockService, *usecase.ClienteService, *usecase.ContaService, *usecase.DividaService, *sql.DB) {
	t.Helper()
	db := newTestDB(t)

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)
	clienteRepo := repository.NewClienteRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	vendaRepo := repository.NewVendaRepository(db, stockRepo, loteRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	dividaRepo := repository.NewDividaRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo)
	clienteSvc := usecase.NewClienteService(clienteRepo)
	dividaSvc := usecase.NewDividaService(dividaRepo, clienteRepo)
	contaSvc := usecase.NewContaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, dividaSvc)

	return produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, dividaSvc, db
}

// criarContaComDivida abre uma conta para o cliente dado, lança um item de
// valor `totalDesejado` e fecha-a sem nenhum pagamento — fica tudo em
// dívida. Devolve o resultado do fecho (para o vendaID/valor).
func criarContaComDivida(t *testing.T, ctx context.Context, produtoSvc *usecase.ProdutoService, armazemSvc *usecase.ArmazemService, stockSvc *usecase.StockService, contaSvc *usecase.ContaService, clienteID int64, totalDesejado float64) usecase.VendaCompleta {
	t.Helper()
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Whisky dose", totalDesejado)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: totalDesejado / 2, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	conta, err := contaSvc.AbrirConta(ctx, "Mesa dívida", &clienteID, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}

	resultado, _, err := contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{
		ArmazemID: armazem.ID, UtilizadorID: 1, // sem pagamentos -> tudo em dívida
	})
	if err != nil {
		t.Fatalf("fechar conta com dívida: %v", err)
	}
	return resultado
}

func TestFecharConta_AlertaLimiteCreditoNaoBloqueiaFecho(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, _, _ := newTestDividaServices(t)

	limite := 500.0
	cliente, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "Carlos Muianga", LimiteCredito: &limite})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}

	// dívida de 700, acima do limite de 500 — tem de FECHAR mesmo assim.
	_ = criarContaComDivida(t, ctx, produtoSvc, armazemSvc, stockSvc, contaSvc, cliente.ID, 700)

	// repete o fluxo manualmente para capturar o alerta desta vez.
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Whisky dose 2", 1)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão 2")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 1, CustoUnitario: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	conta, err := contaSvc.AbrirConta(ctx, "Mesa 2", &cliente.ID, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}
	_, alerta, err := contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{ArmazemID: armazem.ID, UtilizadorID: 1})
	if err != nil {
		t.Fatalf("fechar conta (esperava sucesso, alerta não bloqueia): %v", err)
	}
	if !alerta.Excedido {
		t.Errorf("esperava alerta de limite excedido, obteve %+v", alerta)
	}
	if alerta.LimiteCredito != limite {
		t.Errorf("limite no alerta = %v, want %v", alerta.LimiteCredito, limite)
	}
	if alerta.TotalEmAberto <= limite {
		t.Errorf("total em aberto (%v) devia ultrapassar o limite (%v)", alerta.TotalEmAberto, limite)
	}
}

func TestFecharConta_SemAlertaDentroDoLimite(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, _, _ := newTestDividaServices(t)

	limite := 5000.0
	cliente, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "Rita Chissano", LimiteCredito: &limite})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Whisky dose", 100)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 5, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	conta, err := contaSvc.AbrirConta(ctx, "Mesa", &cliente.ID, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}

	_, alerta, err := contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{ArmazemID: armazem.ID, UtilizadorID: 1})
	if err != nil {
		t.Fatalf("fechar conta: %v", err)
	}
	if alerta.Excedido {
		t.Errorf("não esperava alerta (dívida de 100 está bem dentro do limite de %v), obteve %+v", limite, alerta)
	}
}

func TestRegistarPagamentoDivida_ParcialDepoisTotal(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, dividaSvc, _ := newTestDividaServices(t)

	cliente, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "José Macuácua"})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}
	resultado := criarContaComDivida(t, ctx, produtoSvc, armazemSvc, stockSvc, contaSvc, cliente.ID, 1000)

	dividas, err := dividaSvc.ListarPorCliente(ctx, cliente.ID)
	if err != nil || len(dividas) != 1 {
		t.Fatalf("listar dívidas: %v (dívidas=%+v)", err, dividas)
	}
	divida := dividas[0]
	if divida.ValorEmAberto != 1000 {
		t.Fatalf("valor em aberto inicial = %v, want 1000", divida.ValorEmAberto)
	}
	if divida.VendaID != resultado.Venda.ID {
		t.Errorf("venda_id da dívida = %v, want %v", divida.VendaID, resultado.Venda.ID)
	}

	// pagamento parcial: fica em "parcial", não "paga".
	dividaParcial, pag1, err := dividaSvc.RegistarPagamento(ctx, usecase.RegistarPagamentoDividaParams{
		DividaID: divida.ID, Valor: 400, Metodo: domain.MetodoDinheiro, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("pagamento parcial: %v", err)
	}
	if dividaParcial.Status != domain.StatusDividaParcial {
		t.Errorf("status após pagamento parcial = %v, want parcial", dividaParcial.Status)
	}
	if dividaParcial.ValorEmAberto != 600 {
		t.Errorf("valor em aberto após parcial = %v, want 600", dividaParcial.ValorEmAberto)
	}
	if pag1.Valor != 400 {
		t.Errorf("valor do pagamento registado = %v, want 400", pag1.Valor)
	}

	// completa o pagamento — status muda para "paga", saldo chega a 0.
	dividaPaga, _, err := dividaSvc.RegistarPagamento(ctx, usecase.RegistarPagamentoDividaParams{
		DividaID: divida.ID, Valor: 600, Metodo: domain.MetodoMpesa, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("pagamento final: %v", err)
	}
	if dividaPaga.Status != domain.StatusDividaPaga {
		t.Errorf("status final = %v, want paga", dividaPaga.Status)
	}
	if dividaPaga.ValorEmAberto != 0 {
		t.Errorf("valor em aberto final = %v, want 0", dividaPaga.ValorEmAberto)
	}

	pagamentos, err := dividaSvc.ListarPagamentos(ctx, divida.ID)
	if err != nil {
		t.Fatalf("listar pagamentos: %v", err)
	}
	if len(pagamentos) != 2 {
		t.Fatalf("esperava 2 pagamentos registados (independentes da venda original), obteve %d", len(pagamentos))
	}

	// dívida já paga não aceita mais pagamentos.
	if _, _, err := dividaSvc.RegistarPagamento(ctx, usecase.RegistarPagamentoDividaParams{
		DividaID: divida.ID, Valor: 10, Metodo: domain.MetodoDinheiro, UtilizadorID: 1,
	}); err == nil {
		t.Fatalf("esperava erro ao pagar dívida já paga, obteve nil")
	}
}

func TestRegistarPagamentoDivida_RejeitaValorAcimaDoSaldo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, dividaSvc, _ := newTestDividaServices(t)

	cliente, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "Amélia Nhaca"})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}
	criarContaComDivida(t, ctx, produtoSvc, armazemSvc, stockSvc, contaSvc, cliente.ID, 300)

	dividas, _ := dividaSvc.ListarPorCliente(ctx, cliente.ID)
	divida := dividas[0]

	if _, _, err := dividaSvc.RegistarPagamento(ctx, usecase.RegistarPagamentoDividaParams{
		DividaID: divida.ID, Valor: 500, Metodo: domain.MetodoDinheiro, UtilizadorID: 1,
	}); err == nil {
		t.Fatalf("esperava erro (pagamento de 500 excede saldo de 300), obteve nil")
	}

	// nada deve ter mudado.
	dividaInalterada, err := dividaSvc.ListarPorCliente(ctx, cliente.ID)
	if err != nil || dividaInalterada[0].ValorEmAberto != 300 {
		t.Fatalf("valor em aberto = %+v, want continuar 300", dividaInalterada)
	}
}

func TestListarDividasAntigas_RespeitaJanelaDeDias(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, dividaSvc, db := newTestDividaServices(t)

	clienteAntigo, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "Cliente Antigo"})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}
	clienteRecente, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "Cliente Recente"})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}

	criarContaComDivida(t, ctx, produtoSvc, armazemSvc, stockSvc, contaSvc, clienteAntigo.ID, 200)
	criarContaComDivida(t, ctx, produtoSvc, armazemSvc, stockSvc, contaSvc, clienteRecente.ID, 200)

	// recua manualmente o criado_em da dívida do "cliente antigo" 45 dias —
	// não há forma de fazer isto pela API pública (e não devia haver).
	antigas40 := time.Now().AddDate(0, 0, -45).UTC().Format("2006-01-02T15:04:05.000Z")
	if _, err := db.Exec(
		`UPDATE dividas SET criado_em = ? WHERE cliente_id = ?`,
		antigas40, clienteAntigo.ID,
	); err != nil {
		t.Fatalf("recuar criado_em: %v", err)
	}

	antigas, err := dividaSvc.ListarAntigas(ctx, domain.NegocioID, 30)
	if err != nil {
		t.Fatalf("listar dívidas antigas: %v", err)
	}
	if len(antigas) != 1 || antigas[0].ClienteID != clienteAntigo.ID {
		t.Fatalf("esperava só a dívida do cliente antigo (>30 dias), obteve %+v", antigas)
	}
}
