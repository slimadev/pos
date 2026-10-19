package repository

import (
	"context"
	"database/sql"
	"errors"
	"strconv"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

// VendaRepository junta contas/itens_conta/vendas/itens_venda/pagamentos —
// as cinco tabelas que uma venda direta (SDD 3.5) ou o fecho de uma conta
// (Fase 4) têm de escrever atomicamente, incluindo os movimentos de stock
// resultantes. Por isso carrega um StockRepository e um LoteRepository
// sempre ligados à mesma transação que as suas próprias queries (ver
// WithTx) — consumir por FEFO (Fase 6) tem de decrementar lotes_produto
// e gravar o movimento de stock atomicamente, tal como a ficha técnica
// (Fase 5) tem de expandir ingredientes atomicamente.
type VendaRepository struct {
	db    *sql.DB
	q     *sqlcgen.Queries
	Stock *StockRepository
	Lotes *LoteRepository
}

func NewVendaRepository(db *sql.DB, stock *StockRepository, lotes *LoteRepository) *VendaRepository {
	return &VendaRepository{db: db, q: sqlcgen.New(db), Stock: stock, Lotes: lotes}
}

func (r *VendaRepository) WithTx(ctx context.Context, fn func(*VendaRepository) error) error {
	return runInTx(ctx, r.db, func(tx *sql.Tx) error {
		txStock := &StockRepository{db: r.db, q: sqlcgen.New(tx)}
		txRepo := &VendaRepository{
			db:    r.db,
			q:     r.q.WithTx(tx),
			Stock: txStock,
			Lotes: &LoteRepository{db: r.db, q: sqlcgen.New(tx), Stock: txStock},
		}
		return fn(txRepo)
	})
}

// --- Contas --------------------------------------------------------------

func (r *VendaRepository) CriarConta(ctx context.Context, negocioID, clienteID int64, identificador string, utilizadorAberturaID int64) (domain.Conta, error) {
	row, err := r.q.CriarConta(ctx, sqlcgen.CriarContaParams{
		NegocioID:            negocioID,
		ClienteID:            clienteID,
		Identificador:        identificador,
		UtilizadorAberturaID: utilizadorAberturaID,
	})
	if err != nil {
		return domain.Conta{}, err
	}
	return contaFromRow(row), nil
}

func (r *VendaRepository) FecharConta(ctx context.Context, contaID, utilizadorFechoID int64) (domain.Conta, error) {
	row, err := r.q.FecharConta(ctx, sqlcgen.FecharContaParams{
		UtilizadorFechoID: sql.NullInt64{Int64: utilizadorFechoID, Valid: true},
		ID:                contaID,
	})
	if err != nil {
		return domain.Conta{}, err
	}
	return contaFromRow(row), nil
}

func (r *VendaRepository) CriarItemConta(ctx context.Context, item domain.ItemConta) (domain.ItemConta, error) {
	row, err := r.q.CriarItemConta(ctx, sqlcgen.CriarItemContaParams{
		ContaID:               item.ContaID,
		ProdutoID:             item.ProdutoID,
		UnidadeProdutoID:      item.UnidadeProdutoID,
		Quantidade:            item.Quantidade,
		PrecoUnitarioAplicado: item.PrecoUnitarioAplicado,
		Desconto:              item.Desconto,
		UtilizadorID:          item.UtilizadorID,
	})
	if err != nil {
		return domain.ItemConta{}, err
	}
	return itemContaFromRow(row), nil
}

func (r *VendaRepository) ListarItensConta(ctx context.Context, contaID int64) ([]domain.ItemConta, error) {
	rows, err := r.q.ListarItensConta(ctx, contaID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ItemConta, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemContaFromRow(row))
	}
	return out, nil
}

func (r *VendaRepository) ObterItemConta(ctx context.Context, id int64) (domain.ItemConta, error) {
	row, err := r.q.ObterItemConta(ctx, id)
	if err != nil {
		return domain.ItemConta{}, err
	}
	return itemContaFromRow(row), nil
}

