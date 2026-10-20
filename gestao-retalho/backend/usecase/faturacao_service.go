package usecase

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/repository"
)

// EmissorDocumentos é chamado depois de cada operação que gera um
// documento fiscal. VendaService, ContaService e DividaService recebem-no
// por DefinirEmissor (em vez de no construtor), porque o FaturacaoService
// precisa do VendaService para ler as vendas.
type EmissorDocumentos interface {
	AposVenda(ctx context.Context, vendaID int64) error
	AposPagamentoDivida(ctx context.Context, p domain.PagamentoDivida) error
	AposAnulacao(ctx context.Context, vendaID int64, motivo string) error
}

// emitir chama o emissor sem desfazer a operação que já ficou gravada: se
// a emissão falhar, a venda aparece em "Vendas sem documento" para emitir
// de novo.
func emitir(emissor EmissorDocumentos, fn func(EmissorDocumentos) error) {
	if emissor == nil {
		return
	}
	if err := fn(emissor); err != nil {
		log.Printf("faturação: documento fiscal não emitido: %v", err)
	}
}

// FaturacaoService emite os documentos fiscais — Fatura-Recibo (venda
// paga), Fatura (venda com parte por pagar), Recibo (pagamento de dívida)
// e Nota de Crédito (anulação) — quando a faturação está ligada nas
// configurações. Fica desligada até o software estar aprovado pela AT.
type FaturacaoService struct {
	docs     *repository.DocumentoFiscalRepository
	recibos  *ReciboService
	vendas   *VendaService
	clientes *repository.ClienteRepository
	dividas  *repository.DividaRepository
	configs  *repository.ConfigRepository
	negocios *repository.NegocioRepository
}

func NewFaturacaoService(
	docs *repository.DocumentoFiscalRepository,
	recibos *ReciboService,
	vendas *VendaService,
	clientes *repository.ClienteRepository,
	dividas *repository.DividaRepository,
	configs *repository.ConfigRepository,
	negocios *repository.NegocioRepository,
) *FaturacaoService {
	return &FaturacaoService{docs: docs, recibos: recibos, vendas: vendas, clientes: clientes, dividas: dividas, configs: configs, negocios: negocios}
}

// EstadoFaturacao diz se a emissão está ligada e desde quando (as vendas
// anteriores não têm documento e não contam como "em falta").
type EstadoFaturacao struct {
	Ativa bool
	Desde time.Time
}

func (s *FaturacaoService) Estado(ctx context.Context) (EstadoFaturacao, error) {
	todas, err := s.configs.Listar(ctx)
	if err != nil {
		return EstadoFaturacao{}, err
	}
	e := EstadoFaturacao{Ativa: todas["faturacao_ativa"] == "true"}
	if t, err := time.Parse(time.RFC3339, todas["faturacao_desde"]); err == nil {
		e.Desde = t
	}
	return e, nil
}

// DefinirAtiva liga ou desliga a emissão de documentos fiscais. Ao ligar,
// a data passa a contar para as "vendas sem documento".
func (s *FaturacaoService) DefinirAtiva(ctx context.Context, ativa bool) (EstadoFaturacao, error) {
	valor := "false"
	if ativa {
		valor = "true"
		if err := s.configs.Definir(ctx, "faturacao_desde", time.Now().UTC().Format(time.RFC3339)); err != nil {
			return EstadoFaturacao{}, err
		}
	}
	if err := s.configs.Definir(ctx, "faturacao_ativa", valor); err != nil {
		return EstadoFaturacao{}, err
	}
	return s.Estado(ctx)
}

func (s *FaturacaoService) ativa(ctx context.Context) (bool, error) {
	e, err := s.Estado(ctx)
	return e.Ativa, err
}

