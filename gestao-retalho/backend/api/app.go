// Package api exposes the struct Wails binds to the frontend. Every method
// on App becomes a callable TypeScript function via `wails generate`
// (SDD secção 8.1) — App only forwards to usecase services, no business
// logic lives here.
package api

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

// UtilizadorAtualID fica fixo no utilizador semente (domain.UtilizadorAdminSeedID)
// até existir sessão/login reais — a partir daí este valor passa a vir do
// contexto de autenticação em vez de uma constante. O wizard (Fase 2)
// já preenche os dados reais desse utilizador, só o mecanismo de "sessão
// atual" é que continua adiado.
const UtilizadorAtualID = domain.UtilizadorAdminSeedID

type App struct {
	ctx context.Context
	db  *sql.DB

	ProdutoService      *usecase.ProdutoService
	ArmazemService      *usecase.ArmazemService
	StockService        *usecase.StockService
	ConfigService       *usecase.ConfigService
	ClienteService      *usecase.ClienteService
	VendaService        *usecase.VendaService
	ReciboService       *usecase.ReciboService
	ContaService        *usecase.ContaService
	FichaTecnicaService *usecase.FichaTecnicaService
	FarmaciaService     *usecase.FarmaciaService
}

func NewApp(
	db *sql.DB,
	produto *usecase.ProdutoService,
	armazem *usecase.ArmazemService,
	stock *usecase.StockService,
	cfg *usecase.ConfigService,
	cliente *usecase.ClienteService,
	venda *usecase.VendaService,
	recibo *usecase.ReciboService,
	conta *usecase.ContaService,
	fichas *usecase.FichaTecnicaService,
	farmacia *usecase.FarmaciaService,
) *App {
	return &App{
		db:                  db,
		ProdutoService:      produto,
		ArmazemService:      armazem,
		StockService:        stock,
		ConfigService:       cfg,
		ClienteService:      cliente,
		VendaService:        venda,
		ReciboService:       recibo,
		ContaService:        conta,
		FichaTecnicaService: fichas,
		FarmaciaService:     farmacia,
	}
}

// OnStartup is called by Wails once the frontend is ready to receive
// runtime calls. Kept as a plain method (not part of the Fx graph) so
// main.go can pass it straight to wails.Run's OnStartup hook.
func (a *App) OnStartup(ctx context.Context) {
	a.ctx = ctx
}

// Ping is a scaffolding smoke test from Fase 0: proves the Go↔React
// binding pipeline and the SQLite connection both work end to end.
func (a *App) Ping() (string, error) {
	if err := a.db.PingContext(a.ctx); err != nil {
		return "", fmt.Errorf("ping db: %w", err)
	}
	return "pong", nil
}

// --- Produtos (SDD 4.1) ---------------------------------------------------

func (a *App) ListarProdutos() ([]domain.Produto, error) {
	return a.ProdutoService.Listar(a.ctx, domain.NegocioID)
}

func (a *App) ListarTodosProdutos() ([]domain.Produto, error) {
	return a.ProdutoService.ListarTodos(a.ctx, domain.NegocioID)
}

func (a *App) ObterProduto(id int64) (domain.Produto, error) {
	return a.ProdutoService.Obter(a.ctx, id)
}

func (a *App) ListarUnidadesProduto(produtoID int64) ([]domain.UnidadeProduto, error) {
	return a.ProdutoService.ListarUnidades(a.ctx, produtoID)
}

// CriarProdutoRequest agrupa o produto e as suas unidades de venda/compra
// — um produto só é gravado com pelo menos uma unidade (SDD 4.1).
type CriarProdutoRequest struct {
	Produto  domain.Produto
	Unidades []domain.UnidadeProduto
}

type CriarProdutoResponse struct {
	Produto  domain.Produto
	Unidades []domain.UnidadeProduto
}

func (a *App) CriarProduto(req CriarProdutoRequest) (CriarProdutoResponse, error) {
	produto, unidades, err := a.ProdutoService.Criar(a.ctx, usecase.NovoProduto{
		Produto:  req.Produto,
		Unidades: req.Unidades,
	})
	if err != nil {
		return CriarProdutoResponse{}, err
	}
	return CriarProdutoResponse{Produto: produto, Unidades: unidades}, nil
}

