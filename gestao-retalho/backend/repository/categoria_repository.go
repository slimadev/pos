package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type CategoriaRepository struct {
	q *sqlcgen.Queries
}

func NewCategoriaRepository(db *sql.DB) *CategoriaRepository {
	return &CategoriaRepository{q: sqlcgen.New(db)}
}

func (r *CategoriaRepository) Listar(ctx context.Context, negocioID int64) ([]domain.Categoria, error) {
	rows, err := r.q.ListarCategorias(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Categoria, 0, len(rows))
	for _, row := range rows {
		out = append(out, categoriaFromRow(row))
	}
	return out, nil
}

func (r *CategoriaRepository) Criar(ctx context.Context, c domain.Categoria) (domain.Categoria, error) {
	row, err := r.q.CriarCategoria(ctx, sqlcgen.CriarCategoriaParams{
		NegocioID:      c.NegocioID,
		Nome:           c.Nome,
		CategoriaPaiID: ptrToNullInt64(c.CategoriaPaiID),
	})
	if err != nil {
		return domain.Categoria{}, err
	}
	return categoriaFromRow(row), nil
}

func (r *CategoriaRepository) Atualizar(ctx context.Context, c domain.Categoria) error {
	return r.q.AtualizarCategoria(ctx, sqlcgen.AtualizarCategoriaParams{
		Nome:           c.Nome,
		CategoriaPaiID: ptrToNullInt64(c.CategoriaPaiID),
		ID:             c.ID,
	})
}

func (r *CategoriaRepository) DefinirAtiva(ctx context.Context, id int64, ativo bool) error {
	return r.q.DefinirCategoriaAtiva(ctx, sqlcgen.DefinirCategoriaAtivaParams{
		Ativo: boolToInt64(ativo),
		ID:    id,
	})
}

func categoriaFromRow(row sqlcgen.Categorias) domain.Categoria {
	return domain.Categoria{
		ID:             row.ID,
		NegocioID:      row.NegocioID,
		Nome:           row.Nome,
		CategoriaPaiID: nullInt64ToPtr(row.CategoriaPaiID),
		Ativo:          int64ToBool(row.Ativo),
	}
}
