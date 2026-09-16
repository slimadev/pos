package repository

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"gestao-retalho/backend/config"
)

//go:embed migrations_admin/*.sql
var migrationsAdminFS embed.FS

// NewAdminSQLiteConn abre a base própria do executável de administração
// universal (SDD 8.5, Fase 9) — é só o registo de lojas conhecidas, nunca
// contém dados de negócio, por isso tem o seu próprio schema/migrations em
// vez de reaproveitar o schema completo de uma loja.
func NewAdminSQLiteConn(path config.AdminDBPath) (*sql.DB, error) {
	db, err := sql.Open("sqlite", string(path))
	if err != nil {
		return nil, fmt.Errorf("open admin sqlite at %s: %w", path, err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON;"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := migrateAdmin(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrateAdmin(db *sql.DB) error {
	source, err := iofs.New(migrationsAdminFS, "migrations_admin")
	if err != nil {
		return fmt.Errorf("load embedded admin migrations: %w", err)
	}
	target, err := sqlite.WithInstance(db, &sqlite.Config{})
	if err != nil {
		return fmt.Errorf("attach migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", source, "sqlite", target)
	if err != nil {
		return fmt.Errorf("build migrator: %w", err)
	}
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("apply admin migrations: %w", err)
	}
	return nil
}
