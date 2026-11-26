package impressao

import (
	"bytes"

	"golang.org/x/text/encoding/charmap"
)

// Comandos ESC/POS comuns à Epson TM-T20 e às Xprinter.
var (
	cmdIniciar      = []byte{0x1b, '@'}
	cmdPagina860    = []byte{0x1b, 't', 3} // PC860 (português): ã, ç, õ…
	cmdNegritoOn    = []byte{0x1b, 'E', 1}
	cmdNegritoOff   = []byte{0x1b, 'E', 0}
	cmdGrandeOn     = []byte{0x1d, '!', 0x11} // largura e altura a dobrar
	cmdGrandeOff    = []byte{0x1d, '!', 0x00}
	cmdAvancarCorte = []byte{0x1d, 'V', 66, 3} // avança 3 linhas e corta
)

// ESCPOS codifica o talão para uma impressora térmica de rede: página de
// código PC860, negrito e letra grande onde pedido, e corte do papel.
func (t *Talao) ESCPOS() []byte {
	var b bytes.Buffer
	b.Write(cmdIniciar)
	b.Write(cmdPagina860)
	enc := charmap.CodePage860.NewEncoder()
	for _, l := range t.linhas {
		if l.negrito {
			b.Write(cmdNegritoOn)
		}
		if l.grande {
			b.Write(cmdGrandeOn)
		}
		for _, texto := range t.renderizar(l, true) {
			cod, err := enc.String(texto)
			if err != nil {
				cod = paraASCII(texto)
			}
			b.WriteString(cod)
			b.WriteByte('\n')
		}
		if l.grande {
			b.Write(cmdGrandeOff)
		}
		if l.negrito {
			b.Write(cmdNegritoOff)
		}
	}
	b.Write(cmdAvancarCorte)
	return b.Bytes()
}

// paraASCII troca o que a PC860 não tem por "?" — um talão com um carácter
// estranho é melhor do que um pedido que não sai na cozinha.
func paraASCII(s string) string {
	enc := charmap.CodePage860.NewEncoder()
	var b bytes.Buffer
	for _, r := range s {
		c, err := enc.String(string(r))
		if err != nil {
			b.WriteByte('?')
			continue
		}
		b.WriteString(c)
	}
	return b.String()
}
