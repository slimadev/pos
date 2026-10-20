package admin_test

import (
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestAdminApp_AtualizarIdentidadeLojaMudaNomeELogo(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	app := novoAdminAppTeste(t)
	if _, err := app.ConfigurarPrimeiroArranque(admin.ConfigurarPrimeiroArranqueParams{
		NegocioNome:   "Ferragem Cossa",
		TipoNegocio:   domain.TipoNegocioFerragem,
		AdminNome:     "Nelson Matola",
		AdminUsername: "nelson",
		AdminPassword: "admin123",
	}); err != nil {
		t.Fatalf("configurar primeiro arranque: %v", err)
	}

	// Antes de gravar nada, o IVA é a taxa normal (16%).
	inicial, err := app.ObterLojaAtual()
	if err != nil || inicial.TaxaIVA != 16 || inicial.PapelRecibo != "80" {
		t.Fatalf("loja inicial = %+v, %v; want IVA 16 e papel 80", inicial, err)
	}

	logo := "data:image/png;base64,iVBORw0KGgo="
	loja, err := app.AtualizarIdentidadeLoja(usecase.DadosLoja{
		Nome: "  Ferragem Cossa & Filhos ", Logo: logo,
		NUIT: "400 123 456", Endereco: "Av. Eduardo Mondlane 123, Matola",
		Telefone: "84 123 4567", Email: "geral@cossa.co.mz", TaxaIVA: 16,
	})
	if err != nil {
		t.Fatalf("atualizar identidade: %v", err)
	}
	if loja.Nome != "Ferragem Cossa & Filhos" || loja.Logo != logo {
		t.Fatalf("loja = %+v, want nome novo e logótipo", loja)
	}
	if loja.NUIT != "400123456" || loja.Email != "geral@cossa.co.mz" || loja.Telefone != "84 123 4567" ||
		loja.Endereco != "Av. Eduardo Mondlane 123, Matola" || loja.TaxaIVA != 16 {
		t.Errorf("dados da loja = %+v", loja)
	}
	if loja.TipoNegocio != string(domain.TipoNegocioFerragem) {
		t.Errorf("tipo = %q, want ferragem (não muda)", loja.TipoNegocio)
	}

	// Isento de IVA (0%), sem logótipo e papel de 58 mm.
	loja, err = app.AtualizarIdentidadeLoja(usecase.DadosLoja{Nome: "Ferragem Cossa & Filhos", PapelRecibo: "58", TaxaIVA: 0})
	if err != nil || loja.Logo != "" || loja.PapelRecibo != "58" || loja.TaxaIVA != 0 {
		t.Fatalf("isento/58 mm: %+v, %v", loja, err)
	}

	invalidos := map[string]usecase.DadosLoja{
		"sem nome":       {},
		"logo":           {Nome: "Loja", Logo: "isto não é imagem"},
		"papel":          {Nome: "Loja", PapelRecibo: "100"},
		"NUIT curto":     {Nome: "Loja", NUIT: "12345"},
		"NUIT com letra": {Nome: "Loja", NUIT: "40012345A"},
		"email":          {Nome: "Loja", Email: "geral.cossa.co.mz"},
		"IVA negativo":   {Nome: "Loja", TaxaIVA: -1},
		"IVA acima":      {Nome: "Loja", TaxaIVA: 101},
	}
	for caso, d := range invalidos {
		if _, err := app.AtualizarIdentidadeLoja(d); err == nil {
			t.Errorf("%s: esperava erro", caso)
		}
	}

	app.Logout()
	if _, err := app.AtualizarIdentidadeLoja(usecase.DadosLoja{Nome: "Outra"}); err == nil {
		t.Error("esperava erro sem administrador autenticado")
	}
}
