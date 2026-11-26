package usecase_test

import (
	"context"
	"errors"
	"math"
	"strings"
	"sync"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// impressorFalso guarda o que cada impressora recebeu; as de falhar recusam.
type impressorFalso struct {
	mu       sync.Mutex
	recebido map[string][]string
	falhar   map[string]bool
}

func (f *impressorFalso) Enviar(_ context.Context, endereco string, _ int, dados []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.falhar[endereco] {
		return errors.New("sem resposta")
	}
	f.recebido[endereco] = append(f.recebido[endereco], string(dados))
	return nil
}

func (f *impressorFalso) talões(endereco string) []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string{}, f.recebido[endereco]...)
}

type cenarioRestaurante struct {
	rest      *usecase.RestauranteService
	caixa     *usecase.CaixaService
	operador  *usecase.OperadorService
	impressor *impressorFalso
	armazem   domain.Armazem
	lima      usecase.Operador
	ana       usecase.Operador
	gerente   usecase.Operador
	heineken  domain.UnidadeProduto
	feijoada  domain.UnidadeProduto
	balcao    domain.Impressora
	cozinha   domain.Impressora
}

func novoCenarioRestaurante(t *testing.T) *cenarioRestaurante {
	t.Helper()
	ctx := context.Background()
	db := newTestDB(t)

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)
	clienteRepo := repository.NewClienteRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	vendaRepo := repository.NewVendaRepository(db, stockRepo, loteRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	dividaRepo := repository.NewDividaRepository(db)
	utilRepo := repository.NewUtilizadorRepository(db)
	restRepo := repository.NewRestauranteRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo, loteRepo)
	dividaSvc := usecase.NewDividaService(dividaRepo, clienteRepo)
	contaSvc := usecase.NewContaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, dividaSvc)
	operadorSvc := usecase.NewOperadorService(utilRepo)
	categoriaSvc := usecase.NewCategoriaService(repository.NewCategoriaRepository(db))
	relatorioSvc := usecase.NewRelatorioService(repository.NewRelatorioRepository(db), fichaRepo, produtoRepo, clienteRepo, dividaSvc)

	imp := &impressorFalso{recebido: map[string][]string{}, falhar: map[string]bool{}}
	c := &cenarioRestaurante{
		rest: usecase.NewRestauranteService(restRepo, vendaRepo, contaSvc, operadorSvc, utilRepo,
			repository.NewNegocioRepository(db), repository.NewConfigRepository(db), imp),
		caixa:     usecase.NewCaixaService(restRepo, relatorioSvc, utilRepo),
		operador:  operadorSvc,
		impressor: imp,
	}

	var err error
	if c.lima, err = operadorSvc.Criar(ctx, "Lima", "1111"); err != nil {
		t.Fatal(err)
	}
	if c.ana, err = operadorSvc.Criar(ctx, "Ana", "2222"); err != nil {
		t.Fatal(err)
	}
	if c.gerente, err = operadorSvc.Criar(ctx, "Gerente", "9999"); err != nil {
		t.Fatal(err)
	}
	if err := operadorSvc.DefinirGerente(ctx, c.gerente.ID, true); err != nil {
		t.Fatal(err)
	}

	bebidas, err := categoriaSvc.Criar(ctx, domain.Categoria{NegocioID: domain.NegocioID, Nome: "Bebidas"})
	if err != nil {
		t.Fatal(err)
	}
	pratos, err := categoriaSvc.Criar(ctx, domain.Categoria{NegocioID: domain.NegocioID, Nome: "Pratos"})
	if err != nil {
		t.Fatal(err)
	}
	cerveja, hu := criarProdutoComPreco(t, ctx, produtoSvc, "Heineken", 150)
	prato, fu := criarProdutoComPreco(t, ctx, produtoSvc, "Feijoada", 600)
	cerveja.CategoriaID = &bebidas.ID
	prato.CategoriaID = &pratos.ID
	for _, p := range []domain.Produto{cerveja, prato} {
		if _, err := produtoSvc.Atualizar(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	c.heineken, c.feijoada = hu, fu

	c.armazem = criarArmazem(t, ctx, armazemSvc, "Bar")
	for _, p := range []domain.Produto{cerveja, prato} {
		if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
			ProdutoID: p.ID, ArmazemDestinoID: c.armazem.ID, Quantidade: 100, CustoUnitario: 50, UtilizadorID: 1,
		}); err != nil {
			t.Fatal(err)
		}
	}

	if c.balcao, err = c.rest.GuardarImpressora(ctx, domain.Impressora{Nome: "Balcão", Endereco: "10.0.0.10", Funcao: domain.ImpressoraBalcao}); err != nil {
		t.Fatal(err)
	}
	if c.cozinha, err = c.rest.GuardarImpressora(ctx, domain.Impressora{Nome: "Cozinha", Endereco: "10.0.0.20", Colunas: 32, Funcao: domain.ImpressoraPreparacao}); err != nil {
		t.Fatal(err)
	}
	if err := c.rest.DefinirRotaCategoria(ctx, pratos.ID, &c.cozinha.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.rest.DefinirRotaCategoria(ctx, bebidas.ID, &c.balcao.ID); err == nil {
		t.Fatal("o balcão não devia aceitar rota de categoria")
	}
	return c
}

