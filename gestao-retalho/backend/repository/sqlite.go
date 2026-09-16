// Package repository holds the sqlite-backed data access layer.
package repository

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"

	"gestao-retalho/backend/config"
)

// NewSQLiteConn opens the loja's database with WAL mode and foreign keys
// enabled. A single *sql.DB is shared by every repository via Fx.
func NewSQLiteConn(path config.DBPath) (*sql.DB, error) {
	db, err := sql.Open("sqlite", string(path))
	if err != nil {
		return nil, fmt.Errorf("open sqlite at %s: %w", path, err)
	}

	// SQLite only allows one writer at a time; WAL mode lets readers keep
	// going while a write is in flight, which matters for a POS screen
	// that reads stock while another screen writes a venda.
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if err := Migrate(db); err != nil {
		return nil, err
	}

	return db, nil
}
