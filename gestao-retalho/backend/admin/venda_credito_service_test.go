package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestVendaACredito_CatalogoVendaEAvisoDeLimite(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	vendas := admin.NewVendasService(repo)
	clientes := admin.NewClientesService(repo)

	loja := novaLojaTeste(t, "loja-credito-admin.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}
	produto, unidades, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Varão 12mm", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "barra"},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "barra", QtdEmUnidadeBase: 1, PrecoCusto: 400, PrecoVenda: 650, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 20, CustoUnitario: 400, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	catalogo, err := vendas.CatalogoVenda(ctx, loja.Armazem.ID)
	if err != nil {
		t.Fatalf("catálogo: %v", err)
	}
	var item *admin.ItemCatalogo
	for i := range catalogo {
		if catalogo[i].Produto.ID == produto.ID {
			item = &catalogo[i]
		}
	}
	if item == nil || item.Stock != 20 || len(item.Unidades) != 1 {
		t.Fatalf("item do catálogo = %+v, want Varão com 20 em stock e 1 unidade", item)
	}

	limite := 1000.0
	cliente, err := clientes.Criar(ctx, domain.Cliente{Nome: "Obra Matola", LimiteCredito: &limite})
	if err != nil {
		t.Fatalf("criar cliente: %v", err)
	}
	r, err := vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{
		ArmazemID:  loja.Armazem.ID,
		ClienteID:  &cliente.ID,
		Itens:      []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidades[0].ID, Quantidade: 2}},
		Pagamentos: []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 100}},
	})
	if err != nil {
		t.Fatalf("venda a crédito: %v", err)
	}
	if r.Venda.Venda.Total != 1300 || r.Venda.Venda.Status != domain.StatusVendaParcial {
		t.Fatalf("venda = %+v, want 1300 parcial", r.Venda.Venda)
	}
	if !r.Alerta.Excedido || r.Alerta.TotalEmAberto != 1200 {
		t.Fatalf("alerta = %+v, want excedido com 1200 em aberto (limite 1000)", r.Alerta)
	}

	lista, _ := clientes.Listar(ctx)
	if len(lista) != 1 || lista[0].TotalEmAberto != 1200 {
		t.Fatalf("clientes = %+v, want 1200 em dívida", lista)
	}
}

func TestAdminApp_VendaACreditoExigeSessao(t *testing.T) {
	app := novoAdminAppTeste(t)
	id := int64(2)
	if _, err := app.RegistarVendaACredito(usecase.VendaDiretaParams{ClienteID: &id}); err == nil {
		t.Fatal("esperava erro: venda a crédito sem administrador autenticado")
	}
	if _, err := app.CatalogoVenda(1); err == nil {
		t.Fatal("esperava erro: catálogo sem administrador autenticado")
	}
}
