package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestFichasService_ReceitaCustoDisponibilidadeEProducao(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewFichasService(repo)

	loja := novaLojaTeste(t, "loja-fichas-admin.db", "Bar Central", domain.TipoNegocioBottlestore)
	if _, err := repo.Criar(ctx, "Bar Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}
	criar := func(nome string, tipo domain.TipoProduto, unidade string, preco float64) domain.Produto {
		p, _, err := loja.Produtos.Criar(ctx, usecase.NovoProduto{
			Produto:  domain.Produto{Nome: nome, TipoProduto: tipo, UnidadeBase: unidade},
			Unidades: []domain.UnidadeProduto{{NomeUnidade: unidade, QtdEmUnidadeBase: 1, PrecoVenda: preco, IsPadraoVenda: true, IsPadraoCompra: true}},
		})
		if err != nil {
			t.Fatalf("criar %s: %v", nome, err)
		}
		return p
	}
	rum := criar("Rum", domain.TipoProdutoSimples, "L", 0)
	hortela := criar("Hortelã", domain.TipoProdutoSimples, "molho", 0)
	if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: rum.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 1, CustoUnitario: 1000, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada rum: %v", err)
	}
	if _, err := loja.Stock.RegistarEntrada(ctx, usecase.EntradaParams{ProdutoID: hortela.ID, ArmazemDestinoID: loja.Armazem.ID, Quantidade: 10, CustoUnitario: 20, UtilizadorID: 1}); err != nil {
		t.Fatalf("entrada hortelã: %v", err)
	}
	mojito := criar("Mojito", domain.TipoProdutoComposto, "copo", 350)
	daiquiri := criar("Daiquiri", domain.TipoProdutoComposto, "copo", 380)

	if _, err := svc.DefinirReceita(ctx, mojito.ID, []usecase.IngredienteInput{
		{ProdutoIngredienteID: rum.ID, Quantidade: 0.05},
		{ProdutoIngredienteID: hortela.ID, Quantidade: 1},
	}); err != nil {
		t.Fatalf("receita mojito: %v", err)
	}
	if _, err := svc.DuplicarReceita(ctx, mojito.ID, daiquiri.ID); err != nil {
		t.Fatalf("duplicar: %v", err)
	}

	fichas, err := svc.Listar(ctx, loja.Armazem.ID)
	if err != nil {
		t.Fatalf("listar: %v", err)
	}
	porNome := map[string]admin.FichaResumo{}
	for _, f := range fichas {
		porNome[f.Produto.Nome] = f
	}
	m, ok := porNome["Mojito"]
	if !ok || len(m.Ingredientes) != 2 || m.PrecoVenda != 350 {
		t.Fatalf("mojito = %+v", m)
	}
	if m.CustoTeorico != 70 { // 0,05 L × 1000 + 1 molho × 20
		t.Errorf("custo teórico = %v, want 70", m.CustoTeorico)
	}
	if m.Disponivel != 10 { // hortelã limita: 10 molhos
		t.Errorf("disponível = %v, want 10", m.Disponivel)
	}
	if d := porNome["Daiquiri"]; len(d.Ingredientes) != 2 {
		t.Errorf("daiquiri duplicado com %d ingredientes, want 2", len(d.Ingredientes))
	}

	xarope := criar("Xarope simples", domain.TipoProdutoSemiElaborado, "L", 0)
	if _, err := svc.DefinirReceita(ctx, xarope.ID, []usecase.IngredienteInput{{ProdutoIngredienteID: hortela.ID, Quantidade: 2}}); err != nil {
		t.Fatalf("receita xarope: %v", err)
	}
	if _, err := svc.RegistarProducao(ctx, usecase.ProducaoParams{ProdutoSemiElaboradoID: xarope.ID, ArmazemID: loja.Armazem.ID, Quantidade: 3}); err != nil {
		t.Fatalf("produzir: %v", err)
	}
	fichas, _ = svc.Listar(ctx, loja.Armazem.ID)
	for _, f := range fichas {
		if f.Produto.ID == mojito.ID && f.Disponivel != 4 { // 10 - 3×2 = 4 molhos
			t.Errorf("depois da produção, mojitos possíveis = %v, want 4", f.Disponivel)
		}
	}
}