// TransferirItemConta move uma linha inteira para outra conta aberta (SDD
// 4.10, "transferir itens entre contas") — move a linha completa, não
// fraciona quantidade entre contas.
func (r *VendaRepository) TransferirItemConta(ctx context.Context, itemID, novaContaID int64) error {
	return r.q.TransferirItemConta(ctx, sqlcgen.TransferirItemContaParams{
		ContaID: novaContaID,
		ID:      itemID,
	})
}

func (r *VendaRepository) ObterConta(ctx context.Context, id int64) (domain.Conta, error) {
	row, err := r.q.ObterConta(ctx, id)
	if err != nil {
		return domain.Conta{}, err
	}
	return contaFromRow(row), nil
}

func (r *VendaRepository) ListarContasAbertas(ctx context.Context, negocioID int64) ([]domain.Conta, error) {
	rows, err := r.q.ListarContasAbertas(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Conta, 0, len(rows))
	for _, row := range rows {
		out = append(out, contaFromRow(row))
	}
	return out, nil
}

func (r *VendaRepository) CancelarConta(ctx context.Context, contaID, utilizadorFechoID int64, motivo string) (domain.Conta, error) {
	row, err := r.q.CancelarConta(ctx, sqlcgen.CancelarContaParams{
		UtilizadorFechoID:  sql.NullInt64{Int64: utilizadorFechoID, Valid: true},
		MotivoCancelamento: sql.NullString{String: motivo, Valid: true},
		ID:                 contaID,
	})
	if err != nil {
		return domain.Conta{}, err
	}
	return contaFromRow(row), nil
}

// --- Dívidas (SDD 4.11) -----------------------------------------------------

func (r *VendaRepository) CriarDivida(ctx context.Context, d domain.Divida) (domain.Divida, error) {
	row, err := r.q.CriarDivida(ctx, sqlcgen.CriarDividaParams{
		ClienteID:     d.ClienteID,
		VendaID:       d.VendaID,
		ValorOriginal: d.ValorOriginal,
		ValorPago:     d.ValorPago,
		ValorEmAberto: d.ValorEmAberto,
		Status:        string(d.Status),
	})
	if err != nil {
		return domain.Divida{}, err
	}
	return dividaFromRow(row), nil
}

func (r *VendaRepository) ListarDividasPorCliente(ctx context.Context, clienteID int64) ([]domain.Divida, error) {
	rows, err := r.q.ListarDividasPorCliente(ctx, clienteID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Divida, 0, len(rows))
	for _, row := range rows {
		out = append(out, dividaFromRow(row))
	}
	return out, nil
}

// --- Vendas ----------------------------------------------------------------

func (r *VendaRepository) CriarVenda(ctx context.Context, v domain.Venda) (domain.Venda, error) {
	row, err := r.q.CriarVenda(ctx, sqlcgen.CriarVendaParams{
		NegocioID:     v.NegocioID,
		ArmazemID:     v.ArmazemID,
		ClienteID:     v.ClienteID,
		ContaID:       ptrToNullInt64(v.ContaID),
		TipoVenda:     string(v.TipoVenda),
		Subtotal:      v.Subtotal,
		DescontoTotal: v.DescontoTotal,
		Total:         v.Total,
		Status:        string(v.Status),
		UtilizadorID:  v.UtilizadorID,
	})
	if err != nil {
		return domain.Venda{}, err
	}
	return vendaFromRow(row), nil
}

func (r *VendaRepository) ObterVenda(ctx context.Context, id int64) (domain.Venda, error) {
	row, err := r.q.ObterVenda(ctx, id)
	if err != nil {
		return domain.Venda{}, err
	}
	return vendaFromRow(row), nil
}

// ListarPorCliente cobre o "histórico de compras" do SDD 4.9.
func (r *VendaRepository) ListarPorCliente(ctx context.Context, clienteID int64) ([]domain.Venda, error) {
	rows, err := r.q.ListarVendasPorCliente(ctx, clienteID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Venda, 0, len(rows))
	for _, row := range rows {
		out = append(out, vendaFromRow(row))
	}
	return out, nil
}

