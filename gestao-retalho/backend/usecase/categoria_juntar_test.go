package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func TestCategoriaService_JuntarPassaProdutosESubcategorias(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	catRepo := repository.NewCategoriaRepository(db)
	categorias := usecase.NewCategoriaService(catRepo)
	produtos := usecase.NewProdutoService(repository.NewProdutoRepository(db))

	destino, err := categorias.Criar(ctx, domain.Categoria{Nome: "Elétrico"})
	if err != nil {
		t.Fatalf("criar destino: %v", err)
	}
	// A repetida, como as criadas antes de os nomes repetidos serem recusados.
	origem, err := catRepo.Criar(ctx, domain.Categoria{NegocioID: domain.NegocioID, Nome: "Elétrico"})
	if err != nil {
		t.Fatalf("criar origem: %v", err)
	}
	sub, err := categorias.Criar(ctx, domain.Categoria{Nome: "Lâmpadas", CategoriaPaiID: &origem.ID})
	if err != nil {
		t.Fatalf("criar subcategoria: %v", err)
	}
	for _, nome := range []string{"Cabo 2,5mm", "Tomada"} {
		if _, _, err := produtos.Criar(ctx, usecase.NovoProduto{
			Produto:  domain.Produto{Nome: nome, CategoriaID: &origem.ID, TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
			Unidades: []domain.UnidadeProduto{novaUnidadeTeste("unidade", 1, 10, 20)},
		}); err != nil {
			t.Fatalf("criar %s: %v", nome, err)
		}
	}

	movidos, err := categorias.Juntar(ctx, origem.ID, destino.ID)
	if err != nil {
		t.Fatalf("juntar: %v", err)
	}
	if movidos != 2 {
		t.Errorf("produtos movidos = %d, want 2", movidos)
	}

	todos, _ := produtos.ListarTodos(ctx, domain.NegocioID)
	for _, p := range todos {
		if p.CategoriaID == nil || *p.CategoriaID != destino.ID {
			t.Errorf("%s ficou na categoria %v, want %d", p.Nome, p.CategoriaID, destino.ID)
		}
	}
	if err := categorias.Ativar(ctx, origem.ID); err == nil {
		t.Error("esperava erro ao reativar a repetida com a outra ainda ativa")
	}

	lista, _ := categorias.Listar(ctx, domain.NegocioID)
	for _, c := range lista {
		switch c.ID {
		case origem.ID:
			if c.Ativo {
				t.Error("a categoria de origem continua ativa")
			}
		case sub.ID:
			if c.CategoriaPaiID == nil || *c.CategoriaPaiID != destino.ID {
				t.Errorf("subcategoria ficou em %v, want %d", c.CategoriaPaiID, destino.ID)
			}
		}
	}
}

func TestCategoriaService_JuntarRecusaAMesmaOuDesativada(t *testing.T) {
	ctx := context.Background()
	svc := newTestCategoriaService(t)
	a, _ := svc.Criar(ctx, domain.Categoria{Nome: "Tintas"})
	b, _ := svc.Criar(ctx, domain.Categoria{Nome: "Pintura"})

	if _, err := svc.Juntar(ctx, a.ID, a.ID); err == nil {
		t.Error("esperava erro ao juntar uma categoria com ela própria")
	}
	if err := svc.Desativar(ctx, b.ID); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	if _, err := svc.Juntar(ctx, a.ID, b.ID); err == nil {
		t.Error("esperava erro ao juntar numa categoria desativada")
	}
}
