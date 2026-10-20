package usecase

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/repository"
)

// CotacaoService faz cotações (orçamentos/proformas): não mexem no stock
// nem contam como venda, até serem convertidas numa venda.
type CotacaoService struct {
	cotacoes *repository.CotacaoRepository
	produtos *repository.ProdutoRepository
	clientes *repository.ClienteRepository
	configs  *repository.ConfigRepository
	negocios *repository.NegocioRepository
	vendas   *VendaService
}

func NewCotacaoService(
	cotacoes *repository.CotacaoRepository,
	produtos *repository.ProdutoRepository,
	clientes *repository.ClienteRepository,
	configs *repository.ConfigRepository,
	negocios *repository.NegocioRepository,
	vendas *VendaService,
) *CotacaoService {
	return &CotacaoService{cotacoes: cotacoes, produtos: produtos, clientes: clientes, configs: configs, negocios: negocios, vendas: vendas}
}

// ValidadePorOmissao é quantos dias vale uma cotação se nada for dito.
const ValidadePorOmissao = 15

// ConsumidorFinal é o nome no documento quando a cotação não tem cliente.
const ConsumidorFinal = "Consumidor final"

var ErrCotacaoNaoAberta = errors.New("a cotação já não está aberta (foi convertida ou cancelada)")

type ItemNovaCotacao struct {
	ProdutoID        int64
	UnidadeProdutoID int64
	Quantidade       float64
	Desconto         float64 // na linha, em MT
}

type NovaCotacao struct {
	// ClienteID de um cliente registado; sem ele, usa-se ClienteNome (ex:
	// uma empresa que pediu orçamento por telefone) ou "Consumidor final".
	ClienteID     *int64
	ClienteNome   string
	ValidadeDias  int64 // 0 = ValidadePorOmissao
	Itens         []ItemNovaCotacao
	DescontoTotal float64
	Observacoes   string
	UtilizadorID  int64
}

// CotacaoCompleta é a cotação com os itens e o que o ecrã precisa já
// calculado (código, validade, se expirou).
type CotacaoCompleta struct {
	Cotacao   domain.Cotacao
	Itens     []domain.ItemCotacao
	Codigo    string
	ValidaAte time.Time
	Expirada  bool
}

func completa(c domain.Cotacao, itens []domain.ItemCotacao) CotacaoCompleta {
	if itens == nil {
		itens = []domain.ItemCotacao{}
	}
	return CotacaoCompleta{Cotacao: c, Itens: itens, Codigo: c.Codigo(), ValidaAte: c.ValidaAte(), Expirada: c.Expirada(time.Now())}
}

func arred2(v float64) float64 { return math.Round(v*100) / 100 }

// Criar grava a cotação com os preços de venda atuais (congelados) e o IVA
// incluído no total, segundo a taxa da loja.
func (s *CotacaoService) Criar(ctx context.Context, p NovaCotacao) (CotacaoCompleta, error) {
	if len(p.Itens) == 0 {
		return CotacaoCompleta{}, fmt.Errorf("a cotação precisa de pelo menos um produto")
	}
	if p.ValidadeDias == 0 {
		p.ValidadeDias = ValidadePorOmissao
	}
	if p.ValidadeDias < 1 || p.ValidadeDias > 365 {
		return CotacaoCompleta{}, fmt.Errorf("a validade tem de ser entre 1 e 365 dias")
	}
	if p.DescontoTotal < 0 {
		return CotacaoCompleta{}, fmt.Errorf("o desconto não pode ser negativo")
	}

	c := domain.Cotacao{
		NegocioID:    domain.NegocioID,
		Ano:          int64(time.Now().Year()),
		ValidadeDias: p.ValidadeDias,
		UtilizadorID: p.UtilizadorID,
	}
	if err := s.preencherCliente(ctx, &c, p); err != nil {
		return CotacaoCompleta{}, err
	}
	if obs := strings.TrimSpace(p.Observacoes); obs != "" {
		c.Observacoes = &obs
	}

	itens := make([]domain.ItemCotacao, 0, len(p.Itens))
	var descontoLinhas float64
	for _, i := range p.Itens {
		produto, err := s.produtos.Obter(ctx, i.ProdutoID)
		if err != nil {
			return CotacaoCompleta{}, fmt.Errorf("produto %d: %w", i.ProdutoID, err)
		}
		if !produto.Ativo {
			return CotacaoCompleta{}, fmt.Errorf("%q está desativado", produto.Nome)
		}
		unidade, err := s.produtos.ObterUnidade(ctx, i.UnidadeProdutoID)
		if err != nil || unidade.ProdutoID != produto.ID {
			return CotacaoCompleta{}, fmt.Errorf("%q: unidade inválida", produto.Nome)
		}
		if i.Quantidade <= 0 {
			return CotacaoCompleta{}, fmt.Errorf("%q: a quantidade tem de ser maior que zero", produto.Nome)
		}
		bruto := arred2(i.Quantidade * unidade.PrecoVenda)
		if i.Desconto < 0 || i.Desconto > bruto {
			return CotacaoCompleta{}, fmt.Errorf("%q: desconto inválido", produto.Nome)
		}
		itens = append(itens, domain.ItemCotacao{
			ProdutoID:        produto.ID,
			UnidadeProdutoID: unidade.ID,
			Descricao:        produto.Nome,
			NomeUnidade:      unidade.NomeUnidade,
			Quantidade:       i.Quantidade,
			PrecoUnitario:    unidade.PrecoVenda,
			Desconto:         i.Desconto,
			Subtotal:         arred2(bruto - i.Desconto),
		})
		c.Subtotal += bruto
		descontoLinhas += i.Desconto
	}
	c.Subtotal = arred2(c.Subtotal)
	c.DescontoTotal = arred2(descontoLinhas + p.DescontoTotal)
	c.Total = arred2(c.Subtotal - c.DescontoTotal)
	if c.Total < 0 {
		return CotacaoCompleta{}, fmt.Errorf("o desconto é maior do que o total")
	}

	taxa, _, err := s.configs.Obter(ctx, "iva_taxa")
	if err != nil {
		return CotacaoCompleta{}, err
	}
	c.TaxaIVA = taxaIVA(taxa)
	c.ValorIVA = IVAIncluido(c.Total, c.TaxaIVA)

	criada, criados, err := s.cotacoes.Criar(ctx, c, itens)
	if err != nil {
		return CotacaoCompleta{}, err
	}
	return completa(criada, criados), nil
}

