package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gestao-retalho/backend/config"
)

func TestResolveNovaLojaDBPath_Slugifica(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	caminho, err := config.ResolveNovaLojaDBPath("Mercearia São João")
	if err != nil {
		t.Fatalf("resolver caminho: %v", err)
	}
	if filepath.Base(string(caminho)) != "mercearia-sao-joao.db" {
		t.Errorf("caminho = %q, want basename mercearia-sao-joao.db", caminho)
	}
}

func TestResolveNovaLojaDBPath_DesambiguaColisao(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	primeiro, err := config.ResolveNovaLojaDBPath("Ferragem Cossa")
	if err != nil {
		t.Fatalf("resolver primeiro caminho: %v", err)
	}
	if err := os.WriteFile(string(primeiro), []byte("x"), 0o644); err != nil {
		t.Fatalf("criar ficheiro de teste: %v", err)
	}

	segundo, err := config.ResolveNovaLojaDBPath("Ferragem Cossa")
	if err != nil {
		t.Fatalf("resolver segundo caminho: %v", err)
	}
	if segundo == primeiro {
		t.Fatalf("segundo caminho (%q) devia diferir do primeiro (já ocupado)", segundo)
	}
	if filepath.Base(string(segundo)) != "ferragem-cossa-2.db" {
		t.Errorf("segundo caminho = %q, want basename ferragem-cossa-2.db", segundo)
	}
}

// No Android não há pasta de configuração do utilizador: a app fixa a sua
// pasta privada e tudo (admin.db, lojas, config.json) tem de ficar lá dentro.
func TestDefinirDiretorioBase_TudoDentroDaPastaFixa(t *testing.T) {
	base := t.TempDir()
	config.DefinirDiretorioBase(base)
	t.Cleanup(func() { config.DefinirDiretorioBase("") })

	admin, err := config.ResolveAdminDBPath()
	if err != nil {
		t.Fatalf("admin.db: %v", err)
	}
	loja, err := config.ResolveNovaLojaDBPath("Ferragem Teste")
	if err != nil {
		t.Fatalf("nova loja: %v", err)
	}
	cfg, err := config.ResolveConfigPath()
	if err != nil {
		t.Fatalf("config.json: %v", err)
	}
	for _, c := range []string{string(admin), string(loja), cfg} {
		if !strings.HasPrefix(c, base+string(filepath.Separator)) {
			t.Errorf("%q devia ficar dentro de %q", c, base)
		}
	}
}

func TestResolveNovaLojaDBPath_NomeVazioUsaFallback(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	caminho, err := config.ResolveNovaLojaDBPath("   ")
	if err != nil {
		t.Fatalf("resolver caminho: %v", err)
	}
	if filepath.Base(string(caminho)) != "loja.db" {
		t.Errorf("caminho = %q, want basename loja.db", caminho)
	}
}
