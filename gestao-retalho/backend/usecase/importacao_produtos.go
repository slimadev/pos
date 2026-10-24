package usecase

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/export"
	"gestao-retalho/backend/repository"
)

// Campos que uma coluna do Excel pode alimentar.
const (
	CampoNome       = "nome"
	CampoCategoria  = "categoria"
	CampoUnidade    = "unidade"
	CampoQuantidade = "quantidade"
	CampoCusto      = "custo"
	CampoPreco      = "preco"
	CampoCodigo     = "codigo"
)

// Estados de uma linha na pré-visualização.
const (
	LinhaNova     = "novo"
	LinhaExiste   = "existe"   // já há um produto com este nome: ignorada
	LinhaRepetida = "repetido" // o mesmo nome aparece antes no ficheiro: ignorada
	LinhaIgnorada = "ignorada" // sem nome, linha de totais
	LinhaComErro  = "erro"     // ex: custo negativo, código de barras de outro produto
)

// FolhaImportacao descreve uma folha do Excel: onde está o cabeçalho e que
// coluna parece ser cada campo (-1 = nenhuma).
type FolhaImportacao struct {
	Nome           string
	LinhaCabecalho int // 1 = primeira linha
	Colunas        []string
	Mapeamento     map[string]int
	LinhasDados    int
}

type AnaliseExcel struct {
	Ficheiro string
	Folhas   []FolhaImportacao
}

type PedidoPreVisualizacao struct {
	Ficheiro   string
	Folha      string
	Mapeamento map[string]int
}

// LinhaImportacao é uma linha da folha já interpretada. Na importação só
// contam as linhas "novo"; o preço pode ter sido escrito no ecrã.
type LinhaImportacao struct {
	Linha         int // linha no Excel
	Nome          string
	Categoria     string // texto da coluna Categoria ("" = a escolhida no ecrã)
	CategoriaNova bool   // a categoria não existe e será criada
	Unidade       string
	CodigoBarras  string
	Quantidade    float64
	Custo         float64
	Preco         float64
	Estado        string
	Motivo        string
}

type ImportarProdutosParams struct {
	ArmazemID         int64
	CategoriaPadraoID *int64 // para as linhas sem categoria no Excel
	Referencia        string // ex: "K&KC STOCK.xlsx — CONTAGEM DE STOCK EXISTENTE"
	Linhas            []LinhaImportacao
	UtilizadorID      int64
}

type ResultadoImportacao struct {
	Produtos   int
	Categorias int
}

// ImportacaoService traz produtos novos de uma folha de Excel (nome,
// categoria, unidade, custo, preço, quantidade, código de barras): cria-os
// e dá entrada da quantidade como stock, numa só transação. Produtos que
// já existem não são tocados.
type ImportacaoService struct {
	repo       *repository.ImportacaoRepository
	produtos   *repository.ProdutoRepository
	categorias *repository.CategoriaRepository
	armazens   *repository.ArmazemRepository
}

func NewImportacaoService(repo *repository.ImportacaoRepository, produtos *repository.ProdutoRepository,
	categorias *repository.CategoriaRepository, armazens *repository.ArmazemRepository) *ImportacaoService {
	return &ImportacaoService{repo: repo, produtos: produtos, categorias: categorias, armazens: armazens}
}

func normalizarCabecalho(s string) string {
	return strings.ToUpper(semAcentos.Replace(strings.Join(strings.Fields(strings.ToLower(s)), " ")))
}

// campoDoCabecalho adivinha o campo pelo título da coluna.
func campoDoCabecalho(titulo string) string {
	t := normalizarCabecalho(titulo)
	tem := func(partes ...string) bool {
		for _, p := range partes {
			if strings.Contains(t, p) {
				return true
			}
		}
		return false
	}
	switch {
	case t == "":
		return ""
	case tem("CODIGO", "BARRAS", "EAN"):
		return CampoCodigo
	case tem("CATEG", "FAMILIA", "GRUPO"):
		return CampoCategoria
	case tem("TOT", "MARGEM", "GANHO", "PERDA"):
		return ""
	case tem("CUSTO", "PRECO COMPRA", "PRECO DE COMPRA"):
		return CampoCusto
	case tem("PRECO", "PVP", "VENDA"):
		return CampoPreco
	case tem("QUANT", "QDADE", "QTD", "STOCK", "EXISTENC"):
		return CampoQuantidade
	case strings.HasPrefix(t, "UNIDADE") || t == "UND" || t == "UN":
		return CampoUnidade
	case tem("DESCRI", "PRODUT", "NOME", "ARTIGO", "ITEM", "DESIGNA"):
		return CampoNome
	}
	return ""
}

