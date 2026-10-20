package usecase

import (
	"context"
	"fmt"
	"time"

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
	configs *repository.ConfigRepository
}

// Valores da configuração "recibo_papel": a largura do papel da impressora
// de recibos. Sem valor, o rolo de 80 mm (o mais comum nos terminais).
const (
	PapelRecibo80 = "80"
	PapelRecibo58 = "58"
	PapelReciboA5 = "a5"
)

func NewReciboService(
	vendas *VendaService,
	produto *repository.ProdutoRepository,
	cliente *repository.ClienteRepository,
	negocio *repository.NegocioRepository,
	configs *repository.ConfigRepository,
) *ReciboService {
	return &ReciboService{vendas: vendas, produto: produto, cliente: cliente, negocio: negocio, configs: configs}
}

// ReciboEcra é o recibo como o PDV o mostra no modal (e imprime pela
// janela): os mesmos dados do PDF, mais o papel e o logótipo da loja.
// Tipos próprios em vez de export.DadosRecibo: o Wails gera um namespace
// TypeScript com o nome do pacote, e "export" é palavra reservada.
type ReciboEcra struct {
	NegocioNome   string
	VendaID       int64
	CriadoEm      time.Time
	ClienteNome   string
	Itens         []ItemReciboEcra
	Subtotal      float64
	DescontoTotal float64
	Total         float64
	Pagamentos    []PagamentoReciboEcra
	Papel         string // "80", "58" ou "a5"
	Logo          string // data URL, "" = sem logótipo
}

type ItemReciboEcra struct {
	NomeProduto   string
	NomeUnidade   string
	Quantidade    float64
	PrecoUnitario float64
	Desconto      float64
	Subtotal      float64
}

type PagamentoReciboEcra struct {
	Metodo string
	Valor  float64
}

func (s *ReciboService) ObterParaEcra(ctx context.Context, vendaID int64) (ReciboEcra, error) {
	d, err := s.dados(ctx, vendaID)
	if err != nil {
		return ReciboEcra{}, err
	}
	papel, err := s.papel(ctx)
	if err != nil {
		return ReciboEcra{}, err
	}
	logo, _, err := s.configs.Obter(ctx, "logo")
	if err != nil {
		return ReciboEcra{}, err
	}
	r := ReciboEcra{
		NegocioNome: d.NegocioNome, VendaID: d.VendaID, CriadoEm: d.CriadoEm, ClienteNome: d.ClienteNome,
		Itens:    make([]ItemReciboEcra, 0, len(d.Itens)),
		Subtotal: d.Subtotal, DescontoTotal: d.DescontoTotal, Total: d.Total,
		Pagamentos: make([]PagamentoReciboEcra, 0, len(d.Pagamentos)),
		Papel:      papel, Logo: logo,
	}
	for _, i := range d.Itens {
		r.Itens = append(r.Itens, ItemReciboEcra(i))
	}
	for _, p := range d.Pagamentos {
		r.Pagamentos = append(r.Pagamentos, PagamentoReciboEcra(p))
	}
	return r, nil
}

func (s *ReciboService) papel(ctx context.Context) (string, error) {
	papel, _, err := s.configs.Obter(ctx, "recibo_papel")
	if err != nil {
		return "", err
	}
	if papel != PapelRecibo58 && papel != PapelReciboA5 {
		papel = PapelRecibo80
	}
	return papel, nil
}

func (s *ReciboService) GerarPDF(ctx context.Context, vendaID int64) ([]byte, error) {
	dados, err := s.dados(ctx, vendaID)
	if err != nil {
		return nil, err
	}
	papel, err := s.papel(ctx)
	if err != nil {
		return nil, err
	}
	switch papel {
	case PapelReciboA5:
		return export.GerarReciboPDF(dados)
	case PapelRecibo58:
		return export.GerarReciboTalao(dados, export.PapelRolo58)
	default:
		return export.GerarReciboTalao(dados, export.PapelRolo80)
	}
}

func (s *ReciboService) dados(ctx context.Context, vendaID int64) (export.DadosRecibo, error) {
	completa, err := s.vendas.ObterVendaCompleta(ctx, vendaID)
	if err != nil {
		return export.DadosRecibo{}, fmt.Errorf("obter venda: %w", err)
	}
	negocio, err := s.negocio.Obter(ctx)
	if err != nil {
		return export.DadosRecibo{}, fmt.Errorf("obter negócio: %w", err)
	}
	cliente, err := s.cliente.Obter(ctx, completa.Venda.ClienteID)
	if err != nil {
		return export.DadosRecibo{}, fmt.Errorf("obter cliente: %w", err)
	}

	itens := make([]export.ItemRecibo, 0, len(completa.Itens))
	for _, item := range completa.Itens {
		produto, err := s.produto.Obter(ctx, item.ProdutoID)
		if err != nil {
			return export.DadosRecibo{}, fmt.Errorf("obter produto do item: %w", err)
		}
		unidade, err := s.produto.ObterUnidade(ctx, item.UnidadeProdutoID)
		if err != nil {
			return export.DadosRecibo{}, fmt.Errorf("obter unidade do item: %w", err)
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

	return export.DadosRecibo{
		NegocioNome:   negocio.Nome,
		VendaID:       completa.Venda.ID,
		CriadoEm:      completa.Venda.CriadoEm,
		ClienteNome:   cliente.Nome,
		Itens:         itens,
		Subtotal:      completa.Venda.Subtotal,
		DescontoTotal: completa.Venda.DescontoTotal,
		Total:         completa.Venda.Total,
		Pagamentos:    pagamentos,
	}, nil
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
