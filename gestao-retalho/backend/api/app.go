// Package api exposes the struct Wails binds to the frontend. Every method
// on App becomes a callable TypeScript function via `wails generate`.
package api

import (
	"context"
	"database/sql"
	"fmt"
)

// App is the single binding surface between the Go backend and the React
// frontend (SDD secção 8.1). It starts empty in Fase 0 and grows one field
// per usecase service as later fases land.
type App struct {
	ctx context.Context
	db  *sql.DB
}

// NewApp wires App's dependencies via Fx.
func NewApp(db *sql.DB) *App {
	return &App{db: db}
}

// OnStartup is called by Wails once the frontend is ready to receive
// runtime calls. Kept as a plain method (not part of the Fx graph) so main.go
// can pass it straight to wails.Run's OnStartup hook.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

// Ping is a scaffolding smoke test: it proves the Go↔React binding pipeline
// and the SQLite connection both work end to end. Safe to remove once real
// services (ProdutoService, StockService, ...) exist.
func (a *App) Ping() (string, error) {
	if err := a.db.PingContext(a.ctx); err != nil {
		return "", fmt.Errorf("ping db: %w", err)
	}
	return "pong", nil
}
