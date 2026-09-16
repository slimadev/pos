package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

// DividaRepository cobre o lado de leitura/pagamento de uma dívida — a
// CRIAÇÃO de uma dívida fica em VendaRepository.CriarDivida porque tem de
// acontecer atomicamente com o fecho da conta que a originou (Fase 4).
// Pagar uma dívida é um fluxo independente (SDD 4.11), por isso tem a sua
// própria transação, self-contida (sem precisar de StockRepository).
type DividaRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewDividaRepository(db *sql.DB) *DividaRepository {
	return &DividaRepository{db: db, q: sqlcgen.New(db)}
}

func (r *DividaRepository) WithTx(ctx context.Context, fn func(*DividaRepository) error) error {
	return runInTx(ctx, r.db, func(tx *sql.Tx) error {
		txRepo := &DividaRepository{db: r.db, q: r.q.WithTx(tx)}
		return fn(txRepo)
	})
}

func (r *DividaRepository) Obter(ctx context.Context, id int64) (domain.Divida, error) {
	row, err := r.q.ObterDivida(ctx, id)
	if err != nil {
		return domain.Divida{}, err
	}
	return dividaFromRow(row), nil
}

func (r *DividaRepository) ListarPorCliente(ctx context.Context, clienteID int64) ([]domain.Divida, error) {
	rows, err := r.q.ListarDividasPorCliente(ctx, clienteID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Divida, 0, len(rows))
	for _, row := range rows {
		out = append(out, dividaFromRow(row))
	}
	return out, nil
}

func (r *DividaRepository) ListarAbertas(ctx context.Context, negocioID int64) ([]domain.Divida, error) {
	rows, err := r.q.ListarDividasAbertas(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Divida, 0, len(rows))
	for _, row := range rows {
		out = append(out, dividaFromRow(row))
	}
	return out, nil
}

func (r *DividaRepository) Atualizar(ctx context.Context, id int64, valorPago, valorEmAberto float64, status domain.StatusDivida) (domain.Divida, error) {
	row, err := r.q.AtualizarDivida(ctx, sqlcgen.AtualizarDividaParams{
		ValorPago:     valorPago,
		ValorEmAberto: valorEmAberto,
		Status:        string(status),
		ID:            id,
	})
	if err != nil {
		return domain.Divida{}, err
	}
	return dividaFromRow(row), nil
}

func (r *DividaRepository) CriarPagamento(ctx context.Context, p domain.PagamentoDivida) (domain.PagamentoDivida, error) {
	row, err := r.q.CriarPagamentoDivida(ctx, sqlcgen.CriarPagamentoDividaParams{
		DividaID:     p.DividaID,
		Valor:        p.Valor,
		Metodo:       string(p.Metodo),
		UtilizadorID: p.UtilizadorID,
	})
	if err != nil {
		return domain.PagamentoDivida{}, err
	}
	return pagamentoDividaFromRow(row), nil
}

func (r *DividaRepository) ListarPagamentos(ctx context.Context, dividaID int64) ([]domain.PagamentoDivida, error) {
	rows, err := r.q.ListarPagamentosDivida(ctx, dividaID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.PagamentoDivida, 0, len(rows))
	for _, row := range rows {
		out = append(out, pagamentoDividaFromRow(row))
	}
	return out, nil
}

func pagamentoDividaFromRow(row sqlcgen.PagamentosDivida) domain.PagamentoDivida {
	return domain.PagamentoDivida{
		ID:           row.ID,
		DividaID:     row.DividaID,
		Valor:        row.Valor,
		Metodo:       domain.MetodoPagamento(row.Metodo),
		CriadoEm:     parseTimestamp(row.CriadoEm),
		UtilizadorID: row.UtilizadorID,
	}
}
