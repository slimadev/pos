package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestInventarioService_IniciarContarEFechar(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewInventarioService(repo)
	stock := admin.NewStockService(repo)

	loja := novaLojaTeste(t, "loja-inventario-admin.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}
	produto, _, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Tinta 5L", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "lata"},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "lata", QtdEmUnidadeBase: 1, PrecoCusto: 900, PrecoVenda: 1300, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if _, err := stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 12, CustoUnitario: 900}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	iniciado, err := svc.Iniciar(ctx, loja.Armazem.ID)
	if err != nil {
		t.Fatalf("iniciar: %v", err)
	}
	// A loja de teste já vem com os produtos de exemplo do preset — só o
	// criado aqui tem stock.
	var item domain.InventarioItem
	for _, i := range iniciado.Itens {
		if i.ProdutoID == produto.ID {
			item = i
		}
	}
	if item.ID == 0 || item.QuantidadeSistema != 12 {
		t.Fatalf("item da Tinta = %+v, want 12 no sistema", item)
	}
	if _, err := svc.RegistarContagem(ctx, item.ID, 10); err != nil {
		t.Fatalf("contagem: %v", err)
	}
	movimentos, err := svc.Fechar(ctx, iniciado.Inventario.ID)
	if err != nil || len(movimentos) != 1 {
		t.Fatalf("fechar: %d movimentos, err %v; want 1", len(movimentos), err)
	}

	lista, err := svc.Listar(ctx)
	if err != nil || len(lista) != 1 || lista[0].Status != domain.StatusInventarioFechado {
		t.Fatalf("listar = %+v, err %v; want 1 inventário fechado", lista, err)
	}
	stockFinal, err := stock.ListarPorArmazem(ctx, loja.Armazem.ID)
	if err != nil {
		t.Fatalf("listar stock: %v", err)
	}
	for _, si := range stockFinal {
		if si.ProdutoID == produto.ID && si.Quantidade != 10 {
			t.Fatalf("stock final da Tinta = %v, want 10", si.Quantidade)
		}
	}
}