func (a *App) AtualizarProduto(p domain.Produto) (domain.Produto, error) {
	return a.ProdutoService.Atualizar(a.ctx, p)
}

func (a *App) AtivarProduto(id int64) error {
	return a.ProdutoService.Ativar(a.ctx, id)
}

func (a *App) DesativarProduto(id int64) error {
	return a.ProdutoService.Desativar(a.ctx, id)
}

func (a *App) DuplicarProduto(produtoID int64) (CriarProdutoResponse, error) {
	produto, unidades, err := a.ProdutoService.Duplicar(a.ctx, produtoID)
	if err != nil {
		return CriarProdutoResponse{}, err
	}
	return CriarProdutoResponse{Produto: produto, Unidades: unidades}, nil
}

// --- Armazéns (SDD 4.3) ----------------------------------------------------

func (a *App) ListarArmazens() ([]domain.Armazem, error) {
	return a.ArmazemService.Listar(a.ctx, domain.NegocioID)
}

func (a *App) CriarArmazem(arm domain.Armazem) (domain.Armazem, error) {
	return a.ArmazemService.Criar(a.ctx, arm)
}

func (a *App) AtualizarArmazem(arm domain.Armazem) error {
	return a.ArmazemService.Atualizar(a.ctx, arm)
}

func (a *App) AtivarArmazem(id int64) error {
	return a.ArmazemService.Ativar(a.ctx, id)
}

func (a *App) DesativarArmazem(id int64) error {
	return a.ArmazemService.Desativar(a.ctx, id)
}

// --- Stock (SDD 4.4-4.8) ---------------------------------------------------

func (a *App) ObterStock(produtoID, armazemID int64) (domain.StockItem, error) {
	return a.StockService.ObterStock(a.ctx, produtoID, armazemID)
}

func (a *App) ListarStockPorArmazem(armazemID int64) ([]domain.StockItem, error) {
	return a.StockService.ListarStockPorArmazem(a.ctx, armazemID)
}

func (a *App) ListarMovimentosStock(produtoID int64) ([]domain.MovimentoStock, error) {
	return a.StockService.ListarMovimentos(a.ctx, produtoID)
}

func (a *App) RegistarEntradaStock(p usecase.EntradaParams) (domain.MovimentoStock, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.StockService.RegistarEntrada(a.ctx, p)
}

func (a *App) TransferirStock(p usecase.TransferenciaParams) (domain.MovimentoStock, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.StockService.TransferirStock(a.ctx, p)
}

func (a *App) RegistarPerdaStock(p usecase.SaidaParams) (domain.MovimentoStock, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.StockService.RegistarPerda(a.ctx, p)
}

func (a *App) RegistarConsumoInternoStock(p usecase.SaidaParams) (domain.MovimentoStock, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.StockService.RegistarConsumoInterno(a.ctx, p)
}

// --- Inventário (SDD 4.8) --------------------------------------------------

type IniciarInventarioResponse struct {
	Inventario domain.Inventario
	Itens      []domain.InventarioItem
}

func (a *App) IniciarInventario(armazemID int64) (IniciarInventarioResponse, error) {
	inv, itens, err := a.StockService.IniciarInventario(a.ctx, domain.NegocioID, armazemID, UtilizadorAtualID)
	if err != nil {
		return IniciarInventarioResponse{}, err
	}
	return IniciarInventarioResponse{Inventario: inv, Itens: itens}, nil
}

func (a *App) ListarInventarioItens(inventarioID int64) ([]domain.InventarioItem, error) {
	return a.StockService.ListarInventarioItens(a.ctx, inventarioID)
}

func (a *App) RegistarContagemInventario(itemID int64, quantidadeContada float64) (domain.InventarioItem, error) {
	return a.StockService.RegistarContagem(a.ctx, itemID, quantidadeContada)
}

