package usecase_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

type servicosAnulacao struct {
	db         *sql.DB
	produtos   *usecase.ProdutoService
	armazens   *usecase.ArmazemService
	stock      *usecase.StockService
	vendas     *usecase.VendaService
	fichas     *usecase.FichaTecnicaService
	farmacia   *usecase.FarmaciaService
	clientes   *usecase.ClienteService
	dividas    *usecase.DividaService
	relatorios *usecase.RelatorioService
}

func newTestAnulacao(t *testing.T) servicosAnulacao {
	t.Helper()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	clienteRepo := repository.NewClienteRepository(db)
	dividaSvc := usecase.NewDividaService(repository.NewDividaRepository(db), clienteRepo)
	return servicosAnulacao{
		db:         db,
		produtos:   usecase.NewProdutoService(produtoRepo),
		armazens:   usecase.NewArmazemService(armazemRepo),
		stock:      usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), loteRepo),
		vendas:     usecase.NewVendaService(repository.NewVendaRepository(db, stockRepo, loteRepo), produtoRepo, armazemRepo, fichaRepo, clienteRepo),
		fichas:     usecase.NewFichaTecnicaService(fichaRepo, produtoRepo, stockRepo, loteRepo),
		farmacia:   usecase.NewFarmaciaService(loteRepo, produtoRepo, armazemRepo, repository.NewReceitaMedicaRepository(db)),
		clientes:   usecase.NewClienteService(clienteRepo),
		dividas:    dividaSvc,
		relatorios: usecase.NewRelatorioService(repository.NewRelatorioRepository(db), fichaRepo, produtoRepo, clienteRepo, dividaSvc),
	}
}

func (s servicosAnulacao) vender(t *testing.T, ctx context.Context, armazemID int64, u domain.UnidadeProduto, qtd float64) usecase.VendaCompleta {
	t.Helper()
	v, err := s.vendas.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazemID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: u.ProdutoID, UnidadeProdutoID: u.ID, Quantidade: qtd}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: qtd * u.PrecoVenda}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("vender: %v", err)
	}
	return v
}

func (s servicosAnulacao) quantidade(t *testing.T, ctx context.Context, produtoID, armazemID int64) float64 {
	t.Helper()
	st, err := s.stock.ObterStock(ctx, produtoID, armazemID)
	if err != nil {
		t.Fatalf("stock: %v", err)
	}
	return st.Quantidade
}

func TestAnularVenda_RepoeStockESaiDoFecho(t *testing.T) {
	ctx := context.Background()
	s := newTestAnulacao(t)
	armazem := criarArmazem(t, ctx, s.armazens, "Loja")
	produto, unidade := criarProdutoComPreco(t, ctx, s.produtos, "Martelo", 450)
	if _, err := s.stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 300, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	venda := s.vender(t, ctx, armazem.ID, unidade, 3)

	if err := s.vendas.AnularVenda(ctx, venda.Venda.ID, "  ", 1); !errors.Is(err, domain.ErrMotivoObrigatorio) {
		t.Errorf("sem motivo: esperava ErrMotivoObrigatorio, obteve %v", err)
	}
	if err := s.vendas.AnularVenda(ctx, venda.Venda.ID, "registada por engano", 1); err != nil {
		t.Fatalf("anular: %v", err)
	}
	if q := s.quantidade(t, ctx, produto.ID, armazem.ID); q != 10 {
		t.Errorf("stock depois de anular = %v, want 10", q)
	}
	completa, _ := s.vendas.ObterVendaCompleta(ctx, venda.Venda.ID)
	if completa.Venda.Status != domain.StatusVendaCancelada {
		t.Errorf("estado = %v, want cancelada", completa.Venda.Status)
	}
	fecho, _ := s.relatorios.FechoCaixaDiario(ctx, time.Now())
	if fecho.TotalRecebido != 0 {
		t.Errorf("fecho = %v, want 0 (a venda anulada não conta)", fecho.TotalRecebido)
	}
	if err := s.vendas.AnularVenda(ctx, venda.Venda.ID, "outra vez", 1); err == nil {
		t.Error("anular duas vezes devia ser recusado")
	}
	movs, _ := s.stock.ListarMovimentos(ctx, produto.ID)
	if movs[0].Tipo != domain.TipoMovimentoAnulacaoVenda {
		t.Errorf("último movimento = %v, want anulacao_venda", movs[0].Tipo)
	}
}

func TestAnularVenda_DevolveAosLotesDeOndeSairam(t *testing.T) {
	ctx := context.Background()
	s := newTestAnulacao(t)
	armazem := criarArmazem(t, ctx, s.armazens, "Farmácia")
	p, unidade := criarProdutoLoteControlado(t, ctx, s.produtos, "Ibuprofeno", 80)
	entradaLote(t, ctx, s.farmacia, p.ID, armazem.ID, "CEDO", time.Now().AddDate(0, 2, 0), 3)
	entradaLote(t, ctx, s.farmacia, p.ID, armazem.ID, "TARDE", time.Now().AddDate(1, 0, 0), 10)

	venda := s.vender(t, ctx, armazem.ID, unidade, 5) // 3 do CEDO + 2 do TARDE
	if err := s.vendas.AnularVenda(ctx, venda.Venda.ID, "cliente desistiu", 1); err != nil {
		t.Fatalf("anular: %v", err)
	}
	if q := quantidadeDoLote(t, ctx, s.farmacia, p.ID, armazem.ID, "CEDO"); q != 3 {
		t.Errorf("lote CEDO = %v, want 3", q)
	}
	if q := quantidadeDoLote(t, ctx, s.farmacia, p.ID, armazem.ID, "TARDE"); q != 10 {
		t.Errorf("lote TARDE = %v, want 10", q)
	}
	if q := s.quantidade(t, ctx, p.ID, armazem.ID); q != 13 {
		t.Errorf("stock = %v, want 13 (igual à soma dos lotes)", q)
	}
}

