package repository

import (
	"context"
	"database/sql"
)

// runInTx starts a transaction, runs fn with a tx-scoped *sql.Tx, and
// commits on success or rolls back on any error (including a panic, which
// it re-raises after rollback). Every multi-statement write in this layer
// goes through this so a crash mid-operation never leaves stock half moved.
func runInTx(ctx context.Context, db *sql.DB, fn func(tx *sql.Tx) error) (err error) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
