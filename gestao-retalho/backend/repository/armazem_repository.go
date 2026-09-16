package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type ArmazemRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewArmazemRepository(db *sql.DB) *ArmazemRepository {
	return &ArmazemRepository{db: db, q: sqlcgen.New(db)}
}

func (r *ArmazemRepository) Listar(ctx context.Context, negocioID int64) ([]domain.Armazem, error) {
	rows, err := r.q.ListarArmazens(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Armazem, 0, len(rows))
	for _, row := range rows {
		out = append(out, armazemFromRow(row))
	}
	return out, nil
}

func (r *ArmazemRepository) Obter(ctx context.Context, id int64) (domain.Armazem, error) {
	row, err := r.q.ObterArmazem(ctx, id)
	if err != nil {
		return domain.Armazem{}, err
	}
	return armazemFromRow(row), nil
}

func (r *ArmazemRepository) Criar(ctx context.Context, a domain.Armazem) (domain.Armazem, error) {
	row, err := r.q.CriarArmazem(ctx, sqlcgen.CriarArmazemParams{
		NegocioID: a.NegocioID,
		Nome:      a.Nome,
		Tipo:      string(a.Tipo),
	})
	if err != nil {
		return domain.Armazem{}, err
	}
	return armazemFromRow(row), nil
}

func (r *ArmazemRepository) Atualizar(ctx context.Context, a domain.Armazem) error {
	return r.q.AtualizarArmazem(ctx, sqlcgen.AtualizarArmazemParams{
		Nome: a.Nome,
		Tipo: string(a.Tipo),
		ID:   a.ID,
	})
}

func (r *ArmazemRepository) DefinirAtivo(ctx context.Context, id int64, ativo bool) error {
	return r.q.DefinirArmazemAtivo(ctx, sqlcgen.DefinirArmazemAtivoParams{
		Ativo: boolToInt64(ativo),
		ID:    id,
	})
}

func armazemFromRow(row sqlcgen.Armazens) domain.Armazem {
	return domain.Armazem{
		ID:        row.ID,
		NegocioID: row.NegocioID,
		Nome:      row.Nome,
		Tipo:      domain.TipoArmazem(row.Tipo),
		Ativo:     int64ToBool(row.Ativo),
	}
}
