package domain

import (
	"fmt"
	"time"
)

type TipoDocumento string

const (
	TipoFaturaRecibo TipoDocumento = "FR"
	TipoFatura       TipoDocumento = "FT"
	TipoRecibo       TipoDocumento = "RC"
	TipoNotaCredito  TipoDocumento = "NC"
)

// Nome é como o tipo aparece no documento.
func (t TipoDocumento) Nome() string {
	switch t {
	case TipoFaturaRecibo:
		return "Fatura-Recibo"
	case TipoFatura:
		return "Fatura"
	case TipoRecibo:
		return "Recibo"
	case TipoNotaCredito:
		return "Nota de Crédito"
	}
	return string(t)
}

// DocumentoFiscal é uma fatura, fatura-recibo, recibo ou nota de crédito.
// Depois de emitido nunca muda (a base de dados recusa alterações): o
// emitente e o adquirente ficam como estavam no momento da emissão, e os
// valores incluem IVA (ValorIVA é a parte do total que é imposto).
type DocumentoFiscal struct {
	ID                int64
	NegocioID         int64
	Tipo              TipoDocumento
	Ano               int64
	Numero            int64
	VendaID           *int64
	PagamentoDividaID *int64
	DocumentoOrigemID *int64
	EmitenteNome      string
	EmitenteNUIT      *string
	EmitenteEndereco  *string
	EmitenteContacto  *string
	ClienteID         *int64
	ClienteNome       string
	ClienteNUIT       *string
	ClienteEndereco   *string
	Subtotal          float64
	DescontoTotal     float64
	Total             float64
	TaxaIVA           float64
	ValorIVA          float64
	Motivo            *string
	CriadoEm          time.Time
	UtilizadorID      int64
}

// Codigo é o número como aparece no documento: "FR 2026/0012".
func (d DocumentoFiscal) Codigo() string {
	return fmt.Sprintf("%s %d/%04d", d.Tipo, d.Ano, d.Numero)
}

type LinhaDocumento struct {
	ID            int64
	DocumentoID   int64
	Descricao     string
	Unidade       string
	Quantidade    float64
	PrecoUnitario float64
	Desconto      float64
	Subtotal      float64
}
