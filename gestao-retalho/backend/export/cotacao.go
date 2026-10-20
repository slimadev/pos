package export

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

// ItemCotacaoPDF é uma linha de um documento A4 (cotação ou documento fiscal).
type ItemCotacaoPDF struct {
	Descricao     string
	NomeUnidade   string
	Quantidade    float64
	PrecoUnitario float64
	Desconto      float64
	Subtotal      float64
}

// DadosDocumentoA4 é o que se desenha num documento A4: cotação, fatura,
// fatura-recibo, recibo ou nota de crédito.
type DadosDocumentoA4 struct {
	Titulo          string // "COTAÇÃO", "FATURA-RECIBO"…
	Codigo          string // "COT 2026/0007", "FR 2026/0012"
	CriadoEm        time.Time
	Cabecalho       []string // linhas por baixo da data (validade, documento de origem)
	DiscriminarIVA  bool     // documentos fiscais: base tributável + IVA + total
	Rodape          string
	Loja            DadosRecibo // só os campos da loja (nome, NUIT, endereço, contactos)
	Logo            string      // data URL PNG/JPEG, "" = sem logótipo
	ClienteNome     string
	ClienteNUIT     string
	ClienteContacto string
	Itens           []ItemCotacaoPDF
	Subtotal        float64
	DescontoTotal   float64
	Total           float64
	TaxaIVA         float64
	ValorIVA        float64
	Observacoes     string
}

// DadosCotacaoPDF mantém o nome antigo para as cotações.
type DadosCotacaoPDF = DadosDocumentoA4

// GerarCotacaoPDF produz a cotação em A4, com a nota de que não é fatura.
func GerarCotacaoPDF(d DadosCotacaoPDF, validaAte time.Time) ([]byte, error) {
	d.Titulo = "COTAÇÃO"
	d.Cabecalho = append([]string{"Válida até: " + validaAte.Format("02/01/2006")}, d.Cabecalho...)
	d.Rodape = "Cotação — documento sem valor fiscal, não é fatura. Preços com IVA incluído."
	return GerarDocumentoA4(d)
}