func (c *cenarioRestaurante) lancar(t *testing.T, contaID int64, u domain.UnidadeProduto, q float64, garcom int64) {
	t.Helper()
	if _, err := c.rest.LancarItem(context.Background(), usecase.AdicionarItemContaParams{
		ContaID: contaID, ProdutoID: u.ProdutoID, UnidadeProdutoID: u.ID, Quantidade: q, UtilizadorID: garcom,
	}); err != nil {
		t.Fatalf("lançar: %v", err)
	}
}

func itemDe(t *testing.T, c *cenarioRestaurante, contaID, produtoID int64) domain.ItemMesa {
	t.Helper()
	d, err := c.rest.ObterMesa(context.Background(), contaID)
	if err != nil {
		t.Fatal(err)
	}
	for _, i := range d.Itens {
		if i.ProdutoID == produtoID {
			return i
		}
	}
	t.Fatalf("produto %d não está na mesa", produtoID)
	return domain.ItemMesa{}
}

func TestEnviarPedido_BalcaoRecebeTudoCozinhaSoOsPratos(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)

	mesa, err := c.rest.AbrirMesa(ctx, "5", "", c.lima.ID, 3)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.rest.AbrirMesa(ctx, " 5 ", "", c.ana.ID, 1); err == nil {
		t.Fatal("a mesa 5 já estava aberta")
	}
	c.lancar(t, mesa.ContaID, c.heineken, 2, c.lima.ID)
	c.lancar(t, mesa.ContaID, c.feijoada, 1, c.lima.ID)

	res, err := c.rest.EnviarPedido(ctx, mesa.ContaID, c.lima.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Impressoes) != 2 {
		t.Fatalf("impressões = %+v, queria balcão e cozinha", res.Impressoes)
	}
	balcao := c.impressor.talões("10.0.0.10")
	cozinha := c.impressor.talões("10.0.0.20")
	if len(balcao) != 1 || !strings.Contains(balcao[0], "2 x Heineken") || !strings.Contains(balcao[0], "1 x Feijoada") {
		t.Errorf("balcão recebeu %q", balcao)
	}
	if !strings.Contains(balcao[0], "pediu") || !strings.Contains(balcao[0], "Lima") || !strings.Contains(balcao[0], "MESA 5") {
		t.Errorf("falta o garçom/mesa no talão do balcão: %q", balcao[0])
	}
	if len(cozinha) != 1 || strings.Contains(cozinha[0], "Heineken") || !strings.Contains(cozinha[0], "1 x Feijoada") {
		t.Errorf("cozinha recebeu %q", cozinha)
	}
	if !strings.Contains(res.Texto, "2 x Heineken") {
		t.Errorf("texto para o aparelho: %q", res.Texto)
	}

	if _, err := c.rest.EnviarPedido(ctx, mesa.ContaID, c.lima.ID); !errors.Is(err, usecase.ErrNadaPorEnviar) {
		t.Fatalf("segundo envio sem novidades: %v", err)
	}

	// Mais uma Heineken soma-se à linha — mas só a nova sai, e só no balcão.
	c.lancar(t, mesa.ContaID, c.heineken, 1, c.lima.ID)
	if item := itemDe(t, c, mesa.ContaID, c.heineken.ProdutoID); item.Quantidade != 3 || item.Enviada != 2 {
		t.Fatalf("linha da Heineken = %+v", item)
	}
	if _, err := c.rest.EnviarPedido(ctx, mesa.ContaID, c.lima.ID); err != nil {
		t.Fatal(err)
	}
	balcao = c.impressor.talões("10.0.0.10")
	if len(balcao) != 2 || !strings.Contains(balcao[1], "1 x Heineken") || strings.Contains(balcao[1], "Feijoada") {
		t.Errorf("segundo pedido no balcão: %q", balcao)
	}
	if n := len(c.impressor.talões("10.0.0.20")); n != 1 {
		t.Errorf("a cozinha não devia receber bebidas (%d talões)", n)
	}

	pedidos, err := c.rest.ListarPedidos(ctx, mesa.ContaID)
	if err != nil || len(pedidos) != 2 {
		t.Fatalf("pedidos = %v, %v", pedidos, err)
	}
	if _, err := c.rest.ReimprimirPedido(ctx, pedidos[0].ID, &c.cozinha.ID); err != nil {
		t.Fatal(err)
	}
	if got := c.impressor.talões("10.0.0.20"); len(got) != 2 || !strings.Contains(got[1], "VIA") {
		t.Errorf("reimpressão na cozinha: %q", got)
	}
}

