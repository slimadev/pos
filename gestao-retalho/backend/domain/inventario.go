package domain

import "time"

type StatusInventario string

const (
	StatusInventarioAberto  StatusInventario = "aberto"
	StatusInventarioFechado StatusInventario = "fechado"
)

// Inventario é uma contagem física de um armazém (SDD 4.8): abre com um
// snapshot do stock do sistema, o operador regista o contado, e ao fechar
// gera movimentos `ajuste_inventario` para cada diferença.
type Inventario struct {
	ID           int64
	NegocioID    int64
	ArmazemID    int64
	Data         time.Time
	Status       StatusInventario
	UtilizadorID int64
}

type InventarioItem struct {
	ID                int64
	InventarioID      int64
	ProdutoID         int64
	QuantidadeSistema float64
	QuantidadeContada *float64
	Diferenca         *float64
	Ajustado          bool
}
