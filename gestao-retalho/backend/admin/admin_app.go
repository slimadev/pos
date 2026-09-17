package admin

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// AdminApp é o struct que o Wails liga ao frontend de cmd/admin — mesmo
// papel que api.App tem para cmd/app (SDD 8.1), mas fica no pacote admin
// em vez de api porque admin já depende de api (AbrirLoja devolve
// *api.App) e um import de volta criaria um ciclo.
type AdminApp struct {
	ctx context.Context

	// administradorAtualID começa em 0 ("ninguém autenticado") e só passa
	// a ser um ID real depois de Login()/CriarAdministrador() — este é o
	// único login de todo o sistema (decisão pós-Fase 10, a pedido
	// explícito: um administrador para todas as lojas, não um por loja).
	administradorAtualID int64

	Lojas      *LojasService
	Dashboard  *DashboardService
	Auth       *AuthService
	Categorias *CategoriasService
}

func NewAdminApp(lojas *LojasService, dashboard *DashboardService, auth *AuthService, categorias *CategoriasService) *AdminApp {
	return &AdminApp{Lojas: lojas, Dashboard: dashboard, Auth: auth, Categorias: categorias}
}

func (a *AdminApp) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

// --- Autenticação ------------------------------------------------------------

type SessaoAdministrador struct {
	ID       int64
	Nome     string
	Username string
}

// PrecisaConfigurarAdmin diz ao frontend se deve mostrar o ecrã de
// primeiro arranque (criar o único administrador) em vez do de login.
func (a *AdminApp) PrecisaConfigurarAdmin() (bool, error) {
	return a.Auth.PrecisaConfigurar(a.ctx)
}

func (a *AdminApp) CriarAdministrador(nome, username, password string) (SessaoAdministrador, error) {
	admin, err := a.Auth.CriarAdministrador(a.ctx, nome, username, password)
	if err != nil {
		return SessaoAdministrador{}, err
	}
	a.administradorAtualID = admin.ID
	return SessaoAdministrador{ID: admin.ID, Nome: admin.Nome, Username: admin.Username}, nil
}

// ObterPresetsNegocio expõe a mesma tabela de presets que o wizard do
// cmd/app usa (SDD 9.1) — sem sessão exigida, é só uma lista estática, e o
// primeiro arranque do cmd/admin precisa dela antes de existir qualquer
// administrador autenticado.
func (a *AdminApp) ObterPresetsNegocio() []domain.PresetNegocio {
	return domain.PresetsNegocio
}

// ConfigurarPrimeiroArranqueParams junta, num só ecrã, o que antes estava
// dividido em dois arranques separados (o wizard de uma loja em cmd/app, o
// "Configurar" do administrador em cmd/admin) — pedido explícito do
// utilizador: tipo de negócio + nome + logótipo da primeira loja, e
// user/pass do único administrador de todo o sistema.
type ConfigurarPrimeiroArranqueParams struct {
	NegocioNome   string
	TipoNegocio   domain.TipoNegocio
	Logo          string
	AdminNome     string
	AdminUsername string
	AdminPassword string
}

// ConfigurarPrimeiroArranqueResultado devolve sempre a sessão do
// administrador se este foi criado com sucesso — a criação da loja é
// tratada como um passo separado que pode falhar sem invalidar o
// administrador já criado (evita o "administrador já existe" confuso numa
// nova tentativa só por a loja ter falhado). AvisoLoja fica vazio quando
// tudo correu bem; se não, o frontend mostra o aviso mas continua para o
// dashboard — a loja pode sempre ser adicionada depois em "Lojas".
type ConfigurarPrimeiroArranqueResultado struct {
	Sessao    SessaoAdministrador
	AvisoLoja string
}

func (a *AdminApp) ConfigurarPrimeiroArranque(p ConfigurarPrimeiroArranqueParams) (ConfigurarPrimeiroArranqueResultado, error) {
	sessao, err := a.CriarAdministrador(p.AdminNome, p.AdminUsername, p.AdminPassword)
	if err != nil {
		return ConfigurarPrimeiroArranqueResultado{}, err
	}

	resultado := ConfigurarPrimeiroArranqueResultado{Sessao: sessao}
	if _, err := a.Lojas.CriarComWizard(a.ctx, CriarLojaWizardParams{
		Nome:          p.NegocioNome,
		TipoNegocio:   p.TipoNegocio,
		Logo:          p.Logo,
		AdminNome:     p.AdminNome,
		AdminUsername: p.AdminUsername,
	}); err != nil {
		resultado.AvisoLoja = err.Error()
	}
	return resultado, nil
}

func (a *AdminApp) Login(username, password string) (SessaoAdministrador, error) {
	admin, err := a.Auth.Login(a.ctx, username, password)
	if err != nil {
		return SessaoAdministrador{}, err
	}
	a.administradorAtualID = admin.ID
	return SessaoAdministrador{ID: admin.ID, Nome: admin.Nome, Username: admin.Username}, nil
}

// Logout limpa a sessão em memória — não há nada persistido em disco para
// apagar (sem servidor, sem token; a "sessão" só existe enquanto o
// processo da app está vivo).
func (a *AdminApp) Logout() {
	a.administradorAtualID = 0
}

func (a *AdminApp) exigirSessao() error {
	if a.administradorAtualID == 0 {
		return fmt.Errorf("sessão não iniciada")
	}
	return nil
}

// --- Lojas / dashboard ---------------------------------------------------------
//
// Ao contrário do cmd/app (onde utilizador_id só serve para atribuição em
// auditoria, nunca bloqueia nada), aqui a sessão é mesmo um controlo de
// acesso: sem login, nenhuma destas chamadas expõe dados de lojas nenhumas.

func (a *AdminApp) ListarLojas() ([]repository.Loja, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Lojas.Listar(a.ctx)
}

func (a *AdminApp) CriarLoja(nome, caminhoDB string) (repository.Loja, error) {
	if err := a.exigirSessao(); err != nil {
		return repository.Loja{}, err
	}
	return a.Lojas.Criar(a.ctx, nome, caminhoDB)
}

func (a *AdminApp) AtualizarLoja(id int64, nome, caminhoDB string) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Lojas.Atualizar(a.ctx, id, nome, caminhoDB)
}

func (a *AdminApp) AtivarLoja(id int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Lojas.Ativar(a.ctx, id)
}

func (a *AdminApp) DesativarLoja(id int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Lojas.Desativar(a.ctx, id)
}

func (a *AdminApp) ObterDashboard() (Dashboard, error) {
	if err := a.exigirSessao(); err != nil {
		return Dashboard{}, err
	}
	return a.Dashboard.ObterDashboard(a.ctx)
}

// --- Categorias, por loja -------------------------------------------------------
//
// Categorias são dado de negócio (vivem dentro da base da loja, nunca em
// admin.db) — estes bindings só escolhem qual loja abrir a cada chamada,
// tal como CriarLoja já valida uma loja ligando-se a ela (AbrirLoja).

func (a *AdminApp) ListarCategoriasDaLoja(lojaID int64) ([]domain.Categoria, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Categorias.Listar(a.ctx, lojaID)
}

func (a *AdminApp) CriarCategoriaDaLoja(lojaID int64, c domain.Categoria) (domain.Categoria, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.Categoria{}, err
	}
	return a.Categorias.Criar(a.ctx, lojaID, c)
}
