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

func TestCategoriasService_AtualizarEDefinirAtiva(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	lojasSvc := admin.NewLojasService(repo)
	svc := admin.NewCategoriasService(repo)

	loja := novaLojaTeste(t, "loja-categorias-editar.db", "Ferragem Central", domain.TipoNegocioFerragem)
	registada, err := lojasSvc.Criar(ctx, "Ferragem Central", loja.Caminho)
	if err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	criada, err := svc.Criar(ctx, registada.ID, domain.Categoria{Nome: "Ferramentas"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	if err := svc.Atualizar(ctx, registada.ID, domain.Categoria{ID: criada.ID, Nome: "Ferramentas manuais"}); err != nil {
		t.Fatalf("atualizar categoria: %v", err)
	}
	if err := svc.DefinirAtiva(ctx, registada.ID, criada.ID, false); err != nil {
		t.Fatalf("desativar categoria: %v", err)
	}

	categorias, err := svc.Listar(ctx, registada.ID)
	if err != nil {
		t.Fatalf("listar categorias: %v", err)
	}
	var achada *domain.Categoria
	for i := range categorias {
		if categorias[i].ID == criada.ID {
			achada = &categorias[i]
		}
	}
	if achada == nil {
		t.Fatal("categoria criada não encontrada após listar")
	}
	if achada.Nome != "Ferramentas manuais" {
		t.Errorf("nome = %q, want Ferramentas manuais", achada.Nome)
	}
	if achada.Ativo {
		t.Error("categoria devia estar inativa após DefinirAtiva(false)")
	}
}

func TestCategoriasService_AtualizarRejeitaLojaInexistente(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewCategoriasService(repo)

	if err := svc.Atualizar(ctx, 999, domain.Categoria{ID: 1, Nome: "X"}); err == nil {
		t.Fatal("esperava erro para loja inexistente")
	}
}
