package admin_test

import (
	"context"
	"os"
	"testing"
	"time"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/usecase"
)

func TestRelatoriosService_VendasEExportacao(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewRelatoriosService(repo)

	loja := novaLojaTeste(t, "loja-relatorios-admin.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}
	produto, unidades, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Martelo", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoCusto: 200, PrecoVenda: 350, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 5, CustoUnitario: 200, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	if _, err := loja.Vendas.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    loja.Armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidades[0].ID, Quantidade: 2}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoMpesa, Valor: 700}},
		UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("vender: %v", err)
	}

	f := api.FiltroRelatorioRequest{Desde: time.Now().Add(-time.Hour), Ate: time.Now().Add(time.Hour)}
	vendas, err := svc.VendasPorPeriodo(ctx, f)
	if err != nil || vendas.Resumo.NumeroVendas != 1 || vendas.Resumo.TotalVendido != 700 {
		t.Fatalf("vendas = %+v, err %v; want 1 venda de 700", vendas.Resumo, err)
	}
	metodos, err := svc.VendasPorMetodoPagamento(ctx, f)
	if err != nil || len(metodos) != 1 || metodos[0].Total != 700 {
		t.Fatalf("métodos = %+v, err %v", metodos, err)
	}

	for _, formato := range []string{"xlsx", "pdf"} {
		caminho, err := svc.Exportar(export.Tabela{Titulo: "Vendas por período", Colunas: []string{"Dia", "Total"}, Linhas: [][]string{{"9 de outubro", "700,00 MT"}}}, formato)
		if err != nil {
			t.Fatalf("exportar %s: %v", formato, err)
		}
		if info, err := os.Stat(caminho); err != nil || info.Size() == 0 {
			t.Fatalf("exportar %s: ficheiro %s vazio ou inexistente (%v)", formato, caminho, err)
		}
	}
	if _, err := svc.Exportar(export.Tabela{Titulo: "x"}, "csv"); err == nil {
		t.Fatal("esperava erro para formato desconhecido")
	}
}
