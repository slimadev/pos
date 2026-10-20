package admin_test

import (
	"testing"

	"gestao-retalho/backend/usecase"
)

func TestAdminApp_CotacoesExigemSessao(t *testing.T) {
	app := novoAdminAppTeste(t)
	if _, err := app.ListarCotacoes(); err == nil {
		t.Error("ListarCotacoes sem sessão: esperava erro")
	}
	if _, err := app.CriarCotacao(usecase.NovaCotacao{}); err == nil {
		t.Error("CriarCotacao sem sessão: esperava erro")
	}
	if _, err := app.ConverterCotacao(usecase.ConverterCotacaoParams{CotacaoID: 1}); err == nil {
		t.Error("ConverterCotacao sem sessão: esperava erro")
	}
	if err := app.CancelarCotacao(1); err == nil {
		t.Error("CancelarCotacao sem sessão: esperava erro")
	}
	if _, err := app.AbrirCotacaoPDF(1); err == nil {
		t.Error("AbrirCotacaoPDF sem sessão: esperava erro")
	}
}
