package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

type servicosCotacao struct {
	produtos *usecase.ProdutoService
	stock    *usecase.StockService
	clientes *usecase.ClienteService
	dividas  *repository.DividaRepository
	configs  *repository.ConfigRepository
	cotacoes *usecase.CotacaoService
	armazem  domain.Armazem
	cimento  domain.Produto
	unidade  domain.UnidadeProduto
}

func novosServicosCotacao(t *testing.T) servicosCotacao {
	t.Helper()
	ctx := context.Background()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	clienteRepo := repository.NewClienteRepository(db)
	configRepo := repository.NewConfigRepository(db)
	vendas := usecase.NewVendaService(repository.NewVendaRepository(db, stockRepo, loteRepo), produtoRepo, armazemRepo,
		repository.NewFichaTecnicaRepository(db), clienteRepo)
	s := servicosCotacao{
		produtos: usecase.NewProdutoService(produtoRepo),
		stock:    usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), loteRepo),
		clientes: usecase.NewClienteService(clienteRepo),
		dividas:  repository.NewDividaRepository(db),
		configs:  configRepo,
		cotacoes: usecase.NewCotacaoService(repository.NewCotacaoRepository(db), produtoRepo, clienteRepo, configRepo,
			repository.NewNegocioRepository(db), vendas),
	}
	s.armazem = criarArmazem(t, ctx, usecase.NewArmazemService(armazemRepo), "Principal")
	s.cimento, s.unidade = criarProdutoComPreco(t, ctx, s.produtos, "Cimento 50kg", 620)
	if _, err := s.stock.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: s.cimento.ID, ArmazemDestinoID: s.armazem.ID, Quantidade: 20, CustoUnitario: 480, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	return s
}

func (s servicosCotacao) stockCimento(t *testing.T) float64 {
	t.Helper()
	item, err := s.stock.ObterStock(context.Background(), s.cimento.ID, s.armazem.ID)
	if err != nil {
		t.Fatalf("stock: %v", err)
	}
	return item.Quantidade
}

