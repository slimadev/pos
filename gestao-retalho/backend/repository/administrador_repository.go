package repository

import (
	"context"
	"database/sql"
	"time"
)

// Administrador é o único utilizador do executável de administração
// universal (decisão pós-Fase 10, a pedido explícito: um só administrador
// para todas as lojas, não um por loja) — vive em admin.db, nunca dentro
// de uma base de loja. Não confundir com domain.Utilizador, que continua a
// existir por loja só para atribuição em `utilizador_id` (auditoria),
// nunca para login.
type Administrador struct {
	ID           int64
	Nome         string
	Username     string
	PasswordHash string
	CriadoEm     time.Time
}

// AdministradorRepository é a segunda tabela de admin.db (a par de
// LojaRepository) — mesma razão para usar database/sql diretamente em vez
// de sqlc: schema pequeno demais para justificar geração de código à
// parte.
type AdministradorRepository struct {
	db *sql.DB
}

func NewAdministradorRepository(db *sql.DB) *AdministradorRepository {
	return &AdministradorRepository{db: db}
}

func (r *AdministradorRepository) Existe(ctx context.Context) (bool, error) {
	var n int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM administradores`).Scan(&n); err != nil {
		return false, err
	}
	return n > 0, nil
}

func (r *AdministradorRepository) Criar(ctx context.Context, nome, username, passwordHash string) (Administrador, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO administradores (nome, username, password_hash) VALUES (?, ?, ?)
		 RETURNING id, nome, username, password_hash, criado_em`,
		nome, username, passwordHash,
	)
	return scanAdministrador(row)
}

func (r *AdministradorRepository) ObterPorUsername(ctx context.Context, username string) (Administrador, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, nome, username, password_hash, criado_em FROM administradores WHERE username = ?`,
		username,
	)
	return scanAdministrador(row)
}

func scanAdministrador(s scanner) (Administrador, error) {
	var a Administrador
	var criadoEm string
	if err := s.Scan(&a.ID, &a.Nome, &a.Username, &a.PasswordHash, &criadoEm); err != nil {
		return Administrador{}, err
	}
	a.CriadoEm = parseTimestamp(criadoEm)
	return a, nil
}
