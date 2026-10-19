//go:build !windows

package export

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

func abrirFicheiro(caminho string) error {
	abs, err := filepath.Abs(caminho)
	if err != nil {
		return err
	}
	programa := "xdg-open"
	if runtime.GOOS == "darwin" {
		programa = "open"
	}
	if err := exec.Command(programa, abs).Start(); err != nil {
		return fmt.Errorf("abrir %s: %w", filepath.Base(abs), err)
	}
	return nil
}
