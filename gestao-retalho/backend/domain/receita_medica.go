package domain

import "time"

// ReceitaMedica associa uma venda a uma prescrição médica (SDD 9.4) — nome
// distinto de FichaTecnica (secção 3.3) de propósito: uma é a composição
// de um produto, a outra é a prescrição de um médico.
type ReceitaMedica struct {
	ID            int64
	VendaID       int64
	ClienteID     int64
	Medico        string
	NumeroReceita string
	ValidadoPor   int64
	CriadoEm      time.Time
}
