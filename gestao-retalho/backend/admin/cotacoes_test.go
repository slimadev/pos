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

func TestAdminApp_DocumentosFiscaisExigemSessao(t *testing.T) {
	app := novoAdminAppTeste(t)
	if _, err := app.EstadoFaturacao(); err == nil {
		t.Error("EstadoFaturacao sem sessão: esperava erro")
	}
	if _, err := app.DefinirFaturacaoAtiva(true); err == nil {
		t.Error("DefinirFaturacaoAtiva sem sessão: esperava erro")
	}
	if _, err := app.ListarDocumentosMes(2026, 10); err == nil {
		t.Error("ListarDocumentosMes sem sessão: esperava erro")
	}
	if _, err := app.EmitirDocumentoEmFalta(1); err == nil {
		t.Error("EmitirDocumentoEmFalta sem sessão: esperava erro")
	}
	if _, err := app.ExportarDocumentosMes(2026, 10); err == nil {
		t.Error("ExportarDocumentosMes sem sessão: esperava erro")
	}
}

func TestAdminApp_ImportacaoExigeSessao(t *testing.T) {
	app := novoAdminAppTeste(t)
	if _, err := app.EscolherFicheiroExcel(); err == nil {
		t.Error("EscolherFicheiroExcel sem sessão: esperava erro")
	}
	if _, err := app.AnalisarExcelProdutos("x.xlsx"); err == nil {
		t.Error("AnalisarExcelProdutos sem sessão: esperava erro")
	}
	if _, err := app.ImportarProdutos(usecase.ImportarProdutosParams{}); err == nil {
		t.Error("ImportarProdutos sem sessão: esperava erro")
	}
}
