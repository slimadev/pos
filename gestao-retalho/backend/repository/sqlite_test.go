package repository_test

import (
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/repository"
)

func TestNewSQLiteConn_DefineBusyTimeout(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "loja.db")
	db, err := repository.NewSQLiteConn(config.DBPath(caminho))
	if err != nil {
		t.Fatalf("abrir: %v", err)
	}
	defer db.Close()

	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout;").Scan(&timeout); err != nil {
		t.Fatalf("consultar busy_timeout: %v", err)
	}
	if timeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", timeout)
	}
}

// TestNewSQLiteConn_ConexoesConcorrentesNaoFalhamComDatabaseLocked reproduz
// o cenário real desde que o cmd/app deixou de ser standalone: uma ligação
// de longa duração (o PDV) e várias ligações avulsas abertas e fechadas a
// cada pedido (admin.AbrirLoja, chamado ao clicar em Produtos/Categorias/
// Dashboard) contra o mesmo ficheiro, ao mesmo tempo. Sem PRAGMA
// busy_timeout, isto falhava com "database is locked" (SQLITE_BUSY) de
// imediato em vez de esperar a lib libertar a ligação.
func TestNewSQLiteConn_ConexoesConcorrentesNaoFalhamComDatabaseLocked(t *testing.T) {
	caminho := filepath.Join(t.TempDir(), "loja.db")

	dbLongo, err := repository.NewSQLiteConn(config.DBPath(caminho))
	if err != nil {
		t.Fatalf("abrir ligação longa: %v", err)
	}
	defer dbLongo.Close()

	var wg sync.WaitGroup
	erros := make(chan error, 20)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			db, err := repository.NewSQLiteConn(config.DBPath(caminho))
			if err != nil {
				erros <- fmt.Errorf("abrir ligação avulsa %d: %w", i, err)
				return
			}
			defer db.Close()
			if _, err := db.Exec(
				`INSERT OR REPLACE INTO configuracoes (negocio_id, chave, valor) VALUES (1, ?, ?)`,
				fmt.Sprintf("chave-%d", i), "valor",
			); err != nil {
				erros <- fmt.Errorf("escrever na ligação avulsa %d: %w", i, err)
			}
		}(i)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			if _, err := dbLongo.Exec(
				`INSERT OR REPLACE INTO configuracoes (negocio_id, chave, valor) VALUES (1, ?, ?)`,
				fmt.Sprintf("longo-%d", i), "valor",
			); err != nil {
				erros <- fmt.Errorf("escrever na ligação longa %d: %w", i, err)
			}
		}
	}()

	wg.Wait()
	close(erros)
	for err := range erros {
		t.Error(err)
	}
}
