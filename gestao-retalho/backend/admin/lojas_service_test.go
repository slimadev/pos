package admin_test

import (
	"context"
	"path/filepath"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

// TestLojasService_ListarNuncaDevolveNil apanha a mesma classe de bug do
// TestDashboardService_CamposNuncaSaoNilMesmoSemLojasNemDados: sem nenhuma
// loja registada, `ListarLojas()` marshalla para `null` em JSON se o
// slice Go for nil, e Lojas.tsx faz `.map()` sem guarda nenhuma.
func TestLojasService_ListarNuncaDevolveNil(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewLojasService(repo)

	lojas, err := svc.Listar(ctx)
	if err != nil {
		t.Fatalf("listar lojas: %v", err)
	}
	if lojas == nil {
		t.Error("Listar devolveu nil (want slice vazio) sem nenhuma loja registada")
	}
}

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
