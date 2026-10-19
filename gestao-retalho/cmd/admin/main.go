// Command admin is the universal administration executable (SDD 8.3/8.5,
// Fase 9): a Wails app of its own, separate from cmd/app, that aggregates
// data across every loja registered in its own small admin.db instead of
// serving a single loja.
package main

import (
	"embed"
	"log"

	"go.uber.org/fx"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/config"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	var adminApp *admin.AdminApp

	container := fx.New(
		admin.Module,
		fx.Provide(config.ResolveAdminDBPath),
		fx.Populate(&adminApp),
	)
	if err := container.Err(); err != nil {
		log.Fatalf("wire dependencies: %v", err)
	}

	err := wails.Run(&options.App{
		Title:  "Gestão de Retalho — Administração",
		Width:  1280,
		Height: 800,
		// Maximizado como o PDV: nos terminais de toque o ecrã é pequeno.
		WindowStartState: options.Maximised,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 247, G: 248, B: 250, A: 1},
		OnStartup:        adminApp.OnStartup,
		Bind: []interface{}{
			adminApp,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
