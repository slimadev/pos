package domain

import "time"

// Lote é uma entrada de stock rastreada por número de lote e validade (SDD
// 9.4) — só existe para produtos com LoteControlado=true. A venda consome
// lotes por ordem de validade (FEFO: First-Expire-First-Out).
type Lote struct {
	ID            int64
	ProdutoID     int64
	ArmazemID     int64
	NumeroLote    string
	DataValidade  time.Time
	Quantidade    float64
	CustoUnitario float64
}