func TestCotacao_CriarNumeraCongelaPrecoECalculaIVAIncluido(t *testing.T) {
	ctx := context.Background()
	s := novosServicosCotacao(t)
	str := func(v string) *string { return &v }
	cliente, err := s.clientes.Criar(ctx, domain.Cliente{Nome: "Construções Matola", NIF: str("400555666"), Telefone: str("84 111 2222")})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}

	c1, err := s.cotacoes.Criar(ctx, usecase.NovaCotacao{
		ClienteID:    &cliente.ID,
		Itens:        []usecase.ItemNovaCotacao{{ProdutoID: s.cimento.ID, UnidadeProdutoID: s.unidade.ID, Quantidade: 2, Desconto: 40}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	c := c1.Cotacao
	// 2 × 620 = 1240, menos 40 = 1200; IVA 16% incluído = 1200 − 1200/1,16 = 165,52.
	if c.Subtotal != 1240 || c.DescontoTotal != 40 || c.Total != 1200 || c.TaxaIVA != 16 || c.ValorIVA != 165.52 {
		t.Errorf("totais = %+v", c)
	}
	if c.Numero != 1 || c1.Codigo == "" || c.ValidadeDias != usecase.ValidadePorOmissao || c.Estado != domain.EstadoCotacaoAberta {
		t.Errorf("cotação = %+v (código %q)", c, c1.Codigo)
	}
	if c.ClienteNome != "Construções Matola" || c.ClienteNUIT == nil || *c.ClienteNUIT != "400555666" ||
		c.ClienteContacto == nil || *c.ClienteContacto != "84 111 2222" {
		t.Errorf("cliente congelado = %q %v %v", c.ClienteNome, c.ClienteNUIT, c.ClienteContacto)
	}
	if len(c1.Itens) != 1 || c1.Itens[0].PrecoUnitario != 620 || c1.Itens[0].Descricao != "Cimento 50kg" {
		t.Errorf("itens = %+v", c1.Itens)
	}
	if got := s.stockCimento(t); got != 20 {
		t.Errorf("stock = %v, want 20 (a cotação não mexe no stock)", got)
	}

	// Sem cliente registado e isento de IVA: "Consumidor final", número 2.
	_ = s.configs.Definir(ctx, "iva_taxa", "0")
	c2, err := s.cotacoes.Criar(ctx, usecase.NovaCotacao{
		Itens: []usecase.ItemNovaCotacao{{ProdutoID: s.cimento.ID, UnidadeProdutoID: s.unidade.ID, Quantidade: 1}}, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("criar 2: %v", err)
	}
	if c2.Cotacao.Numero != 2 || c2.Cotacao.ClienteNome != usecase.ConsumidorFinal || c2.Cotacao.ValorIVA != 0 {
		t.Errorf("cotação 2 = %+v", c2.Cotacao)
	}

	if _, err := s.cotacoes.Criar(ctx, usecase.NovaCotacao{UtilizadorID: 1}); err == nil {
		t.Error("esperava erro sem itens")
	}
	if _, err := s.cotacoes.Criar(ctx, usecase.NovaCotacao{
		Itens: []usecase.ItemNovaCotacao{{ProdutoID: s.cimento.ID, UnidadeProdutoID: s.unidade.ID, Quantidade: 1, Desconto: 700}}, UtilizadorID: 1,
	}); err == nil {
		t.Error("esperava erro com desconto maior que a linha")
	}
}

func TestCotacao_ConverterAoPrecoCotadoMesmoQueOPrecoSuba(t *testing.T) {
	ctx := context.Background()
	s := novosServicosCotacao(t)
	cc, err := s.cotacoes.Criar(ctx, usecase.NovaCotacao{
		ClienteNome:  "Obra do Sr. Mabunda",
		Itens:        []usecase.ItemNovaCotacao{{ProdutoID: s.cimento.ID, UnidadeProdutoID: s.unidade.ID, Quantidade: 3}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("criar: %v", err)
	}

	// O cimento sobe de 620 para 700 depois da cotação.
	u := s.unidade
	u.PrecoVenda = 700
	if _, err := s.produtos.AtualizarUnidade(ctx, u); err != nil {
		t.Fatalf("subir preço: %v", err)
	}

	venda, err := s.cotacoes.Converter(ctx, usecase.ConverterCotacaoParams{
		CotacaoID: cc.Cotacao.ID, ArmazemID: s.armazem.ID,
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 1860}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("converter: %v", err)
	}
	if venda.Venda.Total != 1860 || venda.Venda.Status != domain.StatusVendaPaga {
		t.Errorf("venda = %+v, want 1860 paga (3 × 620 cotado)", venda.Venda)
	}
	if got := s.stockCimento(t); got != 17 {
		t.Errorf("stock = %v, want 17", got)
	}
	depois, _ := s.cotacoes.Obter(ctx, cc.Cotacao.ID)
	if depois.Cotacao.Estado != domain.EstadoCotacaoConvertida || depois.Cotacao.VendaID == nil || *depois.Cotacao.VendaID != venda.Venda.ID {
		t.Errorf("cotação depois = %+v", depois.Cotacao)
	}

	if _, err := s.cotacoes.Converter(ctx, usecase.ConverterCotacaoParams{
		CotacaoID: cc.Cotacao.ID, ArmazemID: s.armazem.ID,
		Pagamentos: []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 1860}}, UtilizadorID: 1,
	}); !errors.Is(err, usecase.ErrCotacaoNaoAberta) {
		t.Errorf("segunda conversão: err = %v, want ErrCotacaoNaoAberta", err)
	}
}

func TestCotacao_ConverterACreditoECancelar(t *testing.T) {
	ctx := context.Background()
	s := novosServicosCotacao(t)
	cliente, err := s.clientes.Criar(ctx, domain.Cliente{Nome: "Construções Matola"})
	if err != nil {
		t.Fatalf("cliente: %v", err)
	}
	item := []usecase.ItemNovaCotacao{{ProdutoID: s.cimento.ID, UnidadeProdutoID: s.unidade.ID, Quantidade: 1}}

	// Sem cliente registado, não pode ficar nada por pagar.
	anonima, _ := s.cotacoes.Criar(ctx, usecase.NovaCotacao{Itens: item, UtilizadorID: 1})
	if _, err := s.cotacoes.Converter(ctx, usecase.ConverterCotacaoParams{
		CotacaoID: anonima.Cotacao.ID, ArmazemID: s.armazem.ID, UtilizadorID: 1,
	}); err == nil {
		t.Error("esperava erro: sem cliente e sem pagamento")
	}
	if got := s.stockCimento(t); got != 20 {
		t.Errorf("stock = %v, want 20 (a conversão falhada não vende)", got)
	}

	// Com cliente, sem pagamento: venda a crédito, tudo em dívida.
	cc, _ := s.cotacoes.Criar(ctx, usecase.NovaCotacao{ClienteID: &cliente.ID, Itens: item, UtilizadorID: 1})
	venda, err := s.cotacoes.Converter(ctx, usecase.ConverterCotacaoParams{CotacaoID: cc.Cotacao.ID, ArmazemID: s.armazem.ID, UtilizadorID: 1})
	if err != nil {
		t.Fatalf("converter a crédito: %v", err)
	}
	if venda.Venda.Status != domain.StatusVendaDivida {
		t.Errorf("status = %q, want divida", venda.Venda.Status)
	}
	dividas, _ := s.dividas.ListarPorCliente(ctx, cliente.ID)
	if len(dividas) != 1 || dividas[0].ValorEmAberto != 620 {
		t.Errorf("dívidas = %+v, want uma de 620", dividas)
	}

	// Cancelar: só uma vez, e depois já não converte.
	outra, _ := s.cotacoes.Criar(ctx, usecase.NovaCotacao{Itens: item, UtilizadorID: 1})
	if err := s.cotacoes.Cancelar(ctx, outra.Cotacao.ID); err != nil {
		t.Fatalf("cancelar: %v", err)
	}
	if err := s.cotacoes.Cancelar(ctx, outra.Cotacao.ID); !errors.Is(err, usecase.ErrCotacaoNaoAberta) {
		t.Errorf("cancelar outra vez: err = %v", err)
	}
	if _, err := s.cotacoes.Converter(ctx, usecase.ConverterCotacaoParams{
		CotacaoID: outra.Cotacao.ID, ArmazemID: s.armazem.ID,
		Pagamentos: []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 620}}, UtilizadorID: 1,
	}); !errors.Is(err, usecase.ErrCotacaoNaoAberta) {
		t.Errorf("converter cancelada: err = %v", err)
	}
}
