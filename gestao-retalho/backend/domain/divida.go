package domain

import "time"

type StatusDivida string

const (
	StatusDividaAberta  StatusDivida = "aberta"
	StatusDividaParcial StatusDivida = "parcial"
	StatusDividaPaga    StatusDivida = "paga"
)

// Divida é criada ao fechar uma conta/venda sem cobrir o total (SDD
// 4.10/4.11) — só clientes identificados (não anónimos) podem ficar com
// saldo em aberto. O fluxo de pagar uma dívida chega na Fase 7; por agora
// só o registo é criado.
type Divida struct {
	ID            int64
	ClienteID     int64
	VendaID       int64
	ValorOriginal float64
	ValorPago     float64
	ValorEmAberto float64
	Status        StatusDivida
	CriadoEm      time.Time
}
