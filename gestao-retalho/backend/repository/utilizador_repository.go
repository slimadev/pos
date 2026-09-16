package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type UtilizadorRepository struct {
	q *sqlcgen.Queries
}

func NewUtilizadorRepository(db *sql.DB) *UtilizadorRepository {
	return &UtilizadorRepository{q: sqlcgen.New(db)}
}

// AtualizarAdmin substitui os dados do utilizador semente (id=1, ver
// migration core) pelos dados reais recolhidos no passo 4 do wizard —
// nunca cria um segundo utilizador, só completa o que já existia.
func (r *UtilizadorRepository) AtualizarAdmin(ctx context.Context, id int64, nome, username, passwordHash string) error {
	return r.q.AtualizarUtilizadorAdmin(ctx, sqlcgen.AtualizarUtilizadorAdminParams{
		Nome:         nome,
		Username:     username,
		PasswordHash: passwordHash,
		ID:           id,
	})
}

func (r *UtilizadorRepository) ObterPorUsername(ctx context.Context, username string) (domain.Utilizador, error) {
	row, err := r.q.ObterUtilizadorPorUsername(ctx, username)
	if err != nil {
		return domain.Utilizador{}, err
	}
	return domain.Utilizador{
		ID:           row.ID,
		NegocioID:    nullInt64ToPtr(row.NegocioID),
		Nome:         row.Nome,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		Role:         domain.RoleUtilizador(row.Role),
		Ativo:        int64ToBool(row.Ativo),
	}, nil
}
