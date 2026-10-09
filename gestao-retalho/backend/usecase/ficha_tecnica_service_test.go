package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestFichaTecnicaServices(t *testing.T) (*usecase.ProdutoService, *usecase.ArmazemService, *usecase.StockService, *usecase.VendaService, *usecase.FichaTecnicaService) {
	t.Helper()
	db := newTestDB(t)

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	vendaRepo := repository.NewVendaRepository(db, stockRepo, loteRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo, loteRepo)
	vendaSvc := usecase.NewVendaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, repository.NewClienteRepository(db))
	fichaSvc := usecase.NewFichaTecnicaService(fichaRepo, produtoRepo, stockRepo, loteRepo)

	return produtoSvc, armazemSvc, stockSvc, vendaSvc, fichaSvc
}

// criarIngrediente cria um produto simples com unidade_base == nomeUnidade
// (ex: "L", "kg", "unidade") — conveniente para fichas técnicas, onde a
// quantidade é sempre na unidade base do ingrediente (SDD 3.3).
func criarIngrediente(t *testing.T, ctx context.Context, svc *usecase.ProdutoService, nome, unidadeBase string, precoCusto float64) domain.Produto {
	t.Helper()
	p, _, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: nome, TipoProduto: domain.TipoProdutoSimples, UnidadeBase: unidadeBase},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: unidadeBase, QtdEmUnidadeBase: 1, PrecoCusto: precoCusto, PrecoVenda: precoCusto * 1.3, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar ingrediente %q: %v", nome, err)
	}
	return p
}

func criarComposto(t *testing.T, ctx context.Context, svc *usecase.ProdutoService, nome string, precoVenda float64) (domain.Produto, domain.UnidadeProduto) {
	t.Helper()
	p, unidades, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: nome, TipoProduto: domain.TipoProdutoComposto, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoVenda: precoVenda, IsPadraoVenda: true},
		},
	})
	if err != nil {
		t.Fatalf("criar composto %q: %v", nome, err)
	}
	return p, unidades[0]
}

