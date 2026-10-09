package usecase

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/repository"
)

// ReciboService junta os dados de várias fontes (venda, produtos,
// cliente, negócio) no formato que o pacote export precisa — a geração do
// PDF em si fica inteiramente no export (SDD 2.2/8.1: ExportService separa
// "o que mostrar" de "como desenhar o documento").
type ReciboService struct {
	vendas  *VendaService
	produto *repository.ProdutoRepository
	cliente *repository.ClienteRepository
	negocio *repository.NegocioRepository
}

func NewReciboService(
	vendas *VendaService,
	produto *repository.ProdutoRepository,
	cliente *repository.ClienteRepository,
	negocio *repository.NegocioRepository,
) *ReciboService {
	return &ReciboService{vendas: vendas, produto: produto, cliente: cliente, negocio: negocio}
}

func (s *ReciboService) GerarPDF(ctx context.Context, vendaID int64) ([]byte, error) {
	completa, err := s.vendas.ObterVendaCompleta(ctx, vendaID)
	if err != nil {
		return nil, fmt.Errorf("obter venda: %w", err)
	}
	negocio, err := s.negocio.Obter(ctx)
	if err != nil {
		return nil, fmt.Errorf("obter negócio: %w", err)
	}
	cliente, err := s.cliente.Obter(ctx, completa.Venda.ClienteID)
	if err != nil {
		return nil, fmt.Errorf("obter cliente: %w", err)
	}

	itens := make([]export.ItemRecibo, 0, len(completa.Itens))
	for _, item := range completa.Itens {
		produto, err := s.produto.Obter(ctx, item.ProdutoID)
		if err != nil {
			return nil, fmt.Errorf("obter produto do item: %w", err)
		}
		unidade, err := s.produto.ObterUnidade(ctx, item.UnidadeProdutoID)
		if err != nil {
			return nil, fmt.Errorf("obter unidade do item: %w", err)
		}
		itens = append(itens, export.ItemRecibo{
			NomeProduto:   produto.Nome,
			NomeUnidade:   unidade.NomeUnidade,
			Quantidade:    item.Quantidade,
			PrecoUnitario: item.PrecoUnitario,
			Desconto:      item.Desconto,
			Subtotal:      item.Subtotal,
		})
	}

	pagamentos := make([]export.PagamentoRecibo, 0, len(completa.Pagamentos))
	for _, p := range completa.Pagamentos {
		pagamentos = append(pagamentos, export.PagamentoRecibo{Metodo: rotuloMetodo(p.Metodo), Valor: p.Valor})
	}

	return export.GerarReciboPDF(export.DadosRecibo{
		NegocioNome:   negocio.Nome,
		VendaID:       completa.Venda.ID,
		CriadoEm:      completa.Venda.CriadoEm,
		ClienteNome:   cliente.Nome,
		Itens:         itens,
		Subtotal:      completa.Venda.Subtotal,
		DescontoTotal: completa.Venda.DescontoTotal,
		Total:         completa.Venda.Total,
		Pagamentos:    pagamentos,
	})
}

func rotuloMetodo(m domain.MetodoPagamento) string {
	switch m {
	case domain.MetodoDinheiro:
		return "Dinheiro"
	case domain.MetodoMpesa:
		return "M-Pesa"
	case domain.MetodoEmola:
		return "e-Mola"
	case domain.MetodoCartao:
		return "Cartão"
	case domain.MetodoTransferencia:
		return "Transferência"
	}
	return string(m)
}