// base preenche o emitente (dados atuais da loja, que ficam congelados no
// documento) e a taxa de IVA.
func (s *FaturacaoService) base(ctx context.Context, tipo domain.TipoDocumento, utilizadorID int64) (domain.DocumentoFiscal, error) {
	negocio, err := s.negocios.Obter(ctx)
	if err != nil {
		return domain.DocumentoFiscal{}, err
	}
	todas, err := s.configs.Listar(ctx)
	if err != nil {
		return domain.DocumentoFiscal{}, err
	}
	opt := func(v string) *string {
		if v == "" {
			return nil
		}
		return &v
	}
	var contactos []string
	for _, v := range []string{todas["loja_telefone"], todas["loja_email"]} {
		if v != "" {
			contactos = append(contactos, v)
		}
	}
	return domain.DocumentoFiscal{
		NegocioID:        domain.NegocioID,
		Tipo:             tipo,
		Ano:              int64(time.Now().Year()),
		EmitenteNome:     negocio.Nome,
		EmitenteNUIT:     opt(todas["loja_nuit"]),
		EmitenteEndereco: opt(todas["loja_endereco"]),
		EmitenteContacto: opt(strings.Join(contactos, " · ")),
		TaxaIVA:          taxaIVA(todas["iva_taxa"]),
		UtilizadorID:     utilizadorID,
	}, nil
}

// adquirente copia os dados do cliente (o anónimo é "Consumidor final").
func (s *FaturacaoService) adquirente(ctx context.Context, d *domain.DocumentoFiscal, clienteID int64) error {
	if clienteID == domain.ClienteAnonimoID {
		d.ClienteNome = ConsumidorFinal
		return nil
	}
	c, err := s.clientes.Obter(ctx, clienteID)
	if err != nil {
		return fmt.Errorf("cliente: %w", err)
	}
	id := c.ID
	d.ClienteID = &id
	d.ClienteNome = c.Nome
	d.ClienteNUIT = c.NIF
	d.ClienteEndereco = c.Morada
	return nil
}

// AposVenda emite a Fatura-Recibo (venda paga) ou a Fatura (com parte por
// pagar). Emitir duas vezes para a mesma venda não faz nada.
func (s *FaturacaoService) AposVenda(ctx context.Context, vendaID int64) error {
	if ok, err := s.ativa(ctx); err != nil || !ok {
		return err
	}
	_, existe, err := s.docs.FaturaDaVenda(ctx, vendaID)
	if err != nil || existe {
		return err
	}
	_, err = s.emitirFatura(ctx, vendaID)
	return err
}

func (s *FaturacaoService) emitirFatura(ctx context.Context, vendaID int64) (domain.DocumentoFiscal, error) {
	venda, err := s.vendas.ObterVendaCompleta(ctx, vendaID)
	if err != nil {
		return domain.DocumentoFiscal{}, err
	}
	if venda.Venda.Status == domain.StatusVendaCancelada {
		return domain.DocumentoFiscal{}, fmt.Errorf("a venda #%d está anulada", vendaID)
	}
	tipo := domain.TipoFaturaRecibo
	if venda.Venda.Status != domain.StatusVendaPaga {
		tipo = domain.TipoFatura
	}
	d, err := s.base(ctx, tipo, venda.Venda.UtilizadorID)
	if err != nil {
		return domain.DocumentoFiscal{}, err
	}
	if err := s.adquirente(ctx, &d, venda.Venda.ClienteID); err != nil {
		return domain.DocumentoFiscal{}, err
	}
	dados, err := s.recibos.dados(ctx, vendaID)
	if err != nil {
		return domain.DocumentoFiscal{}, err
	}
	id := vendaID
	d.VendaID = &id
	d.Subtotal = venda.Venda.Subtotal
	d.DescontoTotal = venda.Venda.DescontoTotal
	d.Total = venda.Venda.Total
	d.ValorIVA = IVAIncluido(d.Total, d.TaxaIVA)
	linhas := make([]domain.LinhaDocumento, 0, len(dados.Itens))
	for _, i := range dados.Itens {
		linhas = append(linhas, domain.LinhaDocumento{
			Descricao: i.NomeProduto, Unidade: i.NomeUnidade, Quantidade: i.Quantidade,
			PrecoUnitario: i.PrecoUnitario, Desconto: i.Desconto, Subtotal: i.Subtotal,
		})
	}
	criado, _, err := s.docs.Criar(ctx, d, linhas)
	return criado, err
}

