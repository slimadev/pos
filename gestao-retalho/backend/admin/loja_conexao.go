// Package admin é exclusivo do executável cmd/admin — um administrador
// serve sempre a mesma e única loja (decisão pós-Fase 10, a pedido
// explícito). Nunca é importado por cmd/app.
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
	"gestao-retalho/backend/repository"
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

// abrirLojaUnica resolve e abre a ligação à única loja que este cmd/admin
// serve — usado por todos os serviços que operam "na loja" sem precisarem
// de conhecer o caminho do ficheiro (CategoriasService, ProdutosService,
// LojasService.ObterAtual).
func abrirLojaUnica(ctx context.Context, lojas *repository.LojaRepository) (app *api.App, fechar func(), err error) {
	loja, existe, err := lojas.ObterUnica(ctx)
	if err != nil {
		return nil, nil, err
	}
	if !existe {
		return nil, nil, fmt.Errorf("nenhuma loja configurada")
	}
	return AbrirLoja(loja.CaminhoDB)
}
