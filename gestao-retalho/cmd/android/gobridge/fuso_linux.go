//go:build linux

package gobridge

import (
	"fmt"

	"modernc.org/libc"
)

// definirFusoDoSQLite põe o fuso na libc do SQLite (modernc: musl
// transpilado para Go). É ela que calcula `date(x, 'localtime')` nos
// relatórios/dashboard, e não usa o time.Local do Go: lê a variável TZ do
// seu próprio ambiente — copiado do processo quando o Go arranca, por isso
// um setenv feito depois pelo Kotlin não lhe chega — e, sem
// /usr/share/zoneinfo (o caso do Android), só entende o formato POSIX.
// Sem isto as vendas eram agrupadas por dia em UTC.
func definirFusoDoSQLite(posix string) error {
	tls := libc.NewTLS()
	defer tls.Close()
	nome, err := libc.CString("TZ")
	if err != nil {
		return err
	}
	defer libc.Xfree(tls, nome)
	valor, err := libc.CString(posix)
	if err != nil {
		return err
	}
	defer libc.Xfree(tls, valor)
	if libc.Xsetenv(tls, nome, valor, 1) != 0 {
		return fmt.Errorf("setenv TZ=%q falhou", posix)
	}
	libc.Xtzset(tls)
	return nil
}
