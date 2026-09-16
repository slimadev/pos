// Package api exposes the struct Wails binds to the frontend. Every method
// on App becomes a callable TypeScript function via `wails generate`
// (SDD secção 8.1) — App only forwards to usecase services, no business
// logic lives here.
package api

import (
	"context"
	"database/sql"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

// NegocioID é fixo em 1 porque cada executável cmd/app serve sempre uma
// única loja (SDD secção 2.1) — a linha semeada pela migration core.
// UtilizadorAtualID fica fixo no utilizador de arranque até a Fase 2 trazer
// sessão/login reais; a partir daí este valor passa a vir do contexto de
// autenticação em vez de uma constante.
const (
	NegocioID         int64 = 1
	UtilizadorAtualID int64 = 1
)

type App struct {
	ctx context.Context
	db  *sql.DB

	ProdutoService *usecase.ProdutoService
	ArmazemService *usecase.ArmazemService
	StockService   *usecase.StockService
}

func NewApp(db *sql.DB, produto *usecase.ProdutoService, armazem *usecase.ArmazemService, stock *usecase.StockService) *App {
	return &App{db: db, ProdutoService: produto, ArmazemService: armazem, StockService: stock}
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
	return a.ProdutoService.Listar(a.ctx, NegocioID)
}

func (a *App) ListarTodosProdutos() ([]domain.Produto, error) {
	return a.ProdutoService.ListarTodos(a.ctx, NegocioID)
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
	req.Produto.NegocioID = NegocioID
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
	return a.ArmazemService.Listar(a.ctx, NegocioID)
}

func (a *App) CriarArmazem(arm domain.Armazem) (domain.Armazem, error) {
	arm.NegocioID = NegocioID
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
	inv, itens, err := a.StockService.IniciarInventario(a.ctx, NegocioID, armazemID, UtilizadorAtualID)
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
