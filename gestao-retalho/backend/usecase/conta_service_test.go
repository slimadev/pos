package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestContaServices(t *testing.T) (*usecase.ProdutoService, *usecase.ArmazemService, *usecase.StockService, *usecase.ClienteService, *usecase.ContaService, *repository.VendaRepository) {
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
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo, loteRepo)
	clienteSvc := usecase.NewClienteService(clienteRepo)
	dividaSvc := usecase.NewDividaService(dividaRepo, clienteRepo)
	contaSvc := usecase.NewContaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, dividaSvc)

	return produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, vendaRepo
}

func TestAdicionarItem_NaoMexeNoStockAntesDoFecho(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, contaSvc, _ := newTestContaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "2M 550ml", 180)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 20, CustoUnitario: 130, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	conta, err := contaSvc.AbrirConta(ctx, "Mesa 3", nil, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 5, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}

	// SDD 4.10: stock só desce ao FECHAR a conta, não ao lançar consumo.
	stock, err := stockSvc.ObterStock(ctx, produto.ID, armazem.ID)
	if err != nil {
		t.Fatalf("obter stock: %v", err)
	}
	if stock.Quantidade != 20 {
		t.Errorf("stock = %v, want continuar 20 (item lançado mas conta ainda aberta)", stock.Quantidade)
	}
}

func TestFecharConta_NaoFechaDuasVezes(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, contaSvc, _ := newTestContaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Coca-Cola 500ml", 120)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 70, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	conta, _ := contaSvc.AbrirConta(ctx, "Balcão 2", nil, 1)
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}

	fecharParams := usecase.FecharContaParams{
		ArmazemID:    armazem.ID,
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 120}},
		UtilizadorID: 1,
	}
	if _, _, err := contaSvc.FecharConta(ctx, conta.ID, fecharParams); err != nil {
		t.Fatalf("primeiro fecho: %v", err)
	}

	if _, _, err := contaSvc.FecharConta(ctx, conta.ID, fecharParams); !errors.Is(err, domain.ErrContaNaoAberta) {
		t.Fatalf("esperava ErrContaNaoAberta ao fechar de novo, obteve %v", err)
	}
}

func TestTransferirItem_MantemTotalConsistenteEntreContas(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, contaSvc, _ := newTestContaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Cerveja 2M", 180)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 50, CustoUnitario: 130, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	contaA, err := contaSvc.AbrirConta(ctx, "Mesa 3", nil, 1)
	if err != nil {
		t.Fatalf("abrir conta A: %v", err)
	}
	contaB, err := contaSvc.AbrirConta(ctx, "Mesa 7", nil, 1)
	if err != nil {
		t.Fatalf("abrir conta B: %v", err)
	}

	item, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: contaA.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 4, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("adicionar item: %v", err)
	}

	if err := contaSvc.TransferirItem(ctx, item.ID, contaB.ID); err != nil {
		t.Fatalf("transferir item: %v", err)
	}

	itensA, err := contaSvc.ListarItens(ctx, contaA.ID)
	if err != nil {
		t.Fatalf("listar itens A: %v", err)
	}
	itensB, err := contaSvc.ListarItens(ctx, contaB.ID)
	if err != nil {
		t.Fatalf("listar itens B: %v", err)
	}
	if len(itensA) != 0 {
		t.Errorf("conta A devia ficar sem itens, tem %d", len(itensA))
	}
	if len(itensB) != 1 || itensB[0].Quantidade != 4 {
		t.Fatalf("conta B devia ter 1 item de quantidade 4, tem %+v", itensB)
	}

	// fechar a conta que recebeu o item e confirmar que o total bate
	// exatamente com o que foi transferido (4 * 180 = 720) — nada se
	// perdeu nem duplicou na transferência.
	resultado, _, err := contaSvc.FecharConta(ctx, contaB.ID, usecase.FecharContaParams{
		ArmazemID:    armazem.ID,
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 720}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("fechar conta B: %v", err)
	}
	if resultado.Venda.Total != 720 {
		t.Errorf("total = %v, want 720", resultado.Venda.Total)
	}
}

func TestFecharConta_DividaBloqueadaParaClienteAnonimo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, contaSvc, _ := newTestContaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Whisky dose", 350)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 200, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	conta, err := contaSvc.AbrirConta(ctx, "Balcão 2", nil, 1) // nil = anónimo
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}

	_, _, err = contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{
		ArmazemID:    armazem.ID,
		Pagamentos:   nil, // sem pagamento nenhum -> ficaria tudo em dívida
		UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrClienteAnonimoNaoPodeDivida) {
		t.Fatalf("esperava ErrClienteAnonimoNaoPodeDivida, obteve %v", err)
	}
}

