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
