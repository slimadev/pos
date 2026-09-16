// Package admin é exclusivo do executável cmd/admin (SDD 8.5, Fase 9) —
// agrega dados de várias bases de loja para o dashboard multi-loja. Nunca é
// importado por cmd/app.
package admin

import (
	"context"
	"database/sql"
	"fmt"

	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/config"
	"gestao-retalho/backend/di"
)

// AbrirLoja liga-se à base .db de uma loja e devolve o mesmo *api.App que
// cmd/app usaria — reaproveita di.CoreModule tal e qual (SDD 8.3: "o core é
// 100% partilhado"), só que num container Fx próprio e descartável em vez
// do único container de longa duração que cmd/app mantém. Cada chamada abre
// a sua própria ligação SQLite; fechar() tem de ser chamado para a libertar
// assim que o pedido de agregação terminar — o admin não mantém N ligações
// permanentes abertas, reabre a cada refresh do dashboard.
func AbrirLoja(caminhoDB string) (app *api.App, fechar func(), err error) {
	var db *sql.DB
	container := fx.New(
		fx.WithLogger(func() fxevent.Logger { return fxevent.NopLogger }),
		di.CoreModule,
		fx.Provide(func() config.DBPath { return config.DBPath(caminhoDB) }),
		fx.Populate(&app, &db),
	)
	if err := container.Err(); err != nil {
		return nil, nil, fmt.Errorf("ligar a %s: %w", caminhoDB, err)
	}

	app.OnStartup(context.Background())
	return app, func() { _ = db.Close() }, nil
}