func TestFecharConta_CriaDividaParaClienteIdentificado(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, clienteSvc, contaSvc, vendaRepo := newTestContaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Whisky dose", 350)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 200, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	cliente, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "Carlos Muianga"})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}
	clienteID := cliente.ID

	conta, err := contaSvc.AbrirConta(ctx, "Mesa 5", &clienteID, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	// 2 doses a 350 = 700 total.
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 2, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}

	resultado, _, err := contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{
		ArmazemID:    armazem.ID,
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 300}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("fechar conta com pagamento parcial: %v", err)
	}
	if resultado.Venda.Status != domain.StatusVendaParcial {
		t.Errorf("status da venda = %v, want parcial", resultado.Venda.Status)
	}

	dividas, err := vendaRepo.ListarDividasPorCliente(ctx, clienteID)
	if err != nil {
		t.Fatalf("listar dívidas: %v", err)
	}
	if len(dividas) != 1 {
		t.Fatalf("esperava 1 dívida criada, obteve %d", len(dividas))
	}
	if dividas[0].ValorEmAberto != 400 {
		t.Errorf("valor em aberto = %v, want 400 (700-300)", dividas[0].ValorEmAberto)
	}
	if dividas[0].Status != domain.StatusDividaParcial {
		t.Errorf("status da dívida = %v, want parcial", dividas[0].Status)
	}
}

// Pedir a mesma bebida outra vez soma na mesma linha; o "−" baixa a
// quantidade e a 0 tira a linha; depois de fechada a conta, nada muda.
func TestComanda_JuntaMesmaBebidaEAlteraQuantidade(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, contaSvc, _ := newTestContaServices(t)

	cerveja, uCerveja := criarProdutoComPreco(t, ctx, produtoSvc, "2M 550ml", 180)
	agua, uAgua := criarProdutoComPreco(t, ctx, produtoSvc, "Água 500ml", 60)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	for _, p := range []domain.Produto{cerveja, agua} {
		if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
			ProdutoID: p.ID, ArmazemDestinoID: armazem.ID, Quantidade: 50, CustoUnitario: 40, UtilizadorID: 1,
		}); err != nil {
			t.Fatalf("entrada: %v", err)
		}
	}
	conta, err := contaSvc.AbrirConta(ctx, "Mesa 1", nil, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	lancar := func(produtoID, unidadeID, operador int64) domain.ItemConta {
		t.Helper()
		item, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
			ContaID: conta.ID, ProdutoID: produtoID, UnidadeProdutoID: unidadeID, Quantidade: 1, UtilizadorID: operador,
		})
		if err != nil {
			t.Fatalf("adicionar: %v", err)
		}
		return item
	}

	primeira := lancar(cerveja.ID, uCerveja.ID, 1)
	segunda := lancar(cerveja.ID, uCerveja.ID, 1)
	if segunda.ID != primeira.ID || segunda.Quantidade != 2 {
		t.Fatalf("a 2.ª cerveja devia somar na mesma linha: %+v", segunda)
	}
	lancar(agua.ID, uAgua.ID, 1)

	itens, _ := contaSvc.ListarItens(ctx, conta.ID)
	if len(itens) != 2 {
		t.Fatalf("linhas = %d, want 2 (2× cerveja e a água)", len(itens))
	}

	if it, err := contaSvc.AlterarQuantidadeItem(ctx, primeira.ID, 1); err != nil || it.Quantidade != 1 {
		t.Fatalf("baixar para 1: %+v, %v", it, err)
	}
	if _, err := contaSvc.AlterarQuantidadeItem(ctx, primeira.ID, -1); err == nil {
		t.Error("quantidade negativa devia ser recusada")
	}
	if _, err := contaSvc.AlterarQuantidadeItem(ctx, primeira.ID, 0); err != nil {
		t.Fatalf("tirar linha: %v", err)
	}
	itens, _ = contaSvc.ListarItens(ctx, conta.ID)
	if len(itens) != 1 {
		t.Fatalf("depois de tirar a linha: %d linhas, want 1", len(itens))
	}

	if _, _, err := contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{
		ArmazemID: armazem.ID, UtilizadorID: 1,
		Pagamentos: []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 60}},
	}); err != nil {
		t.Fatalf("fechar: %v", err)
	}
	if _, err := contaSvc.AlterarQuantidadeItem(ctx, itens[0].ID, 5); !errors.Is(err, domain.ErrContaNaoAberta) {
		t.Errorf("conta fechada: err = %v, want ErrContaNaoAberta", err)
	}
}