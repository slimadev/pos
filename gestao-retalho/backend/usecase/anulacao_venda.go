package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// AnularVenda desfaz uma venda registada por engano: repõe todo o stock que
// ela tirou (produtos, ingredientes e lotes, cada um onde estava), apaga a
// dívida que tenha deixado e marca-a "cancelada" — deixa de contar em
// relatórios e no fecho de caixa.
//
// Só no próprio dia: o dinheiro devolve-se ao cliente nesse dia e o fecho
// de caixa desse dia fica certo; fechos de dias anteriores nunca mudam.
// Devolver artigos noutro dia é outra operação (com reembolso no fecho).
func (s *VendaService) AnularVenda(ctx context.Context, vendaID int64, motivo string, utilizadorID int64) error {
	motivo = strings.TrimSpace(motivo)
	if motivo == "" {
		return domain.ErrMotivoObrigatorio
	}
	venda, err := s.vendas.ObterVenda(ctx, vendaID)
	if err != nil {
		return fmt.Errorf("obter venda: %w", err)
	}
	if venda.Status == domain.StatusVendaCancelada {
		return fmt.Errorf("a venda #%d já está anulada", vendaID)
	}
	agora := time.Now()
	if venda.CriadoEm.Local().Format("2006-01-02") != agora.Format("2006-01-02") {
		return fmt.Errorf("só se anula uma venda no próprio dia — a #%d é de %s", vendaID, venda.CriadoEm.Local().Format("02/01/2006"))
	}

	err = s.vendas.WithTx(ctx, func(tx *repository.VendaRepository) error {
		divida, temDivida, err := tx.ObterDividaPorVenda(ctx, vendaID)
		if err != nil {
			return err
		}
		if temDivida {
			n, err := tx.ContarPagamentosDivida(ctx, divida.ID)
			if err != nil {
				return err
			}
			if n > 0 {
				return fmt.Errorf("a dívida desta venda já recebeu pagamentos — não pode ser anulada")
			}
			if err := tx.ApagarDivida(ctx, divida.ID); err != nil {
				return fmt.Errorf("apagar dívida: %w", err)
			}
		}

		movimentos, err := tx.ListarMovimentosDaVenda(ctx, vendaID)
		if err != nil {
			return err
		}
		motivoMov := fmt.Sprintf("Anulação da venda #%d: %s", vendaID, motivo)
		referencia := "anulacao:" + strconv.FormatInt(vendaID, 10)
		for _, m := range movimentos {
			if m.ArmazemOrigemID == nil {
				continue
			}
			armazemID := *m.ArmazemOrigemID

			// O custo médio volta a incluir as unidades ao custo a que saíram.
			atual, err := tx.Stock.ObterStock(ctx, m.ProdutoID, armazemID)
			if err != nil {
				return err
			}
			custo := atual.CustoMedioPonderado
			if m.CustoUnitario != nil {
				custo = *m.CustoUnitario
			}
			novoCusto := custoMedioPonderado(atual.Quantidade, atual.CustoMedioPonderado, m.QuantidadeUnidadeBase, custo)

			loteID, temLote := loteDaReferencia(m.ReferenciaID)
			if !temLote {
				// Vendas anteriores a guardar o lote na referência: sem saber de
				// que lote saíram, repor só o stock deixava os lotes errados.
				produto, err := s.produto.Obter(ctx, m.ProdutoID)
				if err != nil {
					return err
				}
				if produto.LoteControlado {
					return fmt.Errorf("%q controla lotes e esta venda não registou de que lote saiu — não pode ser anulada; corrija com uma entrada c/ lote", produto.Nome)
				}
			}
			if temLote {
				lote, err := tx.Lotes.Obter(ctx, loteID)
				if err != nil {
					return fmt.Errorf("lote da venda: %w", err)
				}
				if err := tx.Lotes.AtualizarQuantidade(ctx, lote.ID, lote.Quantidade+m.QuantidadeUnidadeBase); err != nil {
					return fmt.Errorf("repor lote %s: %w", lote.NumeroLote, err)
				}
			}

			destino := armazemID
			custoCopy := custo
			if _, err := tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
				ProdutoID:        m.ProdutoID,
				ArmazemDestinoID: &destino,
				Tipo:             domain.TipoMovimentoAnulacaoVenda,
				Quantidade:       m.QuantidadeUnidadeBase,
				CustoUnitario:    &custoCopy,
				UnidadeProdutoID: m.UnidadeProdutoID,
				Motivo:           &motivoMov,
				ReferenciaID:     &referencia,
				UtilizadorID:     utilizadorID,
			}); err != nil {
				return fmt.Errorf("repor stock: %w", err)
			}
			if err := tx.Stock.DefinirCustoMedio(ctx, m.ProdutoID, armazemID, novoCusto); err != nil {
				return fmt.Errorf("custo médio: %w", err)
			}
		}

		return tx.AtualizarStatusVenda(ctx, vendaID, domain.StatusVendaCancelada)
	})
	if err != nil {
		return err
	}
	emitir(s.emissor, func(e EmissorDocumentos) error { return e.AposAnulacao(ctx, vendaID, motivo) })
	return nil
}

// loteDaReferencia lê o lote de "<vendaID>;lote:<id>".
func loteDaReferencia(ref *string) (int64, bool) {
	if ref == nil {
		return 0, false
	}
	i := strings.Index(*ref, "lote:")
	if i < 0 {
		return 0, false
	}
	id, err := strconv.ParseInt((*ref)[i+len("lote:"):], 10, 64)
	return id, err == nil
}
