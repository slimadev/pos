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
	loja, err := app.AtualizarIdentidadeLoja("  Ferragem Cossa & Filhos ", logo, "")
	if err != nil {
		t.Fatalf("atualizar identidade: %v", err)
	}
	if loja.Nome != "Ferragem Cossa & Filhos" || loja.Logo != logo {
		t.Fatalf("loja = %+v, want nome novo e logótipo", loja)
	}
	if loja.PapelRecibo != "80" {
		t.Errorf("papel = %q, want 80 por omissão", loja.PapelRecibo)
	}
	if loja.TipoNegocio != string(domain.TipoNegocioFerragem) {
		t.Errorf("tipo = %q, want ferragem (não muda)", loja.TipoNegocio)
	}

	// Sem logótipo volta ao quadrado de cor.
	loja, err = app.AtualizarIdentidadeLoja("Ferragem Cossa & Filhos", "", "")
	if err != nil || loja.Logo != "" {
		t.Fatalf("remover logótipo: %+v, %v", loja, err)
	}

	if _, err := app.AtualizarIdentidadeLoja("", "", ""); err == nil {
		t.Error("esperava erro sem nome")
	}
	if _, err := app.AtualizarIdentidadeLoja("Loja", "isto não é imagem", ""); err == nil {
		t.Error("esperava erro com logótipo que não é imagem")
	}

	loja, err = app.AtualizarIdentidadeLoja("Ferragem Cossa & Filhos", "", "58")
	if err != nil || loja.PapelRecibo != "58" {
		t.Fatalf("papel 58: %+v, %v", loja, err)
	}
	if _, err := app.AtualizarIdentidadeLoja("Loja", "", "100"); err == nil {
		t.Error("esperava erro com papel desconhecido")
	}

	app.Logout()
	if _, err := app.AtualizarIdentidadeLoja("Outra", "", ""); err == nil {
		t.Error("esperava erro sem administrador autenticado")
	}
}
