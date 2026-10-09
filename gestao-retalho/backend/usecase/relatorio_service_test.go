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

func newTestRelatorioServices(t *testing.T) (
	*usecase.ProdutoService, *usecase.ArmazemService, *usecase.StockService,
	*usecase.VendaService, *usecase.ContaService, *usecase.ClienteService,
	*usecase.FichaTecnicaService, *usecase.RelatorioService, *sql.DB,
) {
	t.Helper()
	db := newTestDB(t)

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	vendaRepo := repository.NewVendaRepository(db, stockRepo, loteRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	clienteRepo := repository.NewClienteRepository(db)
	dividaRepo := repository.NewDividaRepository(db)
	relatorioRepo := repository.NewRelatorioRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo, loteRepo)
	vendaSvc := usecase.NewVendaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, repository.NewClienteRepository(db))
	clienteSvc := usecase.NewClienteService(clienteRepo)
	dividaSvc := usecase.NewDividaService(dividaRepo, clienteRepo)
	contaSvc := usecase.NewContaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, dividaSvc)
	fichaSvc := usecase.NewFichaTecnicaService(fichaRepo, produtoRepo, stockRepo)
	relatorioSvc := usecase.NewRelatorioService(relatorioRepo, fichaRepo, produtoRepo, clienteRepo, dividaSvc)

	return produtoSvc, armazemSvc, stockSvc, vendaSvc, contaSvc, clienteSvc, fichaSvc, relatorioSvc, db
}

func janelaAlargada() (time.Time, time.Time) {
	return time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
}

func TestVendasPorPeriodo_AgregaResumoETicketMedio(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 100)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 100, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	for i := 0; i < 2; i++ {
		if _, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
			ArmazemID:    armazem.ID,
			Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
			Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 100}},
			UtilizadorID: 1,
		}); err != nil {
			t.Fatalf("venda %d: %v", i, err)
		}
	}

	desde, ate := janelaAlargada()
	resumo, linhas, err := relatorioSvc.VendasPorPeriodo(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate})
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if resumo.NumeroVendas != 2 {
		t.Errorf("numero vendas = %d, want 2", resumo.NumeroVendas)
	}
	if resumo.TotalVendido != 200 {
		t.Errorf("total vendido = %v, want 200", resumo.TotalVendido)
	}
	if resumo.TicketMedio != 100 {
		t.Errorf("ticket médio = %v, want 100", resumo.TicketMedio)
	}
	if len(linhas) != 1 {
		t.Errorf("linhas por dia = %d, want 1 (mesmo dia)", len(linhas))
	}
}

func TestLucroPorPeriodo_CalculaReceitaCustoMargem(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	// criarProdutoComPreco fixa custo = preço/2 -> 100 venda, 50 custo.
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 100)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	if _, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 2}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 200}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("venda: %v", err)
	}

	desde, ate := janelaAlargada()
	lucro, err := relatorioSvc.LucroPorPeriodo(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate})
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if lucro.Receita != 200 {
		t.Errorf("receita = %v, want 200", lucro.Receita)
	}
	if lucro.Custo != 100 {
		t.Errorf("custo = %v, want 100", lucro.Custo)
	}
	if lucro.Lucro != 100 {
		t.Errorf("lucro = %v, want 100", lucro.Lucro)
	}
	if lucro.MargemPercentual != 50 {
		t.Errorf("margem = %v, want 50", lucro.MargemPercentual)
	}
}

func TestVendasPorProduto_RankingPorValorDescendente(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	caro, unidadeCaro := criarProdutoComPreco(t, ctx, produtoSvc, "Cimento", 600)
	barato, unidadeBarato := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 10)
	for _, p := range []domain.Produto{caro, barato} {
		if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
			ProdutoID: p.ID, ArmazemDestinoID: armazem.ID, Quantidade: 100, CustoUnitario: 5, UtilizadorID: 1,
		}); err != nil {
			t.Fatalf("entrada %s: %v", p.Nome, err)
		}
	}

	if _, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID: armazem.ID,
		Itens: []usecase.ItemVendaDiretaParams{
			{ProdutoID: caro.ID, UnidadeProdutoID: unidadeCaro.ID, Quantidade: 1},
			{ProdutoID: barato.ID, UnidadeProdutoID: unidadeBarato.ID, Quantidade: 5},
		},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 650}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("venda: %v", err)
	}

	desde, ate := janelaAlargada()
	linhas, err := relatorioSvc.VendasPorProduto(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate})
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if len(linhas) != 2 {
		t.Fatalf("linhas = %d, want 2", len(linhas))
	}
	if linhas[0].ProdutoID != caro.ID || linhas[0].ValorTotal != 600 {
		t.Errorf("linha 0 = %+v, want cimento 600 primeiro (ranking por valor)", linhas[0])
	}
	if linhas[1].ProdutoID != barato.ID || linhas[1].ValorTotal != 50 {
		t.Errorf("linha 1 = %+v, want prego 50", linhas[1])
	}
}

