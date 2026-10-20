package domain

import (
	"fmt"
	"time"
)

type EstadoCotacao string

const (
	EstadoCotacaoAberta     EstadoCotacao = "aberta"
	EstadoCotacaoConvertida EstadoCotacao = "convertida"
	EstadoCotacaoCancelada  EstadoCotacao = "cancelada"
)

// Cotacao é um orçamento (proforma) para um cliente: não mexe no stock nem
// conta como venda. Os dados do cliente e os preços ficam congelados no
// momento em que é feita; os preços incluem IVA (ValorIVA é a parte do
// total que é imposto).
type Cotacao struct {
	ID              int64
	NegocioID       int64
	Ano             int64
	Numero          int64
	ClienteID       *int64
	ClienteNome     string
	ClienteNUIT     *string
	ClienteContacto *string
	ValidadeDias    int64
	Subtotal        float64
	DescontoTotal   float64
	Total           float64
	TaxaIVA         float64
	ValorIVA        float64
	Observacoes     *string
	Estado          EstadoCotacao
	VendaID         *int64
	CriadoEm        time.Time
	UtilizadorID    int64
}

// Codigo é o número como aparece no documento: "COT 2026/0007".
func (c Cotacao) Codigo() string {
	return fmt.Sprintf("COT %d/%04d", c.Ano, c.Numero)
}

// ValidaAte é o último dia em que a cotação vale (data local).
func (c Cotacao) ValidaAte() time.Time {
	l := c.CriadoEm.Local()
	dia := time.Date(l.Year(), l.Month(), l.Day(), 0, 0, 0, 0, time.Local)
	return dia.AddDate(0, 0, int(c.ValidadeDias))
}

// Expirada diz se uma cotação ainda aberta já passou da validade.
func (c Cotacao) Expirada(agora time.Time) bool {
	return c.Estado == EstadoCotacaoAberta && agora.After(c.ValidaAte().AddDate(0, 0, 1))
}

type ItemCotacao struct {
	ID               int64
	CotacaoID        int64
	ProdutoID        int64
	UnidadeProdutoID int64
	Descricao        string
	NomeUnidade      string
	Quantidade       float64
	PrecoUnitario    float64
	Desconto         float64
	Subtotal         float64
}