func (r *VendaRepository) ListarPorPeriodo(ctx context.Context, negocioID int64, desde, ate time.Time) ([]domain.Venda, error) {
	rows, err := r.q.ListarVendasPorPeriodo(ctx, sqlcgen.ListarVendasPorPeriodoParams{
		NegocioID:  negocioID,
		CriadoEm:   desde.UTC().Format(timestampLayout),
		CriadoEm_2: ate.UTC().Format(timestampLayout),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Venda, 0, len(rows))
	for _, row := range rows {
		out = append(out, vendaFromRow(row))
	}
	return out, nil
}

func (r *VendaRepository) CriarItemVenda(ctx context.Context, item domain.ItemVenda) (domain.ItemVenda, error) {
	row, err := r.q.CriarItemVenda(ctx, sqlcgen.CriarItemVendaParams{
		VendaID:                item.VendaID,
		ProdutoID:              item.ProdutoID,
		UnidadeProdutoID:       item.UnidadeProdutoID,
		Quantidade:             item.Quantidade,
		PrecoUnitario:          item.PrecoUnitario,
		CustoUnitarioNoMomento: item.CustoUnitarioNoMomento,
		Desconto:               item.Desconto,
		Subtotal:               item.Subtotal,
	})
	if err != nil {
		return domain.ItemVenda{}, err
	}
	return itemVendaFromRow(row), nil
}

func (r *VendaRepository) ListarItensVenda(ctx context.Context, vendaID int64) ([]domain.ItemVenda, error) {
	rows, err := r.q.ListarItensVenda(ctx, vendaID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.ItemVenda, 0, len(rows))
	for _, row := range rows {
		out = append(out, itemVendaFromRow(row))
	}
	return out, nil
}

func (r *VendaRepository) CriarPagamento(ctx context.Context, p domain.Pagamento) (domain.Pagamento, error) {
	row, err := r.q.CriarPagamento(ctx, sqlcgen.CriarPagamentoParams{
		VendaID: p.VendaID,
		Metodo:  string(p.Metodo),
		Valor:   p.Valor,
	})
	if err != nil {
		return domain.Pagamento{}, err
	}
	return pagamentoFromRow(row), nil
}

func (r *VendaRepository) ListarPagamentos(ctx context.Context, vendaID int64) ([]domain.Pagamento, error) {
	rows, err := r.q.ListarPagamentos(ctx, vendaID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Pagamento, 0, len(rows))
	for _, row := range rows {
		out = append(out, pagamentoFromRow(row))
	}
	return out, nil
}

// --- conversões ------------------------------------------------------------

func contaFromRow(row sqlcgen.Contas) domain.Conta {
	var fechadaEm *time.Time
	if row.FechadaEm.Valid {
		t := parseTimestamp(row.FechadaEm.String)
		fechadaEm = &t
	}
	return domain.Conta{
		ID:                   row.ID,
		NegocioID:            row.NegocioID,
		ClienteID:            row.ClienteID,
		Identificador:        row.Identificador,
		Status:               domain.StatusConta(row.Status),
		AbertaEm:             parseTimestamp(row.AbertaEm),
		FechadaEm:            fechadaEm,
		UtilizadorAberturaID: row.UtilizadorAberturaID,
		UtilizadorFechoID:    nullInt64ToPtr(row.UtilizadorFechoID),
		MotivoCancelamento:   nullStringToPtr(row.MotivoCancelamento),
	}
}

func dividaFromRow(row sqlcgen.Dividas) domain.Divida {
	return domain.Divida{
		ID:            row.ID,
		ClienteID:     row.ClienteID,
		VendaID:       row.VendaID,
		ValorOriginal: row.ValorOriginal,
		ValorPago:     row.ValorPago,
		ValorEmAberto: row.ValorEmAberto,
		Status:        domain.StatusDivida(row.Status),
		CriadoEm:      parseTimestamp(row.CriadoEm),
	}
}

func itemContaFromRow(row sqlcgen.ItensConta) domain.ItemConta {
	return domain.ItemConta{
		ID:                    row.ID,
		ContaID:               row.ContaID,
		ProdutoID:             row.ProdutoID,
		UnidadeProdutoID:      row.UnidadeProdutoID,
		Quantidade:            row.Quantidade,
		PrecoUnitarioAplicado: row.PrecoUnitarioAplicado,
		Desconto:              row.Desconto,
		CriadoEm:              parseTimestamp(row.CriadoEm),
		UtilizadorID:          row.UtilizadorID,
	}
}

func vendaFromRow(row sqlcgen.Vendas) domain.Venda {
	return domain.Venda{
		ID:            row.ID,
		NegocioID:     row.NegocioID,
		ArmazemID:     row.ArmazemID,
		ClienteID:     row.ClienteID,
		ContaID:       nullInt64ToPtr(row.ContaID),
		TipoVenda:     domain.TipoVenda(row.TipoVenda),
		Subtotal:      row.Subtotal,
		DescontoTotal: row.DescontoTotal,
		Total:         row.Total,
		Status:        domain.StatusVenda(row.Status),
		CriadoEm:      parseTimestamp(row.CriadoEm),
		UtilizadorID:  row.UtilizadorID,
	}
}

func itemVendaFromRow(row sqlcgen.ItensVenda) domain.ItemVenda {
	return domain.ItemVenda{
		ID:                     row.ID,
		VendaID:                row.VendaID,
		ProdutoID:              row.ProdutoID,
		UnidadeProdutoID:       row.UnidadeProdutoID,
		Quantidade:             row.Quantidade,
		PrecoUnitario:          row.PrecoUnitario,
		CustoUnitarioNoMomento: row.CustoUnitarioNoMomento,
		Desconto:               row.Desconto,
		Subtotal:               row.Subtotal,
	}
}

func pagamentoFromRow(row sqlcgen.Pagamentos) domain.Pagamento {
	return domain.Pagamento{
		ID:       row.ID,
		VendaID:  row.VendaID,
		Metodo:   domain.MetodoPagamento(row.Metodo),
		Valor:    row.Valor,
		CriadoEm: parseTimestamp(row.CriadoEm),
	}
}

// --- Anulação de venda ------------------------------------------------------

// ListarMovimentosDaVenda devolve os movimentos de saída de uma venda: os
// do produto (ou dos ingredientes) com referência "<vendaID>", e os de
// lotes com "<vendaID>;lote:<loteID>".
func (r *VendaRepository) ListarMovimentosDaVenda(ctx context.Context, vendaID int64) ([]domain.MovimentoStock, error) {
	ref := strconv.FormatInt(vendaID, 10)
	rows, err := r.q.ListarMovimentosDaVenda(ctx, sqlcgen.ListarMovimentosDaVendaParams{
		Referencia: sql.NullString{String: ref, Valid: true},
		Prefixo:    sql.NullString{String: ref + ";%", Valid: true},
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.MovimentoStock, 0, len(rows))
	for _, row := range rows {
		out = append(out, movimentoFromRow(row))
	}
	return out, nil
}

// ObterDividaPorVenda devolve ok=false quando a venda não deixou dívida.
func (r *VendaRepository) ObterDividaPorVenda(ctx context.Context, vendaID int64) (domain.Divida, bool, error) {
	row, err := r.q.ObterDividaPorVenda(ctx, vendaID)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Divida{}, false, nil
	}
	if err != nil {
		return domain.Divida{}, false, err
	}
	return dividaFromRow(row), true, nil
}

func (r *VendaRepository) ContarPagamentosDivida(ctx context.Context, dividaID int64) (int, error) {
	rows, err := r.q.ListarPagamentosDivida(ctx, dividaID)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

func (r *VendaRepository) ApagarDivida(ctx context.Context, dividaID int64) error {
	return r.q.ApagarDivida(ctx, dividaID)
}

func (r *VendaRepository) AtualizarStatusVenda(ctx context.Context, vendaID int64, status domain.StatusVenda) error {
	return r.q.AtualizarStatusVenda(ctx, sqlcgen.AtualizarStatusVendaParams{Status: string(status), ID: vendaID})
}
