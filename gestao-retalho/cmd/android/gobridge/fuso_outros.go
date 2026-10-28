//go:build !linux

package gobridge

// Fora de Linux/Android a libc do SQLite usa o fuso do sistema operativo
// (ex: Windows) — nada a fazer. Existe só para o pacote compilar em
// qualquer máquina de desenvolvimento.
func definirFusoDoSQLite(string) error { return nil }
