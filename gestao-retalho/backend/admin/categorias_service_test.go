package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func TestCategoriasService_ListarRejeitaLojaInexistente(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewCategoriasService(repo)

	_, err := svc.Listar(ctx, 999)
	if err == nil {
		t.Fatal("esperava erro para loja inexistente")
	}
}

func TestCategoriasService_ListarNuncaDevolveNil(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	lojasSvc := admin.NewLojasService(repo)
	svc := admin.NewCategoriasService(repo)

	loja := novaLojaTeste(t, "loja-sem-categorias.db", "Ferragem Central", domain.TipoNegocioFerragem)
	registada, err := lojasSvc.Criar(ctx, "Ferragem Central", loja.Caminho)
	if err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	categorias, err := svc.Listar(ctx, registada.ID)
	if err != nil {
		t.Fatalf("listar categorias: %v", err)
	}
	if categorias == nil {
		t.Error("Listar devolveu nil (want slice vazio) sem nenhuma categoria criada")
	}
}

func TestCategoriasService_CriarEListar(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	lojasSvc := admin.NewLojasService(repo)
	svc := admin.NewCategoriasService(repo)

	loja := novaLojaTeste(t, "loja-categorias.db", "Bottlestore XPTO", domain.TipoNegocioBottlestore)
	registada, err := lojasSvc.Criar(ctx, "Bottlestore XPTO", loja.Caminho)
	if err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	criada, err := svc.Criar(ctx, registada.ID, domain.Categoria{Nome: "Bebidas"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}
	if criada.Nome != "Bebidas" {
		t.Errorf("categoria criada = %+v, want nome Bebidas", criada)
	}

	// O wizard já semeia categorias por preset (bottlestore: Cervejas,
	// Refrigerantes) — "Bebidas" só precisa de aparecer entre elas.
	categorias, err := svc.Listar(ctx, registada.ID)
	if err != nil {
		t.Fatalf("listar categorias: %v", err)
	}
	achou := false
	for _, c := range categorias {
		if c.Nome == "Bebidas" {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("categorias = %+v, want incluir Bebidas", categorias)
	}
}