func TestEnviarPedido_ImpressoraEmBaixoNaoPerdeOPedido(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)
	c.impressor.falhar["10.0.0.20"] = true

	mesa, _ := c.rest.AbrirMesa(ctx, "Varanda", "", c.lima.ID, 1)
	c.lancar(t, mesa.ContaID, c.feijoada, 2, c.lima.ID)
	res, err := c.rest.EnviarPedido(ctx, mesa.ContaID, c.lima.ID)
	if err != nil {
		t.Fatal(err)
	}
	var erroCozinha string
	for _, i := range res.Impressoes {
		if i.ImpressoraID == c.cozinha.ID {
			erroCozinha = i.Erro
		}
	}
	if erroCozinha == "" {
		t.Fatalf("devia dizer que a cozinha falhou: %+v", res.Impressoes)
	}
	if item := itemDe(t, c, mesa.ContaID, c.feijoada.ProdutoID); item.Enviada != 2 {
		t.Errorf("o pedido ficou registado como enviado? %+v", item)
	}
}

func TestAnularItem_EnviadoPedeGerente(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)

	mesa, _ := c.rest.AbrirMesa(ctx, "2", "", c.lima.ID, 2)
	c.lancar(t, mesa.ContaID, c.feijoada, 2, c.lima.ID)
	if _, err := c.rest.EnviarPedido(ctx, mesa.ContaID, c.lima.ID); err != nil {
		t.Fatal(err)
	}
	c.lancar(t, mesa.ContaID, c.feijoada, 1, c.lima.ID) // 3, 2 enviadas
	item := itemDe(t, c, mesa.ContaID, c.feijoada.ProdutoID)

	// A que ainda não foi enviada sai sem autorização e sem talão.
	res, err := c.rest.AnularItem(ctx, usecase.AnularItemParams{ItemID: item.ID, Quantidade: 1, GarcomID: c.lima.ID})
	if err != nil || res.PrecisaAutorizacao || res.Anulacao != nil {
		t.Fatalf("anular a não enviada: %+v, %v", res, err)
	}
	if i := itemDe(t, c, mesa.ContaID, c.feijoada.ProdutoID); i.Quantidade != 2 || i.Enviada != 2 {
		t.Fatalf("depois de tirar a não enviada: %+v", i)
	}

	// Já enviada: pede gerente; nada muda até lá.
	res, err = c.rest.AnularItem(ctx, usecase.AnularItemParams{ItemID: item.ID, Quantidade: 1, GarcomID: c.lima.ID})
	if err != nil || !res.PrecisaAutorizacao {
		t.Fatalf("devia pedir autorização: %+v, %v", res, err)
	}
	if _, err := c.rest.AnularItem(ctx, usecase.AnularItemParams{ItemID: item.ID, Quantidade: 1, GarcomID: c.lima.ID,
		PINGerente: "1111", Motivo: "Cliente desistiu"}); !errors.Is(err, usecase.ErrSemAutorizacao) {
		t.Fatalf("PIN de garçom não autoriza: %v", err)
	}
	res, err = c.rest.AnularItem(ctx, usecase.AnularItemParams{ItemID: item.ID, Quantidade: 1, GarcomID: c.lima.ID,
		PINGerente: "9999", Motivo: "Cliente desistiu"})
	if err != nil || res.Anulacao == nil {
		t.Fatalf("anular com gerente: %+v, %v", res, err)
	}
	if i := itemDe(t, c, mesa.ContaID, c.feijoada.ProdutoID); i.Quantidade != 1 || i.Enviada != 1 {
		t.Fatalf("depois da anulação: %+v", i)
	}
	cozinha := c.impressor.talões("10.0.0.20")
	ultimo := cozinha[len(cozinha)-1]
	if !strings.Contains(ultimo, "ANULADO") || !strings.Contains(ultimo, "1 x Feijoada") || !strings.Contains(ultimo, "Gerente") {
		t.Errorf("talão de anulação na cozinha: %q", ultimo)
	}

	// Anular a última tira a linha.
	if _, err := c.rest.AnularItem(ctx, usecase.AnularItemParams{ItemID: item.ID, Quantidade: 1, GarcomID: c.lima.ID,
		PINGerente: "9999", Motivo: "Engano"}); err != nil {
		t.Fatal(err)
	}
	d, _ := c.rest.ObterMesa(ctx, mesa.ContaID)
	if len(d.Itens) != 0 {
		t.Errorf("a linha devia ter saído: %+v", d.Itens)
	}
}

