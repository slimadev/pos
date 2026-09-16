// Command app is the single, generic executable installed in every loja.
// The business type and active modules come from config.json at runtime
// (SDD secção 9), never from a build flag.
package main

import (
	"embed"
	"log"

	"go.uber.org/fx"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/config"
	"gestao-retalho/backend/di"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	var wailsApp *api.App

	container := fx.New(
		di.CoreModule,
		fx.Provide(config.ResolveLocalDBPath),
		fx.Populate(&wailsApp),
	)
	if err := container.Err(); err != nil {
		log.Fatalf("wire dependencies: %v", err)
	}

	err := wails.Run(&options.App{
		Title:  "Gestão de Retalho",
		Width:  1280,
		Height: 800,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 20, G: 18, B: 26, A: 1},
		OnStartup:        wailsApp.OnStartup,
		Bind: []interface{}{
			wailsApp,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}
