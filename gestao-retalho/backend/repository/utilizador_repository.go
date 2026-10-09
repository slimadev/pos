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
	return utilizadorFromRow(row), nil
}

func (r *UtilizadorRepository) Obter(ctx context.Context, id int64) (domain.Utilizador, error) {
	row, err := r.q.ObterUtilizador(ctx, id)
	if err != nil {
		return domain.Utilizador{}, err
	}
	return utilizadorFromRow(row), nil
}

func (r *UtilizadorRepository) Listar(ctx context.Context) ([]domain.Utilizador, error) {
	rows, err := r.q.ListarUtilizadores(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Utilizador, 0, len(rows))
	for _, row := range rows {
		out = append(out, utilizadorFromRow(row))
	}
	return out, nil
}

func (r *UtilizadorRepository) Criar(ctx context.Context, u domain.Utilizador) (domain.Utilizador, error) {
	row, err := r.q.CriarUtilizador(ctx, sqlcgen.CriarUtilizadorParams{
		NegocioID:    ptrToNullInt64(u.NegocioID),
		Nome:         u.Nome,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Role:         string(u.Role),
	})
	if err != nil {
		return domain.Utilizador{}, err
	}
	return utilizadorFromRow(row), nil
}

func (r *UtilizadorRepository) AtualizarNome(ctx context.Context, id int64, nome string) error {
	return r.q.AtualizarNomeUtilizador(ctx, sqlcgen.AtualizarNomeUtilizadorParams{Nome: nome, ID: id})
}

func (r *UtilizadorRepository) AtualizarPassword(ctx context.Context, id int64, hash string) error {
	return r.q.AtualizarPasswordUtilizador(ctx, sqlcgen.AtualizarPasswordUtilizadorParams{PasswordHash: hash, ID: id})
}

func (r *UtilizadorRepository) DefinirAtivo(ctx context.Context, id int64, ativo bool) error {
	return r.q.DefinirUtilizadorAtivo(ctx, sqlcgen.DefinirUtilizadorAtivoParams{Ativo: boolToInt64(ativo), ID: id})
}

func utilizadorFromRow(row sqlcgen.Utilizadores) domain.Utilizador {
	return domain.Utilizador{
		ID:           row.ID,
		NegocioID:    nullInt64ToPtr(row.NegocioID),
		Nome:         row.Nome,
		Username:     row.Username,
		PasswordHash: row.PasswordHash,
		Role:         domain.RoleUtilizador(row.Role),
		Ativo:        int64ToBool(row.Ativo),
	}
}
