package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type InventarioRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewInventarioRepository(db *sql.DB) *InventarioRepository {
	return &InventarioRepository{db: db, q: sqlcgen.New(db)}
}

func (r *InventarioRepository) WithTx(ctx context.Context, fn func(*InventarioRepository) error) error {
	return runInTx(ctx, r.db, func(tx *sql.Tx) error {
		txRepo := &InventarioRepository{db: r.db, q: r.q.WithTx(tx)}
		return fn(txRepo)
	})
}

func (r *InventarioRepository) Criar(ctx context.Context, negocioID, armazemID, utilizadorID int64) (domain.Inventario, error) {
	row, err := r.q.CriarInventario(ctx, sqlcgen.CriarInventarioParams{
		NegocioID:    negocioID,
		ArmazemID:    armazemID,
		UtilizadorID: utilizadorID,
	})
	if err != nil {
		return domain.Inventario{}, err
	}
	return inventarioFromRow(row), nil
}

func (r *InventarioRepository) CriarItem(ctx context.Context, inventarioID, produtoID int64, quantidadeSistema float64) error {
	return r.q.CriarInventarioItem(ctx, sqlcgen.CriarInventarioItemParams{
		InventarioID:      inventarioID,
		ProdutoID:         produtoID,
		QuantidadeSistema: quantidadeSistema,
	})
}

func (r *InventarioRepository) Obter(ctx context.Context, id int64) (domain.Inventario, error) {
	row, err := r.q.ObterInventario(ctx, id)
	if err != nil {
		return domain.Inventario{}, err
	}
	return inventarioFromRow(row), nil
}

func (r *InventarioRepository) Fechar(ctx context.Context, id int64) error {
	return r.q.FecharInventario(ctx, id)
}

func (r *InventarioRepository) ObterItem(ctx context.Context, id int64) (domain.InventarioItem, error) {
	row, err := r.q.ObterInventarioItem(ctx, id)
	if err != nil {
		return domain.InventarioItem{}, err
	}
	return inventarioItemFromRow(row), nil
}

func (r *InventarioRepository) ListarItens(ctx context.Context, inventarioID int64) ([]domain.InventarioItem, error) {
	rows, err := r.q.ListarInventarioItens(ctx, inventarioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.InventarioItem, 0, len(rows))
	for _, row := range rows {
		out = append(out, inventarioItemFromRow(row))
	}
	return out, nil
}

func (r *InventarioRepository) RegistarContagem(ctx context.Context, itemID int64, quantidadeContada, diferenca float64) error {
	return r.q.RegistarContagemInventarioItem(ctx, sqlcgen.RegistarContagemInventarioItemParams{
		QuantidadeContada: sql.NullFloat64{Float64: quantidadeContada, Valid: true},
		Diferenca:         sql.NullFloat64{Float64: diferenca, Valid: true},
		ID:                itemID,
	})
}

func (r *InventarioRepository) MarcarAjustado(ctx context.Context, itemID int64) error {
	return r.q.MarcarInventarioItemAjustado(ctx, itemID)
}

func inventarioFromRow(row sqlcgen.Inventarios) domain.Inventario {
	return domain.Inventario{
		ID:           row.ID,
		NegocioID:    row.NegocioID,
		ArmazemID:    row.ArmazemID,
		Data:         parseTimestamp(row.Data),
		Status:       domain.StatusInventario(row.Status),
		UtilizadorID: row.UtilizadorID,
	}
}

func inventarioItemFromRow(row sqlcgen.InventarioItens) domain.InventarioItem {
	return domain.InventarioItem{
		ID:                row.ID,
		InventarioID:      row.InventarioID,
		ProdutoID:         row.ProdutoID,
		QuantidadeSistema: row.QuantidadeSistema,
		QuantidadeContada: nullFloat64ToPtr(row.QuantidadeContada),
		Diferenca:         nullFloat64ToPtr(row.Diferenca),
		Ajustado:          int64ToBool(row.Ajustado),
	}
}