func TestVendaComposto_ExpandeFichaTecnicaComNMovimentos(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, fichaSvc := newTestFichaTecnicaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")

	vodka := criarIngrediente(t, ctx, produtoSvc, "Vodka", "L", 400)
	limao := criarIngrediente(t, ctx, produtoSvc, "Limão", "unidade", 5)
	acucar := criarIngrediente(t, ctx, produtoSvc, "Açúcar", "kg", 60)
	gelo := criarIngrediente(t, ctx, produtoSvc, "Gelo", "kg", 20)

	for _, e := range []struct {
		produto domain.Produto
		qtd     float64
		custo   float64
	}{
		{vodka, 1, 400}, {limao, 20, 5}, {acucar, 1, 60}, {gelo, 5, 20},
	} {
		if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
			ProdutoID: e.produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: e.qtd, CustoUnitario: e.custo, UtilizadorID: 1,
		}); err != nil {
			t.Fatalf("entrada %s: %v", e.produto.Nome, err)
		}
	}

	caipirinha, unidadeComposto := criarComposto(t, ctx, produtoSvc, "Caipirinha de Vodka", 450)
	if _, err := fichaSvc.DefinirReceita(ctx, caipirinha.ID, []usecase.IngredienteInput{
		{ProdutoIngredienteID: vodka.ID, Quantidade: 0.05},
		{ProdutoIngredienteID: limao.ID, Quantidade: 1},
		{ProdutoIngredienteID: acucar.ID, Quantidade: 0.02},
		{ProdutoIngredienteID: gelo.ID, Quantidade: 0.1},
	}); err != nil {
		t.Fatalf("definir receita: %v", err)
	}

	// custo teórico = 0.05*400 + 1*5 + 0.02*60 + 0.1*20 = 20+5+1.2+2 = 28.2
	custoTeorico, err := fichaSvc.CalcularCustoTeorico(ctx, caipirinha.ID, armazem.ID)
	if err != nil {
		t.Fatalf("calcular custo teórico: %v", err)
	}
	if custoTeorico != 28.2 {
		t.Errorf("custo teórico = %v, want 28.2", custoTeorico)
	}

	resultado, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: caipirinha.ID, UnidadeProdutoID: unidadeComposto.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 450}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("vender composto: %v", err)
	}

	// o cliente só vê 1 item_venda: "1x Caipirinha de Vodka".
	if len(resultado.Itens) != 1 {
		t.Fatalf("esperava 1 item_venda (o composto), obteve %d", len(resultado.Itens))
	}
	if resultado.Itens[0].CustoUnitarioNoMomento != 28.2 {
		t.Errorf("custo unitário no momento = %v, want 28.2", resultado.Itens[0].CustoUnitarioNoMomento)
	}

	// por trás, cada ingrediente tem exatamente 2 movimentos (a entrada +
	// a saída da venda), com sinal negativo (ArmazemOrigemID preenchido,
	// ArmazemDestinoID vazio) e quantidade correta.
	casos := []struct {
		produto     domain.Produto
		qtdEsperada float64
		stockFinal  float64
	}{
		{vodka, 0.05, 0.95},
		{limao, 1, 19},
		{acucar, 0.02, 0.98},
		{gelo, 0.1, 4.9},
	}
	for _, c := range casos {
		movimentos, err := stockSvc.ListarMovimentos(ctx, c.produto.ID)
		if err != nil {
			t.Fatalf("listar movimentos %s: %v", c.produto.Nome, err)
		}
		if len(movimentos) != 2 {
			t.Fatalf("%s: esperava 2 movimentos (entrada+venda), obteve %d", c.produto.Nome, len(movimentos))
		}
		ultimo := movimentos[0] // ORDER BY criado_em DESC, id DESC
		if ultimo.Tipo != domain.TipoMovimentoVenda {
			t.Errorf("%s: tipo do último movimento = %v, want venda", c.produto.Nome, ultimo.Tipo)
		}
		if ultimo.ArmazemOrigemID == nil || *ultimo.ArmazemOrigemID != armazem.ID {
			t.Errorf("%s: movimento de venda devia sair do armazém (ArmazemOrigemID)", c.produto.Nome)
		}
		if ultimo.ArmazemDestinoID != nil {
			t.Errorf("%s: movimento de venda não devia ter ArmazemDestinoID", c.produto.Nome)
		}
		if ultimo.QuantidadeUnidadeBase != c.qtdEsperada {
			t.Errorf("%s: quantidade consumida = %v, want %v", c.produto.Nome, ultimo.QuantidadeUnidadeBase, c.qtdEsperada)
		}

		stock, err := stockSvc.ObterStock(ctx, c.produto.ID, armazem.ID)
		if err != nil {
			t.Fatalf("obter stock %s: %v", c.produto.Nome, err)
		}
		if stock.Quantidade != c.stockFinal {
			t.Errorf("%s: stock final = %v, want %v", c.produto.Nome, stock.Quantidade, c.stockFinal)
		}
	}
}

func TestVendaComposto_BloqueiaSemStockDeIngredienteEAtomico(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, fichaSvc := newTestFichaTecnicaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")

	vodka := criarIngrediente(t, ctx, produtoSvc, "Vodka", "L", 400)
	limao := criarIngrediente(t, ctx, produtoSvc, "Limão", "unidade", 5)

	// vodka tem stock de sobra, limão só tem 0 (sem entrada nenhuma) —
	// a venda tem de falhar e a vodka (processada primeiro) não pode
	// ficar consumida.
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: vodka.ID, ArmazemDestinoID: armazem.ID, Quantidade: 1, CustoUnitario: 400, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada vodka: %v", err)
	}

	caipirinha, unidadeComposto := criarComposto(t, ctx, produtoSvc, "Caipirinha de Vodka", 450)
	if _, err := fichaSvc.DefinirReceita(ctx, caipirinha.ID, []usecase.IngredienteInput{
		{ProdutoIngredienteID: vodka.ID, Quantidade: 0.05},
		{ProdutoIngredienteID: limao.ID, Quantidade: 1},
	}); err != nil {
		t.Fatalf("definir receita: %v", err)
	}

	_, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: caipirinha.ID, UnidadeProdutoID: unidadeComposto.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 450}},
		UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrStockInsuficiente) {
		t.Fatalf("esperava ErrStockInsuficiente, obteve %v", err)
	}

	stockVodka, _ := stockSvc.ObterStock(ctx, vodka.ID, armazem.ID)
	if stockVodka.Quantidade != 1 {
		t.Errorf("vodka = %v, want continuar 1 (venda falhada a meio não pode deixar consumo parcial)", stockVodka.Quantidade)
	}
}