func TestAnularVenda_CompostoRepoeIngredientes(t *testing.T) {
	ctx := context.Background()
	s := newTestAnulacao(t)
	armazem := criarArmazem(t, ctx, s.armazens, "Bar")
	rum := criarIngrediente(t, ctx, s.produtos, "Rum", "L", 900)
	if _, err := s.stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: rum.ID, ArmazemDestinoID: armazem.ID, Quantidade: 2, CustoUnitario: 900, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	mojito, unidade := criarComposto(t, ctx, s.produtos, "Mojito", 300)
	if _, err := s.fichas.DefinirReceita(ctx, mojito.ID, []usecase.IngredienteInput{{ProdutoIngredienteID: rum.ID, Quantidade: 0.05}}); err != nil {
		t.Fatalf("receita: %v", err)
	}
	venda := s.vender(t, ctx, armazem.ID, unidade, 4)
	if err := s.vendas.AnularVenda(ctx, venda.Venda.ID, "engano", 1); err != nil {
		t.Fatalf("anular: %v", err)
	}
	if q := s.quantidade(t, ctx, rum.ID, armazem.ID); q < 1.9999 || q > 2.0001 {
		t.Errorf("rum = %v, want 2", q)
	}
}

func TestAnularVenda_ACreditoApagaDividaSemPagamentos(t *testing.T) {
	ctx := context.Background()
	s := newTestAnulacao(t)
	armazem := criarArmazem(t, ctx, s.armazens, "Loja")
	produto, unidade := criarProdutoComPreco(t, ctx, s.produtos, "Tinta", 1000)
	if _, err := s.stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 5, CustoUnitario: 600, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	cliente, _ := s.clientes.Criar(ctx, domain.Cliente{Nome: "Obra Matola"})
	credito := func() usecase.VendaCompleta {
		v, err := s.vendas.RegistarVendaACredito(ctx, usecase.VendaDiretaParams{
			ArmazemID: armazem.ID, ClienteID: &cliente.ID, UtilizadorID: 1,
			Itens: []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		})
		if err != nil {
			t.Fatalf("venda a crédito: %v", err)
		}
		return v
	}

	sem := credito()
	if err := s.vendas.AnularVenda(ctx, sem.Venda.ID, "engano", 1); err != nil {
		t.Fatalf("anular venda a crédito sem pagamentos: %v", err)
	}
	if ds, _ := s.dividas.ListarPorCliente(ctx, cliente.ID); len(ds) != 0 {
		t.Fatalf("dívidas = %+v, want nenhuma depois de anular", ds)
	}

	com := credito()
	ds, _ := s.dividas.ListarPorCliente(ctx, cliente.ID)
	if _, _, err := s.dividas.RegistarPagamento(ctx, usecase.RegistarPagamentoDividaParams{DividaID: ds[0].ID, Valor: 100, Metodo: domain.MetodoDinheiro, UtilizadorID: 1}); err != nil {
		t.Fatalf("pagar: %v", err)
	}
	if err := s.vendas.AnularVenda(ctx, com.Venda.ID, "engano", 1); err == nil {
		t.Fatal("venda cuja dívida já recebeu pagamentos não pode ser anulada")
	}
	if q := s.quantidade(t, ctx, produto.ID, armazem.ID); q != 4 {
		t.Errorf("stock = %v, want 4 (a primeira voltou, a segunda continua vendida)", q)
	}
}

func TestAnularVenda_SoNoProprioDia(t *testing.T) {
	ctx := context.Background()
	s := newTestAnulacao(t)
	armazem := criarArmazem(t, ctx, s.armazens, "Loja")
	produto, unidade := criarProdutoComPreco(t, ctx, s.produtos, "Pincel", 95)
	if _, err := s.stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 5, CustoUnitario: 50, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	venda := s.vender(t, ctx, armazem.ID, unidade, 1)
	ontem := time.Now().AddDate(0, 0, -1).UTC().Format("2006-01-02T15:04:05.000Z")
	if _, err := s.db.Exec("UPDATE vendas SET criado_em = ? WHERE id = ?", ontem, venda.Venda.ID); err != nil {
		t.Fatalf("mudar data: %v", err)
	}
	if err := s.vendas.AnularVenda(ctx, venda.Venda.ID, "engano", 1); err == nil {
		t.Fatal("venda de ontem não pode ser anulada")
	}
	if q := s.quantidade(t, ctx, produto.ID, armazem.ID); q != 4 {
		t.Errorf("stock = %v, want 4 (nada reposto)", q)
	}
}
