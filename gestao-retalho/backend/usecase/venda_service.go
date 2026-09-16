package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// VendaService implementa o fluxo de venda direta (SDD 4.10, ferragem/
// mercearia/farmácia): internamente abre e fecha uma `conta` na mesma
// transação (SDD 3.5) para reaproveitar, sem duplicar, a mecânica que a
// Fase 4 (bottlestore) vai usar para contas que ficam abertas por horas.
type VendaService struct {
	vendas  *repository.VendaRepository
	produto *repository.ProdutoRepository
	armazem *repository.ArmazemRepository
}

func NewVendaService(
	vendas *repository.VendaRepository,
	produto *repository.ProdutoRepository,
	armazem *repository.ArmazemRepository,
) *VendaService {
	return &VendaService{vendas: vendas, produto: produto, armazem: armazem}
}

type ItemVendaDiretaParams struct {
	ProdutoID        int64
	UnidadeProdutoID int64
	Quantidade       float64
	Desconto         float64
}

type PagamentoVendaParams struct {
	Metodo domain.MetodoPagamento
	Valor  float64
}

type VendaDiretaParams struct {
	ArmazemID     int64
	ClienteID     *int64 // nil = anónimo (SDD 3.2)
	Itens         []ItemVendaDiretaParams
	DescontoTotal float64
	Pagamentos    []PagamentoVendaParams
	UtilizadorID  int64
}

type VendaCompleta struct {
	Venda      domain.Venda
	Itens      []domain.ItemVenda
	Pagamentos []domain.Pagamento
}

// toleranciaPagamento absorve erros de arredondamento de ponto flutuante
// ao comparar soma de pagamentos com o total — não é uma margem de
// crédito, é só folga para 0.1 + 0.2 != 0.3 não bloquear uma venda certa.
const toleranciaPagamento = 0.01

func (s *VendaService) RegistarVendaDireta(ctx context.Context, p VendaDiretaParams) (VendaCompleta, error) {
	if len(p.Itens) == 0 {
		return VendaCompleta{}, fmt.Errorf("venda sem itens")
	}
	if len(p.Pagamentos) == 0 {
		return VendaCompleta{}, fmt.Errorf("venda sem forma de pagamento")
	}

	armazem, err := s.armazem.Obter(ctx, p.ArmazemID)
	if err != nil {
		return VendaCompleta{}, err
	}
	if !armazem.Ativo {
		return VendaCompleta{}, domain.ErrArmazemInativo
	}

	clienteID := domain.ClienteAnonimoID
	if p.ClienteID != nil {
		clienteID = *p.ClienteID
	}

	linhas, subtotal, descontoLinhas, err := s.resolverLinhas(ctx, p.Itens)
	if err != nil {
		return VendaCompleta{}, err
	}

	descontoTotal := descontoLinhas + p.DescontoTotal
	total := subtotal - descontoTotal
	if total < 0 {
		return VendaCompleta{}, fmt.Errorf("desconto maior que o total da venda")
	}

	var somaPagamentos float64
	for _, pag := range p.Pagamentos {
		if pag.Valor <= 0 {
			return VendaCompleta{}, fmt.Errorf("valor de pagamento deve ser positivo")
		}
		somaPagamentos += pag.Valor
	}
	if diff := somaPagamentos - total; diff < -toleranciaPagamento || diff > toleranciaPagamento {
		return VendaCompleta{}, fmt.Errorf("pagamentos (%.2f) não cobrem o total da venda (%.2f)", somaPagamentos, total)
	}

	var resultado VendaCompleta

	err = s.vendas.WithTx(ctx, func(tx *repository.VendaRepository) error {
		conta, err := tx.CriarConta(ctx, domain.NegocioID, clienteID, "Venda direta", p.UtilizadorID)
		if err != nil {
			return fmt.Errorf("abrir conta: %w", err)
		}

		for _, l := range linhas {
			if _, err := tx.CriarItemConta(ctx, domain.ItemConta{
				ContaID:               conta.ID,
				ProdutoID:             l.produto.ID,
				UnidadeProdutoID:      l.unidade.ID,
				Quantidade:            l.params.Quantidade,
				PrecoUnitarioAplicado: l.unidade.PrecoVenda,
				Desconto:              l.params.Desconto,
				UtilizadorID:          p.UtilizadorID,
			}); err != nil {
				return fmt.Errorf("lançar item na conta: %w", err)
			}
		}

		if _, err := tx.FecharConta(ctx, conta.ID, p.UtilizadorID); err != nil {
			return fmt.Errorf("fechar conta: %w", err)
		}

		contaID := conta.ID
		venda, err := tx.CriarVenda(ctx, domain.Venda{
			NegocioID:     domain.NegocioID,
			ArmazemID:     p.ArmazemID,
			ClienteID:     clienteID,
			ContaID:       &contaID,
			TipoVenda:     domain.TipoVendaDireta,
			Subtotal:      subtotal,
			DescontoTotal: descontoTotal,
			Total:         total,
			Status:        domain.StatusVendaPaga,
			UtilizadorID:  p.UtilizadorID,
		})
		if err != nil {
			return fmt.Errorf("criar venda: %w", err)
		}
		resultado.Venda = venda

		for _, l := range linhas {
			stockAtual, err := tx.Stock.ObterStock(ctx, l.produto.ID, p.ArmazemID)
			if err != nil {
				return err
			}
			if stockAtual.Quantidade < l.qtdBase {
				return fmt.Errorf("%w: %s — disponível %.3f, pedido %.3f", domain.ErrStockInsuficiente, l.produto.Nome, stockAtual.Quantidade, l.qtdBase)
			}

			subtotalLinha := l.params.Quantidade*l.unidade.PrecoVenda - l.params.Desconto
			// Custo por unidade vendida (não por unidade base) — ver nota
			// em domain.ItemVenda.
			custoPorUnidadeVendida := stockAtual.CustoMedioPonderado * l.unidade.QtdEmUnidadeBase

			itemGravado, err := tx.CriarItemVenda(ctx, domain.ItemVenda{
				VendaID:                venda.ID,
				ProdutoID:              l.produto.ID,
				UnidadeProdutoID:       l.unidade.ID,
				Quantidade:             l.params.Quantidade,
				PrecoUnitario:          l.unidade.PrecoVenda,
				CustoUnitarioNoMomento: custoPorUnidadeVendida,
				Desconto:               l.params.Desconto,
				Subtotal:               subtotalLinha,
			})
			if err != nil {
				return fmt.Errorf("gravar item de venda: %w", err)
			}
			resultado.Itens = append(resultado.Itens, itemGravado)

			armazemID := p.ArmazemID
			unidadeID := l.unidade.ID
			custo := stockAtual.CustoMedioPonderado
			referenciaID := strconv.FormatInt(venda.ID, 10)
			if _, err := tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
				ProdutoID:        l.produto.ID,
				ArmazemOrigemID:  &armazemID,
				Tipo:             domain.TipoMovimentoVenda,
				Quantidade:       l.qtdBase,
				CustoUnitario:    &custo,
				UnidadeProdutoID: &unidadeID,
				ReferenciaID:     &referenciaID,
				UtilizadorID:     p.UtilizadorID,
			}); err != nil {
				return fmt.Errorf("baixar stock: %w", err)
			}
		}

		for _, pag := range p.Pagamentos {
			pagamentoGravado, err := tx.CriarPagamento(ctx, domain.Pagamento{
				VendaID: venda.ID,
				Metodo:  pag.Metodo,
				Valor:   pag.Valor,
			})
			if err != nil {
				return fmt.Errorf("gravar pagamento: %w", err)
			}
			resultado.Pagamentos = append(resultado.Pagamentos, pagamentoGravado)
		}

		return nil
	})
	if err != nil {
		return VendaCompleta{}, err
	}
	return resultado, nil
}

