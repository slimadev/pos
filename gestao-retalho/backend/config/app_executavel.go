package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// ResolveAppExecutavel encontra o executável do cmd/app (o ecrã de venda)
// para o cmd/admin poder lançá-lo como processo — cmd/app deixou de abrir
// sozinho, só arranca a partir do botão "PDV" (decisão pós-Fase 10, a
// pedido explícito). Três formas, por ordem:
//  1. GESTAO_RETALHO_APP_BIN — override explícito, sempre vence.
//  2. Ao lado do próprio executável do cmd/admin — o layout esperado de
//     uma instalação real, os dois binários lado a lado na mesma pasta.
//  3. A pasta irmã `cmd/app/build/bin` dentro deste repositório — só serve
//     em desenvolvimento, onde `make build-app`/`make build-admin` deixam
//     os binários em pastas separadas dentro do mesmo checkout.
func ResolveAppExecutavel() (string, error) {
	if p := os.Getenv("GESTAO_RETALHO_APP_BIN"); p != "" {
		return p, nil
	}

	nome := "gestao-retalho"
	if runtime.GOOS == "windows" {
		nome += ".exe"
	}

	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolver executável do cmd/admin: %w", err)
	}
	dir := filepath.Dir(exePath)

	candidato := filepath.Join(dir, nome)
	if _, err := os.Stat(candidato); err == nil {
		return candidato, nil
	}

	candidato = filepath.Join(dir, "..", "..", "..", "app", "build", "bin", nome)
	if _, err := os.Stat(candidato); err == nil {
		return candidato, nil
	}

	return "", fmt.Errorf("executável do cmd/app não encontrado ao lado de %s nem em %s — defina GESTAO_RETALHO_APP_BIN", dir, candidato)
}
