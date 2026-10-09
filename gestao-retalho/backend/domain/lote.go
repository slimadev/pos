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

// VencidoEm diz se o lote já passou da validade no dia de `agora` — a
// validade conta até ao fim do próprio dia (é guardada como a data, sem
// hora, em UTC).
func (l Lote) VencidoEm(agora time.Time) bool {
	return l.DataValidade.UTC().Format("2006-01-02") < agora.Format("2006-01-02")
}
