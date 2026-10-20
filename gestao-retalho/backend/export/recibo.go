// Package export gera os documentos que saem da app para fora do ecrã:
// recibos em PDF por agora (SDD 4.10), relatórios em Excel/PDF a partir
// da Fase 8 (SDD secção 6.1).
package export

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/go-pdf/fpdf"
)

type ItemRecibo struct {
	NomeProduto   string
	NomeUnidade   string
	Quantidade    float64
	PrecoUnitario float64
	Desconto      float64
	Subtotal      float64
}

type PagamentoRecibo struct {
	Metodo string
	Valor  float64
}

type DadosRecibo struct {
	NegocioNome   string
	LojaNUIT      string
	LojaEndereco  string
	LojaTelefone  string
	LojaEmail     string
	VendaID       int64
	CriadoEm      time.Time
	ClienteNome   string
	ClienteNUIT   string
	Itens         []ItemRecibo
	Subtotal      float64
	DescontoTotal float64
	Total         float64
	Pagamentos    []PagamentoRecibo
}

// GerarReciboPDF produz um recibo simples em formato de talão (A5) — sem
// logotipo nem layout rico; o SDD 6.1 reserva isso para quando os
// relatórios com gráfico chegarem via HTML+chromedp na Fase 8.
//
// As fontes base do PDF (Helvetica) só conhecem Windows-1252: todo o texto
// passa por `tr`, senão "Farmácia" sai "FarmÃ¡cia".
func GerarReciboPDF(d DadosRecibo) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A5", "")
	pdf.SetMargins(10, 10, 10)
	pdf.AddPage()
	tr := pdf.UnicodeTranslatorFromDescriptor("")

	// A5 tem 148 mm; com 10 mm de margem de cada lado ficam 128 mm.
	const (
		colProduto  = 56.0
		colQtd      = 24.0
		colPreco    = 22.0
		colSubtotal = 26.0
		colRotulo   = colProduto + colQtd + colPreco
	)

	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, tr(d.NegocioNome), "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for _, l := range d.LinhasLoja() {
		pdf.CellFormat(0, 4.5, tr(l), "", 1, "C", false, 0, "")
	}
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Recibo #%d", d.VendaID), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 6, d.CriadoEm.Local().Format("02/01/2006 15:04"), "", 1, "C", false, 0, "")
	pdf.Ln(4)
	pdf.CellFormat(0, 6, tr("Cliente: "+d.ClienteNome), "", 1, "L", false, 0, "")
	if d.ClienteNUIT != "" {
		pdf.CellFormat(0, 6, "NUIT do cliente: "+d.ClienteNUIT, "", 1, "L", false, 0, "")
	}
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(colProduto, 6, "Produto", "B", 0, "L", false, 0, "")
	pdf.CellFormat(colQtd, 6, "Qtd", "B", 0, "R", false, 0, "")
	pdf.CellFormat(colPreco, 6, tr("Preço"), "B", 0, "R", false, 0, "")
	pdf.CellFormat(colSubtotal, 6, "Subtotal", "B", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for _, item := range d.Itens {
		nome := tr(item.NomeProduto)
		for pdf.GetStringWidth(nome) > colProduto-1 && len(nome) > 1 {
			nome = nome[:len(nome)-1]
		}
		pdf.CellFormat(colProduto, 6, nome, "", 0, "L", false, 0, "")
		pdf.CellFormat(colQtd, 6, tr(formatarQtd(item.Quantidade)+" "+item.NomeUnidade), "", 0, "R", false, 0, "")
		pdf.CellFormat(colPreco, 6, formatarValor(item.PrecoUnitario), "", 0, "R", false, 0, "")
		pdf.CellFormat(colSubtotal, 6, formatarValor(item.Subtotal), "", 1, "R", false, 0, "")
	}

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(colRotulo, 6, "Subtotal", "", 0, "R", false, 0, "")
	pdf.CellFormat(colSubtotal, 6, formatarValor(d.Subtotal)+" MT", "", 1, "R", false, 0, "")
	if d.DescontoTotal > 0 {
		pdf.CellFormat(colRotulo, 6, "Desconto", "", 0, "R", false, 0, "")
		pdf.CellFormat(colSubtotal, 6, "-"+formatarValor(d.DescontoTotal)+" MT", "", 1, "R", false, 0, "")
	}

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(colRotulo, 8, "Total", "T", 0, "R", false, 0, "")
	pdf.CellFormat(colSubtotal, 8, formatarValor(d.Total)+" MT", "T", 1, "R", false, 0, "")

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 9)
	for _, pag := range d.Pagamentos {
		pdf.CellFormat(colRotulo, 6, tr(pag.Metodo), "", 0, "R", false, 0, "")
		pdf.CellFormat(colSubtotal, 6, formatarValor(pag.Valor)+" MT", "", 1, "R", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gerar recibo PDF: %w", err)
	}
	return buf.Bytes(), nil
}

// LinhasLoja são as linhas de identificação da loja por baixo do nome
// (NUIT, endereço, contactos), só as que estão preenchidas.
func (d DadosRecibo) LinhasLoja() []string {
	var linhas []string
	if d.LojaNUIT != "" {
		linhas = append(linhas, "NUIT "+d.LojaNUIT)
	}
	if d.LojaEndereco != "" {
		linhas = append(linhas, d.LojaEndereco)
	}
	var contactos []string
	if d.LojaTelefone != "" {
		contactos = append(contactos, "Tel. "+d.LojaTelefone)
	}
	if d.LojaEmail != "" {
		contactos = append(contactos, d.LojaEmail)
	}
	if len(contactos) > 0 {
		linhas = append(linhas, strings.Join(contactos, " · "))
	}
	return linhas
}