// IVAIncluido é a parte do total que é IVA, com os preços já com IVA:
// total − total / (1 + taxa).
func IVAIncluido(total, taxa float64) float64 {
	if taxa <= 0 {
		return 0
	}
	return arred2(total - total/(1+taxa/100))
}

func (s *CotacaoService) preencherCliente(ctx context.Context, c *domain.Cotacao, p NovaCotacao) error {
	if p.ClienteID != nil && *p.ClienteID != domain.ClienteAnonimoID {
		cliente, err := s.clientes.Obter(ctx, *p.ClienteID)
		if err != nil {
			return fmt.Errorf("cliente: %w", err)
		}
		if !cliente.Ativo {
			return fmt.Errorf("o cliente %q está desativado", cliente.Nome)
		}
		id := cliente.ID
		c.ClienteID = &id
		c.ClienteNome = cliente.Nome
		c.ClienteNUIT = cliente.NIF
		var contactos []string
		for _, v := range []*string{cliente.Telefone, cliente.Email} {
			if v != nil && *v != "" {
				contactos = append(contactos, *v)
			}
		}
		if len(contactos) > 0 {
			t := strings.Join(contactos, " · ")
			c.ClienteContacto = &t
		}
		return nil
	}
	c.ClienteNome = strings.TrimSpace(p.ClienteNome)
	if c.ClienteNome == "" {
		c.ClienteNome = ConsumidorFinal
	}
	return nil
}

