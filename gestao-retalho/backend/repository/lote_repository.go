package repository

import (
	"context"
	"database/sql"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

// LoteRepository segue o mesmo padrão do VendaRepository: carrega um
// StockRepository sempre ligado à mesma transação (WithTx), porque
// registar uma entrada com lote ou consumir por FEFO numa venda tem de
// escrever lotes_produto e movimentos_stock atomicamente.
type LoteRepository struct {
	db    *sql.DB
	q     *sqlcgen.Queries
	Stock *StockRepository
}

func NewLoteRepository(db *sql.DB, stock *StockRepository) *LoteRepository {
	return &LoteRepository{db: db, q: sqlcgen.New(db), Stock: stock}
}

func (r *LoteRepository) WithTx(ctx context.Context, fn func(*LoteRepository) error) error {
	return runInTx(ctx, r.db, func(tx *sql.Tx) error {
		txRepo := &LoteRepository{
			db:    r.db,
			q:     r.q.WithTx(tx),
			Stock: &StockRepository{db: r.db, q: sqlcgen.New(tx)},
		}
		return fn(txRepo)
	})
}

func (r *LoteRepository) CriarLote(ctx context.Context, l domain.Lote) (domain.Lote, error) {
	row, err := r.q.CriarLote(ctx, sqlcgen.CriarLoteParams{
		ProdutoID:     l.ProdutoID,
		ArmazemID:     l.ArmazemID,
		NumeroLote:    l.NumeroLote,
		DataValidade:  l.DataValidade.UTC().Format(timestampLayout),
		Quantidade:    l.Quantidade,
		CustoUnitario: l.CustoUnitario,
	})
	if err != nil {
		return domain.Lote{}, err
	}
	return loteFromRow(row), nil
}

// ListarFEFO devolve os lotes com stock, ordenados por validade
// ascendente — o primeiro da lista é sempre o próximo a consumir.
func (r *LoteRepository) ListarFEFO(ctx context.Context, produtoID, armazemID int64) ([]domain.Lote, error) {
	rows, err := r.q.ListarLotesFEFO(ctx, sqlcgen.ListarLotesFEFOParams{
		ProdutoID: produtoID,
		ArmazemID: armazemID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Lote, 0, len(rows))
	for _, row := range rows {
		out = append(out, loteFromRow(row))
	}
	return out, nil
}

func (r *LoteRepository) AtualizarQuantidade(ctx context.Context, loteID int64, novaQuantidade float64) error {
	return r.q.AtualizarQuantidadeLote(ctx, sqlcgen.AtualizarQuantidadeLoteParams{
		Quantidade: novaQuantidade,
		ID:         loteID,
	})
}

// ListarAVencer cobre os alertas de validade próxima (SDD 9.4) — todos os
// lotes com stock cuja validade é até `ate`.
func (r *LoteRepository) ListarAVencer(ctx context.Context, negocioID int64, ate time.Time) ([]domain.Lote, error) {
	rows, err := r.q.ListarLotesAVencer(ctx, sqlcgen.ListarLotesAVencerParams{
		NegocioID:    negocioID,
		DataValidade: ate.UTC().Format(timestampLayout),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Lote, 0, len(rows))
	for _, row := range rows {
		out = append(out, loteFromRow(row))
	}
	return out, nil
}

func loteFromRow(row sqlcgen.LotesProduto) domain.Lote {
	return domain.Lote{
		ID:            row.ID,
		ProdutoID:     row.ProdutoID,
		ArmazemID:     row.ArmazemID,
		NumeroLote:    row.NumeroLote,
		DataValidade:  parseTimestamp(row.DataValidade),
		Quantidade:    row.Quantidade,
		CustoUnitario: row.CustoUnitario,
	}
}
