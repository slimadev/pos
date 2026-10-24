package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

// ProdutoImportar é um produto novo vindo de uma folha de Excel, já
// validado: Produto.CategoriaID aponta para uma categoria existente, ou
// CategoriaNova diz o nome da categoria a criar na mesma importação.
type ProdutoImportar struct {
	Produto       domain.Produto
	Unidade       domain.UnidadeProduto
	CategoriaNova string
	Quantidade    float64 // stock inicial, em unidade base (0 = sem entrada)
}

// ImportacaoRepository grava uma importação inteira numa só transação:
// ou entram as categorias, os produtos e o stock todos, ou nada.
type ImportacaoRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewImportacaoRepository(db *sql.DB) *ImportacaoRepository {
	return &ImportacaoRepository{db: db, q: sqlcgen.New(db)}
}

func (r *ImportacaoRepository) Importar(ctx context.Context, itens []ProdutoImportar, armazemID int64, motivo string, utilizadorID int64) (produtos int, categorias int, err error) {
	err = runInTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		novas := map[string]int64{}
		for _, it := range itens {
			p := it.Produto
			if it.CategoriaNova != "" {
				id, ok := novas[it.CategoriaNova]
				if !ok {
					c, err := q.CriarCategoria(ctx, sqlcgen.CriarCategoriaParams{NegocioID: p.NegocioID, Nome: it.CategoriaNova})
					if err != nil {
						return err
					}
					id = c.ID
					novas[it.CategoriaNova] = id
				}
				p.CategoriaID = &id
			}
			row, err := q.CriarProduto(ctx, sqlcgen.CriarProdutoParams{
				NegocioID:      p.NegocioID,
				CategoriaID:    ptrToNullInt64(p.CategoriaID),
				Nome:           p.Nome,
				TipoProduto:    string(p.TipoProduto),
				UnidadeBase:    p.UnidadeBase,
				StockMinimo:    ptrToNullFloat64(p.StockMinimo),
				ControlaStock:  boolToInt64(p.ControlaStock),
				LoteControlado: boolToInt64(p.LoteControlado),
			})
			if err != nil {
				return err
			}
			u := it.Unidade
			uRow, err := q.CriarUnidadeProduto(ctx, sqlcgen.CriarUnidadeProdutoParams{
				ProdutoID:        row.ID,
				NomeUnidade:      u.NomeUnidade,
				QtdEmUnidadeBase: u.QtdEmUnidadeBase,
				PrecoCusto:       u.PrecoCusto,
				PrecoVenda:       u.PrecoVenda,
				CodigoBarras:     ptrToNullString(u.CodigoBarras),
				IsPadraoVenda:    1,
				IsPadraoCompra:   1,
			})
			if err != nil {
				return err
			}
			if it.Quantidade > 0 {
				destino, custo, m := armazemID, u.PrecoCusto, motivo
				if _, err := q.CriarMovimentoStock(ctx, sqlcgen.CriarMovimentoStockParams{
					ProdutoID:             row.ID,
					ArmazemDestinoID:      ptrToNullInt64(&destino),
					Tipo:                  string(domain.TipoMovimentoEntrada),
					QuantidadeUnidadeBase: it.Quantidade,
					CustoUnitario:         ptrToNullFloat64(&custo),
					UnidadeProdutoID:      ptrToNullInt64(&uRow.ID),
					Motivo:                ptrToNullString(&m),
					UtilizadorID:          utilizadorID,
				}); err != nil {
					return err
				}
				// Produto novo: o custo médio é o custo desta entrada.
				if err := q.DefinirCustoMedioProdutoArmazem(ctx, sqlcgen.DefinirCustoMedioProdutoArmazemParams{
					ProdutoID: row.ID, ArmazemID: armazemID, CustoMedioPonderado: custo,
				}); err != nil {
					return err
				}
			}
			produtos++
		}
		categorias = len(novas)
		return nil
	})
	if err != nil {
		return 0, 0, err
	}
	return produtos, categorias, nil
}
