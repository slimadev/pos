package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func TestLojasService_CriarComWizardCriaEConfiguraLojaDeRaiz(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	loja, err := svc.CriarComWizard(ctx, admin.CriarLojaWizardParams{
		Nome:          "Ferragem Cossa",
		TipoNegocio:   domain.TipoNegocioFerragem,
		Logo:          "data:image/png;base64,iVBORw0KGgo=",
		AdminNome:     "Nelson Matola",
		AdminUsername: "nelson",
	})
	if err != nil {
		t.Fatalf("criar com wizard: %v", err)
	}
	if loja.Nome != "Ferragem Cossa" || !loja.Ativo {
		t.Errorf("loja = %+v, want nome Ferragem Cossa e ativa", loja)
	}

	app, fechar, err := admin.AbrirLoja(loja.CaminhoDB)
	if err != nil {
		t.Fatalf("abrir loja criada: %v", err)
	}
	defer fechar()

	estado, err := app.ObterEstadoConfig()
	if err != nil {
		t.Fatalf("obter estado: %v", err)
	}
	if estado.Negocio.TipoNegocio != domain.TipoNegocioFerragem {
		t.Errorf("tipo de negócio = %q, want ferragem", estado.Negocio.TipoNegocio)
	}
	if estado.Tema != domain.TemaFerragem {
		t.Errorf("tema = %q, want ferragem", estado.Tema)
	}
	if estado.Logo == "" {
		t.Error("logo não foi gravado na loja")
	}
}

func TestLojasService_CriarComWizardRejeitaTipoNegocioVazio(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	_, err := svc.CriarComWizard(ctx, admin.CriarLojaWizardParams{
		Nome:          "Sem Tipo",
		AdminNome:     "Admin",
		AdminUsername: "admin",
	})
	if err == nil {
		t.Fatal("esperava erro sem tipo de negócio")
	}
}

func TestLojasService_ObterAtualSemLojaDevolveErro(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	if _, err := svc.ObterAtual(ctx); err == nil {
		t.Fatal("esperava erro sem nenhuma loja registada")
	}
}

func TestLojasService_ObterAtualDevolveIdentidadeDaLoja(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	if _, err := svc.CriarComWizard(ctx, admin.CriarLojaWizardParams{
		Nome:          "Ferragem Cossa",
		TipoNegocio:   domain.TipoNegocioFerragem,
		AdminNome:     "Nelson Matola",
		AdminUsername: "nelson",
	}); err != nil {
		t.Fatalf("criar com wizard: %v", err)
	}

	atual, err := svc.ObterAtual(ctx)
	if err != nil {
		t.Fatalf("obter atual: %v", err)
	}
	if atual.Nome != "Ferragem Cossa" || atual.TipoNegocio != string(domain.TipoNegocioFerragem) || atual.Tema != string(domain.TemaFerragem) {
		t.Errorf("atual = %+v, want Ferragem Cossa/ferragem/ferragem", atual)
	}
}