// AposPagamentoDivida emite o Recibo do pagamento, ligado à Fatura da
// venda que originou a dívida (se essa venda tiver fatura).
func (s *FaturacaoService) AposPagamentoDivida(ctx context.Context, p domain.PagamentoDivida) error {
	if ok, err := s.ativa(ctx); err != nil || !ok {
		return err
	}
	_, existe, err := s.docs.ReciboDoPagamento(ctx, p.ID)
	if err != nil || existe {
		return err
	}
	divida, err := s.dividas.Obter(ctx, p.DividaID)
	if err != nil {
		return err
	}
	d, err := s.base(ctx, domain.TipoRecibo, p.UtilizadorID)
	if err != nil {
		return err
	}
	if err := s.adquirente(ctx, &d, divida.ClienteID); err != nil {
		return err
	}
	descricao := fmt.Sprintf("Pagamento da venda #%d", divida.VendaID)
	if fatura, ok, err := s.docs.FaturaDaVenda(ctx, divida.VendaID); err != nil {
		return err
	} else if ok {
		origem := fatura.ID
		d.DocumentoOrigemID = &origem
		descricao = "Pagamento da " + fatura.Codigo()
	}
	pagID := p.ID
	d.PagamentoDividaID = &pagID
	d.Subtotal, d.Total = p.Valor, p.Valor
	// O IVA foi liquidado na Fatura; o recibo só regista o pagamento.
	d.TaxaIVA, d.ValorIVA = 0, 0
	_, _, err = s.docs.Criar(ctx, d, []domain.LinhaDocumento{{
		Descricao: descricao, Unidade: rotuloMetodo(p.Metodo), Quantidade: 1, PrecoUnitario: p.Valor, Subtotal: p.Valor,
	}})
	return err
}

// AposAnulacao emite a Nota de Crédito que anula por inteiro a Fatura ou
// Fatura-Recibo da venda (vendas sem documento — anteriores à faturação —
// não geram nota).
func (s *FaturacaoService) AposAnulacao(ctx context.Context, vendaID int64, motivo string) error {
	if ok, err := s.ativa(ctx); err != nil || !ok {
		return err
	}
	fatura, ok, err := s.docs.FaturaDaVenda(ctx, vendaID)
	if err != nil || !ok {
		return err
	}
	if _, existe, err := s.docs.NotaCreditoDaVenda(ctx, vendaID); err != nil || existe {
		return err
	}
	linhas, err := s.docs.ListarLinhas(ctx, fatura.ID)
	if err != nil {
		return err
	}
	d, err := s.base(ctx, domain.TipoNotaCredito, fatura.UtilizadorID)
	if err != nil {
		return err
	}
	// O adquirente e os valores são os da fatura anulada.
	d.ClienteID, d.ClienteNome, d.ClienteNUIT, d.ClienteEndereco = fatura.ClienteID, fatura.ClienteNome, fatura.ClienteNUIT, fatura.ClienteEndereco
	d.Subtotal, d.DescontoTotal, d.Total = fatura.Subtotal, fatura.DescontoTotal, fatura.Total
	d.TaxaIVA, d.ValorIVA = fatura.TaxaIVA, fatura.ValorIVA
	id, origem := vendaID, fatura.ID
	d.VendaID, d.DocumentoOrigemID = &id, &origem
	if m := strings.TrimSpace(motivo); m != "" {
		d.Motivo = &m
	}
	novas := make([]domain.LinhaDocumento, 0, len(linhas))
	for _, l := range linhas {
		l.ID, l.DocumentoID = 0, 0
		novas = append(novas, l)
	}
	_, _, err = s.docs.Criar(ctx, d, novas)
	return err
}

