package domain

import "time"

type StatusConta string

const (
	StatusContaAberta    StatusConta = "aberta"
	StatusContaFechada   StatusConta = "fechada"
	StatusContaCancelada StatusConta = "cancelada"
)

// Conta é o mecanismo genérico de acumulação de itens antes de virar venda
// (SDD 3.5) — a venda direta (Fase 3) abre e fecha uma na mesma transação;
// a bottlestore (Fase 4) mantém-na aberta ao longo da noite.
type Conta struct {
	ID                   int64
	NegocioID            int64
	ClienteID            int64
	Identificador        string
	Status               StatusConta
	AbertaEm             time.Time
	FechadaEm            *time.Time
	UtilizadorAberturaID int64
	UtilizadorFechoID    *int64
	MotivoCancelamento   *string
}

type ItemConta struct {
	ID                    int64
	ContaID               int64
	ProdutoID             int64
	UnidadeProdutoID      int64
	Quantidade            float64
	PrecoUnitarioAplicado float64
	Desconto              float64
	CriadoEm              time.Time
	UtilizadorID          int64
}
