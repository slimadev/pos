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
	"gestao-retalho/backend/copias"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// Uma cópia de segurança escolhida para restaurar só troca os dados
	// agora, antes de qualquer base estar aberta.
	if aplicado, err := copias.AplicarRestauroPendente(); err != nil {
		log.Printf("restauro da cópia de segurança falhou: %v", err)
	} else if aplicado {
		log.Print("cópia de segurança restaurada")
	}

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
