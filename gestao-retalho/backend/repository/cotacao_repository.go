package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type CotacaoRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewCotacaoRepository(db *sql.DB) *CotacaoRepository {
	return &CotacaoRepository{db: db, q: sqlcgen.New(db)}
}

// Criar grava a cotação e os itens numa transação, com o número seguinte
// do ano — lido e usado na mesma transação, para nunca haver dois iguais.
func (r *CotacaoRepository) Criar(ctx context.Context, c domain.Cotacao, itens []domain.ItemCotacao) (domain.Cotacao, []domain.ItemCotacao, error) {
	var criada domain.Cotacao
	criados := make([]domain.ItemCotacao, 0, len(itens))
	err := runInTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		numero, err := q.ProximoNumeroCotacao(ctx, sqlcgen.ProximoNumeroCotacaoParams{NegocioID: c.NegocioID, Ano: c.Ano})
		if err != nil {
			return err
		}
		row, err := q.CriarCotacao(ctx, sqlcgen.CriarCotacaoParams{
			NegocioID:       c.NegocioID,
			Ano:             c.Ano,
			Numero:          numero,
			ClienteID:       ptrToNullInt64(c.ClienteID),
			ClienteNome:     c.ClienteNome,
			ClienteNuit:     ptrToNullString(c.ClienteNUIT),
			ClienteContacto: ptrToNullString(c.ClienteContacto),
			ValidadeDias:    c.ValidadeDias,
			Subtotal:        c.Subtotal,
			DescontoTotal:   c.DescontoTotal,
			Total:           c.Total,
			TaxaIva:         c.TaxaIVA,
			ValorIva:        c.ValorIVA,
			Observacoes:     ptrToNullString(c.Observacoes),
			UtilizadorID:    c.UtilizadorID,
		})
		if err != nil {
			return err
		}
		criada = cotacaoFromRow(row)
		for _, i := range itens {
			ir, err := q.CriarItemCotacao(ctx, sqlcgen.CriarItemCotacaoParams{
				CotacaoID:        criada.ID,
				ProdutoID:        i.ProdutoID,
				UnidadeProdutoID: i.UnidadeProdutoID,
				Descricao:        i.Descricao,
				NomeUnidade:      i.NomeUnidade,
				Quantidade:       i.Quantidade,
				PrecoUnitario:    i.PrecoUnitario,
				Desconto:         i.Desconto,
				Subtotal:         i.Subtotal,
			})
			if err != nil {
				return err
			}
			criados = append(criados, itemCotacaoFromRow(ir))
		}
		return nil
	})
	return criada, criados, err
}

func (r *CotacaoRepository) Listar(ctx context.Context, negocioID int64) ([]domain.Cotacao, error) {
	rows, err := r.q.ListarCotacoes(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Cotacao, 0, len(rows))
	for _, row := range rows {
		out = append(out, cotacaoFromRow(row))
	}
	return out, nil
}

func (r *CotacaoRepository) Obter(ctx context.Context, id int64) (domain.Cotacao, error) {
	row, err := r.q.ObterCotacao(ctx, id)
	if err != nil {
		return domain.Cotacao{}, err
	}
	return cotacaoFromRow(row), nil
}

func (r *CotacaoRepository) ListarItens(ctx context.Context, cotacaoID int64) ([]domain.ItemCotacao, error) {
	rows, err := r.q.ListarItensCotacao(ctx, cotacaoID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ItemCotacao, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemCotacaoFromRow(row))
	}
	return out, nil
}

// Cancelar só muda uma cotação aberta; devolve false se já não estava.
func (r *CotacaoRepository) Cancelar(ctx context.Context, id int64) (bool, error) {
	n, err := r.q.CancelarCotacao(ctx, id)
	return n == 1, err
}

// MarcarConvertida liga a cotação à venda gerada; só uma cotação aberta
// muda, por isso converter duas vezes a mesma falha na segunda.
func (r *CotacaoRepository) MarcarConvertida(ctx context.Context, id, vendaID int64) (bool, error) {
	n, err := r.q.MarcarCotacaoConvertida(ctx, sqlcgen.MarcarCotacaoConvertidaParams{ID: id, VendaID: sql.NullInt64{Int64: vendaID, Valid: true}})
	return n == 1, err
}

func cotacaoFromRow(row sqlcgen.Cotacoes) domain.Cotacao {
	return domain.Cotacao{
		ID:              row.ID,
		NegocioID:       row.NegocioID,
		Ano:             row.Ano,
		Numero:          row.Numero,
		ClienteID:       nullInt64ToPtr(row.ClienteID),
		ClienteNome:     row.ClienteNome,
		ClienteNUIT:     nullStringToPtr(row.ClienteNuit),
		ClienteContacto: nullStringToPtr(row.ClienteContacto),
		ValidadeDias:    row.ValidadeDias,
		Subtotal:        row.Subtotal,
		DescontoTotal:   row.DescontoTotal,
		Total:           row.Total,
		TaxaIVA:         row.TaxaIva,
		ValorIVA:        row.ValorIva,
		Observacoes:     nullStringToPtr(row.Observacoes),
		Estado:          domain.EstadoCotacao(row.Estado),
		VendaID:         nullInt64ToPtr(row.VendaID),
		CriadoEm:        parseTimestamp(row.CriadoEm),
		UtilizadorID:    row.UtilizadorID,
	}
}

func itemCotacaoFromRow(row sqlcgen.CotacaoItens) domain.ItemCotacao {
	return domain.ItemCotacao(row)
}
