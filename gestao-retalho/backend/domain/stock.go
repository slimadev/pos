package domain

import "time"

type TipoMovimento string

const (
	TipoMovimentoEntrada          TipoMovimento = "entrada"
	TipoMovimentoTransferencia    TipoMovimento = "transferencia"
	TipoMovimentoPerda            TipoMovimento = "perda"
	TipoMovimentoConsumoInterno   TipoMovimento = "consumo_interno"
	TipoMovimentoAjusteInventario TipoMovimento = "ajuste_inventario"
	// Venda e Producao só passam a ser emitidos a partir da Fase 3/5, mas
	// já vivem aqui porque o schema é o mesmo para todas as fases (SDD 8.4).
	TipoMovimentoVenda    TipoMovimento = "venda"
	TipoMovimentoProducao TipoMovimento = "producao"
)

// MovimentoStock é o único mecanismo de alteração de stock (SDD 3.2) — tudo
// o resto (stock_atual, relatórios de movimento) deriva daqui, nunca ao
// contrário.
type MovimentoStock struct {
	ID                    int64
	ProdutoID             int64
	ArmazemOrigemID       *int64
	ArmazemDestinoID      *int64
	Tipo                  TipoMovimento
	QuantidadeUnidadeBase float64
	CustoUnitario         *float64
	UnidadeProdutoID      *int64
	Motivo                *string
	ReferenciaID          *string
	UtilizadorID          int64
	CriadoEm              time.Time
}

// StockItem é uma linha de stock_atual: quantidade e custo médio ponderado
// de um produto num armazém específico.
type StockItem struct {
	ProdutoID           int64
	ArmazemID           int64
	Quantidade          float64
	CustoMedioPonderado float64
}
