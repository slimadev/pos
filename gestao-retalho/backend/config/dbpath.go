// Package config resolves where the loja's local files live: the SQLite
// database and, from Fase 2 onward, config.json with the negócio's wizard
// settings.
package config

import (
	"os"
	"path/filepath"
)

// DBPath is the filesystem path to a loja's SQLite database file.
type DBPath string

const appDirName = "gestao-retalho"

// ResolveLocalDBPath returns the path to the current loja's database file,
// creating the containing directory if needed. It lives under the OS user
// config directory so the app works the same on a fresh machine without any
// setup step.
func ResolveLocalDBPath() (DBPath, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return DBPath(filepath.Join(dir, "loja.db")), nil
}

// ResolveConfigPath returns the path to config.json for the current loja.
func ResolveConfigPath() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// ResolveRecibosDir returns the folder recibos (receipt PDFs) are written
// to, creating it if needed.
func ResolveRecibosDir() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	recibosDir := filepath.Join(dir, "recibos")
	if err := os.MkdirAll(recibosDir, 0o755); err != nil {
		return "", err
	}
	return recibosDir, nil
}

// ResolveRelatoriosDir returns the folder exported reports (Excel/PDF,
// Fase 8) are written to, creating it if needed.
func ResolveRelatoriosDir() (string, error) {
	dir, err := dataDir()
	if err != nil {
		return "", err
	}
	relatoriosDir := filepath.Join(dir, "relatorios")
	if err := os.MkdirAll(relatoriosDir, 0o755); err != nil {
		return "", err
	}
	return relatoriosDir, nil
}

func dataDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appDirName), nil
}
