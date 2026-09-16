package repository

import (
	"context"
	"database/sql"
	"errors"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type StockRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewStockRepository(db *sql.DB) *StockRepository {
	return &StockRepository{db: db, q: sqlcgen.New(db)}
}

// WithTx runs fn with a transaction-scoped repository that shares every
// method with the outer one. StockService uses it to read the current
// stock/custo médio and write the resulting movement atomically — e.g. the
// weighted average cost update in RegistarEntrada must see the same
// snapshot it writes back, or a concurrent entrada could compute a wrong
// average.
func (r *StockRepository) WithTx(ctx context.Context, fn func(*StockRepository) error) error {
	return runInTx(ctx, r.db, func(tx *sql.Tx) error {
		txRepo := &StockRepository{db: r.db, q: r.q.WithTx(tx)}
		return fn(txRepo)
	})
}

func (r *StockRepository) ObterStock(ctx context.Context, produtoID, armazemID int64) (domain.StockItem, error) {
	row, err := r.q.ObterStockProdutoArmazem(ctx, sqlcgen.ObterStockProdutoArmazemParams{
		ProdutoID: produtoID,
		ArmazemID: sql.NullInt64{Int64: armazemID, Valid: true},
	})
	if errors.Is(err, sql.ErrNoRows) {
		return domain.StockItem{ProdutoID: produtoID, ArmazemID: armazemID}, nil
	}
	if err != nil {
		return domain.StockItem{}, err
	}
	return stockItemFromRow(row), nil
}

func (r *StockRepository) ListarPorArmazem(ctx context.Context, armazemID int64) ([]domain.StockItem, error) {
	rows, err := r.q.ListarStockPorArmazem(ctx, sql.NullInt64{Int64: armazemID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]domain.StockItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, stockItemFromRow(row))
	}
	return out, nil
}

func (r *StockRepository) ObterCustoMedio(ctx context.Context, produtoID, armazemID int64) (float64, error) {
	row, err := r.q.ObterCustoMedioProdutoArmazem(ctx, sqlcgen.ObterCustoMedioProdutoArmazemParams{
		ProdutoID: produtoID,
		ArmazemID: armazemID,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return row.CustoMedioPonderado, nil
}

func (r *StockRepository) DefinirCustoMedio(ctx context.Context, produtoID, armazemID int64, custo float64) error {
	return r.q.DefinirCustoMedioProdutoArmazem(ctx, sqlcgen.DefinirCustoMedioProdutoArmazemParams{
		ProdutoID:           produtoID,
		ArmazemID:           armazemID,
		CustoMedioPonderado: custo,
	})
}

func (r *StockRepository) ListarMovimentosPorProduto(ctx context.Context, produtoID int64) ([]domain.MovimentoStock, error) {
	rows, err := r.q.ListarMovimentosPorProduto(ctx, produtoID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.MovimentoStock, 0, len(rows))
	for _, row := range rows {
		out = append(out, movimentoFromRow(row))
	}
	return out, nil
}

// NovoMovimento são os campos necessários para registar um movimento de
// stock; o StockService valida as regras de negócio antes de chamar isto
// (SDD secção 4.4-4.8) — este método só persiste.
type NovoMovimento struct {
	ProdutoID        int64
	ArmazemOrigemID  *int64
	ArmazemDestinoID *int64
	Tipo             domain.TipoMovimento
	Quantidade       float64
	CustoUnitario    *float64
	UnidadeProdutoID *int64
	Motivo           *string
	ReferenciaID     *string
	UtilizadorID     int64
}

func (r *StockRepository) CriarMovimento(ctx context.Context, m NovoMovimento) (domain.MovimentoStock, error) {
	row, err := r.q.CriarMovimentoStock(ctx, sqlcgen.CriarMovimentoStockParams{
		ProdutoID:             m.ProdutoID,
		ArmazemOrigemID:       ptrToNullInt64(m.ArmazemOrigemID),
		ArmazemDestinoID:      ptrToNullInt64(m.ArmazemDestinoID),
		Tipo:                  string(m.Tipo),
		QuantidadeUnidadeBase: m.Quantidade,
		CustoUnitario:         ptrToNullFloat64(m.CustoUnitario),
		UnidadeProdutoID:      ptrToNullInt64(m.UnidadeProdutoID),
		Motivo:                ptrToNullString(m.Motivo),
		ReferenciaID:          ptrToNullString(m.ReferenciaID),
		UtilizadorID:          m.UtilizadorID,
	})
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	return movimentoFromRow(row), nil
}

type ItemParaInventario struct {
	ProdutoID         int64
	QuantidadeSistema float64
}

func (r *StockRepository) ListarParaInventario(ctx context.Context, negocioID, armazemID int64) ([]ItemParaInventario, error) {
	rows, err := r.q.ListarStockParaInventario(ctx, sqlcgen.ListarStockParaInventarioParams{
		ArmazemID: sql.NullInt64{Int64: armazemID, Valid: true},
		NegocioID: negocioID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]ItemParaInventario, 0, len(rows))
	for _, row := range rows {
		out = append(out, ItemParaInventario{ProdutoID: row.ProdutoID, QuantidadeSistema: row.QuantidadeSistema})
	}
	return out, nil
}

func stockItemFromRow(row sqlcgen.StockAtual) domain.StockItem {
	return domain.StockItem{
		ProdutoID:           row.ProdutoID,
		ArmazemID:           row.ArmazemID.Int64,
		Quantidade:          row.QuantidadeUnidadeBase.Float64,
		CustoMedioPonderado: row.CustoMedioPonderado,
	}
}

func movimentoFromRow(row sqlcgen.MovimentosStock) domain.MovimentoStock {
	return domain.MovimentoStock{
		ID:                    row.ID,
		ProdutoID:             row.ProdutoID,
		ArmazemOrigemID:       nullInt64ToPtr(row.ArmazemOrigemID),
		ArmazemDestinoID:      nullInt64ToPtr(row.ArmazemDestinoID),
		Tipo:                  domain.TipoMovimento(row.Tipo),
		QuantidadeUnidadeBase: row.QuantidadeUnidadeBase,
		CustoUnitario:         nullFloat64ToPtr(row.CustoUnitario),
		UnidadeProdutoID:      nullInt64ToPtr(row.UnidadeProdutoID),
		Motivo:                nullStringToPtr(row.Motivo),
		ReferenciaID:          nullStringToPtr(row.ReferenciaID),
		UtilizadorID:          row.UtilizadorID,
		CriadoEm:              parseTimestamp(row.CriadoEm),
	}
}
