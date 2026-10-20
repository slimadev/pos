package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type DocumentoFiscalRepository struct {
	db *sql.DB
	q  *sqlcgen.Queries
}

func NewDocumentoFiscalRepository(db *sql.DB) *DocumentoFiscalRepository {
	return &DocumentoFiscalRepository{db: db, q: sqlcgen.New(db)}
}

// Criar grava o documento e as linhas numa transação, com o número seguinte
// do tipo e do ano lido na mesma transação — a numeração não tem falhas
// nem repetições.
func (r *DocumentoFiscalRepository) Criar(ctx context.Context, d domain.DocumentoFiscal, linhas []domain.LinhaDocumento) (domain.DocumentoFiscal, []domain.LinhaDocumento, error) {
	var criado domain.DocumentoFiscal
	criadas := make([]domain.LinhaDocumento, 0, len(linhas))
	err := runInTx(ctx, r.db, func(tx *sql.Tx) error {
		q := r.q.WithTx(tx)
		numero, err := q.ProximoNumeroDocumento(ctx, sqlcgen.ProximoNumeroDocumentoParams{NegocioID: d.NegocioID, Tipo: string(d.Tipo), Ano: d.Ano})
		if err != nil {
			return err
		}
		row, err := q.CriarDocumentoFiscal(ctx, sqlcgen.CriarDocumentoFiscalParams{
			NegocioID:         d.NegocioID,
			Tipo:              string(d.Tipo),
			Ano:               d.Ano,
			Numero:            numero,
			VendaID:           ptrToNullInt64(d.VendaID),
			PagamentoDividaID: ptrToNullInt64(d.PagamentoDividaID),
			DocumentoOrigemID: ptrToNullInt64(d.DocumentoOrigemID),
			EmitenteNome:      d.EmitenteNome,
			EmitenteNuit:      ptrToNullString(d.EmitenteNUIT),
			EmitenteEndereco:  ptrToNullString(d.EmitenteEndereco),
			EmitenteContacto:  ptrToNullString(d.EmitenteContacto),
			ClienteID:         ptrToNullInt64(d.ClienteID),
			ClienteNome:       d.ClienteNome,
			ClienteNuit:       ptrToNullString(d.ClienteNUIT),
			ClienteEndereco:   ptrToNullString(d.ClienteEndereco),
			Subtotal:          d.Subtotal,
			DescontoTotal:     d.DescontoTotal,
			Total:             d.Total,
			TaxaIva:           d.TaxaIVA,
			ValorIva:          d.ValorIVA,
			Motivo:            ptrToNullString(d.Motivo),
			UtilizadorID:      d.UtilizadorID,
		})
		if err != nil {
			return err
		}
		criado = documentoFromRow(row)
		for _, l := range linhas {
			lr, err := q.CriarLinhaDocumento(ctx, sqlcgen.CriarLinhaDocumentoParams{
				DocumentoID:   criado.ID,
				Descricao:     l.Descricao,
				Unidade:       l.Unidade,
				Quantidade:    l.Quantidade,
				PrecoUnitario: l.PrecoUnitario,
				Desconto:      l.Desconto,
				Subtotal:      l.Subtotal,
			})
			if err != nil {
				return err
			}
			criadas = append(criadas, domain.LinhaDocumento(lr))
		}
		return nil
	})
	return criado, criadas, err
}

func (r *DocumentoFiscalRepository) Obter(ctx context.Context, id int64) (domain.DocumentoFiscal, error) {
	row, err := r.q.ObterDocumentoFiscal(ctx, id)
	if err != nil {
		return domain.DocumentoFiscal{}, err
	}
	return documentoFromRow(row), nil
}

func (r *DocumentoFiscalRepository) ListarLinhas(ctx context.Context, documentoID int64) ([]domain.LinhaDocumento, error) {
	rows, err := r.q.ListarLinhasDocumento(ctx, documentoID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.LinhaDocumento, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.LinhaDocumento(row))
	}
	return out, nil
}

// opcional converte "não encontrado" em (zero, false, nil).
func opcional(row sqlcgen.DocumentosFiscais, err error) (domain.DocumentoFiscal, bool, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return domain.DocumentoFiscal{}, false, nil
	}
	if err != nil {
		return domain.DocumentoFiscal{}, false, err
	}
	return documentoFromRow(row), true, nil
}

