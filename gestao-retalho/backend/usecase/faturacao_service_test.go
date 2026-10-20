package usecase_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

type servicosFaturacao struct {
	db        *sql.DB
	vendas    *usecase.VendaService
	dividas   *usecase.DividaService
	clientes  *usecase.ClienteService
	configs   *repository.ConfigRepository
	faturacao *usecase.FaturacaoService
	armazem   domain.Armazem
	produto   domain.Produto
	unidade   domain.UnidadeProduto
	clienteID int64
}

func novosServicosFaturacao(t *testing.T) servicosFaturacao {
	t.Helper()
	ctx := context.Background()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	clienteRepo := repository.NewClienteRepository(db)
	configRepo := repository.NewConfigRepository(db)
	negocioRepo := repository.NewNegocioRepository(db)
	dividaRepo := repository.NewDividaRepository(db)
	vendas := usecase.NewVendaService(repository.NewVendaRepository(db, stockRepo, loteRepo), produtoRepo, armazemRepo,
		repository.NewFichaTecnicaRepository(db), clienteRepo)
	recibos := usecase.NewReciboService(vendas, produtoRepo, clienteRepo, negocioRepo, configRepo)
	s := servicosFaturacao{
		db:        db,
		vendas:    vendas,
		dividas:   usecase.NewDividaService(dividaRepo, clienteRepo),
		clientes:  usecase.NewClienteService(clienteRepo),
		configs:   configRepo,
		faturacao: usecase.NewFaturacaoService(repository.NewDocumentoFiscalRepository(db), recibos, vendas, clienteRepo, dividaRepo, configRepo, negocioRepo),
	}
	vendas.DefinirEmissor(s.faturacao)
	s.dividas.DefinirEmissor(s.faturacao)

	stock := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), loteRepo)
	s.armazem = criarArmazem(t, ctx, usecase.NewArmazemService(armazemRepo), "Principal")
	s.produto, s.unidade = criarProdutoComPreco(t, ctx, usecase.NewProdutoService(produtoRepo), "Cimento 50kg", 580)
	if _, err := stock.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: s.produto.ID, ArmazemDestinoID: s.armazem.ID, Quantidade: 50, CustoUnitario: 400, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	nuit := "400555666"
	c, err := s.clientes.Criar(ctx, domain.Cliente{Nome: "Construções Matola, Lda", NIF: &nuit})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	s.clienteID = c.ID
	_ = configRepo.Definir(ctx, "loja_nuit", "400123456")
	return s
}

func (s servicosFaturacao) vender(t *testing.T, clienteID *int64, qtd, pago float64) usecase.VendaCompleta {
	t.Helper()
	p := usecase.VendaDiretaParams{
		ArmazemID: s.armazem.ID, ClienteID: clienteID, UtilizadorID: 1,
		Itens: []usecase.ItemVendaDiretaParams{{ProdutoID: s.produto.ID, UnidadeProdutoID: s.unidade.ID, Quantidade: qtd}},
	}
	if pago > 0 {
		p.Pagamentos = []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: pago}}
	}
	var v usecase.VendaCompleta
	var err error
	if clienteID != nil && pago < qtd*580 {
		v, err = s.vendas.RegistarVendaACredito(context.Background(), p)
	} else {
		v, err = s.vendas.RegistarVendaDireta(context.Background(), p)
	}
	if err != nil {
		t.Fatalf("venda: %v", err)
	}
	return v
}

func TestFaturacao_DesligadaNaoEmiteNada(t *testing.T) {
	ctx := context.Background()
	s := novosServicosFaturacao(t)
	v := s.vender(t, nil, 1, 580)
	if _, ok, err := s.faturacao.FaturaDaVenda(ctx, v.Venda.ID); err != nil || ok {
		t.Fatalf("fatura com a faturação desligada: ok=%v err=%v", ok, err)
	}
	if e, _ := s.faturacao.Estado(ctx); e.Ativa {
		t.Error("a faturação devia estar desligada por omissão")
	}
}

