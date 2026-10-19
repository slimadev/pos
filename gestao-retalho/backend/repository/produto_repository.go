package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type ProdutoRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewProdutoRepository(db *sql.DB) *ProdutoRepository {
	return &ProdutoRepository{db: db, q: sqlcgen.New(db)}
}

func (r *ProdutoRepository) Listar(ctx context.Context, negocioID int64) ([]domain.Produto, error) {
	rows, err := r.q.ListarProdutos(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	return produtosFromRows(rows), nil
}

func (r *ProdutoRepository) ListarAtivos(ctx context.Context, negocioID int64) ([]domain.Produto, error) {
	rows, err := r.q.ListarProdutosAtivos(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	return produtosFromRows(rows), nil
}

func (r *ProdutoRepository) Obter(ctx context.Context, id int64) (domain.Produto, error) {
	row, err := r.q.ObterProduto(ctx, id)
	if err != nil {
		return domain.Produto{}, err
	}
	return produtoFromRow(row), nil
}

func (r *ProdutoRepository) ListarUnidades(ctx context.Context, produtoID int64) ([]domain.UnidadeProduto, error) {
	rows, err := r.q.ListarUnidadesPorProduto(ctx, produtoID)
	if err != nil {
		return nil, err
	}
	return unidadesFromRows(rows), nil
}

// ListarUnidadesDoNegocio devolve as unidades de todos os produtos numa só
// consulta — o catálogo do PDV não pode pedir as unidades produto a produto.
func (r *ProdutoRepository) ListarUnidadesDoNegocio(ctx context.Context, negocioID int64) ([]domain.UnidadeProduto, error) {
	rows, err := r.q.ListarUnidadesDoNegocio(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	return unidadesFromRows(rows), nil
}

func (r *ProdutoRepository) ObterUnidade(ctx context.Context, id int64) (domain.UnidadeProduto, error) {
	row, err := r.q.ObterUnidadeProduto(ctx, id)
	if err != nil {
		return domain.UnidadeProduto{}, err
	}
	return unidadeFromRow(row), nil
}

// Criar grava o produto e as suas unidades de venda/compra numa única
// transação: um produto sem pelo menos uma unidade não é vendável, por
// isso não faz sentido existir um sem o outro (SDD 3.2/4.1).
func (r *ProdutoRepository) Criar(ctx context.Context, p domain.Produto, unidades []domain.UnidadeProduto) (domain.Produto, []domain.UnidadeProduto, error) {
	var criado domain.Produto
	var criadas []domain.UnidadeProduto

	err := runInTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)

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
		criado = produtoFromRow(row)

		for _, u := range unidades {
			uRow, err := q.CriarUnidadeProduto(ctx, sqlcgen.CriarUnidadeProdutoParams{
				ProdutoID:        criado.ID,
				NomeUnidade:      u.NomeUnidade,
				QtdEmUnidadeBase: u.QtdEmUnidadeBase,
				PrecoCusto:       u.PrecoCusto,
				PrecoVenda:       u.PrecoVenda,
				CodigoBarras:     ptrToNullString(u.CodigoBarras),
				IsPadraoVenda:    boolToInt64(u.IsPadraoVenda),
				IsPadraoCompra:   boolToInt64(u.IsPadraoCompra),
			})
			if err != nil {
				return err
			}
			criadas = append(criadas, unidadeFromRow(uRow))
		}
		return nil
	})
	if err != nil {
		return domain.Produto{}, nil, err
	}
	return criado, criadas, nil
}

func (r *ProdutoRepository) Atualizar(ctx context.Context, p domain.Produto) (domain.Produto, error) {
	row, err := r.q.AtualizarProduto(ctx, sqlcgen.AtualizarProdutoParams{
		CategoriaID:    ptrToNullInt64(p.CategoriaID),
		Nome:           p.Nome,
		StockMinimo:    ptrToNullFloat64(p.StockMinimo),
		LoteControlado: boolToInt64(p.LoteControlado),
		ID:             p.ID,
	})
	if err != nil {
		return domain.Produto{}, err
	}
	return produtoFromRow(row), nil
}