func TestVendasPorMetodoPagamento_AgregaPorMetodo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 100)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	if _, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID: armazem.ID,
		Itens:     []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		Pagamentos: []usecase.PagamentoVendaParams{
			{Metodo: domain.MetodoDinheiro, Valor: 60},
			{Metodo: domain.MetodoMpesa, Valor: 40},
		},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("venda: %v", err)
	}

	desde, ate := janelaAlargada()
	linhas, err := relatorioSvc.VendasPorMetodoPagamento(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate})
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	totais := map[domain.MetodoPagamento]float64{}
	for _, l := range linhas {
		totais[l.Metodo] = l.Total
	}
	if totais[domain.MetodoDinheiro] != 60 {
		t.Errorf("dinheiro = %v, want 60", totais[domain.MetodoDinheiro])
	}
	if totais[domain.MetodoMpesa] != 40 {
		t.Errorf("mpesa = %v, want 40", totais[domain.MetodoMpesa])
	}
}

func TestStockAtual_MarcaAbaixoDoMinimo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	minimo := 20.0
	produto, _, err := produtoSvc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: "Prego", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade", StockMinimo: &minimo},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoCusto: 5, PrecoVenda: 10, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 5, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	linhas, valorTotal, err := relatorioSvc.StockAtual(ctx, nil)
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if len(linhas) != 1 {
		t.Fatalf("linhas = %d, want 1", len(linhas))
	}
	if !linhas[0].AbaixoDoMinimo {
		t.Errorf("esperava AbaixoDoMinimo=true (10 < 20)")
	}
	if linhas[0].ValorStock != 50 {
		t.Errorf("valor stock = %v, want 50 (10*5)", linhas[0].ValorStock)
	}
	if valorTotal != 50 {
		t.Errorf("valor total = %v, want 50", valorTotal)
	}
}

func TestMovimentosPorPeriodo_FiltraPorArmazem(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	balcao := criarArmazem(t, ctx, armazemSvc, "Balcão")
	deposito := criarArmazem(t, ctx, armazemSvc, "Depósito")
	produto, _ := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 10)

	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: balcao.ID, Quantidade: 10, CustoUnitario: 5, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada balcão: %v", err)
	}
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: deposito.ID, Quantidade: 20, CustoUnitario: 5, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada depósito: %v", err)
	}

	desde, ate := janelaAlargada()
	linhas, err := relatorioSvc.MovimentosPorPeriodo(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate, ArmazemID: &balcao.ID})
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if len(linhas) != 1 {
		t.Fatalf("linhas = %d, want 1 (só o movimento do balcão)", len(linhas))
	}
	if linhas[0].QuantidadeUnidadeBase != 10 {
		t.Errorf("quantidade = %v, want 10", linhas[0].QuantidadeUnidadeBase)
	}

	todos, err := relatorioSvc.MovimentosPorPeriodo(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate})
	if err != nil {
		t.Fatalf("relatorio sem filtro: %v", err)
	}
	if len(todos) != 2 {
		t.Errorf("linhas sem filtro = %d, want 2", len(todos))
	}
}

func TestPerdasConsumoInterno_ValorizaAoCusto(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	produto, _ := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 10)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 50, CustoUnitario: 5, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	if _, err := stockSvc.RegistarPerda(ctx, usecase.SaidaParams{
		ProdutoID: produto.ID, ArmazemOrigemID: armazem.ID, Quantidade: 4, Motivo: "quebra", UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("perda: %v", err)
	}

	desde, ate := janelaAlargada()
	linhas, total, err := relatorioSvc.PerdasConsumoInterno(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate})
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if len(linhas) != 1 {
		t.Fatalf("linhas = %d, want 1", len(linhas))
	}
	if linhas[0].Tipo != domain.TipoMovimentoPerda {
		t.Errorf("tipo = %v, want perda", linhas[0].Tipo)
	}
	if linhas[0].ValorTotal != 20 {
		t.Errorf("valor total = %v, want 20 (4*5)", linhas[0].ValorTotal)
	}
	if total != 20 {
		t.Errorf("total geral = %v, want 20", total)
	}
}