func mapear(cabecalho []string) map[string]int {
	m := map[string]int{CampoNome: -1, CampoCategoria: -1, CampoUnidade: -1, CampoQuantidade: -1, CampoCusto: -1, CampoPreco: -1, CampoCodigo: -1}
	for i, c := range cabecalho {
		if campo := campoDoCabecalho(c); campo != "" && m[campo] == -1 {
			m[campo] = i
		}
	}
	return m
}

func lerLinhas(f *excelize.File, folha string) ([][]string, error) {
	return f.GetRows(folha, excelize.Options{RawCellValue: true})
}

// cabecalho encontra a linha de títulos nas primeiras 20: a primeira que
// tenha a coluna do nome e pelo menos mais um campo reconhecido.
func cabecalho(linhas [][]string) int {
	for i := 0; i < len(linhas) && i < 20; i++ {
		m := mapear(linhas[i])
		n := 0
		for _, v := range m {
			if v >= 0 {
				n++
			}
		}
		if m[CampoNome] >= 0 && n >= 2 {
			return i
		}
	}
	return -1
}

// AnalisarExcel lê as folhas do ficheiro e sugere o cabeçalho e as colunas.
func (s *ImportacaoService) AnalisarExcel(caminho string) (AnaliseExcel, error) {
	f, err := excelize.OpenFile(caminho)
	if err != nil {
		return AnaliseExcel{}, fmt.Errorf("abrir o Excel: %w", err)
	}
	defer f.Close()
	a := AnaliseExcel{Ficheiro: caminho, Folhas: []FolhaImportacao{}}
	for _, nome := range f.GetSheetList() {
		linhas, err := lerLinhas(f, nome)
		if err != nil {
			return AnaliseExcel{}, err
		}
		folha := FolhaImportacao{Nome: nome, Colunas: []string{}, Mapeamento: mapear(nil)}
		if i := cabecalho(linhas); i >= 0 {
			folha.LinhaCabecalho = i + 1
			folha.Colunas = linhas[i]
			folha.Mapeamento = mapear(linhas[i])
			folha.LinhasDados = len(linhas) - i - 1
		}
		a.Folhas = append(a.Folhas, folha)
	}
	return a, nil
}

