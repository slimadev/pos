package admin_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
)

func TestProdutosService_ListarRejeitaSemLojaConfigurada(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewProdutosService(repo)

	_, err := svc.Listar(ctx)
	if err == nil {
		t.Fatal("esperava erro sem nenhuma loja registada")
	}
}

func TestProdutosService_CriarComVariasUnidadesEListar(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewProdutosService(repo)

	loja := novaLojaTeste(t, "loja-produtos.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	resultado, err := svc.Criar(ctx, api.CriarProdutoRequest{
		Produto: domain.Produto{Nome: "Cimento 50kg", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "saco"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "saco", QtdEmUnidadeBase: 1, PrecoCusto: 400, PrecoVenda: 620, IsPadraoVenda: true, IsPadraoCompra: true},
			{NomeUnidade: "palete de 40", QtdEmUnidadeBase: 40, PrecoCusto: 15000, PrecoVenda: 22000},
		},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	if len(resultado.Unidades) != 2 {
		t.Fatalf("unidades = %d, want 2 (variações/grupos do mesmo produto)", len(resultado.Unidades))
	}

	produtos, err := svc.Listar(ctx)
	if err != nil {
		t.Fatalf("listar produtos: %v", err)
	}
	achou := false
	for _, p := range produtos {
		if p.Nome == "Cimento 50kg" {
			achou = true
		}
	}
	if !achou {
		t.Fatalf("produtos = %+v, want incluir Cimento 50kg", produtos)
	}

	unidades, err := svc.ListarUnidades(ctx, resultado.Produto.ID)
	if err != nil {
		t.Fatalf("listar unidades: %v", err)
	}
	if len(unidades) != 2 {
		t.Fatalf("unidades listadas = %d, want 2", len(unidades))
	}
}

func TestProdutosService_AdicionarUnidadeAtualizarDuplicarEAtivar(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewProdutosService(repo)

	loja := novaLojaTeste(t, "loja-produtos-editar.db", "Bottlestore XPTO", domain.TipoNegocioBottlestore)
	if _, err := repo.Criar(ctx, "Bottlestore XPTO", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	resultado, err := svc.Criar(ctx, api.CriarProdutoRequest{
		Produto: domain.Produto{Nome: "Cerveja", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			{NomeUnidade: "unidade", QtdEmUnidadeBase: 1, PrecoCusto: 40, PrecoVenda: 80, IsPadraoVenda: true, IsPadraoCompra: true},
		},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}
	produtoID := resultado.Produto.ID

	// Adicionar mais uma unidade a um produto já existente.
	nova, err := svc.AdicionarUnidade(ctx, domain.UnidadeProduto{
		ProdutoID: produtoID, NomeUnidade: "caixa de 24", QtdEmUnidadeBase: 24, PrecoCusto: 900, PrecoVenda: 1800,
	})
	if err != nil {
		t.Fatalf("adicionar unidade: %v", err)
	}
	if nova.ProdutoID != produtoID {
		t.Errorf("ProdutoID da nova unidade = %d, want %d", nova.ProdutoID, produtoID)
	}
	unidades, err := svc.ListarUnidades(ctx, produtoID)
	if err != nil {
		t.Fatalf("listar unidades: %v", err)
	}
	if len(unidades) != 2 {
		t.Fatalf("unidades = %d, want 2 depois de adicionar", len(unidades))
	}

	// Editar (renomear) o produto.
	produto := resultado.Produto
	produto.Nome = "Cerveja Premium"
	atualizado, err := svc.Atualizar(ctx, produto)
	if err != nil {
		t.Fatalf("atualizar produto: %v", err)
	}
	if atualizado.Nome != "Cerveja Premium" {
		t.Errorf("nome = %q, want Cerveja Premium", atualizado.Nome)
	}

	// Duplicar (para uma variante, ex: outro tamanho).
	copia, err := svc.Duplicar(ctx, produtoID)
	if err != nil {
		t.Fatalf("duplicar produto: %v", err)
	}
	if copia.Produto.ID == produtoID {
		t.Error("cópia devia ter ID diferente do original")
	}
	if copia.Produto.Nome != "Cerveja Premium (cópia)" {
		t.Errorf("nome da cópia = %q, want sufixo (cópia)", copia.Produto.Nome)
	}

	// Desativar/ativar.
	if err := svc.DefinirAtivo(ctx, produtoID, false); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	produtos, err := svc.Listar(ctx)
	if err != nil {
		t.Fatalf("listar produtos: %v", err)
	}
	var achado *domain.Produto
	for i := range produtos {
		if produtos[i].ID == produtoID {
			achado = &produtos[i]
		}
	}
	if achado == nil || achado.Ativo {
		t.Fatalf("produto = %+v, want inativo depois de DefinirAtivo(false)", achado)
	}

	if err := svc.DefinirAtivo(ctx, produtoID, true); err != nil {
		t.Fatalf("ativar: %v", err)
	}
}