func TestDividirConta_PassaQuantidadesEMantemEnviados(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)

	mesa, _ := c.rest.AbrirMesa(ctx, "7", "", c.lima.ID, 4)
	c.lancar(t, mesa.ContaID, c.heineken, 3, c.lima.ID)
	if _, err := c.rest.EnviarPedido(ctx, mesa.ContaID, c.lima.ID); err != nil {
		t.Fatal(err)
	}
	c.lancar(t, mesa.ContaID, c.heineken, 1, c.lima.ID) // 4, 3 enviadas
	c.lancar(t, mesa.ContaID, c.feijoada, 1, c.lima.ID)
	h := itemDe(t, c, mesa.ContaID, c.heineken.ProdutoID)
	f := itemDe(t, c, mesa.ContaID, c.feijoada.ProdutoID)

	if _, err := c.rest.DividirConta(ctx, mesa.ContaID, []usecase.ParteDivisao{{ItemID: h.ID, Quantidade: 4}, {ItemID: f.ID, Quantidade: 1}}, ""); err == nil {
		t.Fatal("passar tudo não é dividir")
	}
	nova, err := c.rest.DividirConta(ctx, mesa.ContaID, []usecase.ParteDivisao{{ItemID: h.ID, Quantidade: 2}, {ItemID: f.ID, Quantidade: 1}}, "")
	if err != nil {
		t.Fatal(err)
	}
	if nova.Mesa != "7 (2)" || nova.GarcomID != c.lima.ID {
		t.Errorf("nova conta = %+v", nova)
	}
	if nova.Total != 2*150+600 {
		t.Errorf("total da nova = %v", nova.Total)
	}
	if i := itemDe(t, c, nova.ContaID, c.heineken.ProdutoID); i.Quantidade != 2 || i.Enviada != 2 {
		t.Errorf("Heineken na nova: %+v", i)
	}
	if i := itemDe(t, c, mesa.ContaID, c.heineken.ProdutoID); i.Quantidade != 2 || i.Enviada != 1 {
		t.Errorf("Heineken na original: %+v", i)
	}
	orig, _ := c.rest.ObterMesa(ctx, mesa.ContaID)
	if orig.Mesa.Total != 300 || orig.Mesa.PorEnviar != 1 {
		t.Errorf("original = %+v", orig.Mesa)
	}
	// A feijoada (não enviada) continua por enviar, agora na nova conta.
	if nova.PorEnviar != 1 {
		t.Errorf("por enviar na nova = %v", nova.PorEnviar)
	}
}

