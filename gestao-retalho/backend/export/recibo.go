// Package export gera os documentos que saem da app para fora do ecrã:
// recibos em PDF por agora (SDD 4.10), relatórios em Excel/PDF a partir
// da Fase 8 (SDD secção 6.1).
package export

import (
	"bytes"
	"fmt"
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
	VendaID       int64
	CriadoEm      time.Time
	ClienteNome   string
	Itens         []ItemRecibo
	Subtotal      float64
	DescontoTotal float64
	Total         float64
	Pagamentos    []PagamentoRecibo
}

// GerarReciboPDF produz um recibo simples em formato de talão (A5) — sem
// logotipo nem layout rico; o SDD 6.1 reserva isso para quando os
// relatórios com gráfico chegarem via HTML+chromedp na Fase 8.
func GerarReciboPDF(d DadosRecibo) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A5", "")
	pdf.AddPage()

	pdf.SetFont("Helvetica", "B", 14)
	pdf.CellFormat(0, 8, d.NegocioNome, "", 1, "C", false, 0, "")

	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(0, 6, fmt.Sprintf("Recibo #%d", d.VendaID), "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 6, d.CriadoEm.Format("02/01/2006 15:04"), "", 1, "C", false, 0, "")
	pdf.Ln(4)
	pdf.CellFormat(0, 6, "Cliente: "+d.ClienteNome, "", 1, "L", false, 0, "")
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(70, 6, "Produto", "B", 0, "L", false, 0, "")
	pdf.CellFormat(30, 6, "Qtd", "B", 0, "R", false, 0, "")
	pdf.CellFormat(30, 6, "Preco", "B", 0, "R", false, 0, "")
	pdf.CellFormat(30, 6, "Subtotal", "B", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "", 9)
	for _, item := range d.Itens {
		pdf.CellFormat(70, 6, item.NomeProduto, "", 0, "L", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.2f %s", item.Quantidade, item.NomeUnidade), "", 0, "R", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.2f", item.PrecoUnitario), "", 0, "R", false, 0, "")
		pdf.CellFormat(30, 6, fmt.Sprintf("%.2f", item.Subtotal), "", 1, "R", false, 0, "")
	}

	pdf.Ln(4)
	pdf.SetFont("Helvetica", "", 10)
	pdf.CellFormat(130, 6, "Subtotal", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 6, fmt.Sprintf("%.2f MT", d.Subtotal), "", 1, "R", false, 0, "")
	pdf.CellFormat(130, 6, "Desconto", "", 0, "R", false, 0, "")
	pdf.CellFormat(30, 6, fmt.Sprintf("-%.2f MT", d.DescontoTotal), "", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "B", 12)
	pdf.CellFormat(130, 8, "Total", "T", 0, "R", false, 0, "")
	pdf.CellFormat(30, 8, fmt.Sprintf("%.2f MT", d.Total), "T", 1, "R", false, 0, "")

	pdf.Ln(6)
	pdf.SetFont("Helvetica", "", 9)
	for _, pag := range d.Pagamentos {
		pdf.CellFormat(0, 6, fmt.Sprintf("%s: %.2f MT", pag.Metodo, pag.Valor), "", 1, "L", false, 0, "")
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gerar recibo PDF: %w", err)
	}
	return buf.Bytes(), nil
}