func (a *App) FecharInventario(inventarioID int64) ([]domain.MovimentoStock, error) {
	return a.StockService.FecharInventario(a.ctx, inventarioID, UtilizadorAtualID)
}

// --- Configuração / wizard de 1º arranque (SDD secção 9) -------------------

func (a *App) EstaConfigurado() (bool, error) {
	return a.ConfigService.EstaConfigurado(a.ctx)
}

func (a *App) ObterEstadoConfig() (usecase.EstadoConfig, error) {
	return a.ConfigService.ObterEstado(a.ctx)
}

// ObterPresetsNegocio devolve a tabela de tipos de negócio pré-configurados
// (SDD 9.1) para o passo 2 do wizard — dados estáticos, sem estado.
func (a *App) ObterPresetsNegocio() []domain.PresetNegocio {
	return domain.PresetsNegocio
}

// ObterTodosModulos devolve o checklist completo usado quando o tipo
// escolhido é "Personalizado" (SDD 9.2).
func (a *App) ObterTodosModulos() []domain.Modulo {
	return domain.TodosModulos
}

func (a *App) FinalizarWizard(p usecase.FinalizarWizardParams) error {
	return a.ConfigService.FinalizarWizard(a.ctx, p)
}

// --- Clientes (SDD 4.9) -----------------------------------------------------

func (a *App) ListarClientes() ([]domain.Cliente, error) {
	return a.ClienteService.Listar(a.ctx, domain.NegocioID)
}

func (a *App) ObterCliente(id int64) (domain.Cliente, error) {
	return a.ClienteService.Obter(a.ctx, id)
}

func (a *App) CriarCliente(c domain.Cliente) (domain.Cliente, error) {
	return a.ClienteService.Criar(a.ctx, c)
}

func (a *App) AtualizarCliente(c domain.Cliente) error {
	return a.ClienteService.Atualizar(a.ctx, c)
}

func (a *App) DesativarCliente(id int64) error {
	return a.ClienteService.Desativar(a.ctx, id)
}

// --- Vendas / PDV direto (SDD 4.10) -----------------------------------------

func (a *App) RegistarVendaDireta(p usecase.VendaDiretaParams) (usecase.VendaCompleta, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.VendaService.RegistarVendaDireta(a.ctx, p)
}

func (a *App) ObterVendaCompleta(vendaID int64) (usecase.VendaCompleta, error) {
	return a.VendaService.ObterVendaCompleta(a.ctx, vendaID)
}

// ListarVendasDoDia cobre o "Histórico de vendas do dia" da tela de
// ferragem (SDD 5.3).
func (a *App) ListarVendasDoDia() ([]domain.Venda, error) {
	return a.VendaService.ListarVendasDoDia(a.ctx, domain.NegocioID)
}

// GerarRecibo grava o PDF em disco e abre-o no visualizador por omissão do
// sistema — mais simples do que construir um diálogo "guardar como" só
// para este passo (SDD 4.10, "imprimir/gerar recibo").
func (a *App) GerarRecibo(vendaID int64) (string, error) {
	pdfBytes, err := a.ReciboService.GerarPDF(a.ctx, vendaID)
	if err != nil {
		return "", err
	}

	dir, err := config.ResolveRecibosDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, fmt.Sprintf("recibo-%d.pdf", vendaID))
	if err := os.WriteFile(path, pdfBytes, 0o644); err != nil {
		return "", fmt.Errorf("gravar recibo: %w", err)
	}

	runtime.BrowserOpenURL(a.ctx, "file://"+path)
	return path, nil
}

// --- Contas abertas / bottlestore (SDD 4.10) --------------------------------

func (a *App) ListarContasAbertas() ([]domain.Conta, error) {
	return a.ContaService.ListarAbertas(a.ctx, domain.NegocioID)
}

func (a *App) ObterConta(id int64) (domain.Conta, error) {
	return a.ContaService.ObterConta(a.ctx, id)
}

func (a *App) ListarItensConta(contaID int64) ([]domain.ItemConta, error) {
	return a.ContaService.ListarItens(a.ctx, contaID)
}

type AbrirContaRequest struct {
	Identificador string
	ClienteID     *int64
}

