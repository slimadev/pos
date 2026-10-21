// Loja de demonstração para as capturas do manual (temporário).
package main

import (
	"context"
	"database/sql"
	"encoding/base64"
	"log"
	"os"
	"time"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/api"
	"gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"

	_ "modernc.org/sqlite"
)

func must(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	ctx := context.Background()
	caminho, err := config.ResolveAdminDBPath()
	must(err)
	adb, err := repository.NewAdminSQLiteConn(caminho)
	must(err)
	repo := repository.NewLojaRepository(adb)
	adm := admin.NewAdminApp(
		admin.NewLojasService(repo), admin.NewDashboardService(repo), admin.NewAuthService(repository.NewAdministradorRepository(adb)),
		admin.NewCategoriasService(repo), admin.NewProdutosService(repo), admin.NewArmazensService(repo), admin.NewStockService(repo),
		admin.NewFarmaciaService(repo), admin.NewInventarioService(repo), admin.NewVendasService(repo), admin.NewRelatoriosService(repo),
		admin.NewClientesService(repo), admin.NewFichasService(repo), admin.NewUtilizadoresService(repo), admin.NewPDVService(repo),
	)
	adm.OnStartup(ctx)
	_, err = adm.ConfigurarPrimeiroArranque(admin.ConfigurarPrimeiroArranqueParams{
		NegocioNome: "Ferragem Cossa", TipoNegocio: domain.TipoNegocioFerragem,
		AdminNome: "Nelson Cossa", AdminUsername: "admin", AdminPassword: "admin123",
	})
	must(err)
	loja, _, err := repo.ObterUnica(ctx)
	must(err)

	app, fechar, err := admin.AbrirLoja(loja.CaminhoDB)
	must(err)
	defer fechar()

	logo, err := os.ReadFile(os.Args[1])
	must(err)
	_, err = app.AtualizarIdentidadeLoja(usecase.DadosLoja{
		Nome: "Ferragem Cossa", Logo: "data:image/png;base64," + base64.StdEncoding.EncodeToString(logo),
		NUIT: "400123456", Endereco: "Av. Eduardo Mondlane 123, Matola", Telefone: "84 123 4567",
		Email: "geral@ferragemcossa.co.mz", TaxaIVA: 16, PapelRecibo: "80",
	})
	must(err)

	cats := map[string]int64{}
	lista, _ := app.ListarCategorias()
	for _, c := range lista {
		cats[c.Nome] = c.ID
	}
	for _, n := range []string{"Fixação", "Tintas"} {
		c, err := app.CriarCategoria(domain.Categoria{Nome: n})
		must(err)
		cats[n] = c.ID
	}

	type un struct {
		nome          string
		qtd           float64
		custo, venda  float64
		codigo        string
		venda1, comp1 bool
	}
	type prod struct {
		nome, cat, base string
		min             float64
		uns             []un
	}
	novos := []prod{
		{"Martelo de unha 500g", "Ferramentas", "unidade", 3, []un{{"unidade", 1, 270, 450, "5600000000011", true, true}}},
		{"Chave de fendas", "Ferramentas", "unidade", 5, []un{{"unidade", 1, 108, 180, "5600000000028", true, true}}},
		{"Alicate universal", "Ferramentas", "unidade", 3, []un{{"unidade", 1, 312, 520, "5600000000035", true, true}}},
		{"Fita métrica 5m", "Ferramentas", "unidade", 2, []un{{"unidade", 1, 144, 240, "5600000000042", true, true}}},
		{"Parafuso 4x40", "Fixação", "unidade", 200, []un{{"unidade", 1, 1.8, 3, "", true, false}, {"caixa 100", 100, 180, 250, "5600000000059", false, true}}},
		{"Bucha 6mm", "Fixação", "unidade", 100, []un{{"unidade", 1, 1.2, 2, "", true, true}}},
		{"Prego 2\"", "Fixação", "kg", 5, []un{{"kg", 1, 96, 160, "", true, true}}},
		{"Tinta branca 5L", "Tintas", "lata", 2, []un{{"lata", 1, 870, 1450, "5600000000066", true, true}}},
		{"Pincel 2\"", "Tintas", "unidade", 5, []un{{"unidade", 1, 57, 95, "", true, true}}},
		{"Rolo de pintura", "Tintas", "unidade", 3, []un{{"unidade", 1, 138, 230, "", true, true}}},
		{"Fio elétrico 2,5mm", "Elétrico", "metro", 50, []un{{"metro", 1, 27, 45, "", true, true}}},
		{"Interruptor simples", "Elétrico", "unidade", 5, []un{{"unidade", 1, 90, 150, "", true, true}}},
		{"Tomada dupla", "Elétrico", "unidade", 5, []un{{"unidade", 1, 126, 210, "", true, true}}},
	}
	for _, p := range novos {
		cat := cats[p.cat]
		min := p.min
		req := api.CriarProdutoRequest{Produto: domain.Produto{Nome: p.nome, CategoriaID: &cat, TipoProduto: domain.TipoProdutoSimples, UnidadeBase: p.base, StockMinimo: &min}}
		for _, u := range p.uns {
			var cod *string
			if u.codigo != "" {
				c := u.codigo
				cod = &c
			}
			req.Unidades = append(req.Unidades, domain.UnidadeProduto{NomeUnidade: u.nome, QtdEmUnidadeBase: u.qtd, PrecoCusto: u.custo, PrecoVenda: u.venda, CodigoBarras: cod, IsPadraoVenda: u.venda1, IsPadraoCompra: u.comp1})
		}
		_, err := app.CriarProduto(req)
		must(err)
	}

	armazens, _ := app.ListarArmazens()
	armazem := armazens[0].ID
	produtos, _ := app.ListarTodosProdutos()
	porNome := map[string]domain.Produto{}
	unidade := map[string]domain.UnidadeProduto{}
	var itensRececao []usecase.ItemRececao
	for _, p := range produtos {
		porNome[p.Nome] = p
		us, _ := app.ListarUnidadesProduto(p.ID)
		unidade[p.Nome] = us[0]
		compra := us[0]
		for _, u := range us {
			if u.IsPadraoCompra {
				compra = u
			}
		}
		qtd := 400.0
		if compra.QtdEmUnidadeBase > 1 {
			qtd = 15
		}
		if p.Nome == "Fio elétrico 2,5mm" || p.Nome == "Cabo 2,5mm²" {
			qtd = 500
		}
		if p.Nome == "Fita métrica 5m" {
			qtd = 3
		}
		itensRececao = append(itensRececao, usecase.ItemRececao{ProdutoID: p.ID, UnidadeProdutoID: compra.ID, Quantidade: qtd, CustoUnitario: compra.PrecoCusto})
	}
	_, err = app.StockService.RegistarRececao(ctx, usecase.RececaoParams{ArmazemID: armazem, Referencia: "Fatura 512 — Distribuidora Matola", Itens: itensRececao, UtilizadorID: api.UtilizadorAdminID})
	must(err)

	str := func(s string) *string { return &s }
	lim := 50000.0
	matola, err := app.CriarCliente(domain.Cliente{Nome: "Construções Matola, Lda", NIF: str("400555666"), Telefone: str("84 111 2222"), Email: str("compras@construcoesmatola.co.mz"), Morada: str("Rua do Trabalho 45, Matola"), LimiteCredito: &lim})
	must(err)
	mabunda, err := app.CriarCliente(domain.Cliente{Nome: "João Mabunda", Telefone: str("82 333 4444")})
	must(err)
	escola, err := app.CriarCliente(domain.Cliente{Nome: "Escola Primária da Liberdade", NIF: str("500777888"), Telefone: str("21 700 100")})
	must(err)

	ana, err := app.OperadorService.Criar(ctx, "Ana Sitoe", "1234")
	must(err)
	carlos, err := app.OperadorService.Criar(ctx, "Carlos Nhantumbo", "5678")
	must(err)

	db, err := sql.Open("sqlite", loja.CaminhoDB)
	must(err)
	defer db.Close()
	recuar := func(vendaID int64, dias int, hora int) {
		t := time.Now().AddDate(0, 0, -dias)
		t = time.Date(t.Year(), t.Month(), t.Day(), hora, 15, 0, 0, time.Local)
		_, err := db.Exec(`UPDATE vendas SET criado_em = ? WHERE id = ?`, t.UTC().Format("2006-01-02T15:04:05.000Z"), vendaID)
		must(err)
	}
	type linha struct {
		nome string
		qtd  float64
	}
	vender := func(cliente *int64, op int64, metodo domain.MetodoPagamento, credito bool, pago float64, ls ...linha) usecase.VendaCompleta {
		p := usecase.VendaDiretaParams{ArmazemID: armazem, ClienteID: cliente, UtilizadorID: op}
		var total float64
		for _, l := range ls {
			u := unidade[l.nome]
			p.Itens = append(p.Itens, usecase.ItemVendaDiretaParams{ProdutoID: porNome[l.nome].ID, UnidadeProdutoID: u.ID, Quantidade: l.qtd})
			total += l.qtd * u.PrecoVenda
		}
		if !credito {
			pago = total
		}
		if pago > 0 {
			p.Pagamentos = []usecase.PagamentoVendaParams{{Metodo: metodo, Valor: pago}}
		}
		var v usecase.VendaCompleta
		if credito {
			v, err = app.VendaService.RegistarVendaACredito(ctx, p)
		} else {
			v, err = app.VendaService.RegistarVendaDireta(ctx, p)
		}
		must(err)
		return v
	}

	// Histórico dos últimos 13 dias (faturação ainda desligada).
	cestos := [][]linha{
		{{"Cimento 50kg", 3}, {"Prego 2\"", 1}},
		{{"Martelo de unha 500g", 1}, {"Chave de fendas", 2}},
		{{"Tinta branca 5L", 1}, {"Pincel 2\"", 2}, {"Rolo de pintura", 1}},
		{{"Fio elétrico 2,5mm", 25}, {"Interruptor simples", 2}, {"Tomada dupla", 2}},
		{{"Parafuso 4x40", 50}, {"Bucha 6mm", 50}},
		{{"Lâmpada LED 9W", 4}},
		{{"Cimento 50kg", 6}},
		{{"Alicate universal", 1}, {"Cabo 2,5mm²", 10}},
	}
	metodos := []domain.MetodoPagamento{domain.MetodoDinheiro, domain.MetodoMpesa, domain.MetodoDinheiro, domain.MetodoEmola, domain.MetodoCartao}
	ops := []int64{ana.ID, carlos.ID}
	n := 0
	for dias := 13; dias >= 1; dias-- {
		for k := 0; k < 1+(dias*7)%4; k++ {
			v := vender(nil, ops[n%2], metodos[n%len(metodos)], false, 0, cestos[n%len(cestos)]...)
			recuar(v.Venda.ID, dias, 9+k*2)
			n++
		}
	}
	// Dívidas: a do Sr. Mabunda tem mais de 30 dias; a da Construções Matola foi paga em parte.
	vm := vender(&mabunda.ID, api.UtilizadorAdminID, domain.MetodoDinheiro, true, 0, linha{"Tinta branca 5L", 1}, linha{"Rolo de pintura", 2})
	recuar(vm.Venda.ID, 40, 11)
	_, err = db.Exec(`UPDATE dividas SET criado_em = ? WHERE venda_id = ?`, time.Now().AddDate(0, 0, -40).UTC().Format("2006-01-02T15:04:05.000Z"), vm.Venda.ID)
	must(err)
	vc := vender(&matola.ID, api.UtilizadorAdminID, domain.MetodoDinheiro, true, 2000, linha{"Cimento 50kg", 20}, linha{"Prego 2\"", 5})
	recuar(vc.Venda.ID, 9, 10)

	// Cotação convertida e cotação aberta.
	c1, err := app.CotacaoService.Criar(ctx, usecase.NovaCotacao{ClienteID: &matola.ID, Itens: []usecase.ItemNovaCotacao{
		{ProdutoID: porNome["Cimento 50kg"].ID, UnidadeProdutoID: unidade["Cimento 50kg"].ID, Quantidade: 10, Desconto: 200},
	}, Observacoes: "Entrega na obra incluída.", UtilizadorID: api.UtilizadorAdminID})
	must(err)
	_, err = app.CotacaoService.Converter(ctx, usecase.ConverterCotacaoParams{CotacaoID: c1.Cotacao.ID, ArmazemID: armazem,
		Pagamentos: []usecase.PagamentoVendaParams{{Metodo: domain.MetodoTransferencia, Valor: c1.Cotacao.Total}}, UtilizadorID: api.UtilizadorAdminID})
	must(err)
	_, err = app.CotacaoService.Criar(ctx, usecase.NovaCotacao{ClienteID: &escola.ID, ValidadeDias: 30, Itens: []usecase.ItemNovaCotacao{
		{ProdutoID: porNome["Tinta branca 5L"].ID, UnidadeProdutoID: unidade["Tinta branca 5L"].ID, Quantidade: 12},
		{ProdutoID: porNome["Rolo de pintura"].ID, UnidadeProdutoID: unidade["Rolo de pintura"].ID, Quantidade: 6},
		{ProdutoID: porNome["Pincel 2\""].ID, UnidadeProdutoID: unidade["Pincel 2\""].ID, Quantidade: 10, Desconto: 50},
	}, Observacoes: "Pintura das salas de aula. Preços válidos por 30 dias.", UtilizadorID: api.UtilizadorAdminID})
	must(err)

	// A partir daqui com documentos fiscais.
	_, err = app.FaturacaoService.DefinirAtiva(ctx, true)
	must(err)
	vender(nil, ana.ID, domain.MetodoDinheiro, false, 0, linha{"Cimento 50kg", 2}, linha{"Prego 2\"", 1})
	vender(nil, carlos.ID, domain.MetodoMpesa, false, 0, linha{"Martelo de unha 500g", 1})
	vender(&escola.ID, ana.ID, domain.MetodoDinheiro, false, 0, linha{"Lâmpada LED 9W", 10}, linha{"Interruptor simples", 4})
	vender(&matola.ID, api.UtilizadorAdminID, domain.MetodoDinheiro, true, 1000, linha{"Fio elétrico 2,5mm", 50}, linha{"Tomada dupla", 6})
	dividas, _ := app.DividaService.ListarPorCliente(ctx, matola.ID)
	for _, d := range dividas {
		if d.VendaID == vc.Venda.ID {
			_, _, err = app.DividaService.RegistarPagamento(ctx, usecase.RegistarPagamentoDividaParams{DividaID: d.ID, Valor: 5000, Metodo: domain.MetodoTransferencia, UtilizadorID: api.UtilizadorAdminID})
			must(err)
		}
	}
	// As dívidas ficam com a data da venda que as originou (recuada acima).
	_, err = db.Exec(`UPDATE dividas SET criado_em = (SELECT criado_em FROM vendas WHERE vendas.id = dividas.venda_id)`)
	must(err)
	log.Printf("loja de demonstração em %s", loja.CaminhoDB)
}
