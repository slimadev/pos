package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Loja é uma entrada no registo de lojas conhecidas pelo executável de
// administração universal (SDD 8.5, Fase 9) — não confundir com
// domain.Negocio, que vive dentro de cada base de loja e descreve o
// negócio em si (nome, tipo). Loja só sabe o essencial para lá chegar:
// como se chama no admin e onde está o ficheiro .db.
type Loja struct {
	ID        int64
	Nome      string
	CaminhoDB string
	Ativo     bool
	CriadoEm  time.Time
}

// LojaRepository é a única tabela da base do admin (admin.db) — pequena
// demais para justificar sqlc/schema próprios (sqlc.yaml já aponta só para
// o schema partilhado das lojas), por isso usa database/sql diretamente.
type LojaRepository struct {
	db *sql.DB
}

func NewLojaRepository(db *sql.DB) *LojaRepository {
	return &LojaRepository{db: db}
}

func (r *LojaRepository) Listar(ctx context.Context) ([]Loja, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, nome, caminho_db, ativo, criado_em FROM lojas ORDER BY nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Loja
	for rows.Next() {
		l, err := scanLoja(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *LojaRepository) ListarAtivas(ctx context.Context) ([]Loja, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, nome, caminho_db, ativo, criado_em FROM lojas WHERE ativo = 1 ORDER BY nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Loja
	for rows.Next() {
		l, err := scanLoja(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *LojaRepository) Criar(ctx context.Context, nome, caminhoDB string) (Loja, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO lojas (nome, caminho_db) VALUES (?, ?) RETURNING id, nome, caminho_db, ativo, criado_em`,
		nome, caminhoDB,
	)
	return scanLoja(row)
}

func (r *LojaRepository) Atualizar(ctx context.Context, id int64, nome, caminhoDB string) error {
	res, err := r.db.ExecContext(ctx, `UPDATE lojas SET nome = ?, caminho_db = ? WHERE id = ?`, nome, caminhoDB, id)
	if err != nil {
		return err
	}
	return exigirLinhaAfetada(res)
}

func (r *LojaRepository) DefinirAtiva(ctx context.Context, id int64, ativo bool) error {
	res, err := r.db.ExecContext(ctx, `UPDATE lojas SET ativo = ? WHERE id = ?`, boolToInt64(ativo), id)
	if err != nil {
		return err
	}
	return exigirLinhaAfetada(res)
}

func exigirLinhaAfetada(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("loja não encontrada")
	}
	return nil
}

// scanner cobre tanto *sql.Rows como *sql.Row — ambos implementam Scan.
type scanner interface {
	Scan(dest ...any) error
}

func scanLoja(s scanner) (Loja, error) {
	var l Loja
	var criadoEm string
	var ativo int64
	if err := s.Scan(&l.ID, &l.Nome, &l.CaminhoDB, &ativo, &criadoEm); err != nil {
		return Loja{}, err
	}
	l.Ativo = int64ToBool(ativo)
	l.CriadoEm = parseTimestamp(criadoEm)
	return l, nil
}
