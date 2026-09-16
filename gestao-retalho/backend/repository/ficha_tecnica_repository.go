package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type FichaTecnicaRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewFichaTecnicaRepository(db *sql.DB) *FichaTecnicaRepository {
	return &FichaTecnicaRepository{db: db, q: sqlcgen.New(db)}
}

func (r *FichaTecnicaRepository) ListarIngredientes(ctx context.Context, produtoCompostoID int64) ([]domain.FichaTecnica, error) {
	rows, err := r.q.ListarIngredientes(ctx, produtoCompostoID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.FichaTecnica, 0, len(rows))
	for _, row := range rows {
		out = append(out, fichaTecnicaFromRow(row))
	}
	return out, nil
}

// DefinirReceita substitui a receita inteira (remove todas as linhas
// atuais e grava as novas) numa única transação — mais simples e mais
// seguro do que reconciliar add/remove/editar linha a linha para um
// ecrã de edição de ficha técnica.
func (r *FichaTecnicaRepository) DefinirReceita(ctx context.Context, produtoCompostoID int64, ingredientes []domain.FichaTecnica) ([]domain.FichaTecnica, error) {
	var gravados []domain.FichaTecnica
	err := runInTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		if err := q.RemoverIngredientes(ctx, produtoCompostoID); err != nil {
			return err
		}
		for _, ing := range ingredientes {
			row, err := q.CriarIngrediente(ctx, sqlcgen.CriarIngredienteParams{
				ProdutoCompostoID:    produtoCompostoID,
				ProdutoIngredienteID: ing.ProdutoIngredienteID,
				Quantidade:           ing.Quantidade,
				PerdaPercentual:      ing.PerdaPercentual,
			})
			if err != nil {
				return err
			}
			gravados = append(gravados, fichaTecnicaFromRow(row))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return gravados, nil
}

func fichaTecnicaFromRow(row sqlcgen.FichasTecnicas) domain.FichaTecnica {
	return domain.FichaTecnica{
		ID:                   row.ID,
		ProdutoCompostoID:    row.ProdutoCompostoID,
		ProdutoIngredienteID: row.ProdutoIngredienteID,
		Quantidade:           row.Quantidade,
		PerdaPercentual:      row.PerdaPercentual,
	}
}
