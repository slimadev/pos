//go:build windows

package export

import (
	"fmt"
	"path/filepath"

	"golang.org/x/sys/windows"
)

func abrirFicheiro(caminho string) error {
	abs, err := filepath.Abs(caminho)
	if err != nil {
		return err
	}
	verbo := windows.StringToUTF16Ptr("open")
	if err := windows.ShellExecute(0, verbo, windows.StringToUTF16Ptr(abs), nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("abrir %s: %w", filepath.Base(abs), err)
	}
	return nil
}
