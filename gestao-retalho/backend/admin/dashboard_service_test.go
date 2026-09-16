package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func criarProdutoEStock(t *testing.T, ctx context.Context, loja lojaTeste, nome string, precoVenda, quantidade float64, stockMinimo *float64) (domain.Produto, domain.UnidadeProduto) {
	t.Helper()
	p, unidades, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: nome, TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade", StockMinimo: stockMinimo},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoCusto: precoVenda / 2, PrecoVenda: precoVenda, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto %q: %v", nome, err)
	}
	if quantidade > 0 {
		if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{
			ProdutoID: p.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: quantidade, CustoUnitario: precoVenda / 2, UtilizadorID: 1,
		}); err != nil {
			t.Fatalf("entrada stock %q: %v", nome, err)
		}
	}
	return p, unidades[0]
}

func venderHoje(t *testing.T, ctx context.Context, loja lojaTeste, produto domain.Produto, unidade domain.UnidadeProduto, quantidade, total float64) {
	t.Helper()
	if _, err := loja.Vendas.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    loja.Armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: quantidade}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: total}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("registar venda: %v", err)
	}
}

func TestDashboardService_AgregaVendasDeVariasLojas(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	lojasSvc := admin.NewLojasService(repo)
	dashboardSvc := admin.NewDashboardService(repo)

	bottle := novaLojaTeste(t, "bottlestore.db", "Bottlestore XPTO", domain.TipoNegocioBottlestore)
	produto, unidade := criarProdutoEStock(t, ctx, bottle, "Cerveja", 80, 50, nil)
	venderHoje(t, ctx, bottle, produto, unidade, 2, 160)

	ferragem := novaLojaTeste(t, "ferragem.db", "Ferragem Central", domain.TipoNegocioFerragem)
	produto2, unidade2 := criarProdutoEStock(t, ctx, ferragem, "Cimento", 600, 20, nil)
	venderHoje(t, ctx, ferragem, produto2, unidade2, 1, 600)

	if _, err := lojasSvc.Criar(ctx, "Bottlestore XPTO", bottle.Caminho); err != nil {
		t.Fatalf("registar bottlestore: %v", err)
	}
	if _, err := lojasSvc.Criar(ctx, "Ferragem Central", ferragem.Caminho); err != nil {
		t.Fatalf("registar ferragem: %v", err)
	}

	d, err := dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard: %v", err)
	}

	if d.VendasHoje != 760 {
		t.Errorf("vendas hoje = %v, want 760 (160+600)", d.VendasHoje)
	}
	if len(d.Lojas) != 2 {
		t.Fatalf("lojas = %d, want 2", len(d.Lojas))
	}
	for _, c := range d.Lojas {
		if !c.Online {
			t.Errorf("loja %q devia estar online, erro: %s", c.Nome, c.Erro)
		}
	}
	tipos := map[string]bool{}
	for _, tipo := range d.TiposNegocio {
		tipos[tipo] = true
	}
	if !tipos["bottlestore"] || !tipos["ferragem"] {
		t.Errorf("tipos negocio = %v, want conter bottlestore e ferragem", d.TiposNegocio)
	}

	if len(d.SerieVendasPorLoja) == 0 {
		t.Fatal("série de 14 dias vazia")
	}
	ultimoDia := d.SerieVendasPorLoja[len(d.SerieVendasPorLoja)-1]
	if ultimoDia.PorTipo["bottlestore"] != 160 || ultimoDia.PorTipo["ferragem"] != 600 {
		t.Errorf("série de hoje = %+v, want bottlestore=160 ferragem=600", ultimoDia.PorTipo)
	}
}

