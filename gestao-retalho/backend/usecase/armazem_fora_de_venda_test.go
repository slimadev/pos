package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestRegistarVendaDireta_RecusaArmazemForaDeVenda(t *testing.T) {
	ctx := context.Background()
	produtoSvc, armazemSvc, stockSvc, vendaSvc := newTestVendaServices(t)

	produto, unidade := criarProdutoComPreco(t, ctx, produtoSvc, "Cimento 50kg", 620)
	criarArmazem(t, ctx, armazemSvc, "Balcão")
	deposito, err := armazemSvc.Criar(ctx, domain.Armazem{Nome: "Depósito", Tipo: domain.TipoArmazemDeposito, ForaDeVenda: true})
	if err != nil {
		t.Fatalf("criar depósito: %v", err)
	}
	if !deposito.ForaDeVenda {
		t.Fatal("o depósito devia ter ficado fora de venda")
	}
	if _, err := stockSvc.RegistarEntrada(ctx, usecase.EntradaParams{
		ProdutoID: produto.ID, ArmazemDestinoID: deposito.ID, Quantidade: 20, CustoUnitario: 480, UtilizadorID: 1,
	}); err != nil {
		t.Fatalf("entrada no depósito: %v", err)
	}

	_, err = vendaSvc.RegistarVendaDireta(ctx, usecase.VendaDiretaParams{
		ArmazemID:    deposito.ID,
		Itens:        []usecase.ItemVendaDiretaParams{{ProdutoID: produto.ID, UnidadeProdutoID: unidade.ID, Quantidade: 1}},
		Pagamentos:   []usecase.PagamentoVendaParams{{Metodo: domain.MetodoDinheiro, Valor: 620}},
		UtilizadorID: 1,
	})
	if !errors.Is(err, domain.ErrArmazemForaDeVenda) {
		t.Fatalf("err = %v, want ErrArmazemForaDeVenda", err)
	}
	stock, _ := stockSvc.ObterStock(ctx, produto.ID, deposito.ID)
	if stock.Quantidade != 20 {
		t.Errorf("stock no depósito = %v, want 20 (a venda não podia ter saído)", stock.Quantidade)
	}
}

func TestArmazemService_NaoDeixaFicarSemArmazemParaVenda(t *testing.T) {
	ctx := context.Background()
	_, armazemSvc, _ := newTestServices(t)

	principal := criarArmazem(t, ctx, armazemSvc, "Principal")
	balcao := criarArmazem(t, ctx, armazemSvc, "Balcão")

	// Com outro disponível, pode tirar-se este da venda.
	balcao.ForaDeVenda = true
	if err := armazemSvc.Atualizar(ctx, balcao); err != nil {
		t.Fatalf("tirar o balcão da venda: %v", err)
	}
	// O principal é agora o único — nem sair da venda nem ser desativado.
	principal.ForaDeVenda = true
	if err := armazemSvc.Atualizar(ctx, principal); err == nil {
		t.Error("esperava erro ao tirar da venda o último armazém disponível")
	}
	if err := armazemSvc.Desativar(ctx, principal.ID); err == nil {
		t.Error("esperava erro ao desativar o último armazém disponível para venda")
	}
	// O balcão (fora de venda) pode ser desativado à vontade.
	if err := armazemSvc.Desativar(ctx, balcao.ID); err != nil {
		t.Errorf("desativar armazém fora de venda: %v", err)
	}
}