func TestFaturacao_FRFTRCeNC(t *testing.T) {
	ctx := context.Background()
	s := novosServicosFaturacao(t)
	if _, err := s.faturacao.DefinirAtiva(ctx, true); err != nil {
		t.Fatalf("ativar: %v", err)
	}

	// Venda paga, consumidor final → FR 1, IVA 16% incluído.
	paga := s.vender(t, nil, 2, 1160)
	fr, ok, err := s.faturacao.FaturaDaVenda(ctx, paga.Venda.ID)
	if err != nil || !ok {
		t.Fatalf("FR: ok=%v err=%v", ok, err)
	}
	d := fr.Documento
	if d.Tipo != domain.TipoFaturaRecibo || d.Numero != 1 || d.Total != 1160 || d.ValorIVA != 160 || d.TaxaIVA != 16 {
		t.Errorf("FR = %+v", d)
	}
	if d.ClienteNome != usecase.ConsumidorFinal || d.EmitenteNUIT == nil || *d.EmitenteNUIT != "400123456" {
		t.Errorf("FR adquirente/emitente = %q / %v", d.ClienteNome, d.EmitenteNUIT)
	}
	if len(fr.Linhas) != 1 || fr.Linhas[0].Descricao != "Cimento 50kg" || fr.Linhas[0].Subtotal != 1160 {
		t.Errorf("FR linhas = %+v", fr.Linhas)
	}

	// O NUIT da loja muda depois: a FR já emitida não muda.
	_ = s.configs.Definir(ctx, "loja_nuit", "400999999")
	if again, _ := s.faturacao.Obter(ctx, d.ID); *again.Documento.EmitenteNUIT != "400123456" {
		t.Errorf("emitente mudou depois da emissão: %v", *again.Documento.EmitenteNUIT)
	}

	// Venda a crédito → FT 1 com o NUIT do cliente.
	credito := s.vender(t, &s.clienteID, 1, 0)
	ft, ok, _ := s.faturacao.FaturaDaVenda(ctx, credito.Venda.ID)
	if !ok || ft.Documento.Tipo != domain.TipoFatura || ft.Documento.Numero != 1 ||
		ft.Documento.ClienteNUIT == nil || *ft.Documento.ClienteNUIT != "400555666" {
		t.Fatalf("FT = %+v (ok=%v)", ft.Documento, ok)
	}

	// Pagamento da dívida → RC 1 ligado à FT.
	dividas, _ := s.dividas.ListarPorCliente(ctx, s.clienteID)
	_, pagamento, err := s.dividas.RegistarPagamento(ctx, usecase.RegistarPagamentoDividaParams{
		DividaID: dividas[0].ID, Valor: 300, Metodo: domain.MetodoMpesa, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("pagamento: %v", err)
	}
	docs, _ := s.faturacao.ListarMes(ctx, time.Now().Year(), int(time.Now().Month()))
	var rc usecase.DocumentoCompleto
	for _, c := range docs {
		if c.Documento.Tipo == domain.TipoRecibo {
			rc = c
		}
	}
	if rc.Documento.ID == 0 || rc.Documento.Total != 300 || rc.OrigemCodigo != ft.Codigo ||
		rc.Documento.PagamentoDividaID == nil || *rc.Documento.PagamentoDividaID != pagamento.ID {
		t.Errorf("RC = %+v origem %q", rc.Documento, rc.OrigemCodigo)
	}

	// Anular a venda paga → NC 1 que anula a FR, com o motivo.
	if err := s.vendas.AnularVenda(ctx, paga.Venda.ID, "registada por engano", 1); err != nil {
		t.Fatalf("anular: %v", err)
	}
	docs, _ = s.faturacao.ListarMes(ctx, time.Now().Year(), int(time.Now().Month()))
	var nc usecase.DocumentoCompleto
	for _, c := range docs {
		if c.Documento.Tipo == domain.TipoNotaCredito {
			nc = c
		}
	}
	if nc.Documento.ID == 0 || nc.Documento.Total != 1160 || nc.OrigemCodigo != fr.Codigo ||
		nc.Documento.Motivo == nil || *nc.Documento.Motivo != "registada por engano" {
		t.Errorf("NC = %+v origem %q", nc.Documento, nc.OrigemCodigo)
	}

	// Ficheiro do mês: a NC entra a negativo e o RC não soma.
	tabela, err := s.faturacao.TabelaMes(ctx, time.Now().Year(), int(time.Now().Month()))
	if err != nil {
		t.Fatalf("tabela: %v", err)
	}
	if len(tabela.Linhas) != 4 {
		t.Errorf("linhas = %d, want 4 (FR, FT, RC, NC)", len(tabela.Linhas))
	}
	// FR 1160 + FT 580 − NC 1160 = 580.
	if tabela.Totais[8] != "580,00" {
		t.Errorf("total do mês = %q, want 580,00", tabela.Totais[8])
	}

	// PDF de cada documento.
	for _, c := range docs {
		pdf, nome, err := s.faturacao.GerarPDF(ctx, c.Documento.ID)
		if err != nil || len(pdf) == 0 || nome == "" {
			t.Errorf("PDF de %s: %v", c.Codigo, err)
		}
	}
}

func TestFaturacao_DocumentosNaoSeAlteramNemApagam(t *testing.T) {
	ctx := context.Background()
	s := novosServicosFaturacao(t)
	_, _ = s.faturacao.DefinirAtiva(ctx, true)
	v := s.vender(t, nil, 1, 580)
	fr, _, _ := s.faturacao.FaturaDaVenda(ctx, v.Venda.ID)

	if _, err := s.db.ExecContext(ctx, `UPDATE documentos_fiscais SET total = 1 WHERE id = ?`, fr.Documento.ID); err == nil {
		t.Error("alterar um documento emitido devia falhar")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM documentos_fiscais WHERE id = ?`, fr.Documento.ID); err == nil {
		t.Error("apagar um documento emitido devia falhar")
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM documento_linhas WHERE documento_id = ?`, fr.Documento.ID); err == nil {
		t.Error("apagar as linhas de um documento devia falhar")
	}
}

func TestFaturacao_VendaSemDocumentoEmiteDepois(t *testing.T) {
	ctx := context.Background()
	s := novosServicosFaturacao(t)
	_, _ = s.faturacao.DefinirAtiva(ctx, true)

	// Simula a emissão falhada: venda feita sem emissor.
	s.vendas.DefinirEmissor(nil)
	v := s.vender(t, nil, 1, 580)
	s.vendas.DefinirEmissor(s.faturacao)

	em, err := s.faturacao.VendasSemDocumento(ctx)
	if err != nil || len(em) != 1 || em[0].VendaID != v.Venda.ID {
		t.Fatalf("vendas sem documento = %+v, %v", em, err)
	}
	doc, err := s.faturacao.EmitirEmFalta(ctx, v.Venda.ID)
	if err != nil || doc.Documento.Tipo != domain.TipoFaturaRecibo {
		t.Fatalf("emitir em falta: %+v, %v", doc.Documento, err)
	}
	// Emitir outra vez devolve a mesma, não cria segunda.
	outra, _ := s.faturacao.EmitirEmFalta(ctx, v.Venda.ID)
	if outra.Documento.ID != doc.Documento.ID {
		t.Errorf("segunda emissão criou outro documento (%d ≠ %d)", outra.Documento.ID, doc.Documento.ID)
	}
	if em, _ := s.faturacao.VendasSemDocumento(ctx); len(em) != 0 {
		t.Errorf("ainda há vendas sem documento: %+v", em)
	}
}
