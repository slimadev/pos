package usecase

import (
	"strconv"
	"strings"
)

// qtd formata quantidades para mensagens ao utilizador: "2", "12", "1.5" —
// nunca "2.000", que em português se lê como dois mil.
func qtd(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

var semAcentos = strings.NewReplacer(
	"á", "a", "à", "a", "â", "a", "ã", "a", "é", "e", "ê", "e", "í", "i",
	"ó", "o", "ô", "o", "õ", "o", "ú", "u", "ç", "c",
)

// mesmoNome compara nomes como uma pessoa os leria: sem maiúsculas, acentos
// nem espaços a mais ("Eletrico " é o mesmo que "Elétrico").
func mesmoNome(a, b string) bool {
	norm := func(s string) string {
		return semAcentos.Replace(strings.Join(strings.Fields(strings.ToLower(s)), " "))
	}
	return norm(a) == norm(b)
}
