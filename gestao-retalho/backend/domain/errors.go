package domain

import "errors"

// Erros de regra de negócio partilhados pelos usecases (SDD secção 4).
// São sentinelas comparáveis com errors.Is, não erros de infraestrutura.
var (
	ErrStockInsuficiente  = errors.New("stock insuficiente")
	ErrMotivoObrigatorio  = errors.New("motivo é obrigatório")
	ErrProdutoComposto    = errors.New("produto composto não tem stock próprio")
	ErrProdutoInativo     = errors.New("produto inativo")
	ErrArmazemInativo     = errors.New("armazém inativo")
	ErrArmazemIgual       = errors.New("armazém de origem e destino não podem ser o mesmo")
	ErrInventarioFechado  = errors.New("inventário já está fechado")
	ErrItemNaoPertenceInv = errors.New("item não pertence a este inventário")
	ErrInventarioJaAberto = errors.New("este armazém já tem um inventário em curso")

	ErrLoteNaoEncontrado = errors.New("lote não encontrado")
	ErrLoteObrigatorio   = errors.New("produto controla lotes — escolha o lote")

	// ErrClienteAnonimoNaoPodeDivida: SDD 4.10 — "Registar Dívida (exige
	// cliente identificado, não anónimo)".
	ErrClienteAnonimoNaoPodeDivida = errors.New("cliente anónimo não pode ficar com saldo em dívida")
	ErrContaNaoAberta              = errors.New("conta não está aberta")
)
