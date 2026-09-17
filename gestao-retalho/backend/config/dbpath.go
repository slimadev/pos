// Package config resolves where the loja's local files live: the SQLite
// database and, from Fase 2 onward, config.json with the negócio's wizard
// settings.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DBPath is the filesystem path to a loja's SQLite database file.
type DBPath string

// AdminDBPath is the filesystem path to the admin executable's own
// registry database (Fase 9, SDD 8.5) — distinct type from DBPath so Fx
// never wires one in place of the other by mistake.
type AdminDBPath string

const appDirName = "gestao-retalho"
const adminDirName = "gestao-retalho-admin"

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

// ResolveNovaLojaDBPath returns a fresh .db path for a loja created
// directly from the admin's first-run setup (Fase 10 addendum) — one file
// per loja in a "lojas" subfolder next to the admin's own data, named after
// a slug of nome with a numeric suffix if that slug is already taken.
func ResolveNovaLojaDBPath(nome string) (DBPath, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, adminDirName, "lojas")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	slug := slugify(nome)
	if slug == "" {
		slug = "loja"
	}
	caminho := filepath.Join(dir, slug+".db")
	for i := 2; ; i++ {
		if _, err := os.Stat(caminho); os.IsNotExist(err) {
			break
		}
		caminho = filepath.Join(dir, fmt.Sprintf("%s-%d.db", slug, i))
	}
	return DBPath(caminho), nil
}

var acentos = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "ô", "o", "õ", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

func slugify(s string) string {
	s = acentos.Replace(strings.ToLower(strings.TrimSpace(s)))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// ResolveAdminDBPath returns the path to the admin executable's own
// registry database, creating the containing directory if needed. It lives
// in a sibling folder to any loja's data dir (never inside one) — the
// admin can run on the same machine as a loja without the two databases
// colliding.
func ResolveAdminDBPath() (AdminDBPath, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, adminDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return AdminDBPath(filepath.Join(dir, "admin.db")), nil
}
