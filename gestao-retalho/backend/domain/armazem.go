package domain

type TipoArmazem string

const (
	TipoArmazemPrincipal  TipoArmazem = "principal"
	TipoArmazemPontoVenda TipoArmazem = "ponto_venda"
	TipoArmazemDeposito   TipoArmazem = "deposito"
)

// Armazem representa qualquer depósito/balcão da loja — "principal" ou
// "ponto de venda" é só um atributo Tipo, não uma estrutura diferente
// (SDD secção 2.1, multi-armazém nativo).
type Armazem struct {
	ID        int64
	NegocioID int64
	Nome      string
	Tipo      TipoArmazem
	Ativo     bool
	// ForaDeVenda tira o armazém do PDV e recusa vendas a partir dele (ex:
	// um depósito). Ao contrário de "disponível", para que o valor por
	// omissão (false) mantenha o armazém disponível para venda.
	ForaDeVenda bool
}
