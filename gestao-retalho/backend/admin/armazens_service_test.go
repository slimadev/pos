package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func TestArmazensService_ListarRejeitaSemLojaConfigurada(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewArmazensService(repo)

	_, err := svc.Listar(ctx)
	if err == nil {
		t.Fatal("esperava erro sem nenhuma loja registada")
	}
}

func TestArmazensService_CriarAtualizarEAtivarDesativar(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewArmazensService(repo)

	loja := novaLojaTeste(t, "loja-armazens.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	criado, err := svc.Criar(ctx, domain.Armazem{Nome: "Depósito Norte", Tipo: domain.TipoArmazemDeposito})
	if err != nil {
		t.Fatalf("criar armazém: %v", err)
	}
	if criado.Nome != "Depósito Norte" || !criado.Ativo {
		t.Errorf("armazém criado = %+v, want Depósito Norte ativo", criado)
	}

	criado.Nome = "Depósito Norte (renomeado)"
	if err := svc.Atualizar(ctx, criado); err != nil {
		t.Fatalf("atualizar armazém: %v", err)
	}

	if err := svc.DefinirAtivo(ctx, criado.ID, false); err != nil {
		t.Fatalf("desativar: %v", err)
	}

	armazens, err := svc.Listar(ctx)
	if err != nil {
		t.Fatalf("listar armazéns: %v", err)
	}
	var achado *domain.Armazem
	for i := range armazens {
		if armazens[i].ID == criado.ID {
			achado = &armazens[i]
		}
	}
	if achado == nil {
		t.Fatal("armazém criado não encontrado após listar")
	}
	if achado.Nome != "Depósito Norte (renomeado)" {
		t.Errorf("nome = %q, want Depósito Norte (renomeado)", achado.Nome)
	}
	if achado.Ativo {
		t.Error("armazém devia estar inativo após DefinirAtivo(false)")
	}

	if err := svc.DefinirAtivo(ctx, criado.ID, true); err != nil {
		t.Fatalf("ativar: %v", err)
	}
}
