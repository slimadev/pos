package usecase_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// excelTeste imita a folha do cliente (K&KC STOCK.xlsx): título por cima,
// cabeçalho na linha 4, números escritos como texto em formato inglês e
// uma linha de totais no fim.
func excelTeste(t *testing.T) string {
	t.Helper()
	f := excelize.NewFile()
	folha := "CONTAGEM DE STOCK"
	_ = f.SetSheetName("Sheet1", folha)
	linhas := [][]any{
		{},
		{"", "CONTAGEM DE STOCK EXISTENTE"},
		{},
		{"No.", "DESCRICAO DO PRODUCTO", "CATEGORIA", "QUANTIDADE", "CUSTO UNITARIO", "CUSTO TOTAL", "PRECO UNIT. K&KC"},
		{1, "Alicates de Ponta Fina ", "Ferramentas", 84, " 90.00 ", " 7,560.00 ", " 150.00 "},
		{2, "Extensores 2 metros (5 bocas)", "Elétrico", "64", " 1,000.00 ", "", ""},
		{3, "Cimento 50kg", "Cimento e areia", 10, 480, 4800, 620},
		{4, "alicates de ponta fina", "Ferramentas", 5, 90, 450, 150},
		{5, "Rolos de Lixa", "Abrasivos", 2, "(10.00)", "", 30},
		{"TOTAIS", "", "", "165", "", "", ""},
		{},
		{"", "Nota importante: as colunas de fornecedor ficam para depois.", "", "", "", "", ""},
	}
	for i, l := range linhas {
		cel, _ := excelize.CoordinatesToCellName(1, i+1)
		_ = f.SetSheetRow(folha, cel, &l)
	}
	caminho := filepath.Join(t.TempDir(), "stock.xlsx")
	if err := f.SaveAs(caminho); err != nil {
		t.Fatal(err)
	}
	return caminho
}

type servicosImportacao struct {
	svc        *usecase.ImportacaoService
	produtos   *usecase.ProdutoService
	stock      *usecase.StockService
	categorias *usecase.CategoriaService
	armazem    domain.Armazem
}

func novosServicosImportacao(t *testing.T) servicosImportacao {
	t.Helper()
	ctx := context.Background()
	db := newTestDB(t)
	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	catRepo := repository.NewCategoriaRepository(db)
	s := servicosImportacao{
		svc:        usecase.NewImportacaoService(repository.NewImportacaoRepository(db), produtoRepo, catRepo, armazemRepo),
		produtos:   usecase.NewProdutoService(produtoRepo),
		stock:      usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, repository.NewInventarioRepository(db), repository.NewLoteRepository(db, stockRepo)),
		categorias: usecase.NewCategoriaService(catRepo),
	}
	s.armazem = criarArmazem(t, ctx, usecase.NewArmazemService(armazemRepo), "Principal")
	for _, n := range []string{"Ferramentas", "Elétrico"} {
		if _, err := s.categorias.Criar(ctx, domain.Categoria{Nome: n}); err != nil {
			t.Fatal(err)
		}
	}
	// "Cimento 50kg" já existe: tem de ser ignorado.
	criarProdutoComPreco(t, ctx, s.produtos, "Cimento 50kg", 620)
	return s
}

func TestImportacao_AnalisaCabecalhoEColunas(t *testing.T) {
	s := novosServicosImportacao(t)
	a, err := s.svc.AnalisarExcel(excelTeste(t))
	if err != nil {
		t.Fatalf("analisar: %v", err)
	}
	f := a.Folhas[0]
	m := f.Mapeamento
	if f.LinhaCabecalho != 4 || m[usecase.CampoNome] != 1 || m[usecase.CampoCategoria] != 2 ||
		m[usecase.CampoQuantidade] != 3 || m[usecase.CampoCusto] != 4 || m[usecase.CampoPreco] != 6 {
		t.Errorf("folha = %+v", f)
	}
	if m[usecase.CampoUnidade] != -1 || m[usecase.CampoCodigo] != -1 {
		t.Errorf("colunas inexistentes mapeadas: %+v", m)
	}
}

