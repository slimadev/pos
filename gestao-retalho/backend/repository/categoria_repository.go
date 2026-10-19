package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type CategoriaRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewCategoriaRepository(db *sql.DB) *CategoriaRepository {
	return &CategoriaRepository{db: db, q: sqlcgen.New(db)}
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

// Juntar passa os produtos e as subcategorias de origem para destino e
// desativa origem, numa só transação. Devolve quantos produtos mudaram.
func (r *CategoriaRepository) Juntar(ctx context.Context, origemID, destinoID int64) (int64, error) {
	var movidos int64
	err := runInTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		n, err := q.MoverProdutosDeCategoria(ctx, sqlcgen.MoverProdutosDeCategoriaParams{
			Destino: sql.NullInt64{Int64: destinoID, Valid: true},
			Origem:  sql.NullInt64{Int64: origemID, Valid: true},
		})
		if err != nil {
			return err
		}
		movidos = n
		if err := q.MoverSubcategorias(ctx, sqlcgen.MoverSubcategoriasParams{
			Destino: sql.NullInt64{Int64: destinoID, Valid: true},
			Origem:  sql.NullInt64{Int64: origemID, Valid: true},
		}); err != nil {
			return err
		}
		return q.DefinirCategoriaAtiva(ctx, sqlcgen.DefinirCategoriaAtivaParams{Ativo: 0, ID: origemID})
	})
	return movidos, err
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
