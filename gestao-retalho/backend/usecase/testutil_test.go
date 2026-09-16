package usecase_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/repository"
)

// newTestDB abre uma base SQLite num ficheiro temporário e corre as
// migrations reais (o mesmo caminho que cmd/app usa em produção).
func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dbPath := config.DBPath(filepath.Join(t.TempDir(), "test.db"))
	db, err := repository.NewSQLiteConn(dbPath)
	if err != nil {
		t.Fatalf("abrir base de teste: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
