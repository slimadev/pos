package domain

type RoleUtilizador string

const (
	RoleAdmin       RoleUtilizador = "admin"
	RoleGerente     RoleUtilizador = "gerente"
	RoleOperadorPOS RoleUtilizador = "operador_pos"
)

// Utilizador — permissões por role descritas no SDD 4.13 (RBAC completo
// ainda não impõe limites por role, só guarda quem é quem). O wizard cria
// esta linha (nome/username/password) por loja, mas só para atribuição em
// `utilizador_id` (auditoria de vendas/movimentos, ver api.UtilizadorAtualID)
// — cmd/app nunca pede login com estas credenciais; autenticação real vive
// só no cmd/admin (repository.Administrador), decisão tomada depois da
// Fase 10 a pedido explícito.
type Utilizador struct {
	ID           int64
	NegocioID    *int64
	Nome         string
	Username     string
	PasswordHash string
	Role         RoleUtilizador
	Ativo        bool
}
