package admin_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestVendasService_ListarDetalheERecibo(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewVendasService(repo)

	loja := novaLojaTeste(t, "loja-vendas-admin.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}
	produto, unidades, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Parafuso 6mm", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "caixa", QtdEmUnidadeBase: 100, PrecoCusto: 300, PrecoVenda: 500, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 1000, CustoUnitario: 3, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	venda, err := loja.Vendas.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    loja.Armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidades[0].ID, Quantidade: 2}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 1000}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("vender: %v", err)
	}

	hoje := time.Now().Format("2006-01-02")
	ontem := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	lista, err := svc.Listar(ctx, hoje, hoje)
	if err != nil || len(lista) != 1 || lista[0].Venda.ID != venda.Venda.ID || lista[0].ClienteNome == "" {
		t.Fatalf("listar hoje = %+v, err %v; want a venda com nome de cliente", lista, err)
	}
	if antes, err := svc.Listar(ctx, ontem, ontem); err != nil || len(antes) != 0 {
		t.Fatalf("listar ontem = %d vendas, err %v; want 0", len(antes), err)
	}
	if _, err := svc.Listar(ctx, hoje, ontem); err == nil {
		t.Fatal("esperava erro com a data final anterior à inicial")
	}

	detalhe, err := svc.ObterDetalhe(ctx, venda.Venda.ID)
	if err != nil {
		t.Fatalf("detalhe: %v", err)
	}
	if len(detalhe.Itens) != 1 || detalhe.Itens[0].ProdutoNome != "Parafuso 6mm" || detalhe.Itens[0].UnidadeNome != "caixa" {
		t.Fatalf("itens = %+v, want Parafuso 6mm em caixa", detalhe.Itens)
	}
	if len(detalhe.Pagamentos) != 1 || detalhe.Venda.Total != 1000 {
		t.Fatalf("detalhe = %+v, want 1 pagamento e total 1000", detalhe)
	}

	caminho, err := svc.GerarRecibo(ctx, venda.Venda.ID)
	if err != nil {
		t.Fatalf("recibo: %v", err)
	}
	conteudo, err := os.ReadFile(caminho)
	if err != nil || !strings.HasPrefix(string(conteudo), "%PDF") {
		t.Fatalf("recibo em %s não é um PDF (err %v)", caminho, err)
	}
}
