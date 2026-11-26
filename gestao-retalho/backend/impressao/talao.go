// Package impressao monta talões de texto para impressoras térmicas: os
// pedidos para o balcão/cozinha e a conta da mesa. O mesmo talão sai em
// ESC/POS (impressoras de rede Epson TM-T20, Xprinter…) ou em texto simples
// (a impressora embutida do Sunmi, que imprime linha a linha pelo SDK).
package impressao

import (
	"strings"
	"unicode/utf8"
)

type alinhamento int

const (
	esquerda alinhamento = iota
	centro
)

type linha struct {
	texto   string
	negrito bool
	grande  bool // largura e altura a dobrar
	alinh   alinhamento
	sep     bool
}

// Talao é uma sequência de linhas já partidas à largura do papel.
type Talao struct {
	Colunas int // 48 em 80 mm, 32 em 58 mm
	linhas  []linha
}

func NovoTalao(colunas int) *Talao {
	if colunas < 16 {
		colunas = 32
	}
	return &Talao{Colunas: colunas}
}

func (t *Talao) larg(grande bool) int {
	if grande {
		return t.Colunas / 2
	}
	return t.Colunas
}

func (t *Talao) add(texto string, negrito, grande bool, a alinhamento) *Talao {
	for _, l := range partir(texto, t.larg(grande)) {
		t.linhas = append(t.linhas, linha{texto: l, negrito: negrito, grande: grande, alinh: a})
	}
	return t
}

// Titulo: grande, em negrito e ao centro (ex: "MESA 5").
func (t *Talao) Titulo(s string) *Talao { return t.add(s, true, true, centro) }

// Grande: letra a dobrar, à esquerda — as linhas do pedido na cozinha.
func (t *Talao) Grande(s string) *Talao { return t.add(s, true, true, esquerda) }

func (t *Talao) Centro(s string) *Talao  { return t.add(s, false, false, centro) }
func (t *Talao) Texto(s string) *Talao   { return t.add(s, false, false, esquerda) }
func (t *Talao) Negrito(s string) *Talao { return t.add(s, true, false, esquerda) }

// Par escreve o texto à esquerda e o valor encostado à direita; se não
// couber, o texto parte-se e o valor vai na última linha.
func (t *Talao) Par(esq, dir string, negrito bool) *Talao {
	w := t.Colunas
	nd := utf8.RuneCountInString(dir)
	partes := partir(esq, w)
	if len(partes) == 0 {
		partes = []string{""}
	}
	ultima := partes[len(partes)-1]
	if utf8.RuneCountInString(ultima)+1+nd > w {
		partes = append(partes, "")
		ultima = ""
	}
	for _, p := range partes[:len(partes)-1] {
		t.linhas = append(t.linhas, linha{texto: p, negrito: negrito})
	}
	espacos := w - utf8.RuneCountInString(ultima) - nd
	t.linhas = append(t.linhas, linha{texto: ultima + strings.Repeat(" ", espacos) + dir, negrito: negrito})
	return t
}

func (t *Talao) Separador() *Talao {
	t.linhas = append(t.linhas, linha{sep: true})
	return t
}

func (t *Talao) Vazio() *Talao {
	t.linhas = append(t.linhas, linha{})
	return t
}

// TextoSimples devolve o talão como texto (uma linha por linha impressa),
// com o centrado feito a espaços — para a impressora do aparelho e para o
// pré-visualizar no ecrã.
func (t *Talao) TextoSimples() string {
	var b strings.Builder
	for _, l := range t.linhas {
		b.WriteString(t.renderizar(l, false))
		b.WriteByte('\n')
	}
	return b.String()
}

func (t *Talao) renderizar(l linha, escpos bool) string {
	if l.sep {
		return strings.Repeat("-", t.Colunas)
	}
	texto := l.texto
	// Em ESC/POS a letra grande ocupa duas colunas por carácter: o
	// centrado faz-se na largura reduzida. Em texto simples não há letra
	// grande, centra-se na largura toda.
	w := t.Colunas
	if escpos && l.grande {
		w = t.Colunas / 2
	}
	if l.alinh == centro {
		n := utf8.RuneCountInString(texto)
		if n < w {
			texto = strings.Repeat(" ", (w-n)/2) + texto
		}
	}
	return texto
}

// partir divide o texto em linhas de no máximo w caracteres, cortando nos
// espaços (uma palavra maior que a linha é cortada à força).
func partir(s string, w int) []string {
	if w < 1 {
		w = 1
	}
	var out []string
	for _, paragrafo := range strings.Split(s, "\n") {
		atual := ""
		for _, palavra := range strings.Fields(paragrafo) {
			for utf8.RuneCountInString(palavra) > w {
				if atual != "" {
					out = append(out, atual)
					atual = ""
				}
				r := []rune(palavra)
				out = append(out, string(r[:w]))
				palavra = string(r[w:])
			}
			switch {
			case atual == "":
				atual = palavra
			case utf8.RuneCountInString(atual)+1+utf8.RuneCountInString(palavra) <= w:
				atual += " " + palavra
			default:
				out = append(out, atual)
				atual = palavra
			}
		}
		out = append(out, atual)
	}
	return out
}