func TestImportacao_PreVisualizarEImportar(t *testing.T) {
	ctx := context.Background()
	s := novosServicosImportacao(t)
	caminho := excelTeste(t)
	a, _ := s.svc.AnalisarExcel(caminho)
	linhas, err := s.svc.PreVisualizar(ctx, usecase.PedidoPreVisualizacao{Ficheiro: caminho, Folha: a.Folhas[0].Nome, Mapeamento: a.Folhas[0].Mapeamento})
	if err != nil {
		t.Fatalf("pré-visualizar: %v", err)
	}
	if len(linhas) != 5 {
		t.Fatalf("linhas = %d, want 5 (a de TOTAIS fica de fora): %+v", len(linhas), linhas)
	}
	estado := func(i int) string { return linhas[i].Estado }
	if estado(0) != usecase.LinhaNova || linhas[0].Nome != "Alicates de Ponta Fina" || linhas[0].Quantidade != 84 || linhas[0].Custo != 90 || linhas[0].Preco != 150 {
		t.Errorf("linha 0 = %+v", linhas[0])
	}
	if estado(1) != usecase.LinhaNova || linhas[1].Custo != 1000 || linhas[1].Preco != 0 || linhas[1].Categoria != "Elétrico" {
		t.Errorf("linha 1 = %+v", linhas[1])
	}
	if estado(2) != usecase.LinhaExiste || estado(3) != usecase.LinhaRepetida || estado(4) != usecase.LinhaComErro {
		t.Errorf("estados = %q %q %q", estado(2), estado(3), estado(4))
	}

	// Sem preço na linha 1: a importação recusa e nada entra.
	if _, err := s.svc.Importar(ctx, usecase.ImportarProdutosParams{ArmazemID: s.armazem.ID, Linhas: linhas, UtilizadorID: 1}); err == nil {
		t.Fatal("esperava erro: falta o preço de venda")
	}
	if todos, _ := s.produtos.ListarTodos(ctx, domain.NegocioID); len(todos) != 1 {
		t.Fatalf("produtos = %d, want 1 (nada importado)", len(todos))
	}

	// Com o preço escrito no ecrã e uma categoria nova numa linha.
	linhas[1].Preco = 1350
	linhas[0].Categoria = "Ferramentas manuais"
	linhas, _ = s.svc.PreVisualizar(ctx, usecase.PedidoPreVisualizacao{Ficheiro: caminho, Folha: a.Folhas[0].Nome, Mapeamento: a.Folhas[0].Mapeamento})
	linhas[1].Preco = 1350
	linhas[0].Categoria, linhas[0].CategoriaNova = "Ferramentas manuais", true
	r, err := s.svc.Importar(ctx, usecase.ImportarProdutosParams{
		ArmazemID: s.armazem.ID, Referencia: "stock.xlsx — CONTAGEM", Linhas: linhas, UtilizadorID: 1,
	})
	if err != nil {
		t.Fatalf("importar: %v", err)
	}
	if r.Produtos != 2 || r.Categorias != 1 {
		t.Errorf("resultado = %+v, want 2 produtos e 1 categoria nova", r)
	}

	todos, _ := s.produtos.ListarTodos(ctx, domain.NegocioID)
	cats, _ := s.categorias.Listar(ctx, domain.NegocioID)
	nomeCat := func(id *int64) string {
		for _, c := range cats {
			if id != nil && c.ID == *id {
				return c.Nome
			}
		}
		return ""
	}
	for _, p := range todos {
		st, _ := s.stock.ObterStock(ctx, p.ID, s.armazem.ID)
		switch p.Nome {
		case "Alicates de Ponta Fina":
			if st.Quantidade != 84 || st.CustoMedioPonderado != 90 || nomeCat(p.CategoriaID) != "Ferramentas manuais" {
				t.Errorf("alicates: stock %v a %v, categoria %q", st.Quantidade, st.CustoMedioPonderado, nomeCat(p.CategoriaID))
			}
			us, _ := s.produtos.ListarUnidades(ctx, p.ID)
			if len(us) != 1 || us[0].PrecoVenda != 150 || us[0].NomeUnidade != "unidade" {
				t.Errorf("unidades = %+v", us)
			}
		case "Extensores 2 metros (5 bocas)":
			if st.Quantidade != 64 || nomeCat(p.CategoriaID) != "Elétrico" {
				t.Errorf("extensores: stock %v, categoria %q", st.Quantidade, nomeCat(p.CategoriaID))
			}
		}
	}
	if len(todos) != 3 {
		t.Errorf("produtos = %d, want 3", len(todos))
	}
}

func TestImportacao_CategoriaPadraoParaLinhasSemCategoria(t *testing.T) {
	ctx := context.Background()
	s := novosServicosImportacao(t)
	cats, _ := s.categorias.Listar(ctx, domain.NegocioID)
	padrao := cats[0].ID
	r, err := s.svc.Importar(ctx, usecase.ImportarProdutosParams{
		ArmazemID: s.armazem.ID, CategoriaPadraoID: &padrao, UtilizadorID: 1,
		Linhas: []usecase.LinhaImportacao{{Linha: 5, Nome: "Enxada", Quantidade: 3, Custo: 200, Preco: 350, Estado: usecase.LinhaNova}},
	})
	if err != nil || r.Produtos != 1 {
		t.Fatalf("importar: %+v %v", r, err)
	}
	todos, _ := s.produtos.ListarTodos(ctx, domain.NegocioID)
	for _, p := range todos {
		if p.Nome == "Enxada" && (p.CategoriaID == nil || *p.CategoriaID != padrao) {
			t.Errorf("categoria = %v, want %d", p.CategoriaID, padrao)
		}
	}
}