func TestImprimirConta_ValorPorPessoa(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)

	mesa, _ := c.rest.AbrirMesa(ctx, "3", "", c.lima.ID, 1)
	c.lancar(t, mesa.ContaID, c.heineken, 2, c.lima.ID)
	c.lancar(t, mesa.ContaID, c.feijoada, 1, c.lima.ID)

	conta, err := c.rest.ImprimirConta(ctx, mesa.ContaID, 3, &c.balcao.ID)
	if err != nil {
		t.Fatal(err)
	}
	if conta.Total != 900 || conta.Pessoas != 3 || math.Abs(conta.PorPessoa-300) > 1e-9 {
		t.Fatalf("conta = %+v", conta)
	}
	if !strings.Contains(conta.Texto, "3 pessoas") || !strings.Contains(conta.Texto, "300,00") || !strings.Contains(conta.Texto, "900,00") {
		t.Errorf("texto da conta:\n%s", conta.Texto)
	}
	if conta.Impressao == nil || conta.Impressao.Erro != "" || len(c.impressor.talões("10.0.0.10")) != 1 {
		t.Errorf("devia ter impresso no balcão: %+v", conta.Impressao)
	}
	for _, l := range strings.Split(conta.Texto, "\n") {
		if len([]rune(l)) > 32 {
			t.Errorf("linha maior que o papel do aparelho: %q", l)
		}
	}
	m, _ := c.rest.ObterMesa(ctx, mesa.ContaID)
	if m.Mesa.Pessoas != 3 {
		t.Errorf("pessoas não ficou guardado: %v", m.Mesa.Pessoas)
	}
}

func TestTransferirEJuntarMesas(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)

	m1, _ := c.rest.AbrirMesa(ctx, "1", "", c.lima.ID, 2)
	m2, _ := c.rest.AbrirMesa(ctx, "2", "", c.lima.ID, 2)
	c.lancar(t, m1.ContaID, c.heineken, 2, c.lima.ID)
	c.lancar(t, m2.ContaID, c.feijoada, 1, c.lima.ID)

	if err := c.rest.TransferirMesa(ctx, m1.ContaID, c.ana.ID); err != nil {
		t.Fatal(err)
	}
	daAna, _ := c.rest.ListarMesas(ctx, &c.ana.ID)
	if len(daAna) != 1 || daAna[0].ContaID != m1.ContaID || daAna[0].GarcomNome != "Ana" {
		t.Fatalf("mesas da Ana = %+v", daAna)
	}

	if err := c.rest.JuntarMesas(ctx, m2.ContaID, m1.ContaID, c.lima.ID); err != nil {
		t.Fatal(err)
	}
	todas, _ := c.rest.ListarMesas(ctx, nil)
	if len(todas) != 1 || todas[0].Total != 900 || todas[0].Pessoas != 4 {
		t.Fatalf("depois de juntar: %+v", todas)
	}
}

func TestCaixa_FecharMesaPedeCaixaEFechoConfere(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)

	mesa, _ := c.rest.AbrirMesa(ctx, "9", "", c.lima.ID, 3)
	c.lancar(t, mesa.ContaID, c.heineken, 2, c.lima.ID)
	c.lancar(t, mesa.ContaID, c.feijoada, 1, c.lima.ID)
	fechar := usecase.FecharContaParams{
		ArmazemID: c.armazem.ID, UtilizadorID: c.ana.ID,
		Pagamentos: []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 600}, {Metodo: domain.MetodoMpesa, Valor: 300}},
	}
	if _, _, err := c.rest.FecharMesa(ctx, mesa.ContaID, fechar); !errors.Is(err, usecase.ErrCaixaFechado) {
		t.Fatalf("sem caixa aberto: %v", err)
	}

	if _, err := c.caixa.Abrir(ctx, c.ana.ID, 1000); err != nil {
		t.Fatal(err)
	}
	if _, err := c.caixa.Abrir(ctx, c.ana.ID, 1000); !errors.Is(err, usecase.ErrCaixaJaAberto) {
		t.Fatalf("segundo caixa: %v", err)
	}
	if _, _, err := c.rest.FecharMesa(ctx, mesa.ContaID, fechar); err != nil {
		t.Fatal(err)
	}
	if abertas, _ := c.rest.ListarMesas(ctx, nil); len(abertas) != 0 {
		t.Fatalf("a mesa devia ter fechado: %+v", abertas)
	}

	est, err := c.caixa.Atual(ctx)
	if err != nil || !est.Aberto || est.DinheiroEsperado != 1600 || est.Recebido.TotalRecebido != 900 {
		t.Fatalf("estado do caixa = %+v, %v", est, err)
	}

	fecho, err := c.caixa.Fechar(ctx, c.gerente.ID, map[domain.MetodoPagamento]float64{
		domain.MetodoDinheiro: 1550, domain.MetodoMpesa: 300,
	}, "faltam 50")
	if err != nil {
		t.Fatal(err)
	}
	if len(fecho.Linhas) != 2 || fecho.Linhas[0].Metodo != domain.MetodoDinheiro {
		t.Fatalf("linhas = %+v", fecho.Linhas)
	}
	if l := fecho.Linhas[0]; l.Esperado != 1600 || l.Contado != 1550 || l.Diferenca != -50 {
		t.Errorf("dinheiro = %+v", l)
	}
	if l := fecho.Linhas[1]; l.Esperado != 300 || l.Diferenca != 0 {
		t.Errorf("mpesa = %+v", l)
	}
	if fecho.DiferencaTotal != -50 || fecho.FechadoPor != "Gerente" {
		t.Errorf("fecho = %+v", fecho)
	}
	if est, _ := c.caixa.Atual(ctx); est.Aberto {
		t.Error("o caixa devia estar fechado")
	}
	again, err := c.caixa.ObterFecho(ctx, fecho.Sessao.ID)
	if err != nil || again.DiferencaTotal != -50 {
		t.Errorf("reabrir o fecho: %+v, %v", again, err)
	}
}

