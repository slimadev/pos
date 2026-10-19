package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func TestCategoriasService_ListarRejeitaSemLojaConfigurada(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewCategoriasService(repo)

	_, err := svc.Listar(ctx)
	if err == nil {
		t.Fatal("esperava erro sem nenhuma loja registada")
	}
}

func TestCategoriasService_ListarNuncaDevolveNil(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewCategoriasService(repo)

	loja := novaLojaTeste(t, "loja-sem-categorias.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	categorias, err := svc.Listar(ctx)
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
	svc := admin.NewCategoriasService(repo)

	loja := novaLojaTeste(t, "loja-categorias.db", "Bottlestore XPTO", domain.TipoNegocioBottlestore)
	if _, err := repo.Criar(ctx, "Bottlestore XPTO", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	criada, err := svc.Criar(ctx, domain.Categoria{Nome: "Bebidas"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}
	if criada.Nome != "Bebidas" {
		t.Errorf("categoria criada = %+v, want nome Bebidas", criada)
	}

	// O wizard já semeia categorias por preset (bottlestore: Cervejas,
	// Refrigerantes) — "Bebidas" só precisa de aparecer entre elas.
	categorias, err := svc.Listar(ctx)
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
	svc := admin.NewCategoriasService(repo)

	loja := novaLojaTeste(t, "loja-categorias-editar.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	// "Ferramentas" já vem no preset de ferragem — nomes repetidos são recusados.
	criada, err := svc.Criar(ctx, domain.Categoria{Nome: "Jardinagem"})
	if err != nil {
		t.Fatalf("criar categoria: %v", err)
	}

	if err := svc.Atualizar(ctx, domain.Categoria{ID: criada.ID, Nome: "Jardinagem e rega"}); err != nil {
		t.Fatalf("atualizar categoria: %v", err)
	}
	if err := svc.DefinirAtiva(ctx, criada.ID, false); err != nil {
		t.Fatalf("desativar categoria: %v", err)
	}

	categorias, err := svc.Listar(ctx)
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
	if achada.Nome != "Jardinagem e rega" {
		t.Errorf("nome = %q, want Ferramentas manuais", achada.Nome)
	}
	if achada.Ativo {
		t.Error("categoria devia estar inativa após DefinirAtiva(false)")
	}
}

func TestCategoriasService_AtualizarRejeitaSemLojaConfigurada(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewCategoriasService(repo)

	if err := svc.Atualizar(ctx, domain.Categoria{ID: 1, Nome: "X"}); err == nil {
		t.Fatal("esperava erro sem nenhuma loja registada")
	}
}