// lerNumero aceita números do Excel em bruto ("1000.5") e escritos como
// texto em qualquer formato ("1,000.00", "1 234,56", "(900.00)").
func lerNumero(s string) (float64, bool) {
	s = strings.TrimSpace(strings.ReplaceAll(s, " ", " "))
	if s == "" || s == "-" {
		return 0, true
	}
	negativo := strings.HasPrefix(s, "(") && strings.HasSuffix(s, ")")
	s = strings.Trim(s, "()")
	s = strings.ReplaceAll(strings.ReplaceAll(s, " ", ""), "MT", "")
	virgula, ponto := strings.LastIndex(s, ","), strings.LastIndex(s, ".")
	switch {
	case virgula >= 0 && ponto >= 0:
		if virgula > ponto { // 1.234,56
			s = strings.ReplaceAll(s, ".", "")
			s = strings.Replace(s, ",", ".", 1)
		} else { // 1,234.56
			s = strings.ReplaceAll(s, ",", "")
		}
	case virgula >= 0:
		// "1,000" (milhares) ou "12,5" (decimal).
		if len(s)-virgula-1 == 3 && strings.Count(s, ",") >= 1 && !strings.HasPrefix(s, "0,") {
			s = strings.ReplaceAll(s, ",", "")
		} else {
			s = strings.Replace(s, ",", ".", 1)
		}
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	if negativo {
		v = -v
	}
	return math.Round(v*1000) / 1000, true
}

func celula(linha []string, i int) string {
	if i < 0 || i >= len(linha) {
		return ""
	}
	return strings.TrimSpace(linha[i])
}

// PreVisualizar interpreta as linhas da folha com o mapeamento escolhido e
// diz o que acontece a cada uma.
func (s *ImportacaoService) PreVisualizar(ctx context.Context, p PedidoPreVisualizacao) ([]LinhaImportacao, error) {
	f, err := excelize.OpenFile(p.Ficheiro)
	if err != nil {
		return nil, fmt.Errorf("abrir o Excel: %w", err)
	}
	defer f.Close()
	linhas, err := lerLinhas(f, p.Folha)
	if err != nil {
		return nil, err
	}
	if p.Mapeamento[CampoNome] < 0 {
		return nil, fmt.Errorf("escolha a coluna com o nome do produto")
	}
	inicio := cabecalho(linhas) + 1
	if inicio <= 0 {
		inicio = 0
	}

	out := []LinhaImportacao{}
	for i := inicio; i < len(linhas); i++ {
		l := linhas[i]
		li := LinhaImportacao{
			Linha:        i + 1,
			Nome:         strings.Join(strings.Fields(celula(l, p.Mapeamento[CampoNome])), " "),
			Categoria:    celula(l, p.Mapeamento[CampoCategoria]),
			Unidade:      celula(l, p.Mapeamento[CampoUnidade]),
			CodigoBarras: celula(l, p.Mapeamento[CampoCodigo]),
		}
		var erros []string
		for campo, destino := range map[string]*float64{CampoQuantidade: &li.Quantidade, CampoCusto: &li.Custo, CampoPreco: &li.Preco} {
			if v, ok := lerNumero(celula(l, p.Mapeamento[campo])); ok {
				*destino = v
			} else {
				erros = append(erros, campo+" não é um número")
			}
		}
		norm := normalizarCabecalho(li.Nome)
		if li.Nome == "" || strings.HasPrefix(norm, "TOTA") {
			continue // linhas vazias e de totais nem aparecem
		}
		// Notas e títulos soltos no meio ou no fim da folha: texto sem
		// nenhum número nas outras colunas.
		semNumeros := true
		for _, campo := range []string{CampoQuantidade, CampoCusto, CampoPreco} {
			if celula(l, p.Mapeamento[campo]) != "" {
				semNumeros = false
			}
		}
		if semNumeros {
			continue
		}
		if len(erros) > 0 {
			li.Estado, li.Motivo = LinhaComErro, strings.Join(erros, "; ")
		}
		out = append(out, li)
	}
	return s.classificar(ctx, out)
}

// classificar marca cada linha (novo / existe / repetido / erro) contra os
// produtos e categorias da loja. Usada na pré-visualização e outra vez ao
// importar, porque entretanto alguém pode ter criado um produto.
func (s *ImportacaoService) classificar(ctx context.Context, linhas []LinhaImportacao) ([]LinhaImportacao, error) {
	produtos, err := s.produtos.Listar(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	unidades, err := s.produtos.ListarUnidadesDoNegocio(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	categorias, err := s.categorias.Listar(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	codigos := map[string]bool{}
	for _, u := range unidades {
		if u.CodigoBarras != nil {
			codigos[*u.CodigoBarras] = true
		}
	}
	vistos := []string{}
	for i := range linhas {
		li := &linhas[i]
		if li.Estado == LinhaComErro {
			continue
		}
		li.Estado, li.Motivo, li.CategoriaNova = LinhaNova, "", false
		existe := false
		for _, p := range produtos {
			if mesmoNome(p.Nome, li.Nome) {
				existe = true
				break
			}
		}
		repetido := false
		for _, v := range vistos {
			if mesmoNome(v, li.Nome) {
				repetido = true
				break
			}
		}
		switch {
		case existe:
			li.Estado, li.Motivo = LinhaExiste, "já existe — ignorado"
			continue
		case repetido:
			li.Estado, li.Motivo = LinhaRepetida, "aparece antes no ficheiro — ignorado"
			continue
		case li.Quantidade < 0 || li.Custo < 0 || li.Preco < 0:
			li.Estado, li.Motivo = LinhaComErro, "valores negativos"
			continue
		case li.CodigoBarras != "" && codigos[li.CodigoBarras]:
			li.Estado, li.Motivo = LinhaComErro, "código de barras já está noutro produto"
			continue
		}
		vistos = append(vistos, li.Nome)
		if li.CodigoBarras != "" {
			codigos[li.CodigoBarras] = true
		}
		if li.Categoria != "" {
			li.CategoriaNova = true
			for _, c := range categorias {
				if mesmoNome(c.Nome, li.Categoria) {
					li.Categoria, li.CategoriaNova = c.Nome, false
					break
				}
			}
		}
	}
	return linhas, nil
}

// Importar cria os produtos novos (e as categorias que faltam) e dá entrada
// das quantidades no armazém, tudo ou nada. Todas as linhas novas têm de
// ter preço de venda.
func (s *ImportacaoService) Importar(ctx context.Context, p ImportarProdutosParams) (ResultadoImportacao, error) {
	armazem, err := s.armazens.Obter(ctx, p.ArmazemID)
	if err != nil {
		return ResultadoImportacao{}, fmt.Errorf("armazém: %w", err)
	}
	if !armazem.Ativo {
		return ResultadoImportacao{}, domain.ErrArmazemInativo
	}
	if p.CategoriaPadraoID != nil {
		if _, err := s.categoriaPorID(ctx, *p.CategoriaPadraoID); err != nil {
			return ResultadoImportacao{}, err
		}
	}
	linhas := make([]LinhaImportacao, 0, len(p.Linhas))
	for _, l := range p.Linhas {
		if l.Estado == LinhaNova {
			l.Estado = ""
			linhas = append(linhas, l)
		}
	}
	if len(linhas) == 0 {
		return ResultadoImportacao{}, fmt.Errorf("não há produtos novos para importar")
	}
	linhas, err = s.classificar(ctx, linhas)
	if err != nil {
		return ResultadoImportacao{}, err
	}
	categorias, err := s.categorias.Listar(ctx, domain.NegocioID)
	if err != nil {
		return ResultadoImportacao{}, err
	}

	itens := make([]repository.ProdutoImportar, 0, len(linhas))
	for _, l := range linhas {
		if l.Estado != LinhaNova {
			return ResultadoImportacao{}, fmt.Errorf("linha %d (%s): %s", l.Linha, l.Nome, l.Motivo)
		}
		if l.Preco <= 0 {
			return ResultadoImportacao{}, fmt.Errorf("linha %d (%s): falta o preço de venda", l.Linha, l.Nome)
		}
		unidade := strings.TrimSpace(l.Unidade)
		if unidade == "" {
			unidade = "unidade"
		}
		it := repository.ProdutoImportar{
			Produto: domain.Produto{
				NegocioID: domain.NegocioID, Nome: l.Nome, TipoProduto: domain.TipoProdutoSimples,
				UnidadeBase: unidade, ControlaStock: true, Ativo: true,
			},
			Unidade:    domain.UnidadeProduto{NomeUnidade: unidade, QtdEmUnidadeBase: 1, PrecoCusto: l.Custo, PrecoVenda: l.Preco},
			Quantidade: l.Quantidade,
		}
		if l.CodigoBarras != "" {
			codigo := l.CodigoBarras
			it.Unidade.CodigoBarras = &codigo
		}
		switch {
		case l.Categoria == "":
			it.Produto.CategoriaID = p.CategoriaPadraoID
		case l.CategoriaNova:
			it.CategoriaNova = l.Categoria
		default:
			for _, c := range categorias {
				if mesmoNome(c.Nome, l.Categoria) {
					id := c.ID
					it.Produto.CategoriaID = &id
					break
				}
			}
		}
		itens = append(itens, it)
	}

	motivo := "Importação de produtos"
	if ref := strings.TrimSpace(p.Referencia); ref != "" {
		motivo += ": " + ref
	}
	n, c, err := s.repo.Importar(ctx, itens, p.ArmazemID, motivo, p.UtilizadorID)
	if err != nil {
		return ResultadoImportacao{}, erroCodigoBarras(err)
	}
	return ResultadoImportacao{Produtos: n, Categorias: c}, nil
}

func (s *ImportacaoService) categoriaPorID(ctx context.Context, id int64) (domain.Categoria, error) {
	todas, err := s.categorias.Listar(ctx, domain.NegocioID)
	if err != nil {
		return domain.Categoria{}, err
	}
	for _, c := range todas {
		if c.ID == id {
			return c, nil
		}
	}
	return domain.Categoria{}, fmt.Errorf("categoria não encontrada")
}

// ModeloImportacao é a folha de Excel que se dá ao cliente para preencher.
func ModeloImportacao() export.Tabela {
	return export.Tabela{
		Titulo:  "Importação de produtos",
		Colunas: []string{"Nome", "Categoria", "Unidade", "Custo", "Preço de venda", "Quantidade", "Código de barras"},
		Linhas: [][]string{
			{"Martelo de unha 500g", "Ferramentas", "unidade", "270", "450", "12", ""},
			{"Cimento 50kg", "Cimento e areia", "saco", "480", "620", "40", "5600000000011"},
		},
	}
}

// NomeReferencia é "ficheiro — folha", para o motivo dos movimentos.
func NomeReferencia(ficheiro, folha string) string {
	return filepath.Base(ficheiro) + " — " + folha
}
