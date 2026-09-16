package repository

import (
	"context"
	"database/sql"
	"errors"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

// ConfigRepository lê/escreve a tabela `configuracoes` (SDD 3.2/9.3) — o
// espelho consultável em runtime por qualquer service, sem reler ficheiros.
type ConfigRepository struct {
	q *sqlcgen.Queries
}

func NewConfigRepository(db *sql.DB) *ConfigRepository {
	return &ConfigRepository{q: sqlcgen.New(db)}
}

func (r *ConfigRepository) Obter(ctx context.Context, chave string) (valor string, ok bool, err error) {
	row, err := r.q.ObterConfiguracao(ctx, sqlcgen.ObterConfiguracaoParams{
		NegocioID: domain.NegocioID,
		Chave:     chave,
	})
	if errors.Is(err, sql.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.Valor, true, nil
}

func (r *ConfigRepository) Definir(ctx context.Context, chave, valor string) error {
	return r.q.DefinirConfiguracao(ctx, sqlcgen.DefinirConfiguracaoParams{
		NegocioID: domain.NegocioID,
		Chave:     chave,
		Valor:     valor,
	})
}

func (r *ConfigRepository) Listar(ctx context.Context) (map[string]string, error) {
	rows, err := r.q.ListarConfiguracoes(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	out := make(map[string]string, len(rows))
	for _, row := range rows {
		out[row.Chave] = row.Valor
	}
	return out, nil
}
