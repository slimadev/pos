package domain

// ClienteAnonimoID é o registo fixo semeado pela migration de vendas — ver
// nota em 000002_vendas.up.sql sobre a escolha de registo fixo em vez de FK
// nullable para "sem cliente identificado".
const ClienteAnonimoID int64 = 1

type Cliente struct {
	ID            int64
	NegocioID     int64
	Nome          string
	Telefone      *string
	NIF           *string
	Morada        *string
	Email         *string
	LimiteCredito *float64
	IsAnonimo     bool
	Ativo         bool
}
