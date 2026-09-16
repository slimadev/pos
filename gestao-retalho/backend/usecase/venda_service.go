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
// transação (SDD 3.5). O mecanismo de resolver linhas e gerar itens_venda
// + movimentos de stock (linhaResolvida / criarItensVendaEMovimentos, mais
// abaixo) é o mesmo que ContaService.FecharConta usa para a bottlestore —
// só a origem das linhas muda (parâmetros frescos vs. itens_conta já
// persistidos ao longo da noite).
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
				ProdutoID:             l.ProdutoID,
				UnidadeProdutoID:      l.UnidadeProdutoID,
				Quantidade:            l.Quantidade,
				PrecoUnitarioAplicado: l.PrecoUnitario,
				Desconto:              l.Desconto,
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

		itens, err := criarItensVendaEMovimentos(ctx, tx, venda.ID, p.ArmazemID, p.UtilizadorID, linhas)
		if err != nil {
			return err
		}
		resultado.Itens = itens

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

// linhaResolvida é uma linha de venda já validada e pronta a virar
// itens_venda + movimento de stock — venha ela de parâmetros frescos
// (venda direta) ou de itens_conta já persistidos (fecho de conta aberta).
type linhaResolvida struct {
	ProdutoID        int64
	ProdutoNome      string
	UnidadeProdutoID int64
	QtdEmUnidadeBase float64
	PrecoUnitario    float64
	Quantidade       float64
	Desconto         float64
	QtdBase          float64
}

func (s *VendaService) resolverLinhas(ctx context.Context, itens []ItemVendaDiretaParams) ([]linhaResolvida, float64, float64, error) {
	linhas := make([]linhaResolvida, 0, len(itens))
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

		subtotal += item.Quantidade * unidade.PrecoVenda
		descontoLinhas += item.Desconto

		linhas = append(linhas, linhaResolvida{
			ProdutoID:        produto.ID,
			ProdutoNome:      produto.Nome,
			UnidadeProdutoID: unidade.ID,
			QtdEmUnidadeBase: unidade.QtdEmUnidadeBase,
			PrecoUnitario:    unidade.PrecoVenda,
			Quantidade:       item.Quantidade,
			Desconto:         item.Desconto,
			QtdBase:          item.Quantidade * unidade.QtdEmUnidadeBase,
		})
	}

	return linhas, subtotal, descontoLinhas, nil
}

// criarItensVendaEMovimentos grava um itens_venda e um movimento de stock
// tipo `venda` por cada linha, dentro da transação corrente. Falha (e
// aborta tudo, incluindo linhas já processadas nesta chamada — o chamador
// está sempre dentro de um WithTx) se alguma linha não tiver stock
// suficiente no armazém indicado.
func criarItensVendaEMovimentos(ctx context.Context, tx *repository.VendaRepository, vendaID, armazemID, utilizadorID int64, linhas []linhaResolvida) ([]domain.ItemVenda, error) {
	itens := make([]domain.ItemVenda, 0, len(linhas))

	for _, l := range linhas {
		stockAtual, err := tx.Stock.ObterStock(ctx, l.ProdutoID, armazemID)
		if err != nil {
			return nil, err
		}
		if stockAtual.Quantidade < l.QtdBase {
			return nil, fmt.Errorf("%w: %s — disponível %.3f, pedido %.3f", domain.ErrStockInsuficiente, l.ProdutoNome, stockAtual.Quantidade, l.QtdBase)
		}

		subtotalLinha := l.Quantidade*l.PrecoUnitario - l.Desconto
		// Custo por unidade vendida (não por unidade base) — ver nota em
		// domain.ItemVenda.
		custoPorUnidadeVendida := stockAtual.CustoMedioPonderado * l.QtdEmUnidadeBase

		itemGravado, err := tx.CriarItemVenda(ctx, domain.ItemVenda{
			VendaID:                vendaID,
			ProdutoID:              l.ProdutoID,
			UnidadeProdutoID:       l.UnidadeProdutoID,
			Quantidade:             l.Quantidade,
			PrecoUnitario:          l.PrecoUnitario,
			CustoUnitarioNoMomento: custoPorUnidadeVendida,
			Desconto:               l.Desconto,
			Subtotal:               subtotalLinha,
		})
		if err != nil {
			return nil, fmt.Errorf("gravar item de venda: %w", err)
		}
		itens = append(itens, itemGravado)

		armazemIDCopy := armazemID
		unidadeIDCopy := l.UnidadeProdutoID
		custoCopy := stockAtual.CustoMedioPonderado
		referenciaID := strconv.FormatInt(vendaID, 10)
		if _, err := tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        l.ProdutoID,
			ArmazemOrigemID:  &armazemIDCopy,
			Tipo:             domain.TipoMovimentoVenda,
			Quantidade:       l.QtdBase,
			CustoUnitario:    &custoCopy,
			UnidadeProdutoID: &unidadeIDCopy,
			ReferenciaID:     &referenciaID,
			UtilizadorID:     utilizadorID,
		}); err != nil {
			return nil, fmt.Errorf("baixar stock: %w", err)
		}
	}

	return itens, nil
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
