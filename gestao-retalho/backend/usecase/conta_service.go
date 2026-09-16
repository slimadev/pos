package usecase

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// ContaService cobre contas que ficam abertas ao longo da noite (SDD 4.10,
// bottlestore): abrir, lançar consumo incremental, listar as abertas,
// transferir itens entre contas, cancelar, e fechar — gerando a venda,
// possivelmente com dívida. O fecho reutiliza a mesma mecânica que
// VendaService usa para a venda direta (criarItensVendaEMovimentos em
// venda_service.go); só a origem das linhas muda.
type ContaService struct {
	vendas  *repository.VendaRepository
	produto *repository.ProdutoRepository
	armazem *repository.ArmazemRepository
}

func NewContaService(
	vendas *repository.VendaRepository,
	produto *repository.ProdutoRepository,
	armazem *repository.ArmazemRepository,
) *ContaService {
	return &ContaService{vendas: vendas, produto: produto, armazem: armazem}
}

func (s *ContaService) ListarAbertas(ctx context.Context, negocioID int64) ([]domain.Conta, error) {
	return s.vendas.ListarContasAbertas(ctx, negocioID)
}

func (s *ContaService) ObterConta(ctx context.Context, id int64) (domain.Conta, error) {
	return s.vendas.ObterConta(ctx, id)
}

func (s *ContaService) ListarItens(ctx context.Context, contaID int64) ([]domain.ItemConta, error) {
	return s.vendas.ListarItensConta(ctx, contaID)
}

func (s *ContaService) AbrirConta(ctx context.Context, identificador string, clienteID *int64, utilizadorID int64) (domain.Conta, error) {
	if identificador == "" {
		return domain.Conta{}, fmt.Errorf("identificador da conta é obrigatório")
	}
	cid := domain.ClienteAnonimoID
	if clienteID != nil {
		cid = *clienteID
	}
	return s.vendas.CriarConta(ctx, domain.NegocioID, cid, identificador, utilizadorID)
}

type AdicionarItemContaParams struct {
	ContaID          int64
	ProdutoID        int64
	UnidadeProdutoID int64
	Quantidade       float64
	Desconto         float64
	UtilizadorID     int64
}

// AdicionarItem só lança o item na comanda — SDD 4.10 (bottlestore): o
// stock só é verificado/baixado ao FECHAR a conta, não a cada item
// lançado. Uma conta aberta às 19h e fechada às 2h da manhã não pode
// bloquear o operador a meio da noite por causa de uma leitura de stock
// que pode já ter mudado quando finalmente for cobrada.
func (s *ContaService) AdicionarItem(ctx context.Context, p AdicionarItemContaParams) (domain.ItemConta, error) {
	if p.Quantidade <= 0 {
		return domain.ItemConta{}, fmt.Errorf("quantidade deve ser positiva")
	}
	if p.Desconto < 0 {
		return domain.ItemConta{}, fmt.Errorf("desconto não pode ser negativo")
	}

	conta, err := s.vendas.ObterConta(ctx, p.ContaID)
	if err != nil {
		return domain.ItemConta{}, err
	}
	if conta.Status != domain.StatusContaAberta {
		return domain.ItemConta{}, domain.ErrContaNaoAberta
	}

	produto, err := s.produto.Obter(ctx, p.ProdutoID)
	if err != nil {
		return domain.ItemConta{}, err
	}
	if !produto.Ativo {
		return domain.ItemConta{}, domain.ErrProdutoInativo
	}
	if produto.TipoProduto == domain.TipoProdutoComposto {
		return domain.ItemConta{}, domain.ErrProdutoComposto
	}

	unidade, err := s.produto.ObterUnidade(ctx, p.UnidadeProdutoID)
	if err != nil {
		return domain.ItemConta{}, err
	}
	if unidade.ProdutoID != produto.ID {
		return domain.ItemConta{}, fmt.Errorf("unidade %d não pertence ao produto %q", unidade.ID, produto.Nome)
	}

	return s.vendas.CriarItemConta(ctx, domain.ItemConta{
		ContaID:               p.ContaID,
		ProdutoID:             produto.ID,
		UnidadeProdutoID:      unidade.ID,
		Quantidade:            p.Quantidade,
		PrecoUnitarioAplicado: unidade.PrecoVenda,
		Desconto:              p.Desconto,
		UtilizadorID:          p.UtilizadorID,
	})
}

// TransferirItem move uma linha inteira de uma conta aberta para outra
// (SDD 4.10, "útil se o grupo se dividir") — move a linha completa, não
// fraciona quantidade entre contas.
func (s *ContaService) TransferirItem(ctx context.Context, itemID, novaContaID int64) error {
	item, err := s.vendas.ObterItemConta(ctx, itemID)
	if err != nil {
		return err
	}
	if item.ContaID == novaContaID {
		return fmt.Errorf("conta de origem e destino são a mesma")
	}

	origem, err := s.vendas.ObterConta(ctx, item.ContaID)
	if err != nil {
		return err
	}
	if origem.Status != domain.StatusContaAberta {
		return fmt.Errorf("conta de origem: %w", domain.ErrContaNaoAberta)
	}

	destino, err := s.vendas.ObterConta(ctx, novaContaID)
	if err != nil {
		return err
	}
	if destino.Status != domain.StatusContaAberta {
		return fmt.Errorf("conta de destino: %w", domain.ErrContaNaoAberta)
	}

	return s.vendas.TransferirItemConta(ctx, itemID, novaContaID)
}