func TestContasPorPeriodo_CalculaValorMedio(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, contaSvc, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Cerveja", 80)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 20, CustoUnitario: 40, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	conta, err := contaSvc.AbrirConta(ctx, "Mesa 1", nil, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 2, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}
	if _, _, err := contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{
		ArmazemID:    armazem.ID,
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 160}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("fechar conta: %v", err)
	}

	desde, ate := janelaAlargada()
	resumo, linhas, err := relatorioSvc.ContasPorPeriodo(ctx, desde, ate)
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if resumo.NumeroContas != 1 {
		t.Fatalf("numero contas = %d, want 1", resumo.NumeroContas)
	}
	if resumo.ValorMedio != 160 {
		t.Errorf("valor médio = %v, want 160", resumo.ValorMedio)
	}
	if len(linhas) != 1 || linhas[0].ValorTotal != 160 {
		t.Errorf("linhas = %+v, want 1 linha com valor 160", linhas)
	}
	if linhas[0].TempoAbertoHoras == nil {
		t.Errorf("esperava TempoAbertoHoras preenchido para conta fechada")
	}
}

func TestDividasEmAberto_CalculaIdadeDias(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, contaSvc, clienteSvc, _, relatorioSvc, db := newTestRelatorioServices(t)

	cliente, err := clienteSvc.Criar(ctx, domain.Cliente{Nome: "João"})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Cimento", 100)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	conta, err := contaSvc.AbrirConta(ctx, "Dívida João", &cliente.ID, 1)
	if err != nil {
		t.Fatalf("abrir conta: %v", err)
	}
	if _, err := contaSvc.AdicionarItem(ctx, usecase.AdicionarItemContaParams{
		ContaID: conta.ID, ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("adicionar item: %v", err)
	}
	// Zero pagamentos -> gera dívida do valor total (SDD 4.10/4.11).
	if _, _, err := contaSvc.FecharConta(ctx, conta.ID, usecase.FecharContaParams{ArmazemID: armazem.ID, UtilizadorID: 1}); err != nil {
		t.Fatalf("fechar conta: %v", err)
	}

	// Não há forma de fazer isto pela API pública (e não devia haver) —
	// mesma técnica exclusiva de teste da Fase 7 (divida_service_test.go)
	// para simular uma dívida antiga.
	antiga := time.Now().Add(-45 * 24 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z")
	if _, err := db.Exec("UPDATE dividas SET criado_em = ? WHERE cliente_id = ?", antiga, cliente.ID); err != nil {
		t.Fatalf("recuar criado_em: %v", err)
	}

	linhas, err := relatorioSvc.DividasEmAberto(ctx)
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if len(linhas) != 1 {
		t.Fatalf("linhas = %d, want 1", len(linhas))
	}
	if linhas[0].ClienteNome != "João" {
		t.Errorf("cliente nome = %q, want João", linhas[0].ClienteNome)
	}
	if linhas[0].IdadeDias < 44 || linhas[0].IdadeDias > 46 {
		t.Errorf("idade dias = %d, want ~45", linhas[0].IdadeDias)
	}
}

func TestFechoCaixaDiario_SomaPorMetodo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, _, _, _, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 100)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	if _, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID: armazem.ID,
		Itens:     []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		Pagamentos: []usecase.PagamentoVendaParams{
			{Metodo: domain.MetodoDinheiro, Valor: 70},
			{Metodo: domain.MetodoMpesa, Valor: 30},
		},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("venda: %v", err)
	}

	fecho, err := relatorioSvc.FechoCaixaDiario(ctx, time.Now())
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if fecho.TotalVendido != 100 {
		t.Errorf("total vendido = %v, want 100", fecho.TotalVendido)
	}
	if len(fecho.PorMetodo) != 2 {
		t.Errorf("por método = %d linhas, want 2", len(fecho.PorMetodo))
	}
}

