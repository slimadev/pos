package repository

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

// Loja é o registo, em admin.db, da única loja que este cmd/admin serve
// (decisão pós-Fase 10, a pedido explícito — um administrador, uma loja,
// que pode ter vários armazéns internamente). Não confundir com
// domain.Negocio, que vive dentro da base da loja e descreve o negócio em
// si (nome, tipo); Loja só sabe o essencial para lá chegar: como se chama
// no admin e onde está o ficheiro .db.
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

// ObterUnica devolve a única loja registada (existe=false antes do
// primeiro arranque terminar de a criar) — a tabela nunca tem mais do que
// uma linha, já que só ConfigurarPrimeiroArranque grava aqui.
func (r *LojaRepository) ObterUnica(ctx context.Context) (loja Loja, existe bool, err error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, nome, caminho_db, ativo, criado_em FROM lojas LIMIT 1`)
	loja, err = scanLoja(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Loja{}, false, nil
	}
	if err != nil {
		return Loja{}, false, err
	}
	return loja, true, nil
}

func (r *LojaRepository) Criar(ctx context.Context, nome, caminhoDB string) (Loja, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO lojas (nome, caminho_db) VALUES (?, ?) RETURNING id, nome, caminho_db, ativo, criado_em`,
		nome, caminhoDB,
	)
	return scanLoja(row)
}

// AtualizarNome acompanha a mudança de nome do negócio dentro da loja —
// o registo do admin mostra o mesmo nome.
func (r *LojaRepository) AtualizarNome(ctx context.Context, id int64, nome string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE lojas SET nome = ? WHERE id = ?`, nome, id)
	return err
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