func (s *ContaService) CancelarConta(ctx context.Context, contaID, utilizadorID int64, motivo string) (domain.Conta, error) {
	if motivo == "" {
		return domain.Conta{}, fmt.Errorf("motivo do cancelamento é obrigatório")
	}
	conta, err := s.vendas.ObterConta(ctx, contaID)
	if err != nil {
		return domain.Conta{}, err
	}
	if conta.Status != domain.StatusContaAberta {
		return domain.Conta{}, domain.ErrContaNaoAberta
	}
	return s.vendas.CancelarConta(ctx, contaID, utilizadorID, motivo)
}

type FecharContaParams struct {
	ArmazemID     int64
	DescontoTotal float64
	Pagamentos    []PagamentoVendaParams
	UtilizadorID  int64
}

// FecharConta gera a venda a partir dos itens já lançados (SDD 4.10, passo
// 6). Ao contrário da venda direta (Fase 3), o pagamento não precisa de
// cobrir o total — a diferença vira dívida, mas só se o cliente da conta
// estiver identificado (não for o anónimo).
func (s *ContaService) FecharConta(ctx context.Context, contaID int64, p FecharContaParams) (VendaCompleta, error) {
	conta, err := s.vendas.ObterConta(ctx, contaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	if conta.Status != domain.StatusContaAberta {
		return VendaCompleta{}, domain.ErrContaNaoAberta
	}

	armazem, err := s.armazem.Obter(ctx, p.ArmazemID)
	if err != nil {
		return VendaCompleta{}, err
	}
	if !armazem.Ativo {
		return VendaCompleta{}, domain.ErrArmazemInativo
	}

	itensConta, err := s.vendas.ListarItensConta(ctx, contaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	if len(itensConta) == 0 {
		return VendaCompleta{}, fmt.Errorf("conta sem itens lançados")
	}

	linhas := make([]linhaResolvida, 0, len(itensConta))
	var subtotal, descontoLinhas float64
	for _, item := range itensConta {
		produto, err := s.produto.Obter(ctx, item.ProdutoID)
		if err != nil {
			return VendaCompleta{}, err
		}
		unidade, err := s.produto.ObterUnidade(ctx, item.UnidadeProdutoID)
		if err != nil {
			return VendaCompleta{}, err
		}

		subtotal += item.Quantidade * item.PrecoUnitarioAplicado
		descontoLinhas += item.Desconto

		linhas = append(linhas, linhaResolvida{
			ProdutoID:        produto.ID,
			ProdutoNome:      produto.Nome,
			UnidadeProdutoID: unidade.ID,
			QtdEmUnidadeBase: unidade.QtdEmUnidadeBase,
			// Preço congelado no momento em que o item foi lançado na
			// conta (itens_conta.preco_unitario_aplicado), não o preço
			// de catálogo atual — o cliente vê o preço de quando pediu.
			PrecoUnitario: item.PrecoUnitarioAplicado,
			Quantidade:    item.Quantidade,
			Desconto:      item.Desconto,
			QtdBase:       item.Quantidade * unidade.QtdEmUnidadeBase,
		})
	}

	descontoTotal := descontoLinhas + p.DescontoTotal
	total := subtotal - descontoTotal
	if total < 0 {
		return VendaCompleta{}, fmt.Errorf("desconto maior que o total da conta")
	}

	var somaPagamentos float64
	for _, pag := range p.Pagamentos {
		if pag.Valor <= 0 {
			return VendaCompleta{}, fmt.Errorf("valor de pagamento deve ser positivo")
		}
		somaPagamentos += pag.Valor
	}
	if somaPagamentos > total+toleranciaPagamento {
		return VendaCompleta{}, fmt.Errorf("pagamentos (%.2f) excedem o total da conta (%.2f)", somaPagamentos, total)
	}

	faltaPagar := total - somaPagamentos
	temDivida := faltaPagar > toleranciaPagamento
	if temDivida && conta.ClienteID == domain.ClienteAnonimoID {
		return VendaCompleta{}, domain.ErrClienteAnonimoNaoPodeDivida
	}

	status := domain.StatusVendaPaga
	if temDivida {
		status = domain.StatusVendaDivida
		if somaPagamentos > toleranciaPagamento {
			status = domain.StatusVendaParcial
		}
	}

	var resultado VendaCompleta
	err = s.vendas.WithTx(ctx, func(tx *repository.VendaRepository) error {
		if _, err := tx.FecharConta(ctx, contaID, p.UtilizadorID); err != nil {
			return fmt.Errorf("fechar conta: %w", err)
		}

		cID := contaID
		venda, err := tx.CriarVenda(ctx, domain.Venda{
			NegocioID:     domain.NegocioID,
			ArmazemID:     p.ArmazemID,
			ClienteID:     conta.ClienteID,
			ContaID:       &cID,
			TipoVenda:     domain.TipoVendaContaFechada,
			Subtotal:      subtotal,
			DescontoTotal: descontoTotal,
			Total:         total,
			Status:        status,
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

		if temDivida {
			dividaStatus := domain.StatusDividaAberta
			if somaPagamentos > toleranciaPagamento {
				dividaStatus = domain.StatusDividaParcial
			}
			if _, err := tx.CriarDivida(ctx, domain.Divida{
				ClienteID:     conta.ClienteID,
				VendaID:       venda.ID,
				ValorOriginal: total,
				ValorPago:     somaPagamentos,
				ValorEmAberto: faltaPagar,
				Status:        dividaStatus,
			}); err != nil {
				return fmt.Errorf("registar dívida: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return VendaCompleta{}, err
	}
	return resultado, nil
}
