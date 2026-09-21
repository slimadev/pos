package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"gestao-retalho/backend/config"
)

func TestResolveAppExecutavel_UsaOverrideDeAmbiente(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "app-fake")
	if err := os.WriteFile(caminho, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatalf("criar executável fake: %v", err)
	}
	t.Setenv("GESTAO_RETALHO_APP_BIN", caminho)

	resolvido, err := config.ResolveAppExecutavel()
	if err != nil {
		t.Fatalf("resolver executável: %v", err)
	}
	if resolvido != caminho {
		t.Errorf("resolvido = %q, want %q", resolvido, caminho)
	}
}

func TestResolveAppExecutavel_SemOverrideNemBinarioDevolveErro(t *testing.T) {
	t.Setenv("GESTAO_RETALHO_APP_BIN", "")

	if _, err := config.ResolveAppExecutavel(); err == nil {
		t.Fatal("esperava erro sem override e sem o binário do cmd/app presente")
	}
}
