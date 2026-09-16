// Package di assembles the Fx dependency graph for a single loja (SDD
// secção 8.2). It is used as a builder only — fx.Populate extracts the
// finished *api.App once, it never keeps the container running, because
// something else owns the process lifecycle instead: wails.Run for
// cmd/app, or admin.AbrirLoja's caller for cmd/admin (Fase 9, SDD 8.5),
// which opens one of these containers per loja per dashboard refresh and
// closes it again afterwards.
package di

import (
	"go.uber.org/fx"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// CoreModule provides every repository, usecase and the api.App on top of
// them for one loja's database. cmd/app wires it once, for its own
// config.DBPath, and keeps it running for the process lifetime. cmd/admin
// (Fase 9) never wires it directly — it goes through admin.AbrirLoja, which
// wires a fresh instance of this same module per loja, every time the
// dashboard aggregates data, and tears it down right after.
var CoreModule = fx.Options(
	fx.Provide(repository.NewSQLiteConn),

	fx.Provide(repository.NewProdutoRepository),
	fx.Provide(repository.NewArmazemRepository),
	fx.Provide(repository.NewStockRepository),
	fx.Provide(repository.NewInventarioRepository),
	fx.Provide(repository.NewNegocioRepository),
	fx.Provide(repository.NewConfigRepository),
	fx.Provide(repository.NewUtilizadorRepository),
	fx.Provide(repository.NewCategoriaRepository),
	fx.Provide(repository.NewClienteRepository),
	fx.Provide(repository.NewLoteRepository),
	fx.Provide(repository.NewReceitaMedicaRepository),
	fx.Provide(repository.NewVendaRepository),
	fx.Provide(repository.NewFichaTecnicaRepository),
	fx.Provide(repository.NewDividaRepository),
	fx.Provide(repository.NewRelatorioRepository),

	fx.Provide(usecase.NewProdutoService),
	fx.Provide(usecase.NewArmazemService),
	fx.Provide(usecase.NewStockService),
	fx.Provide(usecase.NewConfigService),
	fx.Provide(usecase.NewClienteService),
	fx.Provide(usecase.NewVendaService),
	fx.Provide(usecase.NewReciboService),
	fx.Provide(usecase.NewDividaService),
	fx.Provide(usecase.NewContaService),
	fx.Provide(usecase.NewFichaTecnicaService),
	fx.Provide(usecase.NewFarmaciaService),
	fx.Provide(usecase.NewRelatorioService),

	fx.Provide(api.NewApp),
)
