package domain

type RoleUtilizador string

const (
	RoleAdmin       RoleUtilizador = "admin"
	RoleGerente     RoleUtilizador = "gerente"
	RoleOperadorPOS RoleUtilizador = "operador_pos"
)

// Utilizador — permissões por role descritas no SDD 4.13 (RBAC completo
// ainda não impõe limites por role, só guarda quem é quem). O utilizador
// semente (id 1, o administrador) recebe tudo o que o cmd/admin faz; os
// operadores (role operador_pos) identificam-se no PDV com um PIN e ficam
// com as vendas/movimentos do seu turno (ver api.App.utilizadorAtual). O
// login completo continua a existir só no cmd/admin (repository.Administrador).
type Utilizador struct {
	ID           int64
	NegocioID    *int64
	Nome         string
	Username     string
	PasswordHash string
	Role         RoleUtilizador
	Ativo        bool
	// Caixa: no restaurante fecha contas, vende direto e mexe nas mesas
	// dos outros garçons (o gerente também).
	Caixa bool
}

// PodeGerirMesas: caixa ou gerente — fecha contas e mexe nas mesas dos
// outros (anular, dividir, passar a outro garçom), mas não lança nelas.
func (u Utilizador) PodeGerirMesas() bool {
	return u.Caixa || u.Role == RoleGerente || u.Role == RoleAdmin
}