// EmitirEmFalta emite a fatura de uma venda que ficou sem ela (a emissão
// falhou na altura).
func (s *FaturacaoService) EmitirEmFalta(ctx context.Context, vendaID int64) (DocumentoCompleto, error) {
	if ok, err := s.ativa(ctx); err != nil {
		return DocumentoCompleto{}, err
	} else if !ok {
		return DocumentoCompleto{}, fmt.Errorf("a emissão de documentos fiscais está desligada")
	}
	if d, existe, err := s.docs.FaturaDaVenda(ctx, vendaID); err != nil {
		return DocumentoCompleto{}, err
	} else if existe {
		return s.Obter(ctx, d.ID)
	}
	d, err := s.emitirFatura(ctx, vendaID)
	if err != nil {
		return DocumentoCompleto{}, err
	}
	return s.Obter(ctx, d.ID)
}

// DocumentoCompleto é o documento com as linhas e o que o ecrã mostra.
type DocumentoCompleto struct {
	Documento    domain.DocumentoFiscal
	Linhas       []domain.LinhaDocumento
	Codigo       string
	NomeTipo     string
	OrigemCodigo string // documento anulado (NC) ou pago (RC)
}

func (s *FaturacaoService) completo(ctx context.Context, d domain.DocumentoFiscal, comLinhas bool) (DocumentoCompleto, error) {
	c := DocumentoCompleto{Documento: d, Codigo: d.Codigo(), NomeTipo: d.Tipo.Nome(), Linhas: []domain.LinhaDocumento{}}
	if d.DocumentoOrigemID != nil {
		origem, err := s.docs.Obter(ctx, *d.DocumentoOrigemID)
		if err != nil {
			return c, err
		}
		c.OrigemCodigo = origem.Codigo()
	}
	if comLinhas {
		linhas, err := s.docs.ListarLinhas(ctx, d.ID)
		if err != nil {
			return c, err
		}
		c.Linhas = linhas
	}
	return c, nil
}

func (s *FaturacaoService) Obter(ctx context.Context, id int64) (DocumentoCompleto, error) {
	d, err := s.docs.Obter(ctx, id)
	if err != nil {
		return DocumentoCompleto{}, err
	}
	return s.completo(ctx, d, true)
}

// FaturaDaVenda devolve o documento (FR/FT) da venda, se existir.
func (s *FaturacaoService) FaturaDaVenda(ctx context.Context, vendaID int64) (DocumentoCompleto, bool, error) {
	d, ok, err := s.docs.FaturaDaVenda(ctx, vendaID)
	if err != nil || !ok {
		return DocumentoCompleto{}, false, err
	}
	c, err := s.completo(ctx, d, true)
	return c, err == nil, err
}

