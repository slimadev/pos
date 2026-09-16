package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type ReceitaMedicaRepository struct {
	q *sqlcgen.Queries
}

func NewReceitaMedicaRepository(db *sql.DB) *ReceitaMedicaRepository {
	return &ReceitaMedicaRepository{q: sqlcgen.New(db)}
}

func (r *ReceitaMedicaRepository) Criar(ctx context.Context, rm domain.ReceitaMedica) (domain.ReceitaMedica, error) {
	row, err := r.q.CriarReceitaMedica(ctx, sqlcgen.CriarReceitaMedicaParams{
		VendaID:       rm.VendaID,
		ClienteID:     rm.ClienteID,
		Medico:        rm.Medico,
		NumeroReceita: rm.NumeroReceita,
		ValidadoPor:   rm.ValidadoPor,
	})
	if err != nil {
		return domain.ReceitaMedica{}, err
	}
	return receitaMedicaFromRow(row), nil
}

func (r *ReceitaMedicaRepository) ListarPorVenda(ctx context.Context, vendaID int64) ([]domain.ReceitaMedica, error) {
	rows, err := r.q.ListarReceitasPorVenda(ctx, vendaID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ReceitaMedica, 0, len(rows))
	for _, row := range rows {
		out = append(out, receitaMedicaFromRow(row))
	}
	return out, nil
}

func receitaMedicaFromRow(row sqlcgen.ReceitasMedicas) domain.ReceitaMedica {
	return domain.ReceitaMedica{
		ID:            row.ID,
		VendaID:       row.VendaID,
		ClienteID:     row.ClienteID,
		Medico:        row.Medico,
		NumeroReceita: row.NumeroReceita,
		ValidadoPor:   row.ValidadoPor,
		CriadoEm:      parseTimestamp(row.CriadoEm),
	}
}
