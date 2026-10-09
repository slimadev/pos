package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func novoAdminAppTeste(t *testing.T) *admin.AdminApp {
	t.Helper()
	repo := novoLojaRepositoryTeste(t)
	administradores := novoAdministradorRepositoryTeste(t)
	auth := admin.NewAuthService(administradores)
	app := admin.NewAdminApp(
		admin.NewLojasService(repo),
		admin.NewDashboardService(repo),
		auth,
		admin.NewCategoriasService(repo),
		admin.NewProdutosService(repo),
		admin.NewArmazensService(repo),
		admin.NewStockService(repo),
		admin.NewFarmaciaService(repo),
		admin.NewInventarioService(repo),
		admin.NewVendasService(repo),
		admin.NewRelatoriosService(repo),
		admin.NewClientesService(repo),
		admin.NewPDVServiceComExecutavel(repo, func() (string, error) { return "/bin/echo", nil }),
	)
	app.OnStartup(context.Background())
	return app
}

// TestAdminApp_ConfigurarPrimeiroArranqueCriaAdministradorELoja cobre o
// fluxo pedido: configuração (tipo de negócio + nome + logótipo + user/pass
// do administrador) → sessão ativa de imediato → "Admin Geral" já vê a
// primeira loja registada.
func TestAdminApp_ConfigurarPrimeiroArranqueCriaAdministradorELoja(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := novoAdminAppTeste(t)

	resultado, err := app.ConfigurarPrimeiroArranque(admin.ConfigurarPrimeiroArranqueParams{
		NegocioNome:   "Ferragem Cossa",
		TipoNegocio:   domain.TipoNegocioFerragem,
		Logo:          "data:image/png;base64,iVBORw0KGgo=",
		AdminNome:     "Nelson Matola",
		AdminUsername: "nelson",
		AdminPassword: "admin123",
	})
	if err != nil {
		t.Fatalf("configurar primeiro arranque: %v", err)
	}
	if resultado.AvisoLoja != "" {
		t.Errorf("avisoLoja = %q, want vazio (loja devia ter sido criada)", resultado.AvisoLoja)
	}
	if resultado.Sessao.Username != "nelson" {
		t.Errorf("sessão = %+v, want username nelson", resultado.Sessao)
	}

	// A sessão já deve estar ativa — sem precisar de Login() depois.
	loja, err := app.ObterLojaAtual()
	if err != nil {
		t.Fatalf("obter loja atual com sessão recém-criada: %v", err)
	}
	if loja.Nome != "Ferragem Cossa" || loja.TipoNegocio != string(domain.TipoNegocioFerragem) {
		t.Fatalf("loja = %+v, want Ferragem Cossa/ferragem", loja)
	}
}

// TestAdminApp_ConfigurarPrimeiroArranqueComLojaInvalidaMantemAdministrador
// verifica que uma falha ao criar a loja não deixa o sistema sem
// administrador nenhum — evita o "administrador já existe" confuso numa
// nova tentativa só porque a parte da loja falhou.
func TestAdminApp_ConfigurarPrimeiroArranqueComLojaInvalidaMantemAdministrador(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := novoAdminAppTeste(t)

	resultado, err := app.ConfigurarPrimeiroArranque(admin.ConfigurarPrimeiroArranqueParams{
		NegocioNome:   "Sem Tipo",
		TipoNegocio:   "", // força CriarComWizard a falhar
		AdminNome:     "Nelson Matola",
		AdminUsername: "nelson",
		AdminPassword: "admin123",
	})
	if err != nil {
		t.Fatalf("configurar primeiro arranque: %v", err)
	}
	if resultado.AvisoLoja == "" {
		t.Error("esperava AvisoLoja preenchido quando a loja falha")
	}

	precisa, err := app.PrecisaConfigurarAdmin()
	if err != nil {
		t.Fatalf("precisa configurar: %v", err)
	}
	if precisa {
		t.Error("administrador devia ter sido criado mesmo com a loja a falhar")
	}
}

// TestAdminApp_ObterLojaAtualNaoExigeSessao cobre o pedido de colorir o
// ecrã de Login com a cor do tipo de negócio antes de autenticar — nome e
// tipo da loja não são dado sensível (ao contrário do dashboard), por isso
// esta chamada tem de funcionar mesmo sem login nenhum.
func TestAdminApp_ObterLojaAtualNaoExigeSessao(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := novoAdminAppTeste(t)

	if _, err := app.ConfigurarPrimeiroArranque(admin.ConfigurarPrimeiroArranqueParams{
		NegocioNome:   "Ferragem Cossa",
		TipoNegocio:   domain.TipoNegocioFerragem,
		AdminNome:     "Nelson Matola",
		AdminUsername: "nelson",
		AdminPassword: "admin123",
	}); err != nil {
		t.Fatalf("configurar primeiro arranque: %v", err)
	}
	app.Logout()

	loja, err := app.ObterLojaAtual()
	if err != nil {
		t.Fatalf("obter loja atual sem sessão: %v", err)
	}
	if loja.Nome != "Ferragem Cossa" || loja.TipoNegocio != string(domain.TipoNegocioFerragem) {
		t.Fatalf("loja = %+v, want Ferragem Cossa/ferragem", loja)
	}
}
