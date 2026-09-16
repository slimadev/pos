package domain

import "time"

type TipoVenda string

const (
	TipoVendaDireta       TipoVenda = "direta"
	TipoVendaContaFechada TipoVenda = "conta_fechada"
)

type StatusVenda string

const (
	StatusVendaPaga      StatusVenda = "paga"
	StatusVendaDivida    StatusVenda = "divida"
	StatusVendaParcial   StatusVenda = "parcial"
	StatusVendaCancelada StatusVenda = "cancelada"
)

type MetodoPagamento string

const (
	MetodoDinheiro      MetodoPagamento = "dinheiro"
	MetodoMpesa         MetodoPagamento = "mpesa"
	MetodoEmola         MetodoPagamento = "emola"
	MetodoCartao        MetodoPagamento = "cartao"
	MetodoTransferencia MetodoPagamento = "transferencia"
)

type Venda struct {
	ID            int64
	NegocioID     int64
	ArmazemID     int64
	ClienteID     int64
	ContaID       *int64
	TipoVenda     TipoVenda
	Subtotal      float64
	DescontoTotal float64
	Total         float64
	Status        StatusVenda
	CriadoEm      time.Time
	UtilizadorID  int64
}

// ItemVenda guarda CustoUnitarioNoMomento (por unidade vendida, não por
// unidade base) para o lucro real por venda ficar correto mesmo que o
// custo médio do produto mude depois (SDD 3.2).
type ItemVenda struct {
	ID                     int64
	VendaID                int64
	ProdutoID              int64
	UnidadeProdutoID       int64
	Quantidade             float64
	PrecoUnitario          float64
	CustoUnitarioNoMomento float64
	Desconto               float64
	Subtotal               float64
}

type Pagamento struct {
	ID       int64
	VendaID  int64
	Metodo   MetodoPagamento
	Valor    float64
	CriadoEm time.Time
}
