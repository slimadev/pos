package domain

import "time"

type TipoProduto string

const (
	TipoProdutoSimples       TipoProduto = "simples"
	TipoProdutoComposto      TipoProduto = "composto"
	TipoProdutoSemiElaborado TipoProduto = "semi_elaborado"
)

// Produto é a entidade partilhada por todos os tipos de negócio (SDD 3.2).
// Só produtos Simples/SemiElaborado têm ControlaStock=true — Composto nunca
// guarda stock próprio, é sempre inferido via ficha técnica (Fase 5).
// LoteControlado ativa rastreio por lote e FEFO (Fase 6, SDD 9.4) — só faz
// sentido para simples/semi_elaborado, nunca para composto.
type Produto struct {
	ID             int64
	NegocioID      int64
	CategoriaID    *int64
	Nome           string
	TipoProduto    TipoProduto
	UnidadeBase    string
	StockMinimo    *float64
	ControlaStock  bool
	LoteControlado bool
	Ativo          bool
	CriadoEm       time.Time
	AtualizadoEm   time.Time
}

// UnidadeProduto cobre o fracionamento do mesmo produto (caixa/unidade,
// garrafa/dose simples/dose dupla — SDD 3.2).
type UnidadeProduto struct {
	ID               int64
	ProdutoID        int64
	NomeUnidade      string
	QtdEmUnidadeBase float64
	PrecoCusto       float64
	PrecoVenda       float64
	CodigoBarras     *string
	IsPadraoVenda    bool
	IsPadraoCompra   bool
}
