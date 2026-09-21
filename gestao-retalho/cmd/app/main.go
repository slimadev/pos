// Command app is the single, generic executable installed in every loja —
// o ecrã de venda (PDV). Deixou de abrir sozinho: só arranca a partir do
// botão "PDV" do cmd/admin (decisão pós-Fase 10, a pedido explícito), que
// passa o caminho da base via -db para garantir que os dois processos
// usam sempre o mesmo ficheiro .db. O tipo de negócio e módulos ativos
// continuam a vir de config.json em runtime (SDD secção 9), nunca de uma
// flag de build.
package main

import (
	"embed"
	"flag"
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
	caminhoDB := flag.String("db", "", "caminho para o ficheiro loja.db (por omissão, resolve o caminho local desta máquina — usado só para correr o cmd/app sozinho em desenvolvimento)")
	flag.Parse()

	resolverDBPath := func() (config.DBPath, error) {
		if *caminhoDB != "" {
			return config.DBPath(*caminhoDB), nil
		}
		return config.ResolveLocalDBPath()
	}

	var wailsApp *api.App

	container := fx.New(
		di.CoreModule,
		fx.Provide(resolverDBPath),
		fx.Populate(&wailsApp),
	)
	if err := container.Err(); err != nil {
		log.Fatalf("wire dependencies: %v", err)
	}

	err := wails.Run(&options.App{
		Title:            "Gestão de Retalho",
		Width:            1280,
		Height:           800,
		WindowStartState: options.Maximised,
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
