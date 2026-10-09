package repository

import (
	"context"
	"database/sql"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

// RelatorioRepository (Fase 8, SDD secção 6) só lê — nenhum destes métodos
// participa de nenhuma transação de escrita, por isso não segue o padrão
// WithTx dos outros repositórios. Cada linha devolvida já vem com o nome do
// produto/armazém juntado em SQL para o frontend não ter de fazer N
// pedidos extra só para rotular uma tabela.
type RelatorioRepository struct {
	q *sqlcgen.Queries
}

func NewRelatorioRepository(db *sql.DB) *RelatorioRepository {
	return &RelatorioRepository{q: sqlcgen.New(db)}
}

type PeriodoFiltro struct {
	NegocioID int64
	Desde     time.Time
	Ate       time.Time
	ArmazemID *int64
}

func (f PeriodoFiltro) desdeStr() string { return f.Desde.UTC().Format(timestampLayout) }
func (f PeriodoFiltro) ateStr() string   { return f.Ate.UTC().Format(timestampLayout) }

type LinhaVendasPorDia struct {
	Dia          string
	NumeroVendas int64
	TotalVendido float64
}

// VendasPorDia agrupa pelo dia do relógio da loja (date(criado_em,
// 'localtime') na query), não pelo dia UTC em que criado_em é guardado.
func (r *RelatorioRepository) VendasPorDia(ctx context.Context, f PeriodoFiltro) ([]LinhaVendasPorDia, error) {
	rows, err := r.q.RelatorioVendasPorDia(ctx, sqlcgen.RelatorioVendasPorDiaParams{
		NegocioID: f.NegocioID,
		Desde:     f.desdeStr(),
		Ate:       f.ateStr(),
		ArmazemID: interfaceArmazemFiltro(f.ArmazemID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]LinhaVendasPorDia, 0, len(rows))
	for _, row := range rows {
		dia, _ := row.Dia.(string)
		out = append(out, LinhaVendasPorDia{
			Dia:          dia,
			NumeroVendas: row.NumeroVendas,
			TotalVendido: anyToFloat64(row.TotalVendido),
		})
	}
	return out, nil
}

type LucroPeriodo struct {
	Receita float64
	Custo   float64
}

func (r *RelatorioRepository) LucroPorPeriodo(ctx context.Context, f PeriodoFiltro) (LucroPeriodo, error) {
	row, err := r.q.RelatorioLucroPorPeriodo(ctx, sqlcgen.RelatorioLucroPorPeriodoParams{
		NegocioID: f.NegocioID,
		Desde:     f.desdeStr(),
		Ate:       f.ateStr(),
		ArmazemID: interfaceArmazemFiltro(f.ArmazemID),
	})
	if err != nil {
		return LucroPeriodo{}, err
	}
	return LucroPeriodo{Receita: anyToFloat64(row.Receita), Custo: anyToFloat64(row.Custo)}, nil
}

type LinhaVendasPorProduto struct {
	ProdutoID       int64
	ProdutoNome     string
	QuantidadeTotal float64
	ValorTotal      float64
}

func (r *RelatorioRepository) VendasPorProduto(ctx context.Context, f PeriodoFiltro) ([]LinhaVendasPorProduto, error) {
	rows, err := r.q.RelatorioVendasPorProduto(ctx, sqlcgen.RelatorioVendasPorProdutoParams{
		NegocioID: f.NegocioID,
		Desde:     f.desdeStr(),
		Ate:       f.ateStr(),
		ArmazemID: interfaceArmazemFiltro(f.ArmazemID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]LinhaVendasPorProduto, 0, len(rows))
	for _, row := range rows {
		out = append(out, LinhaVendasPorProduto{
			ProdutoID:       row.ProdutoID,
			ProdutoNome:     row.ProdutoNome,
			QuantidadeTotal: anyToFloat64(row.QuantidadeTotal),
			ValorTotal:      anyToFloat64(row.ValorTotal),
		})
	}
	return out, nil
}

type LinhaVendasPorMetodo struct {
	Metodo domain.MetodoPagamento
	Total  float64
}

func (r *RelatorioRepository) VendasPorMetodoPagamento(ctx context.Context, f PeriodoFiltro) ([]LinhaVendasPorMetodo, error) {
	rows, err := r.q.RelatorioVendasPorMetodoPagamento(ctx, sqlcgen.RelatorioVendasPorMetodoPagamentoParams{
		NegocioID: f.NegocioID,
		Desde:     f.desdeStr(),
		Ate:       f.ateStr(),
		ArmazemID: interfaceArmazemFiltro(f.ArmazemID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]LinhaVendasPorMetodo, 0, len(rows))
	for _, row := range rows {
		out = append(out, LinhaVendasPorMetodo{Metodo: domain.MetodoPagamento(row.Metodo), Total: anyToFloat64(row.Total)})
	}
	return out, nil
}

// ItensVendaPorPeriodo alimenta o relatório de consumo teórico vs real
// (RelatorioService, em usecase) — devolve os itens de venda em bruto do
// período para o usecase expandir fichas técnicas produto a produto.
func (r *RelatorioRepository) ItensVendaPorPeriodo(ctx context.Context, f PeriodoFiltro) ([]domain.ItemVenda, error) {
	rows, err := r.q.ItensVendaPorPeriodo(ctx, sqlcgen.ItensVendaPorPeriodoParams{
		NegocioID: f.NegocioID,
		Desde:     f.desdeStr(),
		Ate:       f.ateStr(),
		ArmazemID: interfaceArmazemFiltro(f.ArmazemID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.ItemVenda, 0, len(rows))
	for _, row := range rows {
		out = append(out, domain.ItemVenda{
			ID:                     row.ID,
			VendaID:                row.VendaID,
			ProdutoID:              row.ProdutoID,
			UnidadeProdutoID:       row.UnidadeProdutoID,
			Quantidade:             row.Quantidade,
			PrecoUnitario:          row.PrecoUnitario,
			CustoUnitarioNoMomento: row.CustoUnitarioNoMomento,
			Desconto:               row.Desconto,
			Subtotal:               row.Subtotal,
		})
	}
	return out, nil
}

type LinhaMovimento struct {
	ID                    int64
	ProdutoID             int64
	ProdutoNome           string
	ArmazemOrigemID       *int64
	ArmazemDestinoID      *int64
	Tipo                  domain.TipoMovimento
	QuantidadeUnidadeBase float64
	CustoUnitario         *float64
	Motivo                *string
	CriadoEm              string
}

type MovimentosFiltro struct {
	Desde     time.Time
	Ate       time.Time
	ArmazemID *int64
}

func (f MovimentosFiltro) desdeStr() string { return f.Desde.UTC().Format(timestampLayout) }
func (f MovimentosFiltro) ateStr() string   { return f.Ate.UTC().Format(timestampLayout) }

func (r *RelatorioRepository) MovimentosPorPeriodo(ctx context.Context, f MovimentosFiltro) ([]LinhaMovimento, error) {
	rows, err := r.q.RelatorioMovimentosPorPeriodo(ctx, sqlcgen.RelatorioMovimentosPorPeriodoParams{
		Desde:     f.desdeStr(),
		Ate:       f.ateStr(),
		ArmazemID: interfaceArmazemFiltro(f.ArmazemID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]LinhaMovimento, 0, len(rows))
	for _, row := range rows {
		out = append(out, LinhaMovimento{
			ID:                    row.ID,
			ProdutoID:             row.ProdutoID,
			ProdutoNome:           row.ProdutoNome,
			ArmazemOrigemID:       nullInt64ToPtr(row.ArmazemOrigemID),
			ArmazemDestinoID:      nullInt64ToPtr(row.ArmazemDestinoID),
			Tipo:                  domain.TipoMovimento(row.Tipo),
			QuantidadeUnidadeBase: row.QuantidadeUnidadeBase,
			CustoUnitario:         nullFloat64ToPtr(row.CustoUnitario),
			Motivo:                nullStringToPtr(row.Motivo),
			CriadoEm:              row.CriadoEm,
		})
	}
	return out, nil
}

type LinhaPerdaConsumo struct {
	Tipo            domain.TipoMovimento
	ProdutoID       int64
	ProdutoNome     string
	QuantidadeTotal float64
	ValorTotal      float64
}

func (r *RelatorioRepository) PerdasConsumoInterno(ctx context.Context, f MovimentosFiltro) ([]LinhaPerdaConsumo, error) {
	rows, err := r.q.RelatorioPerdasConsumoInterno(ctx, sqlcgen.RelatorioPerdasConsumoInternoParams{
		Desde:     f.desdeStr(),
		Ate:       f.ateStr(),
		ArmazemID: interfaceArmazemFiltro(f.ArmazemID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]LinhaPerdaConsumo, 0, len(rows))
	for _, row := range rows {
		out = append(out, LinhaPerdaConsumo{
			Tipo:            domain.TipoMovimento(row.Tipo),
			ProdutoID:       row.ProdutoID,
			ProdutoNome:     row.ProdutoNome,
			QuantidadeTotal: anyToFloat64(row.QuantidadeTotal),
			ValorTotal:      anyToFloat64(row.ValorTotal),
		})
	}
	return out, nil
}

type LinhaStockAtual struct {
	ProdutoID             int64
	ProdutoNome           string
	ArmazemID             int64
	QuantidadeUnidadeBase float64
	CustoMedioPonderado   float64
	StockMinimo           *float64
}

func (r *RelatorioRepository) StockAtual(ctx context.Context, negocioID int64, armazemID *int64) ([]LinhaStockAtual, error) {
	rows, err := r.q.RelatorioStockAtual(ctx, sqlcgen.RelatorioStockAtualParams{
		NegocioID: negocioID,
		ArmazemID: interfaceArmazemFiltro(armazemID),
	})
	if err != nil {
		return nil, err
	}
	out := make([]LinhaStockAtual, 0, len(rows))
	for _, row := range rows {
		out = append(out, LinhaStockAtual{
			ProdutoID:             row.ProdutoID,
			ProdutoNome:           row.ProdutoNome,
			ArmazemID:             row.ArmazemID.Int64,
			QuantidadeUnidadeBase: row.QuantidadeUnidadeBase.Float64,
			CustoMedioPonderado:   row.CustoMedioPonderado,
			StockMinimo:           nullFloat64ToPtr(row.StockMinimo),
		})
	}
	return out, nil
}

func (r *RelatorioRepository) ContasPorPeriodo(ctx context.Context, negocioID int64, desde, ate time.Time) ([]domain.Conta, error) {
	rows, err := r.q.RelatorioContasPorPeriodo(ctx, sqlcgen.RelatorioContasPorPeriodoParams{
		NegocioID: negocioID,
		Desde:     desde.UTC().Format(timestampLayout),
		Ate:       ate.UTC().Format(timestampLayout),
	})
	if err != nil {
		return nil, err
	}
	out := make([]domain.Conta, 0, len(rows))
	for _, row := range rows {
		out = append(out, contaFromRow(row))
	}
	return out, nil
}

// ValorPorConta devolve o valor total (soma de itens_conta) de TODAS as
// contas já lançadas, indexado por conta_id — RelatorioService cruza isto
// com ContasPorPeriodo em Go em vez de um JOIN, porque só interessam as
// poucas contas already filtradas pelo período.
func (r *RelatorioRepository) ValorPorConta(ctx context.Context) (map[int64]float64, error) {
	rows, err := r.q.RelatorioValorContas(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[int64]float64, len(rows))
	for _, row := range rows {
		out[row.ContaID] = anyToFloat64(row.ValorTotal)
	}
	return out, nil
}
