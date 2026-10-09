package admin_test

import (
	"context"
	"testing"
	"time"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestOperadores_PDVExigeTurnoEAtribuiVendaAoOperador(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewUtilizadoresService(repo)

	loja := novaLojaTeste(t, "loja-operadores.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}
	produto, unidades, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Fita isoladora", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "rolo"},
		Unidades: []domain.UnidadeProduto{{NomeUnidade: "rolo", QtdEmUnidadeBase: 1, PrecoCusto: 20, PrecoVenda: 45, IsPadraoVenda: true, IsPadraoCompra: true}},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: produto.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 10, CustoUnitario: 20, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada: %v", err)
	}

	// O PDV é outro processo com a sua própria instância de api.App.
	pdv, fechar, err := admin.AbrirLoja(loja.Caminho)
	if err != nil {
		t.Fatalf("abrir loja: %v", err)
	}
	defer fechar()
	vender := func() (usecase.VendaCompleta, error) {
		return pdv.RegistarVendaDireta(usecase.VendaDiretaParams{
			ArmazemID:  loja.Armazem.ID,
			Itens:      []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidades[0].ID, Quantidade: 1}},
			Pagamentos: []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 45}},
		})
	}

	// Sem operadores, o PDV funciona como antes.
	if v, err := vender(); err != nil || v.Venda.UtilizadorID != domain.UtilizadorAdminSeedID {
		t.Fatalf("sem operadores: venda = %+v, err %v; want atribuída ao administrador", v.Venda, err)
	}

	op, err := svc.Criar(ctx, "Carla Nhantumbo", "2468")
	if err != nil {
		t.Fatalf("criar operador: %v", err)
	}
	if _, err := vender(); err == nil {
		t.Fatal("com operadores e sem turno, a venda devia ser recusada")
	}
	if _, err := pdv.IniciarTurno(op.ID, "1111"); err == nil {
		t.Fatal("PIN errado não pode iniciar o turno")
	}
	if _, err := pdv.IniciarTurno(op.ID, "2468"); err != nil {
		t.Fatalf("iniciar turno: %v", err)
	}
	v, err := vender()
	if err != nil || v.Venda.UtilizadorID != op.ID {
		t.Fatalf("venda no turno = %+v, err %v; want atribuída à Carla (%d)", v.Venda, err, op.ID)
	}

	hoje := time.Now().Format("2006-01-02")
	vendas, err := admin.NewVendasService(repo).Listar(ctx, hoje, hoje)
	if err != nil || len(vendas) != 2 {
		t.Fatalf("vendas = %+v, err %v; want 2", vendas, err)
	}
	nomes := map[string]bool{}
	for _, v := range vendas {
		nomes[v.OperadorNome] = true
	}
	if !nomes["Carla Nhantumbo"] || !nomes["Administrador"] {
		t.Errorf("operadores nas vendas = %v, want Carla e Administrador", nomes)
	}
	porOperador, err := admin.NewRelatoriosService(repo).VendasPorOperador(ctx, hoje, hoje)
	if err != nil || len(porOperador) != 2 || porOperador[0].NumeroVendas != 1 || porOperador[0].Total != 45 {
		t.Fatalf("vendas por operador = %+v, err %v; want 2 linhas de 1 venda de 45", porOperador, err)
	}

	if err := svc.DefinirAtivo(ctx, op.ID, false); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	if atual, _ := pdv.ObterOperadorAtual(); atual.ID != 0 {
		t.Errorf("operador desativado a meio do turno devia terminar o turno, ainda é %+v", atual)
	}
	if _, err := vender(); err != nil {
		t.Errorf("sem operadores ativos o PDV volta a vender sem turno: %v", err)
	}

	if err := svc.RedefinirPIN(ctx, op.ID, "12"); err == nil {
		t.Error("PIN de 2 algarismos devia ser recusado")
	}
	lista, err := svc.Listar(ctx)
	if err != nil || len(lista) != 1 || lista[0].Ativo {
		t.Fatalf("lista = %+v, err %v; want a Carla inativa", lista, err)
	}
}