// ListarMes devolve os documentos emitidos num mês (hora local).
func (s *FaturacaoService) ListarMes(ctx context.Context, ano, mes int) ([]DocumentoCompleto, error) {
	desde := time.Date(ano, time.Month(mes), 1, 0, 0, 0, 0, time.Local)
	docs, err := s.docs.ListarEntre(ctx, domain.NegocioID, desde, desde.AddDate(0, 1, 0))
	if err != nil {
		return nil, err
	}
	out := make([]DocumentoCompleto, 0, len(docs))
	for _, d := range docs {
		c, err := s.completo(ctx, d, false)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// VendasSemDocumento são as vendas feitas com a faturação ligada que
// ficaram sem FR/FT.
func (s *FaturacaoService) VendasSemDocumento(ctx context.Context) ([]repository.VendaSemFatura, error) {
	e, err := s.Estado(ctx)
	if err != nil || !e.Ativa {
		return []repository.VendaSemFatura{}, err
	}
	return s.docs.VendasSemFatura(ctx, domain.NegocioID, e.Desde)
}

// TabelaMes é o ficheiro mensal dos documentos emitidos. A estrutura é
// provisória: a AT ainda não publicou a do SAF-T (Moz); ajusta-se quando
// a tivermos.
func (s *FaturacaoService) TabelaMes(ctx context.Context, ano, mes int) (export.Tabela, error) {
	docs, err := s.ListarMes(ctx, ano, mes)
	if err != nil {
		return export.Tabela{}, err
	}
	t := export.Tabela{
		Titulo:  fmt.Sprintf("Documentos fiscais %02d-%d", mes, ano),
		Colunas: []string{"Tipo", "Número", "Data", "NUIT adquirente", "Adquirente", "Base tributável", "Taxa IVA", "IVA", "Total", "Documento de origem", "Motivo"},
	}
	var base, iva, total float64
	for _, c := range docs {
		d := c.Documento
		sinal := 1.0
		if d.Tipo == domain.TipoNotaCredito {
			sinal = -1
		}
		nuit, motivo := "", ""
		if d.ClienteNUIT != nil {
			nuit = *d.ClienteNUIT
		}
		if d.Motivo != nil {
			motivo = *d.Motivo
		}
		b := sinal * (d.Total - d.ValorIVA)
		t.Linhas = append(t.Linhas, []string{
			c.NomeTipo, c.Codigo, d.CriadoEm.Local().Format("02/01/2006 15:04"), nuit, d.ClienteNome,
			export.FormatarValor(b), fmt.Sprintf("%s%%", qtd(d.TaxaIVA)), export.FormatarValor(sinal * d.ValorIVA),
			export.FormatarValor(sinal * d.Total), c.OrigemCodigo, motivo,
		})
		// Os recibos não somam: o valor já está na fatura que pagam.
		if d.Tipo != domain.TipoRecibo {
			base += b
			iva += sinal * d.ValorIVA
			total += sinal * d.Total
		}
	}
	t.Totais = []string{"Total (sem recibos)", "", "", "", "", export.FormatarValor(base), "", export.FormatarValor(iva), export.FormatarValor(total), "", ""}
	return t, nil
}

// GerarPDF produz o documento em A4 com o emitente e o adquirente tal
// como ficaram na emissão.
func (s *FaturacaoService) GerarPDF(ctx context.Context, id int64) ([]byte, string, error) {
	c, err := s.Obter(ctx, id)
	if err != nil {
		return nil, "", err
	}
	logo, _, err := s.configs.Obter(ctx, "logo")
	if err != nil {
		return nil, "", err
	}
	d := c.Documento
	val := func(p *string) string {
		if p == nil {
			return ""
		}
		return *p
	}
	a4 := export.DadosDocumentoA4{
		Titulo:         strings.ToUpper(c.NomeTipo),
		Codigo:         c.Codigo,
		CriadoEm:       d.CriadoEm,
		DiscriminarIVA: d.Tipo != domain.TipoRecibo,
		Rodape:         "Processado por computador — Gestão de Retalho. Valores com IVA incluído.",
		Loja: export.DadosRecibo{
			NegocioNome: d.EmitenteNome, LojaNUIT: val(d.EmitenteNUIT), LojaEndereco: val(d.EmitenteEndereco),
			LojaTelefone: val(d.EmitenteContacto),
		},
		Logo:            logo,
		ClienteNome:     d.ClienteNome,
		ClienteNUIT:     val(d.ClienteNUIT),
		ClienteContacto: val(d.ClienteEndereco),
		Subtotal:        d.Subtotal,
		DescontoTotal:   d.DescontoTotal,
		Total:           d.Total,
		TaxaIVA:         d.TaxaIVA,
		ValorIVA:        d.ValorIVA,
		Observacoes:     val(d.Motivo),
	}
	if c.OrigemCodigo != "" {
		rotulo := "Refere-se a: "
		if d.Tipo == domain.TipoNotaCredito {
			rotulo = "Anula: "
		}
		a4.Cabecalho = append(a4.Cabecalho, rotulo+c.OrigemCodigo)
	}
	if d.Motivo != nil {
		a4.Observacoes = "Motivo: " + *d.Motivo
	}
	for _, l := range c.Linhas {
		a4.Itens = append(a4.Itens, export.ItemCotacaoPDF{
			Descricao: l.Descricao, NomeUnidade: l.Unidade, Quantidade: l.Quantidade,
			PrecoUnitario: l.PrecoUnitario, Desconto: l.Desconto, Subtotal: l.Subtotal,
		})
	}
	pdf, err := export.GerarDocumentoA4(a4)
	if err != nil {
		return nil, "", err
	}
	nome := strings.NewReplacer(" ", "-", "/", "-").Replace(c.Codigo) + ".pdf"
	return pdf, nome, nil
}
