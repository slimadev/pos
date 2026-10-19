package admin_test

import (
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func TestAdminApp_AtualizarIdentidadeLojaMudaNomeELogo(t *testing.T) {
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

	logo := "data:image/png;base64,iVBORw0KGgo="
	loja, err := app.AtualizarIdentidadeLoja("  Ferragem Cossa & Filhos ", logo)
	if err != nil {
		t.Fatalf("atualizar identidade: %v", err)
	}
	if loja.Nome != "Ferragem Cossa & Filhos" || loja.Logo != logo {
		t.Fatalf("loja = %+v, want nome novo e logótipo", loja)
	}
	if loja.TipoNegocio != string(domain.TipoNegocioFerragem) {
		t.Errorf("tipo = %q, want ferragem (não muda)", loja.TipoNegocio)
	}

	// Sem logótipo volta ao quadrado de cor.
	loja, err = app.AtualizarIdentidadeLoja("Ferragem Cossa & Filhos", "")
	if err != nil || loja.Logo != "" {
		t.Fatalf("remover logótipo: %+v, %v", loja, err)
	}

	if _, err := app.AtualizarIdentidadeLoja("", ""); err == nil {
		t.Error("esperava erro sem nome")
	}
	if _, err := app.AtualizarIdentidadeLoja("Loja", "isto não é imagem"); err == nil {
		t.Error("esperava erro com logótipo que não é imagem")
	}

	app.Logout()
	if _, err := app.AtualizarIdentidadeLoja("Outra", ""); err == nil {
		t.Error("esperava erro sem administrador autenticado")
	}
}
