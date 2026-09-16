package domain

import "time"

// PagamentoDivida é um pagamento contra uma dívida existente, independente
// de qualquer venda nova (SDD 3.2/4.11).
type PagamentoDivida struct {
	ID           int64
	DividaID     int64
	Valor        float64
	Metodo       MetodoPagamento
	CriadoEm     time.Time
	UtilizadorID int64
}
