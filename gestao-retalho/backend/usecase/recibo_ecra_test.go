package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func TestReciboService_ObterParaEcraTrazDadosPapelELogo(t *testing.T) {
	ctx := context.Background()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	clienteRepo := repository.NewClienteRepository(db)
	configRepo := repository.NewConfigRepository(db)
	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), loteRepo)
	vendaSvc := usecase.NewVendaService(repository.NewVendaRepository(db, stockRepo, loteRepo), produtoRepo, armazemRepo,
		repository.NewFichaTecnicaRepository(db), clienteRepo)
	recibos := usecase.NewReciboService(vendaSvc, produtoRepo, clienteRepo, repository.NewNegocioRepository(db), configRepo)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Cimento 50kg", 620)
	armazem := criarArmazem(t, ctx, armazemSvc, "Balcão")
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: armazem.ID, Quantidade: 10, CustoUnitario: 480, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada: %v", err)
	}
	venda, err := vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    armazem.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 2}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoMpesa, Valor: 1240}},
		UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("venda: %v", err)
	}

	r, err := recibos.ObterParaEcra(ctx, venda.Venda.ID)
	if err != nil {
		t.Fatalf("obter recibo: %v", err)
	}
	if r.Papel != "80" || r.Logo != "" {
		t.Errorf("papel/logo = %q/%q, want 80 por omissão e sem logótipo", r.Papel, r.Logo)
	}
	d := r
	if d.VendaID != venda.Venda.ID || d.Total != 1240 || len(d.Itens) != 1 || d.Itens[0].NomeProduto != "Cimento 50kg" {
		t.Errorf("dados = %+v", d)
	}
	if len(d.Pagamentos) != 1 || d.Pagamentos[0].Metodo != "M-Pesa" {
		t.Errorf("pagamentos = %+v, want M-Pesa", d.Pagamentos)
	}

	if err := configRepo.Definir(ctx, "recibo_papel", "58"); err != nil {
		t.Fatal(err)
	}
	if r, _ := recibos.ObterParaEcra(ctx, venda.Venda.ID); r.Papel != "58" {
		t.Errorf("papel = %q, want 58", r.Papel)
	}
}
