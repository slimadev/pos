package repository

import (
	"database/sql"
	"time"
)

// Formato produzido por strftime('%Y-%m-%dT%H:%M:%fZ','now') nas migrations.
const timestampLayout = "2006-01-02T15:04:05.000Z"

func parseTimestamp(s string) time.Time {
	t, err := time.Parse(timestampLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func nullInt64ToPtr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

func ptrToNullInt64(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func nullFloat64ToPtr(n sql.NullFloat64) *float64 {
	if !n.Valid {
		return nil
	}
	v := n.Float64
	return &v
}

func ptrToNullFloat64(p *float64) sql.NullFloat64 {
	if p == nil {
		return sql.NullFloat64{}
	}
	return sql.NullFloat64{Float64: *p, Valid: true}
}

func nullStringToPtr(n sql.NullString) *string {
	if !n.Valid {
		return nil
	}
	v := n.String
	return &v
}

func ptrToNullString(p *string) sql.NullString {
	if p == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *p, Valid: true}
}

func boolToInt64(b bool) int64 {
	if b {
		return 1
	}
	return 0
}

func int64ToBool(i int64) bool {
	return i != 0
}

// anyToFloat64 lê o resultado de um COALESCE(SUM(coluna_real), 0) — SQLite
// devolve int64 quando a agregação não tocou nenhuma linha (o literal `0`
// vence por tipagem dinâmica) e float64 quando somou pelo menos uma linha
// REAL; os relatórios (Fase 8) usam isto em todas as colunas agregadas.
func anyToFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int64:
		return float64(n)
	default:
		return 0
	}
}

// interfaceArmazemFiltro converte um filtro opcional de armazém para o tipo
// `interface{}` que o sqlc gera para `sqlc.narg` usado em condições OR
// contra colunas de tabelas diferentes — nil desativa o filtro.
func interfaceArmazemFiltro(armazemID *int64) any {
	if armazemID == nil {
		return nil
	}
	return *armazemID
}
