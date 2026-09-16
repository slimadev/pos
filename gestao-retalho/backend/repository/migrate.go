package repository

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies every pending migration in backend/repository/migrations
// to db. It is called once at startup (see NewSQLiteConn) so every consumer
// of *sql.DB always sees the current schema — there is no separate "run
// migrations" step to forget in dev or in a fresh loja install.
func Migrate(db *sql.DB) error {
	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("load embedded migrations: %w", err)
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
		return fmt.Errorf("apply migrations: %w", err)
	}
	return nil
}
