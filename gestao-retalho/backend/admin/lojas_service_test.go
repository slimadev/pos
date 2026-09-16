package admin_test

import (
	"context"
	"path/filepath"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func TestLojasService_CriarRejeitaFicheiroInexistente(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	_, err := svc.Criar(ctx, "Loja Fantasma", filepath.Join(t.TempDir(), "nao-existe.db"))
	if err == nil {
		t.Fatal("esperava erro para ficheiro inexistente")
	}
}

func TestLojasService_CriarRejeitaLojaNaoConfigurada(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	caminho := novaBaseNaoConfigurada(t, "loja-crua.db")
	_, err := svc.Criar(ctx, "Loja Crua", caminho)
	if err == nil {
		t.Fatal("esperava erro para loja sem wizard concluído")
	}
}

func TestLojasService_CriarEListar(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	loja := novaLojaTeste(t, "loja-ok.db", "Bottlestore XPTO", domain.TipoNegocioBottlestore)

	criada, err := svc.Criar(ctx, "Bottlestore XPTO", loja.Caminho)
	if err != nil {
		t.Fatalf("criar loja: %v", err)
	}
	if !criada.Ativo {
		t.Errorf("loja recém-criada devia começar ativa")
	}

	lojas, err := svc.Listar(ctx)
	if err != nil {
		t.Fatalf("listar lojas: %v", err)
	}
	if len(lojas) != 1 || lojas[0].CaminhoDB != loja.Caminho {
		t.Fatalf("lojas = %+v, want 1 com caminho %q", lojas, loja.Caminho)
	}
}

func TestLojasService_DesativarEAtivar(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	loja := novaLojaTeste(t, "loja-toggle.db", "Ferragem Central", domain.TipoNegocioFerragem)
	criada, err := svc.Criar(ctx, "Ferragem Central", loja.Caminho)
	if err != nil {
		t.Fatalf("criar loja: %v", err)
	}

	if err := svc.Desativar(ctx, criada.ID); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	lojas, _ := svc.Listar(ctx)
	if lojas[0].Ativo {
		t.Fatal("loja devia estar inativa após Desativar")
	}

	if err := svc.Ativar(ctx, criada.ID); err != nil {
		t.Fatalf("ativar: %v", err)
	}
	lojas, _ = svc.Listar(ctx)
	if !lojas[0].Ativo {
		t.Fatal("loja devia estar ativa após Ativar")
	}
}