func TestConsumirFichaTecnica_AplicaPerdaPercentual(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc, fichaSvc := newTestFichaTecnicaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Cozinha")

	batata := criarIngrediente(t, ctx, produtoSvc, "Batata", "kg", 40)
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: batata.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 40, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada batata: %v", err)
	}

	prato, unidadeComposto := criarComposto(t, ctx, produtoSvc, "Batata frita", 200)
	// 0.15kg líquidos, 10% de perda no descasque -> consumo real = 0.15/0.9 = 0.1666...7
	if _, err := fichaSvc.DefinirReceita(ctx, prato.ID, []usecase.IngredienteInput{
		{ProdutoIngredienteID: batata.ID, Quantidade: 0.15, PerdaPercentual: 10},
	}); err != nil {
		t.Fatalf("definir receita: %v", err)
	}

	if _, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: prato.ID, UnidadeProdutoID: unidadeComposto.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 200}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("vender: %v", err)
	}

	stockBatata, err := stockSvc.ObterStock(ctx, batata.ID, armazem.ID)
	if err != nil {
		t.Fatalf("obter stock: %v", err)
	}
	consumoEsperado := 0.15 / 0.9
	stockEsperado := 10 - consumoEsperado
	if diff := stockBatata.Quantidade - stockEsperado; diff < -0.0001 || diff > 0.0001 {
		t.Errorf("stock final = %v, want %v (perda de 10%% aumenta o consumo real)", stockBatata.Quantidade, stockEsperado)
	}
}

func TestSimularDisponibilidade_RespeitaIngredienteMaisRestritivo(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, fichaSvc := newTestFichaTecnicaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")

	vodka := criarIngrediente(t, ctx, produtoSvc, "Vodka", "L", 400)
	limao := criarIngrediente(t, ctx, produtoSvc, "Limão", "unidade", 5)

	// vodka dá para 1L/0.05L = 20 doses; limão só dá para 8/1 = 8 doses.
	// o limão é o ingrediente mais restritivo.
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: vodka.ID, ArmazemDestinoID: armazem.ID, Quantidade: 1, CustoUnitario: 400, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada vodka: %v", err)
	}
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: limao.ID, ArmazemDestinoID: armazem.ID, Quantidade: 8, CustoUnitario: 5, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada limão: %v", err)
	}

	caipirinha, _ := criarComposto(t, ctx, produtoSvc, "Caipirinha de Vodka", 450)
	if _, err := fichaSvc.DefinirReceita(ctx, caipirinha.ID, []usecase.IngredienteInput{
		{ProdutoIngredienteID: vodka.ID, Quantidade: 0.05},
		{ProdutoIngredienteID: limao.ID, Quantidade: 1},
	}); err != nil {
		t.Fatalf("definir receita: %v", err)
	}

	disponivel, err := fichaSvc.SimularDisponibilidade(ctx, caipirinha.ID, armazem.ID)
	if err != nil {
		t.Fatalf("simular disponibilidade: %v", err)
	}
	if disponivel != 8 {
		t.Errorf("disponível = %v, want 8 (limitado pelo limão, não pela vodka)", disponivel)
	}
}

