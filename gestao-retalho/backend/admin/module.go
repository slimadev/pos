package admin

import (
	"go.uber.org/fx"

	"gestao-retalho/backend/repository"
)

// Module wires cmd/admin's own small graph: admin.db (só o registo de
// lojas, SDD 8.5) e os serviços de agregação. Vive neste pacote, não em
// backend/di, porque AbrirLoja (neste pacote) já importa backend/di para
// reaproveitar CoreModule por loja — um di.AdminModule que importasse este
// pacote de volta criaria um ciclo.
var Module = fx.Options(
	fx.Provide(repository.NewAdminSQLiteConn),
	fx.Provide(repository.NewLojaRepository),
	fx.Provide(repository.NewAdministradorRepository),

	fx.Provide(NewLojasService),
	fx.Provide(NewDashboardService),
	fx.Provide(NewAuthService),
	fx.Provide(NewCategoriasService),
	fx.Provide(NewProdutosService),
	fx.Provide(NewPDVService),

	fx.Provide(NewAdminApp),
)
