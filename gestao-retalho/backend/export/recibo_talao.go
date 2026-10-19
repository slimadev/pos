package export

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
)

// Larguras de rolo das impressoras de talão (térmicas) mais comuns.
const (
	PapelRolo80 = 80.0
	PapelRolo58 = 58.0
)

// GerarReciboTalao produz o recibo no formato de rolo de uma impressora de
// talão (80 ou 58 mm de largura): uma só coluna, cada produto em duas
// linhas (nome; quantidade × preço e subtotal) e a página com a altura
// exata do conteúdo, para a impressora não puxar papel em branco. Em A5
// (GerarReciboPDF) o driver da impressora térmica reduzia tudo a letras
// ilegíveis.
func GerarReciboTalao(d DadosRecibo, larguraMM float64) ([]byte, error) {
	margem := 4.0
	tamanho, linha := 9.0, 4.2
	if larguraMM < 70 {
		margem, tamanho, linha = 2.5, 8.0, 3.8
	}
	util := larguraMM - 2*margem

	// Primeira passagem só para medir: a página tem de ser criada já com a
	// altura final.
	medir := fpdf.New("P", "mm", "A4", "")
	tr := medir.UnicodeTranslatorFromDescriptor("")
	medir.SetFont("Helvetica", "B", tamanho+3)
	linhasLoja := len(partirTexto(medir, tr, d.NegocioNome, util))
	medir.SetFont("Helvetica", "B", tamanho)
	linhasNome := func(p *fpdf.Fpdf, nome string) []string {
		return partirTexto(p, tr, nome, util)
	}
	linhas := 0
	for _, item := range d.Itens {
		linhas += len(linhasNome(medir, item.NomeProduto)) + 1
	}
	if d.DescontoTotal > 0 {
		linhas++
	}
	linhas += len(d.Pagamentos)
	altura := 2*margem + float64(linhasLoja)*5 + 1 + 3*linha /* recibo, data, cliente */ + 6 + /* separadores e totais */
		float64(linhas)*linha + 3*linha + 7 /* total */ + 10 /* rodapé */

	pdf := fpdf.NewCustom(&fpdf.InitType{
		OrientationStr: "P",
		UnitStr:        "mm",
		Size:           fpdf.SizeType{Wd: larguraMM, Ht: altura},
	})
	pdf.SetMargins(margem, margem, margem)
	pdf.SetAutoPageBreak(false, 0)
	pdf.AddPage()
	tr = pdf.UnicodeTranslatorFromDescriptor("")

	separador := func() {
		y := pdf.GetY() + 1
		pdf.SetDashPattern([]float64{0.8, 0.8}, 0)
		pdf.Line(margem, y, larguraMM-margem, y)
		pdf.SetDashPattern([]float64{}, 0)
		pdf.SetY(y + 1.5)
	}
	duasColunas := func(esq, dir string) {
		w := pdf.GetStringWidth(dir) + 1
		pdf.CellFormat(util-w, linha, esq, "", 0, "L", false, 0, "")
		pdf.CellFormat(w, linha, dir, "", 1, "R", false, 0, "")
	}

	pdf.SetFont("Helvetica", "B", tamanho+3)
	for _, parte := range partirTexto(pdf, tr, d.NegocioNome, util) {
		pdf.CellFormat(util, 5, parte, "", 1, "C", false, 0, "")
	}
	pdf.Ln(1)
	pdf.SetFont("Helvetica", "", tamanho)
	pdf.CellFormat(util, linha, fmt.Sprintf("Recibo #%d", d.VendaID), "", 1, "C", false, 0, "")
	pdf.CellFormat(util, linha, d.CriadoEm.Local().Format("02/01/2006 15:04"), "", 1, "C", false, 0, "")
	pdf.CellFormat(util, linha, tr("Cliente: "+d.ClienteNome), "", 1, "L", false, 0, "")
	separador()

	for _, item := range d.Itens {
		pdf.SetFont("Helvetica", "B", tamanho)
		for _, parte := range linhasNome(pdf, item.NomeProduto) {
			pdf.CellFormat(util, linha, parte, "", 1, "L", false, 0, "")
		}
		pdf.SetFont("Helvetica", "", tamanho)
		detalhe := fmt.Sprintf("%s %s x %s", formatarQtd(item.Quantidade), item.NomeUnidade, formatarValor(item.PrecoUnitario))
		if item.Desconto > 0 {
			detalhe += " -" + formatarValor(item.Desconto)
		}
		duasColunas(tr(detalhe), formatarValor(item.Subtotal))
	}
	separador()

	duasColunas("Subtotal", formatarValor(d.Subtotal)+" MT")
	if d.DescontoTotal > 0 {
		duasColunas("Desconto", "-"+formatarValor(d.DescontoTotal)+" MT")
	}
	pdf.SetFont("Helvetica", "B", tamanho+3)
	w := pdf.GetStringWidth(formatarValor(d.Total)+" MT") + 1
	pdf.CellFormat(util-w, 7, "TOTAL", "", 0, "L", false, 0, "")
	pdf.CellFormat(w, 7, formatarValor(d.Total)+" MT", "", 1, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", tamanho)
	for _, pag := range d.Pagamentos {
		duasColunas(tr(pag.Metodo), formatarValor(pag.Valor)+" MT")
	}
	separador()
	pdf.CellFormat(util, linha, tr("Obrigado pela preferência!"), "", 1, "C", false, 0, "")

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gerar recibo de talão: %w", err)
	}
	return buf.Bytes(), nil
}

// partirTexto divide texto em linhas que cabem em largura (mm) com a fonte
// atual, palavra a palavra, e devolve-as já convertidas para a codificação
// do PDF. O SplitText do fpdf mede letra a letra num texto UTF-8 e rebenta
// com caracteres fora do Latin-1 (ex: "—").
func partirTexto(p *fpdf.Fpdf, tr func(string) string, texto string, largura float64) []string {
	var linhas []string
	atual := ""
	for _, palavra := range strings.Fields(texto) {
		tentativa := palavra
		if atual != "" {
			tentativa = atual + " " + palavra
		}
		if atual != "" && p.GetStringWidth(tr(tentativa)) > largura {
			linhas = append(linhas, tr(atual))
			atual = palavra
		} else {
			atual = tentativa
		}
	}
	if atual != "" || len(linhas) == 0 {
		linhas = append(linhas, tr(atual))
	}
	return linhas
}
