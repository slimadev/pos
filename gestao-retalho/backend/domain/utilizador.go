package domain

type RoleUtilizador string

const (
	RoleAdmin       RoleUtilizador = "admin"
	RoleGerente     RoleUtilizador = "gerente"
	RoleOperadorPOS RoleUtilizador = "operador_pos"
)

// Utilizador — permissões por role descritas no SDD 4.13; o mecanismo de
// sessão/login em si só chega com a Fase 7.
type Utilizador struct {
	ID           int64
	NegocioID    *int64
	Nome         string
	Username     string
	PasswordHash string
	Role         RoleUtilizador
	Ativo        bool
}
