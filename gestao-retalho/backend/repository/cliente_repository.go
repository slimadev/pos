package repository

import (
	"context"
	"database/sql"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository/sqlcgen"
)

type ClienteRepository struct {
	q *sqlcgen.Queries
}

func NewClienteRepository(db *sql.DB) *ClienteRepository {
	return &ClienteRepository{q: sqlcgen.New(db)}
}

// Listar exclui o registo fixo "Anónimo" (SDD 3.2) — é gestão de clientes
// reais, o anónimo nunca aparece numa lista para editar/apagar.
func (r *ClienteRepository) Listar(ctx context.Context, negocioID int64) ([]domain.Cliente, error) {
	rows, err := r.q.ListarClientes(ctx, negocioID)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Cliente, 0, len(rows))
	for _, row := range rows {
		out = append(out, clienteFromRow(row))
	}
	return out, nil
}

func (r *ClienteRepository) Obter(ctx context.Context, id int64) (domain.Cliente, error) {
	row, err := r.q.ObterCliente(ctx, id)
	if err != nil {
		return domain.Cliente{}, err
	}
	return clienteFromRow(row), nil
}

func (r *ClienteRepository) Criar(ctx context.Context, c domain.Cliente) (domain.Cliente, error) {
	row, err := r.q.CriarCliente(ctx, sqlcgen.CriarClienteParams{
		NegocioID:     c.NegocioID,
		Nome:          c.Nome,
		Telefone:      ptrToNullString(c.Telefone),
		Nif:           ptrToNullString(c.NIF),
		Morada:        ptrToNullString(c.Morada),
		LimiteCredito: ptrToNullFloat64(c.LimiteCredito),
	})
	if err != nil {
		return domain.Cliente{}, err
	}
	return clienteFromRow(row), nil
}

func (r *ClienteRepository) Atualizar(ctx context.Context, c domain.Cliente) error {
	return r.q.AtualizarCliente(ctx, sqlcgen.AtualizarClienteParams{
		Nome:          c.Nome,
		Telefone:      ptrToNullString(c.Telefone),
		Nif:           ptrToNullString(c.NIF),
		Morada:        ptrToNullString(c.Morada),
		LimiteCredito: ptrToNullFloat64(c.LimiteCredito),
		ID:            c.ID,
	})
}

func (r *ClienteRepository) DefinirAtivo(ctx context.Context, id int64, ativo bool) error {
	return r.q.DefinirClienteAtivo(ctx, sqlcgen.DefinirClienteAtivoParams{
		Ativo: boolToInt64(ativo),
		ID:    id,
	})
}

func clienteFromRow(row sqlcgen.Clientes) domain.Cliente {
	return domain.Cliente{
		ID:            row.ID,
		NegocioID:     row.NegocioID,
		Nome:          row.Nome,
		Telefone:      nullStringToPtr(row.Telefone),
		NIF:           nullStringToPtr(row.Nif),
		Morada:        nullStringToPtr(row.Morada),
		LimiteCredito: nullFloat64ToPtr(row.LimiteCredito),
		IsAnonimo:     int64ToBool(row.IsAnonimo),
		Ativo:         int64ToBool(row.Ativo),
	}
}
