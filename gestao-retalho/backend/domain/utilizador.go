package domain

type RoleUtilizador string

const (
	RoleAdmin       RoleUtilizador = "admin"
	RoleGerente     RoleUtilizador = "gerente"
	RoleOperadorPOS RoleUtilizador = "operador_pos"
)

// Utilizador — permissões por role descritas no SDD 4.13 (RBAC completo
// ainda não impõe limites por role, só guarda quem é quem); o mecanismo de
// sessão/login em si (usecase.AuthService, api.App.Login) chegou depois da
// Fase 10, a pedido explícito.
type Utilizador struct {
	ID           int64
	NegocioID    *int64
	Nome         string
	Username     string
	PasswordHash string
	Role         RoleUtilizador
	Ativo        bool
}
