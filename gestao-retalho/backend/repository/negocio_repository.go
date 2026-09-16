package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type NegocioRepository struct {
	q *sqlcgen.Queries
}

func NewNegocioRepository(db *sql.DB) *NegocioRepository {
	return &NegocioRepository{q: sqlcgen.New(db)}
}

func (r *NegocioRepository) Obter(ctx context.Context) (domain.Negocio, error) {
	row, err := r.q.ObterNegocio(ctx, domain.NegocioID)
	if err != nil {
		return domain.Negocio{}, err
	}
	return negocioFromRow(row), nil
}

func (r *NegocioRepository) Atualizar(ctx context.Context, nome string, tipo domain.TipoNegocio) (domain.Negocio, error) {
	row, err := r.q.AtualizarNegocio(ctx, sqlcgen.AtualizarNegocioParams{
		Nome:        nome,
		TipoNegocio: string(tipo),
		ID:          domain.NegocioID,
	})
	if err != nil {
		return domain.Negocio{}, err
	}
	return negocioFromRow(row), nil
}

func negocioFromRow(row sqlcgen.Negocios) domain.Negocio {
	return domain.Negocio{
		ID:          row.ID,
		Nome:        row.Nome,
		TipoNegocio: domain.TipoNegocio(row.TipoNegocio),
		Ativo:       int64ToBool(row.Ativo),
	}
}