func TestOperador_GerenteListadoEAutoriza(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)
	todos, err := c.operador.Listar(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var gerentes int
	for _, o := range todos {
		if o.Gerente {
			gerentes++
		}
	}
	if len(todos) != 3 || gerentes != 1 {
		t.Fatalf("operadores = %+v", todos)
	}
	if g, err := c.operador.AutorizarGerente(ctx, "9999"); err != nil || g.ID != c.gerente.ID {
		t.Fatalf("autorizar: %+v, %v", g, err)
	}
	if _, err := c.operador.Autenticar(ctx, c.gerente.ID, "9999"); err != nil {
		t.Fatalf("o gerente também entra no PDV: %v", err)
	}
}

func TestNovaConta_MesmaMesaContasSeparadasComEtiqueta(t *testing.T) {
	ctx := context.Background()
	c := novoCenarioRestaurante(t)

	a, err := c.rest.AbrirMesa(ctx, "8", "  camisa   azul ", c.lima.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if a.Etiqueta != "camisa azul" {
		t.Errorf("etiqueta = %q", a.Etiqueta)
	}
	b, err := c.rest.NovaConta(ctx, a.ContaID, "aniversariante", c.ana.ID)
	if err != nil {
		t.Fatal(err)
	}
	if b.Mesa != "8 (2)" || b.Etiqueta != "aniversariante" || b.GarcomID != c.ana.ID {
		t.Errorf("nova conta = %+v", b)
	}
	// A partir da "8 (2)" a seguinte é "8 (3)", não "8 (2) (2)".
	d, err := c.rest.NovaConta(ctx, b.ContaID, "", c.lima.ID)
	if err != nil {
		t.Fatal(err)
	}
	if d.Mesa != "8 (3)" {
		t.Errorf("terceira conta = %q", d.Mesa)
	}
	if _, err := c.rest.NovaConta(ctx, a.ContaID, strings.Repeat("x", usecase.MaxEtiqueta+1), c.lima.ID); err == nil {
		t.Error("aceitou etiqueta comprida")
	}

	c.lancar(t, b.ContaID, c.feijoada, 1, c.ana.ID)
	if _, err := c.rest.EnviarPedido(ctx, b.ContaID, c.ana.ID); err != nil {
		t.Fatal(err)
	}
	cozinha := c.impressor.talões("10.0.0.20")
	if len(cozinha) != 1 || !strings.Contains(cozinha[0], "MESA 8 (2)") || !strings.Contains(cozinha[0], "ANIVERSARIANTE") {
		t.Errorf("cozinha recebeu %q", cozinha)
	}

	if err := c.rest.DefinirEtiqueta(ctx, d.ContaID, "senhora de óculos"); err != nil {
		t.Fatal(err)
	}
	c.lancar(t, d.ContaID, c.heineken, 2, c.lima.ID)
	conta, err := c.rest.ImprimirConta(ctx, d.ContaID, 0, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(conta.Texto, "MESA 8 (3)") || !strings.Contains(conta.Texto, "senhora de óculos") {
		t.Errorf("conta impressa:\n%s", conta.Texto)
	}
}
