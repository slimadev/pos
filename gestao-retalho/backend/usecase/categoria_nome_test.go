package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func TestCategoria_NomeRepetidoERecusado(t *testing.T) {
	ctx := context.Background()
	svc := usecase.NewCategoriaService(repository.NewCategoriaRepository(newTestDB(t)))

	eletrico, err := svc.Criar(ctx, domain.Categoria{Nome: "Elétrico"})
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	for _, nome := range []string{"Elétrico", "eletrico", "  ELÉTRICO  "} {
		if _, err := svc.Criar(ctx, domain.Categoria{Nome: nome}); err == nil {
			t.Errorf("criar %q devia ser recusado (já existe Elétrico)", nome)
		}
	}
	tintas, err := svc.Criar(ctx, domain.Categoria{Nome: "Tintas"})
	if err != nil {
		t.Fatalf("criar tintas: %v", err)
	}
	eletrico.Nome = "Elétrico "
	if err := svc.Atualizar(ctx, eletrico); err != nil {
		t.Errorf("guardar a própria categoria com o mesmo nome devia passar: %v", err)
	}
	tintas.Nome = "Eletrico"
	if err := svc.Atualizar(ctx, tintas); err == nil {
		t.Error("renomear Tintas para Eletrico devia ser recusado")
	}
}