func (r *DocumentoFiscalRepository) FaturaDaVenda(ctx context.Context, vendaID int64) (domain.DocumentoFiscal, bool, error) {
	return opcional(r.q.ObterFaturaDaVenda(ctx, sql.NullInt64{Int64: vendaID, Valid: true}))
}

func (r *DocumentoFiscalRepository) NotaCreditoDaVenda(ctx context.Context, vendaID int64) (domain.DocumentoFiscal, bool, error) {
	return opcional(r.q.ObterNotaCreditoDaVenda(ctx, sql.NullInt64{Int64: vendaID, Valid: true}))
}

func (r *DocumentoFiscalRepository) ReciboDoPagamento(ctx context.Context, pagamentoID int64) (domain.DocumentoFiscal, bool, error) {
	return opcional(r.q.ObterReciboDoPagamento(ctx, sql.NullInt64{Int64: pagamentoID, Valid: true}))
}

func (r *DocumentoFiscalRepository) ListarEntre(ctx context.Context, negocioID int64, desde, ate time.Time) ([]domain.DocumentoFiscal, error) {
	rows, err := r.q.ListarDocumentosEntre(ctx, sqlcgen.ListarDocumentosEntreParams{
		NegocioID: negocioID,
		Desde:     desde.UTC().Format(timestampLayout),
		Ate:       ate.UTC().Format(timestampLayout),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.DocumentoFiscal, 0, len(rows))
	for _, row := range rows {
		out = append(out, documentoFromRow(row))
	}
	return out, nil
}

// VendaSemFatura é uma venda feita com a faturação ligada que ficou sem
// FR/FT (a emissão falhou) — para emitir de novo.
type VendaSemFatura struct {
	VendaID  int64
	Total    float64
	Status   string
	CriadoEm time.Time
}

func (r *DocumentoFiscalRepository) VendasSemFatura(ctx context.Context, negocioID int64, desde time.Time) ([]VendaSemFatura, error) {
	rows, err := r.q.ListarVendasSemFatura(ctx, sqlcgen.ListarVendasSemFaturaParams{
		NegocioID: negocioID,
		Desde:     desde.UTC().Format(timestampLayout),
	})
	if err != nil {
		return nil, err
	}
	out := make([]VendaSemFatura, 0, len(rows))
	for _, row := range rows {
		out = append(out, VendaSemFatura{VendaID: row.ID, Total: row.Total, Status: row.Status, CriadoEm: parseTimestamp(row.CriadoEm)})
	}
	return out, nil
}

func documentoFromRow(row sqlcgen.DocumentosFiscais) domain.DocumentoFiscal {
	return domain.DocumentoFiscal{
		ID:                row.ID,
		NegocioID:         row.NegocioID,
		Tipo:              domain.TipoDocumento(row.Tipo),
		Ano:               row.Ano,
		Numero:            row.Numero,
		VendaID:           nullInt64ToPtr(row.VendaID),
		PagamentoDividaID: nullInt64ToPtr(row.PagamentoDividaID),
		DocumentoOrigemID: nullInt64ToPtr(row.DocumentoOrigemID),
		EmitenteNome:      row.EmitenteNome,
		EmitenteNUIT:      nullStringToPtr(row.EmitenteNuit),
		EmitenteEndereco:  nullStringToPtr(row.EmitenteEndereco),
		EmitenteContacto:  nullStringToPtr(row.EmitenteContacto),
		ClienteID:         nullInt64ToPtr(row.ClienteID),
		ClienteNome:       row.ClienteNome,
		ClienteNUIT:       nullStringToPtr(row.ClienteNuit),
		ClienteEndereco:   nullStringToPtr(row.ClienteEndereco),
		Subtotal:          row.Subtotal,
		DescontoTotal:     row.DescontoTotal,
		Total:             row.Total,
		TaxaIVA:           row.TaxaIva,
		ValorIVA:          row.ValorIva,
		Motivo:            nullStringToPtr(row.Motivo),
		CriadoEm:          parseTimestamp(row.CriadoEm),
		UtilizadorID:      row.UtilizadorID,
	}
}
