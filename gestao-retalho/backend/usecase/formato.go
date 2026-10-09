package usecase

import "strconv"

// qtd formata quantidades para mensagens ao utilizador: "2", "12", "1.5" —
// nunca "2.000", que em português se lê como dois mil.
func qtd(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}
