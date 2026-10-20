package export

import (
	"strconv"
	"strings"
)

// formatarValor escreve dinheiro como em pt-PT: "1 234,56".
func formatarValor(v float64) string {
	negativo := v < 0
	if negativo {
		v = -v
	}
	partes := strings.SplitN(strconv.FormatFloat(v, 'f', 2, 64), ".", 2)
	inteiro := partes[0]
	var agrupado strings.Builder
	for i, c := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			agrupado.WriteByte(' ')
		}
		agrupado.WriteRune(c)
	}
	s := agrupado.String() + "," + partes[1]
	if negativo {
		s = "-" + s
	}
	return s
}

// formatarQtd mostra só as casas decimais necessárias: "3", "1,5".
func formatarQtd(v float64) string {
	return strings.Replace(strconv.FormatFloat(v, 'f', -1, 64), ".", ",", 1)
}

// FormatarValor é formatarValor para quem monta tabelas fora deste pacote
// ("1 234,56").
func FormatarValor(v float64) string { return formatarValor(v) }
