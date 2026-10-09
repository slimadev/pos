package admin

import (
	"context"
	"fmt"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
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
	Produtos   *ProdutosService
	Armazens   *ArmazensService
	Stock      *StockService
	Farmacia   *FarmaciaService
	Inventario *InventarioService
	PDV        *PDVService
}

func NewAdminApp(lojas *LojasService, dashboard *DashboardService, auth *AuthService, categorias *CategoriasService, produtos *ProdutosService, armazens *ArmazensService, stock *StockService, farmacia *FarmaciaService, inventario *InventarioService, pdv *PDVService) *AdminApp {
	return &AdminApp{Lojas: lojas, Dashboard: dashboard, Auth: auth, Categorias: categorias, Produtos: produtos, Armazens: armazens, Stock: stock, Farmacia: farmacia, Inventario: inventario, PDV: pdv}
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
// dashboard, que fica vazio até a loja ficar configurada.
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

// ObterLojaAtual identifica a única loja que este cmd/admin serve — nome e
// tipo de negócio, não dados de vendas/clientes, por isso não exige sessão
// (tal como ObterPresetsNegocio): o ecrã de Login já mostra a identidade e
// a cor da loja antes de autenticar, tal como o resto da app depois de
// entrar (decisão pós-Fase 10, a pedido explícito).
func (a *AdminApp) ObterLojaAtual() (LojaAtual, error) {
	return a.Lojas.ObterAtual(a.ctx)
}

// --- Dashboard ---------------------------------------------------------------
//
// Ao contrário do cmd/app (onde utilizador_id só serve para atribuição em
// auditoria, nunca bloqueia nada), aqui a sessão é mesmo um controlo de
// acesso: sem login, nenhuma destas chamadas expõe dados da loja nenhuns.

func (a *AdminApp) ObterDashboard() (Dashboard, error) {
	if err := a.exigirSessao(); err != nil {
		return Dashboard{}, err
	}
	return a.Dashboard.ObterDashboard(a.ctx)
}

// --- Categorias ------------------------------------------------------------
//
// Categorias são dado de negócio (vivem dentro da base da loja, nunca em
// admin.db) — estes bindings só abrem a ligação à única loja a cada
// chamada, nunca mantêm uma ligação persistente (ver AbrirLoja).

func (a *AdminApp) ListarCategorias() ([]domain.Categoria, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Categorias.Listar(a.ctx)
}

func (a *AdminApp) CriarCategoria(c domain.Categoria) (domain.Categoria, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.Categoria{}, err
	}
	return a.Categorias.Criar(a.ctx, c)
}

func (a *AdminApp) AtualizarCategoria(c domain.Categoria) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Categorias.Atualizar(a.ctx, c)
}

func (a *AdminApp) AtivarCategoria(categoriaID int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Categorias.DefinirAtiva(a.ctx, categoriaID, true)
}

func (a *AdminApp) DesativarCategoria(categoriaID int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Categorias.DefinirAtiva(a.ctx, categoriaID, false)
}

// --- Produtos ----------------------------------------------------------------
//
// Produtos (e as suas unidades de venda/compra — "variações ou grupos" do
// mesmo produto) são dado de negócio (vivem dentro da base da loja, nunca
// em admin.db) — estes bindings só abrem a ligação à única loja a cada
// chamada, nunca mantêm uma ligação persistente (ver AbrirLoja).

func (a *AdminApp) ListarProdutos() ([]domain.Produto, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Produtos.Listar(a.ctx)
}

func (a *AdminApp) ListarUnidadesProduto(produtoID int64) ([]domain.UnidadeProduto, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Produtos.ListarUnidades(a.ctx, produtoID)
}

func (a *AdminApp) CriarProduto(req api.CriarProdutoRequest) (api.CriarProdutoResponse, error) {
	if err := a.exigirSessao(); err != nil {
		return api.CriarProdutoResponse{}, err
	}
	return a.Produtos.Criar(a.ctx, req)
}

func (a *AdminApp) AtualizarProduto(p domain.Produto) (domain.Produto, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.Produto{}, err
	}
	return a.Produtos.Atualizar(a.ctx, p)
}

func (a *AdminApp) AtivarProduto(produtoID int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Produtos.DefinirAtivo(a.ctx, produtoID, true)
}

func (a *AdminApp) DesativarProduto(produtoID int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Produtos.DefinirAtivo(a.ctx, produtoID, false)
}

func (a *AdminApp) DuplicarProduto(produtoID int64) (api.CriarProdutoResponse, error) {
	if err := a.exigirSessao(); err != nil {
		return api.CriarProdutoResponse{}, err
	}
	return a.Produtos.Duplicar(a.ctx, produtoID)
}

func (a *AdminApp) AdicionarUnidadeProduto(u domain.UnidadeProduto) (domain.UnidadeProduto, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.UnidadeProduto{}, err
	}
	return a.Produtos.AdicionarUnidade(a.ctx, u)
}

// --- Armazéns ----------------------------------------------------------------
//
// Armazéns são dado de negócio (vivem dentro da base da loja, nunca em
// admin.db) — estes bindings só abrem a ligação à única loja a cada
// chamada, nunca mantêm uma ligação persistente (ver AbrirLoja).

func (a *AdminApp) ListarArmazens() ([]domain.Armazem, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Armazens.Listar(a.ctx)
}

func (a *AdminApp) CriarArmazem(arm domain.Armazem) (domain.Armazem, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.Armazem{}, err
	}
	return a.Armazens.Criar(a.ctx, arm)
}

func (a *AdminApp) AtualizarArmazem(arm domain.Armazem) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Armazens.Atualizar(a.ctx, arm)
}

