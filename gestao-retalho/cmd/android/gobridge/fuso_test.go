package gobridge

import (
	"database/sql"
	"runtime"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestTzPosix(t *testing.T) {
	inverno := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC)
	casos := []struct{ fuso, quer string }{
		{"Africa/Maputo", "<+0200>-2"},
		{"Asia/Shanghai", "<+0800>-8"},
		{"UTC", "<+0000>0"},
		{"Asia/Kolkata", "<+0530>-5:30"},
		{"America/New_York", "<-0500>5"},
	}
	for _, c := range casos {
		loc, err := time.LoadLocation(c.fuso)
		if err != nil {
			t.Fatal(err)
		}
		if got := tzPosix(loc, inverno); got != c.quer {
			t.Errorf("tzPosix(%s) = %q, want %q", c.fuso, got, c.quer)
		}
	}
}

// O bug visto no terminal: uma venda às 04:16 (+08:00) a 28/10 aparecia no
// dia 27 no gráfico, porque `date(x, 'localtime')` no SQLite estava em UTC.
func TestSQLiteAgrupaPeloDiaLocal(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("só a libc musl do modernc (Linux/Android) precisa disto")
	}
	t.Cleanup(func() { _ = definirFusoDoSQLite("<+0000>0") })
	if err := definirFusoDoSQLite("<+0800>-8"); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var dia string
	if err := db.QueryRow(`SELECT date('2026-10-27T20:16:06Z', 'localtime')`).Scan(&dia); err != nil {
		t.Fatal(err)
	}
	if dia != "2026-10-28" {
		t.Errorf("dia local = %s, want 2026-10-28", dia)
	}
}