func TestDashboardService_LojaOfflineNaoInterrompeAgregacao(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	lojasSvc := admin.NewLojasService(repo)
	dashboardSvc := admin.NewDashboardService(repo)

	loja := novaLojaTeste(t, "loja-viva.db", "Mercearia D. Ana", domain.TipoNegocioMercearia)
	produto, unidade := criarProdutoEStock(t, ctx, loja, "Arroz 5kg", 250, 30, nil)
	venderHoje(t, ctx, loja, produto, unidade, 1, 250)

	if _, err := lojasSvc.Criar(ctx, "Mercearia D. Ana", loja.Caminho); err != nil {
		t.Fatalf("registar loja viva: %v", err)
	}
	// Insere diretamente no registo (contorna a validação de
	// LojasService.Criar de propósito) para simular uma loja cujo
	// ficheiro desapareceu depois de registada — o dashboard tem de
	// continuar a agregar as outras em vez de falhar tudo.
	if _, err := repo.Criar(ctx, "Loja Desaparecida", "/caminho/que/nao/existe.db"); err != nil {
		t.Fatalf("registar loja fantasma: %v", err)
	}

	d, err := dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard: %v", err)
	}
	if len(d.Lojas) != 2 {
		t.Fatalf("lojas = %d, want 2", len(d.Lojas))
	}

	var online, offline int
	for _, c := range d.Lojas {
		if c.Online {
			online++
		} else {
			offline++
			if c.Erro == "" {
				t.Errorf("loja offline devia ter Erro preenchido")
			}
		}
	}
	if online != 1 || offline != 1 {
		t.Errorf("online=%d offline=%d, want 1 e 1", online, offline)
	}
	if d.VendasHoje != 250 {
		t.Errorf("vendas hoje = %v, want 250 (só a loja viva)", d.VendasHoje)
	}
}

func TestDashboardService_AlertaStockAbaixoDoMinimo(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	lojasSvc := admin.NewLojasService(repo)
	dashboardSvc := admin.NewDashboardService(repo)

	loja := novaLojaTeste(t, "loja-stock.db", "Farmácia Sommerschield", domain.TipoNegocioFarmacia)
	minimo := 50.0
	criarProdutoEStock(t, ctx, loja, "Paracetamol", 20, 5, &minimo)

	if _, err := lojasSvc.Criar(ctx, "Farmácia Sommerschield", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	d, err := dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard: %v", err)
	}
	if len(d.Lojas) != 1 || d.Lojas[0].ProdutosAbaixoMinimo != 1 {
		t.Fatalf("lojas = %+v, want 1 loja com 1 produto abaixo do mínimo", d.Lojas)
	}

	achouAlerta := false
	for _, a := range d.Alertas {
		if a.Severidade == "alta" && a.LojaNome == "Farmácia Sommerschield" {
			achouAlerta = true
		}
	}
	if !achouAlerta {
		t.Errorf("alertas = %+v, want um alerta alta para a farmácia", d.Alertas)
	}
}

// TestDashboardService_CamposNuncaSaoNilMesmoSemLojasNemDados apanha uma
// classe de bug real: em Go, um slice construído só por `append` fica nil
// se nunca crescer, e nil marshalla para `null` em JSON, não `[]`. O
// frontend faz `d.Alertas.length`/`.map()` sem guarda nenhuma — um `null`
// aí não é "lista vazia", é um ecrã inteiro em branco (sem error boundary).
// Isto tem de valer tanto com zero lojas registadas como com uma loja sem
// nenhum dado (o caso mais comum: nenhum alerta a apresentar).
func TestDashboardService_CamposNuncaSaoNilMesmoSemLojasNemDados(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	dashboardSvc := admin.NewDashboardService(repo)

	d, err := dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard sem lojas: %v", err)
	}
	if d.Lojas == nil {
		t.Error("Lojas é nil (want slice vazio) sem nenhuma loja registada")
	}
	if d.Alertas == nil {
		t.Error("Alertas é nil (want slice vazio) sem nenhuma loja registada")
	}
	if d.SerieVendasPorLoja == nil {
		t.Error("SerieVendasPorLoja é nil (want slice vazio) sem nenhuma loja registada")
	}
	if d.MaisVendidosHoje == nil {
		t.Error("MaisVendidosHoje é nil (want slice vazio) sem nenhuma loja registada")
	}
	if d.TiposNegocio == nil {
		t.Error("TiposNegocio é nil (want slice vazio) sem nenhuma loja registada")
	}

	// Uma loja registada mas sem nenhuma venda/stock-mínimo/dívida tem
	// exatamente zero alertas — o caso normal do dia a dia, não uma
	// exceção.
	lojasSvc := admin.NewLojasService(repo)
	loja := novaLojaTeste(t, "loja-silenciosa.db", "Mercearia Silenciosa", domain.TipoNegocioMercearia)
	if _, err := lojasSvc.Criar(ctx, "Mercearia Silenciosa", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	d, err = dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard com loja sem dados: %v", err)
	}
	if d.Alertas == nil {
		t.Error("Alertas é nil (want slice vazio) com loja sem nenhum alerta a disparar")
	}
	if d.MaisVendidosHoje == nil {
		t.Error("MaisVendidosHoje é nil (want slice vazio) com loja sem vendas hoje")
	}
}