type linhaVenda struct {
	produto domain.Produto
	unidade domain.UnidadeProduto
	params  ItemVendaDiretaParams
	qtdBase float64
}

func (s *VendaService) resolverLinhas(ctx context.Context, itens []ItemVendaDiretaParams) ([]linhaVenda, float64, float64, error) {
	linhas := make([]linhaVenda, 0, len(itens))
	var subtotal, descontoLinhas float64

	for _, item := range itens {
		if item.Quantidade <= 0 {
			return nil, 0, 0, fmt.Errorf("quantidade deve ser positiva")
		}
		if item.Desconto < 0 {
			return nil, 0, 0, fmt.Errorf("desconto não pode ser negativo")
		}

		produto, err := s.produto.Obter(ctx, item.ProdutoID)
		if err != nil {
			return nil, 0, 0, err
		}
		if !produto.Ativo {
			return nil, 0, 0, domain.ErrProdutoInativo
		}
		// Fichas técnicas (expansão de receita ao vender) só chegam na
		// Fase 5 — até lá, produto composto não é vendável, tal como já
		// não é possível mexer o seu stock diretamente (SDD 3.2).
		if produto.TipoProduto == domain.TipoProdutoComposto {
			return nil, 0, 0, domain.ErrProdutoComposto
		}

		unidade, err := s.produto.ObterUnidade(ctx, item.UnidadeProdutoID)
		if err != nil {
			return nil, 0, 0, err
		}
		if unidade.ProdutoID != produto.ID {
			return nil, 0, 0, fmt.Errorf("unidade %d não pertence ao produto %q", unidade.ID, produto.Nome)
		}

		subtotalLinha := item.Quantidade * unidade.PrecoVenda
		subtotal += subtotalLinha
		descontoLinhas += item.Desconto

		linhas = append(linhas, linhaVenda{
			produto: produto,
			unidade: unidade,
			params:  item,
			qtdBase: item.Quantidade * unidade.QtdEmUnidadeBase,
		})
	}

	return linhas, subtotal, descontoLinhas, nil
}

// ObterVendaCompleta é usado pelo recibo e pelo histórico do dia (SDD 5.3).
func (s *VendaService) ObterVendaCompleta(ctx context.Context, vendaID int64) (VendaCompleta, error) {
	venda, err := s.vendas.ObterVenda(ctx, vendaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	itens, err := s.vendas.ListarItensVenda(ctx, vendaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	pagamentos, err := s.vendas.ListarPagamentos(ctx, vendaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	return VendaCompleta{Venda: venda, Itens: itens, Pagamentos: pagamentos}, nil
}

// ListarVendasDoDia cobre o "Histórico de vendas do dia" da tela de
// ferragem (SDD 5.3) — usa o relógio local do computador da loja, que é
// o que faz sentido para um "dia de operação" de um balcão físico.
func (s *VendaService) ListarVendasDoDia(ctx context.Context, negocioID int64) ([]domain.Venda, error) {
	agora := time.Now()
	inicio := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, agora.Location())
	fim := inicio.Add(24 * time.Hour)
	return s.vendas.ListarPorPeriodo(ctx, negocioID, inicio, fim)
}