func (r *ProdutoRepository) DefinirAtivo(ctx context.Context, id int64, ativo bool) error {
	return r.q.DefinirProdutoAtivo(ctx, sqlcgen.DefinirProdutoAtivoParams{
		Ativo: boolToInt64(ativo),
		ID:    id,
	})
}

func (r *ProdutoRepository) CriarUnidade(ctx context.Context, u domain.UnidadeProduto) (domain.UnidadeProduto, error) {
	row, err := r.q.CriarUnidadeProduto(ctx, sqlcgen.CriarUnidadeProdutoParams{
		ProdutoID:        u.ProdutoID,
		NomeUnidade:      u.NomeUnidade,
		QtdEmUnidadeBase: u.QtdEmUnidadeBase,
		PrecoCusto:       u.PrecoCusto,
		PrecoVenda:       u.PrecoVenda,
		CodigoBarras:     ptrToNullString(u.CodigoBarras),
		IsPadraoVenda:    boolToInt64(u.IsPadraoVenda),
		IsPadraoCompra:   boolToInt64(u.IsPadraoCompra),
	})
	if err != nil {
		return domain.UnidadeProduto{}, err
	}
	return unidadeFromRow(row), nil
}

// AtualizarUnidade grava nome, conversão, preços, código de barras e
// padrões de uma unidade existente — o produto a que pertence não muda.
func (r *ProdutoRepository) AtualizarUnidade(ctx context.Context, u domain.UnidadeProduto) (domain.UnidadeProduto, error) {
	err := r.q.AtualizarUnidadeProduto(ctx, sqlcgen.AtualizarUnidadeProdutoParams{
		ID:               u.ID,
		NomeUnidade:      u.NomeUnidade,
		QtdEmUnidadeBase: u.QtdEmUnidadeBase,
		PrecoCusto:       u.PrecoCusto,
		PrecoVenda:       u.PrecoVenda,
		CodigoBarras:     ptrToNullString(u.CodigoBarras),
		IsPadraoVenda:    boolToInt64(u.IsPadraoVenda),
		IsPadraoCompra:   boolToInt64(u.IsPadraoCompra),
	})
	if err != nil {
		return domain.UnidadeProduto{}, err
	}
	return r.ObterUnidade(ctx, u.ID)
}

func produtoFromRow(row sqlcgen.Produtos) domain.Produto {
	return domain.Produto{
		ID:             row.ID,
		NegocioID:      row.NegocioID,
		CategoriaID:    nullInt64ToPtr(row.CategoriaID),
		Nome:           row.Nome,
		TipoProduto:    domain.TipoProduto(row.TipoProduto),
		UnidadeBase:    row.UnidadeBase,
		StockMinimo:    nullFloat64ToPtr(row.StockMinimo),
		ControlaStock:  int64ToBool(row.ControlaStock),
		LoteControlado: int64ToBool(row.LoteControlado),
		Ativo:          int64ToBool(row.Ativo),
		CriadoEm:       parseTimestamp(row.CriadoEm),
		AtualizadoEm:   parseTimestamp(row.AtualizadoEm),
	}
}

func produtosFromRows(rows []sqlcgen.Produtos) []domain.Produto {
	out := make([]domain.Produto, 0, len(rows))
	for _, row := range rows {
		out = append(out, produtoFromRow(row))
	}
	return out
}

func unidadeFromRow(row sqlcgen.UnidadesProduto) domain.UnidadeProduto {
	return domain.UnidadeProduto{
		ID:               row.ID,
		ProdutoID:        row.ProdutoID,
		NomeUnidade:      row.NomeUnidade,
		QtdEmUnidadeBase: row.QtdEmUnidadeBase,
		PrecoCusto:       row.PrecoCusto,
		PrecoVenda:       row.PrecoVenda,
		CodigoBarras:     nullStringToPtr(row.CodigoBarras),
		IsPadraoVenda:    int64ToBool(row.IsPadraoVenda),
		IsPadraoCompra:   int64ToBool(row.IsPadraoCompra),
	}
}

func unidadesFromRows(rows []sqlcgen.UnidadesProduto) []domain.UnidadeProduto {
	out := make([]domain.UnidadeProduto, 0, len(rows))
	for _, row := range rows {
		out = append(out, unidadeFromRow(row))
	}
	return out
}
