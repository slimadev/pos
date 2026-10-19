package usecase

import (
	"context"
	"fmt"
	"strings"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// ItemRececao é uma linha da guia/fatura do fornecedor: a quantidade e o
// custo vêm na unidade de compra (ex: 3 caixas de 100 a 250 MT a caixa).
type ItemRececao struct {
	ProdutoID        int64
	UnidadeProdutoID int64
	Quantidade       float64
	CustoUnitario    float64
}

type RececaoParams struct {
	ArmazemID int64
	// Referencia identifica o documento do fornecedor (ex: "Fatura 123 —
	// Distribuidora Matola"); fica no motivo de cada movimento.
	Referencia   string
	Itens        []ItemRececao
	UtilizadorID int64
}

// RegistarRececao dá entrada de toda a mercadoria de um documento do
// fornecedor numa só transação — ou entram todas as linhas, ou nenhuma.
// Cada linha é convertida para a unidade base (quantidade × conteúdo da
// unidade, custo ÷ conteúdo) e atualiza o custo médio ponderado como uma
// entrada normal. Produtos com lotes ficam de fora: precisam de número de
// lote e validade, e entram pela entrada em lote.
func (s *StockService) RegistarRececao(ctx context.Context, p RececaoParams) ([]domain.MovimentoStock, error) {
	if len(p.Itens) == 0 {
		return nil, fmt.Errorf("a receção precisa de pelo menos um produto")
	}
	if _, err := s.validarArmazemAtivo(ctx, p.ArmazemID); err != nil {
		return nil, err
	}

	type linha struct {
		item    ItemRececao
		produto domain.Produto
		unidade domain.UnidadeProduto
	}
	linhas := make([]linha, 0, len(p.Itens))
	for _, item := range p.Itens {
		produto, err := s.validarProdutoParaStock(ctx, item.ProdutoID)
		if err != nil {
			return nil, err
		}
		if produto.LoteControlado {
			return nil, fmt.Errorf("%q controla lotes — registe-o na entrada em lote, com número de lote e validade", produto.Nome)
		}
		if item.Quantidade <= 0 {
			return nil, fmt.Errorf("%q: a quantidade tem de ser maior que zero", produto.Nome)
		}
		if item.CustoUnitario < 0 {
			return nil, fmt.Errorf("%q: o custo não pode ser negativo", produto.Nome)
		}
		unidade, err := s.produto.ObterUnidade(ctx, item.UnidadeProdutoID)
		if err != nil || unidade.ProdutoID != produto.ID {
			return nil, fmt.Errorf("%q: unidade de compra inválida", produto.Nome)
		}
		linhas = append(linhas, linha{item: item, produto: produto, unidade: unidade})
	}

	motivo := "Receção de mercadoria"
	if ref := strings.TrimSpace(p.Referencia); ref != "" {
		motivo += ": " + ref
	}

	movimentos := make([]domain.MovimentoStock, 0, len(linhas))
	err := s.stock.WithTx(ctx, func(tx *repository.StockRepository) error {
		for _, l := range linhas {
			qtdBase := l.item.Quantidade * l.unidade.QtdEmUnidadeBase
			custoBase := l.item.CustoUnitario / l.unidade.QtdEmUnidadeBase

			// Lido dentro da transação: o mesmo produto pode vir em duas
			// linhas, e a segunda tem de ver a primeira.
			atual, err := tx.ObterStock(ctx, l.produto.ID, p.ArmazemID)
			if err != nil {
				return err
			}
			novoCusto := custoMedioPonderado(atual.Quantidade, atual.CustoMedioPonderado, qtdBase, custoBase)

			destino := p.ArmazemID
			unidadeID := l.unidade.ID
			m, err := tx.CriarMovimento(ctx, repository.NovoMovimento{
				ProdutoID:        l.produto.ID,
				ArmazemDestinoID: &destino,
				Tipo:             domain.TipoMovimentoEntrada,
				Quantidade:       qtdBase,
				CustoUnitario:    &custoBase,
				UnidadeProdutoID: &unidadeID,
				Motivo:           &motivo,
				UtilizadorID:     p.UtilizadorID,
			})
			if err != nil {
				return err
			}
			if err := tx.DefinirCustoMedio(ctx, l.produto.ID, p.ArmazemID, novoCusto); err != nil {
				return err
			}
			movimentos = append(movimentos, m)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return movimentos, nil
}
