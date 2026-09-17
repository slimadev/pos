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

func TestDashboardService_AgregaDadosDaLoja(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	dashboardSvc := admin.NewDashboardService(repo)

	ferragem := novaLojaTeste(t, "ferragem.db", "Ferragem Central", domain.TipoNegocioFerragem)
	produto, unidade := criarProdutoEStock(t, ctx, ferragem, "Cimento", 600, 20, nil)
	venderHoje(t, ctx, ferragem, produto, unidade, 1, 600)

	if _, err := repo.Criar(ctx, "Ferragem Central", ferragem.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	d, err := dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard: %v", err)
	}

	if !d.TemLoja {
		t.Fatal("TemLoja = false, want true com a loja registada")
	}
	if d.NegocioNome != "Ferragem Central" || d.TipoNegocio != "ferragem" {
		t.Errorf("negócio = %q/%q, want Ferragem Central/ferragem", d.NegocioNome, d.TipoNegocio)
	}
	if d.VendasHoje != 600 {
		t.Errorf("vendas hoje = %v, want 600", d.VendasHoje)
	}
	if len(d.SerieVendas14Dias) == 0 {
		t.Fatal("série de 14 dias vazia")
	}
	ultimoDia := d.SerieVendas14Dias[len(d.SerieVendas14Dias)-1]
	if ultimoDia.Total != 600 {
		t.Errorf("total do último dia = %v, want 600", ultimoDia.Total)
	}
}

func TestDashboardService_LojaOfflineDevolveErro(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	dashboardSvc := admin.NewDashboardService(repo)

	// Insere diretamente no registo (contorna qualquer validação de
	// caminho) para simular uma loja cujo ficheiro desapareceu — o
	// dashboard tem de devolver um erro claro, não rebentar.
	if _, err := repo.Criar(ctx, "Loja Desaparecida", "/caminho/que/nao/existe.db"); err != nil {
		t.Fatalf("registar loja fantasma: %v", err)
	}

	if _, err := dashboardSvc.ObterDashboard(ctx); err == nil {
		t.Fatal("esperava erro para loja cujo ficheiro não existe")
	}
}

func TestDashboardService_AlertaStockAbaixoDoMinimo(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	dashboardSvc := admin.NewDashboardService(repo)

	loja := novaLojaTeste(t, "loja-stock.db", "Farmácia Sommerschield", domain.TipoNegocioFarmacia)
	minimo := 50.0
	criarProdutoEStock(t, ctx, loja, "Paracetamol", 20, 5, &minimo)

	if _, err := repo.Criar(ctx, "Farmácia Sommerschield", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	d, err := dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard: %v", err)
	}
	if d.ProdutosAbaixoMinimo != 1 {
		t.Fatalf("ProdutosAbaixoMinimo = %d, want 1", d.ProdutosAbaixoMinimo)
	}

	achouAlerta := false
	for _, a := range d.Alertas {
		if a.Severidade == "alta" {
			achouAlerta = true
		}
	}
	if !achouAlerta {
		t.Errorf("alertas = %+v, want um alerta alta de stock", d.Alertas)
	}
}

// TestDashboardService_CamposNuncaSaoNilMesmoSemLojaNemDados apanha uma
// classe de bug real: em Go, um slice construído só por `append` fica nil
// se nunca crescer, e nil marshalla para `null` em JSON, não `[]`. O
// frontend faz `d.Alertas.length`/`.map()` sem guarda nenhuma — um `null`
// aí não é "lista vazia", é um ecrã inteiro em branco (sem error boundary).
// Isto tem de valer tanto sem loja nenhuma registada como com uma loja sem
// nenhum dado (o caso mais comum: nenhum alerta a apresentar).
func TestDashboardService_CamposNuncaSaoNilMesmoSemLojaNemDados(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	dashboardSvc := admin.NewDashboardService(repo)

	d, err := dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard sem loja: %v", err)
	}
	if d.TemLoja {
		t.Error("TemLoja = true, want false sem nenhuma loja registada")
	}
	if d.Alertas == nil {
		t.Error("Alertas é nil (want slice vazio) sem nenhuma loja registada")
	}
	if d.SerieVendas14Dias == nil {
		t.Error("SerieVendas14Dias é nil (want slice vazio) sem nenhuma loja registada")
	}
	if d.MaisVendidosHoje == nil {
		t.Error("MaisVendidosHoje é nil (want slice vazio) sem nenhuma loja registada")
	}

	// Uma loja registada mas sem nenhuma venda/stock-mínimo/dívida tem
	// exatamente zero alertas — o caso normal do dia a dia, não uma
	// exceção.
	loja := novaLojaTeste(t, "loja-silenciosa.db", "Mercearia Silenciosa", domain.TipoNegocioMercearia)
	if _, err := repo.Criar(ctx, "Mercearia Silenciosa", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	d, err = dashboardSvc.ObterDashboard(ctx)
	if err != nil {
		t.Fatalf("obter dashboard com loja sem dados: %v", err)
	}
	if !d.TemLoja {
		t.Error("TemLoja = false, want true com a loja registada")
	}
	if d.Alertas == nil {
		t.Error("Alertas é nil (want slice vazio) com loja sem nenhum alerta a disparar")
	}
	if d.MaisVendidosHoje == nil {
		t.Error("MaisVendidosHoje é nil (want slice vazio) com loja sem vendas hoje")
	}
}