func (s *CotacaoService) Listar(ctx context.Context) ([]CotacaoCompleta, error) {
	lista, err := s.cotacoes.Listar(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	out := make([]CotacaoCompleta, 0, len(lista))
	for _, c := range lista {
		out = append(out, completa(c, nil))
	}
	return out, nil
}

func (s *CotacaoService) Obter(ctx context.Context, id int64) (CotacaoCompleta, error) {
	c, err := s.cotacoes.Obter(ctx, id)
	if err != nil {
		return CotacaoCompleta{}, err
	}
	itens, err := s.cotacoes.ListarItens(ctx, id)
	if err != nil {
		return CotacaoCompleta{}, err
	}
	return completa(c, itens), nil
}

func (s *CotacaoService) Cancelar(ctx context.Context, id int64) error {
	ok, err := s.cotacoes.Cancelar(ctx, id)
	if err != nil {
		return err
	}
	if !ok {
		return ErrCotacaoNaoAberta
	}
	return nil
}

type ConverterCotacaoParams struct {
	CotacaoID    int64
	ArmazemID    int64
	Pagamentos   []PagamentoVendaParams // o que o cliente paga agora; o resto fica em dívida
	UtilizadorID int64
}

// Converter transforma a cotação numa venda, ao preço cotado: se o preço
// atual subiu, a diferença entra como desconto; se desceu, fica o preço
// atual (mais barato para o cliente). Paga por inteiro é uma venda
// direta; com parte por pagar é uma venda a crédito (só com cliente
// registado). O stock sai nesse momento, do armazém escolhido.
func (s *CotacaoService) Converter(ctx context.Context, p ConverterCotacaoParams) (VendaCompleta, error) {
	cc, err := s.Obter(ctx, p.CotacaoID)
	if err != nil {
		return VendaCompleta{}, err
	}
	c := cc.Cotacao
	if c.Estado != domain.EstadoCotacaoAberta {
		return VendaCompleta{}, ErrCotacaoNaoAberta
	}

	itens := make([]ItemVendaDiretaParams, 0, len(cc.Itens))
	for _, i := range cc.Itens {
		unidade, err := s.produtos.ObterUnidade(ctx, i.UnidadeProdutoID)
		if err != nil {
			return VendaCompleta{}, fmt.Errorf("%q: %w", i.Descricao, err)
		}
		desconto := i.Desconto
		if subiu := unidade.PrecoVenda - i.PrecoUnitario; subiu > 0 {
			desconto += arred2(i.Quantidade * subiu)
		}
		itens = append(itens, ItemVendaDiretaParams{
			ProdutoID:        i.ProdutoID,
			UnidadeProdutoID: i.UnidadeProdutoID,
			Quantidade:       i.Quantidade,
			Desconto:         desconto,
		})
	}

	// O desconto global da cotação é o total dos descontos menos os das linhas.
	var descontoLinhas float64
	for _, i := range cc.Itens {
		descontoLinhas += i.Desconto
	}
	venda := VendaDiretaParams{
		ArmazemID:     p.ArmazemID,
		ClienteID:     c.ClienteID,
		Itens:         itens,
		DescontoTotal: arred2(c.DescontoTotal - descontoLinhas),
		Pagamentos:    p.Pagamentos,
		UtilizadorID:  p.UtilizadorID,
	}

	var pago float64
	for _, pg := range p.Pagamentos {
		pago += pg.Valor
	}
	var completaVenda VendaCompleta
	if pago+toleranciaPagamento >= c.Total {
		completaVenda, err = s.vendas.RegistarVendaDireta(ctx, venda)
	} else {
		if c.ClienteID == nil {
			return VendaCompleta{}, fmt.Errorf("sem cliente registado a venda tem de ser paga por inteiro — falta %s", qtd(c.Total-pago))
		}
		completaVenda, err = s.vendas.RegistarVendaACredito(ctx, venda)
	}
	if err != nil {
		return VendaCompleta{}, err
	}

	ok, err := s.cotacoes.MarcarConvertida(ctx, c.ID, completaVenda.Venda.ID)
	if err != nil {
		return completaVenda, fmt.Errorf("venda #%d registada, mas a cotação não ficou marcada: %w", completaVenda.Venda.ID, err)
	}
	if !ok {
		return completaVenda, fmt.Errorf("venda #%d registada, mas a cotação já tinha sido convertida", completaVenda.Venda.ID)
	}
	return completaVenda, nil
}

// GerarPDF produz a cotação em A4 com os dados atuais da loja (nome,
// NUIT, contactos, logótipo) e os do cliente congelados na cotação.
func (s *CotacaoService) GerarPDF(ctx context.Context, id int64) ([]byte, string, error) {
	cc, err := s.Obter(ctx, id)
	if err != nil {
		return nil, "", err
	}
	negocio, err := s.negocios.Obter(ctx)
	if err != nil {
		return nil, "", err
	}
	todas, err := s.configs.Listar(ctx)
	if err != nil {
		return nil, "", err
	}
	c := cc.Cotacao
	d := export.DadosCotacaoPDF{
		Codigo:    cc.Codigo,
		CriadoEm:  c.CriadoEm,
		ValidaAte: cc.ValidaAte,
		Loja: export.DadosRecibo{
			NegocioNome:  negocio.Nome,
			LojaNUIT:     todas["loja_nuit"],
			LojaEndereco: todas["loja_endereco"],
			LojaTelefone: todas["loja_telefone"],
			LojaEmail:    todas["loja_email"],
		},
		Logo:          todas["logo"],
		ClienteNome:   c.ClienteNome,
		Subtotal:      c.Subtotal,
		DescontoTotal: c.DescontoTotal,
		Total:         c.Total,
		TaxaIVA:       c.TaxaIVA,
		ValorIVA:      c.ValorIVA,
	}
	if c.ClienteNUIT != nil {
		d.ClienteNUIT = *c.ClienteNUIT
	}
	if c.ClienteContacto != nil {
		d.ClienteContacto = *c.ClienteContacto
	}
	if c.Observacoes != nil {
		d.Observacoes = *c.Observacoes
	}
	for _, i := range cc.Itens {
		d.Itens = append(d.Itens, export.ItemCotacaoPDF{
			Descricao: i.Descricao, NomeUnidade: i.NomeUnidade, Quantidade: i.Quantidade,
			PrecoUnitario: i.PrecoUnitario, Desconto: i.Desconto, Subtotal: i.Subtotal,
		})
	}
	pdf, err := export.GerarCotacaoPDF(d)
	if err != nil {
		return nil, "", err
	}
	nome := strings.NewReplacer(" ", "-", "/", "-").Replace(cc.Codigo) + ".pdf"
	return pdf, nome, nil
}
