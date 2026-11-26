package domain

import "time"

// MesaAberta é uma conta de restaurante: a mesa, o garçom que a serve e o
// número de pessoas (para dividir a conta na impressão).
type MesaAberta struct {
	ContaID    int64
	Mesa       string
	GarcomID   int64
	GarcomNome string
	Pessoas    int64
	AbertaEm   time.Time
	Total      float64
	PorEnviar  float64 // quantidade de itens ainda não enviada para preparação
}

// ItemMesa é uma linha da conta com quanto dela já foi enviado (impresso
// no balcão/cozinha).
type ItemMesa struct {
	ID               int64
	ContaID          int64
	ProdutoID        int64
	UnidadeProdutoID int64
	Produto          string
	Unidade          string
	CategoriaID      *int64
	Quantidade       float64
	Enviada          float64
	PrecoUnitario    float64
	Desconto         float64
	Subtotal         float64
}

type FuncaoImpressora string

const (
	// ImpressoraBalcao recebe todos os pedidos e imprime as contas.
	ImpressoraBalcao FuncaoImpressora = "balcao"
	// ImpressoraPreparacao (cozinha, bar…) só recebe os itens das
	// categorias que lhe estão atribuídas.
	ImpressoraPreparacao FuncaoImpressora = "preparacao"
)

// Impressora é uma impressora térmica de rede (ESC/POS, porta 9100).
type Impressora struct {
	ID       int64
	Nome     string
	Endereco string
	Porta    int64
	Colunas  int64 // 48 em 80 mm, 32 em 58 mm
	Funcao   FuncaoImpressora
	Ativa    bool
}

type TipoPedido string

const (
	TipoPedidoEnvio    TipoPedido = "pedido"
	TipoPedidoAnulacao TipoPedido = "anulacao"
)

// Pedido é um envio de itens para preparação (ou a anulação de itens já
// enviados), tal como foi impresso — dá para reimprimir.
type Pedido struct {
	ID            int64
	ContaID       int64
	Tipo          TipoPedido
	Mesa          string
	GarcomID      int64
	Motivo        *string
	AutorizadoPor *int64
	CriadoEm      time.Time
	Linhas        []LinhaPedido
}

type LinhaPedido struct {
	ProdutoID    int64
	Descricao    string
	Unidade      string
	Quantidade   float64
	ImpressoraID *int64 // impressora de preparação; nil = só balcão
}

// SessaoCaixa é um turno de caixa, da abertura (com o fundo inicial) ao
// fecho (com a contagem por método de pagamento).
type SessaoCaixa struct {
	ID                   int64
	AbertaEm             time.Time
	UtilizadorAberturaID int64
	FundoInicial         float64
	FechadaEm            *time.Time
	UtilizadorFechoID    *int64
	Contagem             map[MetodoPagamento]float64
	Observacoes          *string
}
