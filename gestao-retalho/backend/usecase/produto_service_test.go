package usecase_test

import (
	"context"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestProdutoService(t *testing.T) *usecase.ProdutoService {
	t.Helper()
	db := newTestDB(t)
	return usecase.NewProdutoService(repository.NewProdutoRepository(db))
}

func novaUnidadeTeste(nome string, qtd, custo, venda float64) domain.UnidadeProduto {
	return domain.UnidadeProduto{
		NomeUnidade:      nome,
		QtdEmUnidadeBase: qtd,
		PrecoCusto:       custo,
		PrecoVenda:       venda,
		IsPadraoVenda:    true,
		IsPadraoCompra:   true,
	}
}

func TestProdutoService_CriarExigePeloMenosUmaUnidade(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	_, _, err := svc.Criar(ctx, usecase.NovoProduto{Produto: domain.Produto{Nome: "Cerveja"}})
	if err == nil {
		t.Fatal("esperava erro sem nenhuma unidade")
	}
}

func TestProdutoService_CriarComVariasUnidades(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	produto, unidades, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto: domain.Produto{Nome: "Cerveja", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{
			novaUnidadeTeste("unidade", 1, 40, 80),
			novaUnidadeTeste("caixa de 24", 24, 900, 1800),
		},
	})
	if err != nil {
		t.Fatalf("criar produto com várias unidades: %v", err)
	}
	if len(unidades) != 2 {
		t.Fatalf("unidades = %d, want 2 (variações/grupos do mesmo produto)", len(unidades))
	}

	listadas, err := svc.ListarUnidades(ctx, produto.ID)
	if err != nil {
		t.Fatalf("listar unidades: %v", err)
	}
	if len(listadas) != 2 {
		t.Fatalf("unidades listadas = %d, want 2", len(listadas))
	}
}

func TestProdutoService_AdicionarUnidadeAProdutoExistente(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	produto, _, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Cerveja", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{novaUnidadeTeste("unidade", 1, 40, 80)},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}

	nova := novaUnidadeTeste("caixa de 24", 24, 900, 1800)
	nova.ProdutoID = produto.ID
	criada, err := svc.AdicionarUnidade(ctx, nova)
	if err != nil {
		t.Fatalf("adicionar unidade: %v", err)
	}
	if criada.ProdutoID != produto.ID {
		t.Errorf("ProdutoID = %d, want %d", criada.ProdutoID, produto.ID)
	}

	unidades, err := svc.ListarUnidades(ctx, produto.ID)
	if err != nil {
		t.Fatalf("listar unidades: %v", err)
	}
	if len(unidades) != 2 {
		t.Fatalf("unidades = %d, want 2 depois de adicionar", len(unidades))
	}
}

func TestProdutoService_AdicionarUnidadeValidaCampos(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	if _, err := svc.AdicionarUnidade(ctx, domain.UnidadeProduto{ProdutoID: 1, NomeUnidade: "", QtdEmUnidadeBase: 1}); err == nil {
		t.Fatal("esperava erro sem nome da unidade")
	}
	if _, err := svc.AdicionarUnidade(ctx, domain.UnidadeProduto{ProdutoID: 1, NomeUnidade: "caixa", QtdEmUnidadeBase: 0}); err == nil {
		t.Fatal("esperava erro com quantidade zero")
	}
}

func TestProdutoService_DuplicarCriaCopiaComPropriasUnidades(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	original, _, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Cerveja 330ml", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{novaUnidadeTeste("unidade", 1, 40, 80)},
	})
	if err != nil {
		t.Fatalf("criar produto original: %v", err)
	}

	copia, unidadesCopia, err := svc.Duplicar(ctx, original.ID)
	if err != nil {
		t.Fatalf("duplicar produto: %v", err)
	}
	if copia.ID == original.ID {
		t.Error("cópia devia ter ID diferente do original")
	}
	if copia.Nome != "Cerveja 330ml (cópia)" {
		t.Errorf("nome da cópia = %q, want sufixo (cópia)", copia.Nome)
	}
	if len(unidadesCopia) != 1 {
		t.Fatalf("unidades da cópia = %d, want 1", len(unidadesCopia))
	}

	todos, err := svc.ListarTodos(ctx, domain.NegocioID)
	if err != nil {
		t.Fatalf("listar todos: %v", err)
	}
	if len(todos) != 2 {
		t.Fatalf("produtos = %d, want 2 (original + cópia, útil para variantes)", len(todos))
	}
}

func TestProdutoService_AtualizarRenomeiaProduto(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	produto, _, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Cerveja", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{novaUnidadeTeste("unidade", 1, 40, 80)},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}

	produto.Nome = "Cerveja Premium"
	atualizado, err := svc.Atualizar(ctx, produto)
	if err != nil {
		t.Fatalf("atualizar produto: %v", err)
	}
	if atualizado.Nome != "Cerveja Premium" {
		t.Errorf("nome = %q, want Cerveja Premium", atualizado.Nome)
	}
}

func TestProdutoService_AtualizarUnidadeMudaPrecoECodigo(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	_, unidades, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Cerveja", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{novaUnidadeTeste("unidade", 1, 40, 80)},
	})
	if err != nil {
		t.Fatalf("criar produto: %v", err)
	}

	u := unidades[0]
	u.PrecoVenda = 95
	codigo := "5601234567890"
	u.CodigoBarras = &codigo
	atualizada, err := svc.AtualizarUnidade(ctx, u)
	if err != nil {
		t.Fatalf("atualizar unidade: %v", err)
	}
	if atualizada.PrecoVenda != 95 || atualizada.CodigoBarras == nil || *atualizada.CodigoBarras != codigo {
		t.Errorf("unidade = %+v, want preço 95 e código %s", atualizada, codigo)
	}
	if atualizada.ProdutoID != u.ProdutoID {
		t.Errorf("ProdutoID = %d, want %d (a unidade não muda de produto)", atualizada.ProdutoID, u.ProdutoID)
	}

	todas, err := svc.ListarUnidadesDoNegocio(ctx)
	if err != nil || len(todas) != 1 || todas[0].PrecoVenda != 95 {
		t.Fatalf("unidades do negócio = %+v, %v; want uma com preço 95", todas, err)
	}
}

func TestProdutoService_AtualizarUnidadeRecusaCodigoRepetidoEPrecoNegativo(t *testing.T) {
	ctx := context.Background()
	svc := newTestProdutoService(t)

	codigo := "5601234567890"
	comCodigo := novaUnidadeTeste("unidade", 1, 40, 80)
	comCodigo.CodigoBarras = &codigo
	if _, _, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Cerveja", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{comCodigo},
	}); err != nil {
		t.Fatalf("criar cerveja: %v", err)
	}
	_, unidades, err := svc.Criar(ctx, usecase.NovoProduto{
		Produto:  domain.Produto{Nome: "Sumo", TipoProduto: domain.TipoProdutoSimples, UnidadeBase: "unidade"},
		Unidades: []domain.UnidadeProduto{novaUnidadeTeste("unidade", 1, 20, 50)},
	})
	if err != nil {
		t.Fatalf("criar sumo: %v", err)
	}

	u := unidades[0]
	u.CodigoBarras = &codigo
	if _, err := svc.AtualizarUnidade(ctx, u); err == nil || err.Error() != "este código de barras já está noutro produto ou unidade" {
		t.Errorf("err = %v, want aviso de código de barras repetido", err)
	}

	u = unidades[0]
	u.PrecoVenda = -1
	if _, err := svc.AtualizarUnidade(ctx, u); err == nil {
		t.Error("esperava erro com preço negativo")
	}
}
