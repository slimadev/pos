package admin

import (
	"context"

	"gestao-retalho/backend/repository"
)

// AdminApp é o struct que o Wails liga ao frontend de cmd/admin — mesmo
// papel que api.App tem para cmd/app (SDD 8.1), mas fica no pacote admin
// em vez de api porque admin já depende de api (AbrirLoja devolve
// *api.App) e um import de volta criaria um ciclo.
type AdminApp struct {
	ctx       context.Context
	Lojas     *LojasService
	Dashboard *DashboardService
}

func NewAdminApp(lojas *LojasService, dashboard *DashboardService) *AdminApp {
	return &AdminApp{Lojas: lojas, Dashboard: dashboard}
}

func (a *AdminApp) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

func (a *AdminApp) ListarLojas() ([]repository.Loja, error) {
	return a.Lojas.Listar(a.ctx)
}

func (a *AdminApp) CriarLoja(nome, caminhoDB string) (repository.Loja, error) {
	return a.Lojas.Criar(a.ctx, nome, caminhoDB)
}

func (a *AdminApp) AtualizarLoja(id int64, nome, caminhoDB string) error {
	return a.Lojas.Atualizar(a.ctx, id, nome, caminhoDB)
}

func (a *AdminApp) AtivarLoja(id int64) error {
	return a.Lojas.Ativar(a.ctx, id)
}

func (a *AdminApp) DesativarLoja(id int64) error {
	return a.Lojas.Desativar(a.ctx, id)
}

func (a *AdminApp) ObterDashboard() (Dashboard, error) {
	return a.Dashboard.ObterDashboard(a.ctx)
}
