package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestCategoriaService(t *testing.T) *usecase.CategoriaService {
	t.Helper()
	db := newTestDB(t)
	return usecase.NewCategoriaService(repository.NewCategoriaRepository(db))
}

func TestCategoriaService_CriarExigeNome(t *testing.T) {
	ctx := context.Background()
	svc := newTestCategoriaService(t)

	if _, err := svc.Criar(ctx, domain.Categoria{}); err == nil {
		t.Fatal("esperava erro para categoria sem nome")
	}
}

func TestCategoriaService_CriarEListar(t *testing.T) {
	ctx := context.Background()
	svc := newTestCategoriaService(t)

	pai, err := svc.Criar(ctx, domain.Categoria{Nome: "Bebidas"})
	if err != nil {
		t.Fatalf("criar categoria pai: %v", err)
	}
	if pai.NegocioID != domain.NegocioID {
		t.Errorf("NegocioID = %d, want %d (forçado pelo service)", pai.NegocioID, domain.NegocioID)
	}

	filha, err := svc.Criar(ctx, domain.Categoria{Nome: "Cervejas", CategoriaPaiID: &pai.ID})
	if err != nil {
		t.Fatalf("criar subcategoria: %v", err)
	}
	if filha.CategoriaPaiID == nil || *filha.CategoriaPaiID != pai.ID {
		t.Errorf("CategoriaPaiID = %v, want %d", filha.CategoriaPaiID, pai.ID)
	}

	categorias, err := svc.Listar(ctx, domain.NegocioID)
	if err != nil {
		t.Fatalf("listar categorias: %v", err)
	}
	if len(categorias) != 2 {
		t.Fatalf("categorias = %d, want 2", len(categorias))
	}
}

func TestCategoriaService_CriarComecaAtiva(t *testing.T) {
	ctx := context.Background()
	svc := newTestCategoriaService(t)

	c, err := svc.Criar(ctx, domain.Categoria{Nome: "Bebidas"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}
	if !c.Ativo {
		t.Error("categoria recém-criada devia começar ativa")
	}
}

func TestCategoriaService_AtualizarExigeNome(t *testing.T) {
	ctx := context.Background()
	svc := newTestCategoriaService(t)

	c, err := svc.Criar(ctx, domain.Categoria{Nome: "Bebidas"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	if err := svc.Atualizar(ctx, domain.Categoria{ID: c.ID, Nome: ""}); err == nil {
		t.Fatal("esperava erro para nome vazio")
	}
}

func TestCategoriaService_AtualizarRenomeia(t *testing.T) {
	ctx := context.Background()
	svc := newTestCategoriaService(t)

	c, err := svc.Criar(ctx, domain.Categoria{Nome: "Bebidas"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	if err := svc.Atualizar(ctx, domain.Categoria{ID: c.ID, Nome: "Bebidas e Sumos"}); err != nil {
		t.Fatalf("atualizar categoria: %v", err)
	}

	categorias, err := svc.Listar(ctx, domain.NegocioID)
	if err != nil {
		t.Fatalf("listar categorias: %v", err)
	}
	if len(categorias) != 1 || categorias[0].Nome != "Bebidas e Sumos" {
		t.Fatalf("categorias = %+v, want 1 categoria renomeada", categorias)
	}
}

func TestCategoriaService_DesativarEAtivar(t *testing.T) {
	ctx := context.Background()
	svc := newTestCategoriaService(t)

	c, err := svc.Criar(ctx, domain.Categoria{Nome: "Bebidas"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	if err := svc.Desativar(ctx, c.ID); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	categorias, _ := svc.Listar(ctx, domain.NegocioID)
	if categorias[0].Ativo {
		t.Fatal("categoria devia estar inativa após Desativar")
	}

	if err := svc.Ativar(ctx, c.ID); err != nil {
		t.Fatalf("ativar: %v", err)
	}
	categorias, _ = svc.Listar(ctx, domain.NegocioID)
	if !categorias[0].Ativo {
		t.Fatal("categoria devia estar ativa após Ativar")
	}
}
