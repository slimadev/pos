// Package di assembles the Fx dependency graph shared by cmd/app and
// cmd/admin (SDD secção 8.2). It is used as a builder only — fx.Populate
// extracts the finished *api.App once, it never keeps the container running,
// because wails.Run owns the process lifecycle instead.
package di

import (
	"go.uber.org/fx"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/repository"
)

// CoreModule provides every repository, usecase and the api.App on top of
// them. cmd/app and cmd/admin only differ in how they provide config.DBPath
// (one loja vs. many), everything else is 100% shared.
var CoreModule = fx.Options(
	fx.Provide(repository.NewSQLiteConn),
	fx.Provide(api.NewApp),
)