func (a *App) AbrirConta(req AbrirContaRequest) (domain.Conta, error) {
	return a.ContaService.AbrirConta(a.ctx, req.Identificador, req.ClienteID, UtilizadorAtualID)
}

func (a *App) AdicionarItemConta(p usecase.AdicionarItemContaParams) (domain.ItemConta, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.ContaService.AdicionarItem(a.ctx, p)
}

func (a *App) TransferirItemConta(itemID, novaContaID int64) error {
	return a.ContaService.TransferirItem(a.ctx, itemID, novaContaID)
}

func (a *App) CancelarConta(contaID int64, motivo string) (domain.Conta, error) {
	return a.ContaService.CancelarConta(a.ctx, contaID, UtilizadorAtualID, motivo)
}

func (a *App) FecharContaAberta(contaID int64, p usecase.FecharContaParams) (usecase.VendaCompleta, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.ContaService.FecharConta(a.ctx, contaID, p)
}

// --- Fichas técnicas / produtos compostos e semi-elaborados (SDD 3.3) ------

func (a *App) ListarIngredientes(produtoCompostoID int64) ([]domain.FichaTecnica, error) {
	return a.FichaTecnicaService.ListarIngredientes(a.ctx, produtoCompostoID)
}

func (a *App) DefinirReceita(produtoCompostoID int64, ingredientes []usecase.IngredienteInput) ([]domain.FichaTecnica, error) {
	return a.FichaTecnicaService.DefinirReceita(a.ctx, produtoCompostoID, ingredientes)
}

func (a *App) DuplicarReceita(produtoOrigemID, produtoDestinoID int64) ([]domain.FichaTecnica, error) {
	return a.FichaTecnicaService.DuplicarReceita(a.ctx, produtoOrigemID, produtoDestinoID)
}

func (a *App) CalcularCustoTeorico(produtoCompostoID, armazemID int64) (float64, error) {
	return a.FichaTecnicaService.CalcularCustoTeorico(a.ctx, produtoCompostoID, armazemID)
}

func (a *App) SimularDisponibilidade(produtoCompostoID, armazemID int64) (float64, error) {
	return a.FichaTecnicaService.SimularDisponibilidade(a.ctx, produtoCompostoID, armazemID)
}

func (a *App) RegistarProducao(p usecase.ProducaoParams) (domain.MovimentoStock, error) {
	p.UtilizadorID = UtilizadorAtualID
	return a.FichaTecnicaService.RegistarProducao(a.ctx, p)
}

// --- Farmácia: lotes/validade e receitas médicas (SDD 9.4) ------------------

type EntradaComLoteResponse struct {
	Lote      domain.Lote
	Movimento domain.MovimentoStock
}

func (a *App) RegistarEntradaComLote(p usecase.EntradaComLoteParams) (EntradaComLoteResponse, error) {
	p.UtilizadorID = UtilizadorAtualID
	lote, movimento, err := a.FarmaciaService.RegistarEntradaComLote(a.ctx, p)
	if err != nil {
		return EntradaComLoteResponse{}, err
	}
	return EntradaComLoteResponse{Lote: lote, Movimento: movimento}, nil
}

func (a *App) ListarLotes(produtoID, armazemID int64) ([]domain.Lote, error) {
	return a.FarmaciaService.ListarLotes(a.ctx, produtoID, armazemID)
}

func (a *App) ListarLotesAVencer(dentroDeDias int) ([]domain.Lote, error) {
	return a.FarmaciaService.ListarLotesAVencer(a.ctx, domain.NegocioID, dentroDeDias)
}

func (a *App) RegistarReceitaMedica(p usecase.ReceitaMedicaInput) (domain.ReceitaMedica, error) {
	p.ValidadoPor = UtilizadorAtualID
	return a.FarmaciaService.RegistarReceitaMedica(a.ctx, p)
}

func (a *App) ListarReceitasPorVenda(vendaID int64) ([]domain.ReceitaMedica, error) {
	return a.FarmaciaService.ListarReceitasPorVenda(a.ctx, vendaID)
}