// GerarDocumentoA4 produz o documento em A4: loja e logótipo no topo,
// título, número e datas, cliente, tabela de produtos e totais com o IVA.
func GerarDocumentoA4(d DadosDocumentoA4) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(15, 15, 15)
	pdf.SetAutoPageBreak(true, 18)
	tr := pdf.UnicodeTranslatorFromDescriptor("")
	pdf.SetFooterFunc(func() {
		pdf.SetY(-13)
		pdf.SetFont("Helvetica", "I", 8)
		pdf.SetTextColor(110, 110, 110)
		pdf.CellFormat(0, 5, tr(d.Rodape), "", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
	})
	pdf.AddPage()

	const largura = 180.0
	x0, y0 := 15.0, 15.0
	textoX := x0
	if img, tipo := imagemDeDataURL(d.Logo); img != nil {
		pdf.RegisterImageOptionsReader("logo", fpdf.ImageOptions{ImageType: tipo}, bytes.NewReader(img))
		if pdf.Ok() {
			pdf.ImageOptions("logo", x0, y0, 0, 22, false, fpdf.ImageOptions{ImageType: tipo}, 0, "")
			textoX = x0 + 28
		}
		pdf.ClearError()
	}

	// Loja (esquerda) e título (direita).
	pdf.SetXY(textoX, y0)
	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(100, 7, tr(d.Loja.NegocioNome), "", 2, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	for _, l := range d.Loja.LinhasLoja() {
		pdf.CellFormat(100, 4.5, tr(l), "", 2, "L", false, 0, "")
	}
	yLoja := pdf.GetY()

	pdf.SetXY(x0+largura-70, y0)
	pdf.SetFont("Helvetica", "B", 18)
	pdf.CellFormat(70, 9, tr(d.Titulo), "", 2, "R", false, 0, "")
	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(70, 6, d.Codigo, "", 2, "R", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(70, 5, "Data: "+d.CriadoEm.Local().Format("02/01/2006"), "", 2, "R", false, 0, "")
	for _, l := range d.Cabecalho {
		pdf.CellFormat(70, 5, tr(l), "", 2, "R", false, 0, "")
	}

	y := maxF(yLoja, pdf.GetY(), y0+24) + 6

	// Cliente.
	pdf.SetXY(x0, y)
	pdf.SetFillColor(244, 245, 247)
	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(largura, 6, "  Cliente", "", 1, "L", true, 0, "")
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(largura, 6, "  "+tr(d.ClienteNome), "", 1, "L", false, 0, "")
	if d.ClienteNUIT != "" {
		pdf.CellFormat(largura, 5, "  NUIT: "+d.ClienteNUIT, "", 1, "L", false, 0, "")
	}
	if d.ClienteContacto != "" {
		pdf.CellFormat(largura, 5, "  "+tr(d.ClienteContacto), "", 1, "L", false, 0, "")
	}
	pdf.Ln(5)

	// Produtos.
	cols := []float64{78, 22, 20, 22, 18, 20}
	cab := []string{"Produto", "Qtd", "Unidade", "Preço", "Desc.", "Total"}
	alin := []string{"L", "R", "L", "R", "R", "R"}
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(30, 30, 30)
	pdf.SetTextColor(255, 255, 255)
	for i, c := range cab {
		pdf.CellFormat(cols[i], 7, tr(c), "", 0, alin[i], true, 0, "")
	}
	pdf.Ln(-1)
	pdf.SetTextColor(0, 0, 0)
	pdf.SetFont("Helvetica", "", 9)
	for n, it := range d.Itens {
		fundo := n%2 == 1
		pdf.SetFillColor(248, 248, 250)
		nome := tr(it.Descricao)
		for pdf.GetStringWidth(nome) > cols[0]-2 && len(nome) > 1 {
			nome = nome[:len(nome)-1]
		}
		desc := "—"
		if it.Desconto > 0 {
			desc = formatarValor(it.Desconto)
		}
		valores := []string{nome, formatarQtd(it.Quantidade), tr(it.NomeUnidade), formatarValor(it.PrecoUnitario), tr(desc), formatarValor(it.Subtotal)}
		for i, v := range valores {
			pdf.CellFormat(cols[i], 6.5, v, "", 0, alin[i], fundo, 0, "")
		}
		pdf.Ln(-1)
	}
	pdf.Line(x0, pdf.GetY(), x0+largura, pdf.GetY())
	pdf.Ln(3)

	// Totais.
	linhaTotal := func(rotulo, valor string, negrito bool) {
		estilo := ""
		tam := 10.0
		if negrito {
			estilo, tam = "B", 12
		}
		pdf.SetFont("Helvetica", estilo, tam)
		pdf.SetX(x0 + largura - 80)
		pdf.CellFormat(45, 6.5, tr(rotulo), "", 0, "L", false, 0, "")
		pdf.CellFormat(35, 6.5, valor, "", 1, "R", false, 0, "")
	}
	linhaTotal("Subtotal", formatarValor(d.Subtotal)+" MT", false)
	if d.DescontoTotal > 0 {
		linhaTotal("Desconto", "-"+formatarValor(d.DescontoTotal)+" MT", false)
	}
	if d.DiscriminarIVA {
		linhaTotal("Base tributável", formatarValor(d.Total-d.ValorIVA)+" MT", false)
		if d.TaxaIVA > 0 {
			linhaTotal(fmt.Sprintf("IVA (%s%%)", formatarQtd(d.TaxaIVA)), formatarValor(d.ValorIVA)+" MT", false)
		} else {
			linhaTotal("IVA (isento)", formatarValor(0)+" MT", false)
		}
	}
	linhaTotal("TOTAL", formatarValor(d.Total)+" MT", true)
	pdf.SetFont("Helvetica", "", 9)
	pdf.SetX(x0 + largura - 80)
	// Sem discriminação (cotação), uma nota com o IVA que o total inclui.
	switch {
	case d.DiscriminarIVA:
	case d.TaxaIVA > 0:
		pdf.CellFormat(80, 5, tr(fmt.Sprintf("Inclui IVA (%s%%): %s MT", formatarQtd(d.TaxaIVA), formatarValor(d.ValorIVA))), "", 1, "R", false, 0, "")
	default:
		pdf.CellFormat(80, 5, tr("Isento de IVA"), "", 1, "R", false, 0, "")
	}

	if obs := strings.TrimSpace(d.Observacoes); obs != "" {
		pdf.Ln(6)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.CellFormat(largura, 5, tr("Observações"), "", 1, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 9)
		pdf.MultiCell(largura, 4.5, tr(obs), "", "L", false)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gerar documento PDF: %w", err)
	}
	return buf.Bytes(), nil
}

func maxF(vs ...float64) float64 {
	m := vs[0]
	for _, v := range vs[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// imagemDeDataURL devolve os bytes e o tipo ("PNG"/"JPG") de um logótipo
// em data URL; outros formatos (ex: SVG, WebP) ficam de fora do PDF.
func imagemDeDataURL(url string) ([]byte, string) {
	var tipo string
	switch {
	case strings.HasPrefix(url, "data:image/png;base64,"):
		tipo = "PNG"
	case strings.HasPrefix(url, "data:image/jpeg;base64,"), strings.HasPrefix(url, "data:image/jpg;base64,"):
		tipo = "JPG"
	default:
		return nil, ""
	}
	b, err := base64.StdEncoding.DecodeString(url[strings.Index(url, ",")+1:])
	if err != nil {
		return nil, ""
	}
	return b, tipo
}