func TestConsumoTeoricoVsReal_SemDesvioQuandoConsumoSeguiuFicha(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, _, _, fichaSvc, relatorioSvc, _ := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	vodka := criarIngrediente(t, ctx, produtoSvc, "Vodka", "L", 400)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: vodka.ID, ArmazemDestinoID: armazem.ID, Quantidade: 5, CustoUnitario: 400, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada vodka: %v", err)
	}

	caipirinha, unidadeComposto := criarComposto(t, ctx, produtoSvc, "Caipirinha", 100)
	if _, err := fichaSvc.DefinirReceita(ctx, caipirinha.ID, []usecase.IngredienteInput{
		{ProdutoIngredienteID: vodka.ID, Quantidade: 0.05},
	}); err != nil {
		t.Fatalf("definir receita: %v", err)
	}

	if _, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: caipirinha.ID, UnidadeProdutoID: unidadeComposto.ID, Quantidade: 10}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 1000}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("venda: %v", err)
	}

	desde, ate := janelaAlargada()
	linhas, err := relatorioSvc.ConsumoTeoricoVsReal(ctx, usecase.FiltroPeriodo{Desde: desde, Ate: ate})
	if err != nil {
		t.Fatalf("relatorio: %v", err)
	}
	if len(linhas) != 1 {
		t.Fatalf("linhas = %d, want 1 (só a vodka é ingrediente)", len(linhas))
	}
	// 10 caipirinhas * 0.05L cada = 0.5L teórico; sem perda registada,
	// o real (movimento tipo=venda do ingrediente) segue exatamente igual.
	if linhas[0].ConsumoTeorico != 0.5 {
		t.Errorf("consumo teórico = %v, want 0.5", linhas[0].ConsumoTeorico)
	}
	if linhas[0].ConsumoReal != 0.5 {
		t.Errorf("consumo real = %v, want 0.5", linhas[0].ConsumoReal)
	}
	if linhas[0].Desvio != 0 {
		t.Errorf("desvio = %v, want 0", linhas[0].Desvio)
	}
}

// Uma venda às 00:30 da hora local é guardada (em UTC) com a data da
// véspera fora de UTC; o relatório por dia e o fecho de caixa têm de a
// contar no dia do relógio da loja.
func TestRelatorios_DiaDaVendaEODoRelogioLocal(t *testing.T) {
	if _, offset := time.Now().Zone(); offset <= 0 {
		t.Skip("só distingue dia local de dia UTC num fuso a leste de UTC")
	}
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, _, _, _, relatorioSvc, db := newTestRelatorioServices(t)

	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Prego", 100)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 50, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	venda, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 100}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("venda: %v", err)
	}

	agora := time.Now()
	meiaNoiteLocal := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, time.Local)
	madrugada := meiaNoiteLocal.Add(30 * time.Minute)
	for _, tabela := range []string{"vendas", "pagamentos"} {
		if _, err := db.Exec("UPDATE "+tabela+" SET criado_em = ? WHERE "+map[string]string{"vendas": "id", "pagamentos": "venda_id"}[tabela]+" = ?",
			madrugada.UTC().Format("2006-01-02T15:04:05.000Z"), venda.Venda.ID); err != nil {
			t.Fatalf("mudar hora da venda: %v", err)
		}
	}

	_, linhas, err := relatorioSvc.VendasPorPeriodo(ctx, usecase.FiltroPeriodo{Desde: meiaNoiteLocal, Ate: meiaNoiteLocal.AddDate(0, 0, 1)})
	if err != nil {
		t.Fatalf("vendas por período: %v", err)
	}
	if len(linhas) != 1 || linhas[0].Dia != meiaNoiteLocal.Format("2006-01-02") {
		t.Fatalf("linhas = %+v, want um único dia %s", linhas, meiaNoiteLocal.Format("2006-01-02"))
	}

	// O ecrã envia a meia-noite local já convertida para UTC.
	fecho, err := relatorioSvc.FechoCaixaDiario(ctx, meiaNoiteLocal.UTC())
	if err != nil {
		t.Fatalf("fecho: %v", err)
	}
	if fecho.Dia != meiaNoiteLocal.Format("2006-01-02") || fecho.TotalVendido != 100 {
		t.Fatalf("fecho = %+v, want dia %s com 100", fecho, meiaNoiteLocal.Format("2006-01-02"))
	}
}
