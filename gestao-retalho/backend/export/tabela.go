package export

import (
	"bytes"
	"fmt"

	"github.com/go-pdf/fpdf"
	"github.com/xuri/excelize/v2"
)

// Tabela é a forma genérica que qualquer um dos relatórios da Fase 8 (SDD
// secção 6) assume antes de ser exportado — o ecrã de relatórios já
// formatou cada valor como string para o preview em tabela, por isso a
// exportação reutiliza exatamente as mesmas linhas em vez de reconstruir a
// formatação a partir dos dados tipados outra vez.
type Tabela struct {
	Titulo  string
	Colunas []string
	Linhas  [][]string
	// Totais é opcional — vazio para omitir, senão do mesmo tamanho que
	// Colunas (string vazia nas colunas sem total) e sai a negrito.
	Totais []string
}

// GerarTabelaExcel produz um .xlsx de uma folha só: título, cabeçalho a
// negrito, linhas de dados, e totais a negrito no fim quando presentes
// (SDD 6.1).
func GerarTabelaExcel(t Tabela) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const folha = "Relatório"
	f.SetSheetName(f.GetSheetName(0), folha)

	negrito, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return nil, fmt.Errorf("estilo negrito: %w", err)
	}
	tituloEstilo, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true, Size: 14}})
	if err != nil {
		return nil, fmt.Errorf("estilo título: %w", err)
	}

	linha := 1
	if t.Titulo != "" {
		cell, _ := excelize.CoordinatesToCellName(1, linha)
		f.SetCellValue(folha, cell, t.Titulo)
		f.SetCellStyle(folha, cell, cell, tituloEstilo)
		linha += 2
	}

	cabecalhoLinha := linha
	for col, nome := range t.Colunas {
		cell, _ := excelize.CoordinatesToCellName(col+1, cabecalhoLinha)
		f.SetCellValue(folha, cell, nome)
	}
	if len(t.Colunas) > 0 {
		fim, _ := excelize.CoordinatesToCellName(len(t.Colunas), cabecalhoLinha)
		inicio, _ := excelize.CoordinatesToCellName(1, cabecalhoLinha)
		f.SetCellStyle(folha, inicio, fim, negrito)
	}
	linha++

	for _, valores := range t.Linhas {
		for col, valor := range valores {
			cell, _ := excelize.CoordinatesToCellName(col+1, linha)
			f.SetCellValue(folha, cell, valor)
		}
		linha++
	}

	if len(t.Totais) > 0 {
		for col, valor := range t.Totais {
			if valor == "" {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(col+1, linha)
			f.SetCellValue(folha, cell, valor)
		}
		inicio, _ := excelize.CoordinatesToCellName(1, linha)
		fim, _ := excelize.CoordinatesToCellName(len(t.Colunas), linha)
		f.SetCellStyle(folha, inicio, fim, negrito)
	}

	for col := range t.Colunas {
		letra, _ := excelize.ColumnNumberToName(col + 1)
		f.SetColWidth(folha, letra, letra, 18)
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("gerar excel: %w", err)
	}
	return buf.Bytes(), nil
}

// GerarTabelaPDF produz um PDF tabular simples em A4 paisagem — layout
// consistente com o resto dos relatórios, sem gráficos (SDD 6.1 reserva
// gráficos para uma geração via HTML/chromedp que fica fora do âmbito
// desta fase).
func GerarTabelaPDF(t Tabela) ([]byte, error) {
	pdf := fpdf.New("L", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetMargins(10, 10, 10)

	if t.Titulo != "" {
		pdf.SetFont("Helvetica", "B", 14)
		pdf.CellFormat(0, 10, t.Titulo, "", 1, "L", false, 0, "")
		pdf.Ln(2)
	}

	larguraDisponivel := 277.0
	numCols := len(t.Colunas)
	if numCols == 0 {
		numCols = 1
	}
	largura := larguraDisponivel / float64(numCols)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	for _, nome := range t.Colunas {
		pdf.CellFormat(largura, 7, nome, "1", 0, "L", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Helvetica", "", 8)
	for _, valores := range t.Linhas {
		for col := 0; col < numCols; col++ {
			valor := ""
			if col < len(valores) {
				valor = valores[col]
			}
			pdf.CellFormat(largura, 6, valor, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
		if pdf.GetY() > 190 {
			pdf.AddPage()
		}
	}

	if len(t.Totais) > 0 {
		pdf.SetFont("Helvetica", "B", 8)
		for col := 0; col < numCols; col++ {
			valor := ""
			if col < len(t.Totais) {
				valor = t.Totais[col]
			}
			pdf.CellFormat(largura, 6, valor, "1", 0, "L", false, 0, "")
		}
		pdf.Ln(-1)
	}

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("gerar relatório PDF: %w", err)
	}
	return buf.Bytes(), nil
}