func (a *AdminApp) AtivarArmazem(armazemID int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Armazens.DefinirAtivo(a.ctx, armazemID, true)
}

func (a *AdminApp) DesativarArmazem(armazemID int64) error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.Armazens.DefinirAtivo(a.ctx, armazemID, false)
}

// --- Stock ---------------------------------------------------------------
//
// Stock é dado de negócio (vive dentro da base da loja, nunca em admin.db)
// — estes bindings só abrem a ligação à única loja a cada chamada, nunca
// mantêm uma ligação persistente (ver AbrirLoja).

func (a *AdminApp) ListarStockPorArmazem(armazemID int64) ([]domain.StockItem, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Stock.ListarPorArmazem(a.ctx, armazemID)
}

func (a *AdminApp) ListarMovimentosStock(produtoID int64) ([]domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Stock.ListarMovimentos(a.ctx, produtoID)
}

func (a *AdminApp) RegistarEntradaStock(p usecase.EntradaParams) (domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.MovimentoStock{}, err
	}
	return a.Stock.RegistarEntrada(a.ctx, p)
}

func (a *AdminApp) TransferirStock(p usecase.TransferenciaParams) (domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.MovimentoStock{}, err
	}
	return a.Stock.Transferir(a.ctx, p)
}

func (a *AdminApp) RegistarPerdaStock(p usecase.SaidaParams) (domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.MovimentoStock{}, err
	}
	return a.Stock.RegistarPerda(a.ctx, p)
}

func (a *AdminApp) RegistarConsumoInternoStock(p usecase.SaidaParams) (domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.MovimentoStock{}, err
	}
	return a.Stock.RegistarConsumoInterno(a.ctx, p)
}

func (a *AdminApp) RegistarEntradaComLoteStock(p usecase.EntradaComLoteParams) (api.EntradaComLoteResponse, error) {
	if err := a.exigirSessao(); err != nil {
		return api.EntradaComLoteResponse{}, err
	}
	return a.Stock.RegistarEntradaComLote(a.ctx, p)
}

func (a *AdminApp) ListarLotesStock(produtoID, armazemID int64) ([]domain.Lote, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Stock.ListarLotes(a.ctx, produtoID, armazemID)
}

func (a *AdminApp) RegistarPerdaDeLoteStock(p usecase.SaidaLoteParams) (domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.MovimentoStock{}, err
	}
	return a.Stock.RegistarPerdaDeLote(a.ctx, p)
}

func (a *AdminApp) RegistarConsumoInternoDeLoteStock(p usecase.SaidaLoteParams) (domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.MovimentoStock{}, err
	}
	return a.Stock.RegistarConsumoInternoDeLote(a.ctx, p)
}

func (a *AdminApp) TransferirLoteStock(p usecase.TransferenciaLoteParams) (domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.MovimentoStock{}, err
	}
	return a.Stock.TransferirLote(a.ctx, p)
}

// --- Inventário --------------------------------------------------------------

func (a *AdminApp) ListarInventarios() ([]domain.Inventario, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Inventario.Listar(a.ctx)
}

func (a *AdminApp) IniciarInventario(armazemID int64) (api.IniciarInventarioResponse, error) {
	if err := a.exigirSessao(); err != nil {
		return api.IniciarInventarioResponse{}, err
	}
	return a.Inventario.Iniciar(a.ctx, armazemID)
}

func (a *AdminApp) ListarInventarioItens(inventarioID int64) ([]domain.InventarioItem, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Inventario.ListarItens(a.ctx, inventarioID)
}

func (a *AdminApp) RegistarContagemInventario(itemID int64, quantidadeContada float64) (domain.InventarioItem, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.InventarioItem{}, err
	}
	return a.Inventario.RegistarContagem(a.ctx, itemID, quantidadeContada)
}

func (a *AdminApp) FecharInventario(inventarioID int64) ([]domain.MovimentoStock, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Inventario.Fechar(a.ctx, inventarioID)
}

// --- Farmácia ----------------------------------------------------------------
//
// Receitas médicas são opcionais: nunca bloqueiam a venda, registam-se
// depois, associadas a uma venda do dia.

func (a *AdminApp) ListarLotesAVencer(dentroDeDias int) ([]domain.Lote, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Farmacia.ListarLotesAVencer(a.ctx, dentroDeDias)
}

func (a *AdminApp) ListarVendasDoDia() ([]domain.Venda, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Farmacia.ListarVendasDoDia(a.ctx)
}

func (a *AdminApp) RegistarReceitaMedica(p usecase.ReceitaMedicaInput) (domain.ReceitaMedica, error) {
	if err := a.exigirSessao(); err != nil {
		return domain.ReceitaMedica{}, err
	}
	return a.Farmacia.RegistarReceitaMedica(a.ctx, p)
}

func (a *AdminApp) ListarReceitasPorVenda(vendaID int64) ([]domain.ReceitaMedica, error) {
	if err := a.exigirSessao(); err != nil {
		return nil, err
	}
	return a.Farmacia.ListarReceitasPorVenda(a.ctx, vendaID)
}

// --- PDV -----------------------------------------------------------------
//
// AbrirPDV lança o cmd/app como processo separado (decisão pós-Fase 10, a
// pedido explícito: cmd/app deixou de abrir sozinho, só arranca a partir
// daqui) — nunca renderiza nada dentro do cmd/admin, só arranca a janela.

func (a *AdminApp) AbrirPDV() error {
	if err := a.exigirSessao(); err != nil {
		return err
	}
	return a.PDV.Abrir(a.ctx)
}
