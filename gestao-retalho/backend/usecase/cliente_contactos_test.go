package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func TestClienteService_GuardaEmailEValidaNUIT(t *testing.T) {
	ctx := context.Background()
	svc := usecase.NewClienteService(repository.NewClienteRepository(newTestDB(t)))
	str := func(s string) *string { return &s }

	c, err := svc.Criar(ctx, domain.Cliente{Nome: "Construções Matola", NIF: str(" 400 555 666 "), Email: str(" compras@matola.co.mz ")})
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	obtido, err := svc.Obter(ctx, c.ID)
	if err != nil {
		t.Fatalf("obter: %v", err)
	}
	if obtido.NIF == nil || *obtido.NIF != "400555666" || obtido.Email == nil || *obtido.Email != "compras@matola.co.mz" {
		t.Errorf("cliente = NIF %v, email %v; want 400555666 e compras@matola.co.mz", obtido.NIF, obtido.Email)
	}

	if _, err := svc.Criar(ctx, domain.Cliente{Nome: "X", NIF: str("123")}); err == nil {
		t.Error("esperava erro com NUIT de 3 algarismos")
	}
	if _, err := svc.Criar(ctx, domain.Cliente{Nome: "X", Email: str("sem-arroba")}); err == nil {
		t.Error("esperava erro com email sem @")
	}
	obtido.Email = str("")
	if err := svc.Atualizar(ctx, obtido); err != nil {
		t.Fatalf("limpar email: %v", err)
	}
	if o, _ := svc.Obter(ctx, c.ID); o.Email != nil {
		t.Errorf("email = %v, want nil depois de limpar", *o.Email)
	}
}
