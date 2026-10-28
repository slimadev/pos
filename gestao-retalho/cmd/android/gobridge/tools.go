//go:build tools

package gobridge

// O `gomobile bind` precisa de golang.org/x/mobile/bind no go.mod do
// projeto, mas nenhum código o importa — este ficheiro (nunca compilado:
// tag "tools") impede o `go mod tidy` de o remover. Versão fixada a uma
// compatível com o Go do projeto (1.25): as mais recentes exigem 1.26.
import _ "golang.org/x/mobile/bind"