func TestRegistarProducao_ConsomeIngredientesECriaStockNaMesmaTransacao(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, _, fichaSvc := newTestFichaTecnicaServices(t)
	armazem := criarArmazem(t, ctx, armazemSvc, "Cozinha")

	arrozCru := criarIngrediente(t, ctx, produtoSvc, "Arroz cru", "kg", 90)
	agua := criarIngrediente(t, ctx, produtoSvc, "Água", "L", 1)
	sal := criarIngrediente(t, ctx, produtoSvc, "Sal", "kg", 30)

	for _, e := range []struct {
		produto domain.Produto
		qtd     float64
		custo   float64
	}{
		{arrozCru, 10, 90}, {agua, 20, 1}, {sal, 2, 30},
	} {
		if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
			ProdutoID: e.produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: e.qtd, CustoUnitario: e.custo, UtilizadorID: 1,
		}); err != nil {
			t.Fatalf("entrada %s: %v", e.produto.Nome, err)
		}
	}

	arrozCozinhado, _, err := produtoSvc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: "Arroz Cozinhado", TipoProduto: domain.TipoProdutoSemiElaborado, UnidadeBase: "kg"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "kg", QtdEmUnidadeBase: 1, PrecoCusto: 0, PrecoVenda: 0, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar semi-elaborado: %v", err)
	}

	// ficha técnica é sempre "por unidade produzida" (mesma convenção do
	// composto: por unidade vendida) — por kg de arroz cozinhado: 0.24kg
	// arroz cru, 0.4L água, 0.02kg sal. Para um lote de 5kg isso dá os
	// 1.2kg/2L/0.1kg do exemplo do SDD 3.3.2.
	if _, err := fichaSvc.DefinirReceita(ctx, arrozCozinhado.ID, []usecase.IngredienteInput{
		{ProdutoIngredienteID: arrozCru.ID, Quantidade: 0.24},
		{ProdutoIngredienteID: agua.ID, Quantidade: 0.4},
		{ProdutoIngredienteID: sal.ID, Quantidade: 0.02},
	}); err != nil {
		t.Fatalf("definir receita: %v", err)
	}

	movimento, err := fichaSvc.RegistarProducao(ctx, usecase.ProducaoParams{
		ProdutoSemiElaboradoID: arrozCozinhado.ID, ArmazemID: armazem.ID, Quantidade: 5, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("registar produção: %v", err)
	}
	if movimento.Tipo != domain.TipoMovimentoProducao {
		t.Errorf("tipo do movimento = %v, want producao", movimento.Tipo)
	}
	if movimento.ArmazemDestinoID == nil || *movimento.ArmazemDestinoID != armazem.ID {
		t.Errorf("movimento de produção devia criar stock no armazém (ArmazemDestinoID)")
	}
	if movimento.QuantidadeUnidadeBase != 5 {
		t.Errorf("quantidade produzida = %v, want 5", movimento.QuantidadeUnidadeBase)
	}

	// ingredientes consumidos corretamente.
	stockArroz, _ := stockSvc.ObterStock(ctx, arrozCru.ID, armazem.ID)
	if stockArroz.Quantidade != 8.8 { // 10 - 1.2
		t.Errorf("arroz cru final = %v, want 8.8", stockArroz.Quantidade)
	}
	stockAgua, _ := stockSvc.ObterStock(ctx, agua.ID, armazem.ID)
	if stockAgua.Quantidade != 18 { // 20 - 2
		t.Errorf("água final = %v, want 18", stockAgua.Quantidade)
	}
	stockSal, _ := stockSvc.ObterStock(ctx, sal.ID, armazem.ID)
	if diff := stockSal.Quantidade - 1.9; diff < -0.0001 || diff > 0.0001 { // 2 - 0.1
		t.Errorf("sal final = %v, want 1.9", stockSal.Quantidade)
	}

	// stock do semi-elaborado produzido e custo médio calculados.
	stockProduzido, err := stockSvc.ObterStock(ctx, arrozCozinhado.ID, armazem.ID)
	if err != nil {
		t.Fatalf("obter stock produzido: %v", err)
	}
	if stockProduzido.Quantidade != 5 {
		t.Errorf("stock produzido = %v, want 5", stockProduzido.Quantidade)
	}
	// custo total ingredientes = 1.2*90 + 2*1 + 0.1*30 = 108+2+3 = 113; /5kg = 22.6/kg
	if stockProduzido.CustoMedioPonderado != 22.6 {
		t.Errorf("custo médio do produzido = %v, want 22.6", stockProduzido.CustoMedioPonderado)
	}
}
