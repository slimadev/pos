package usecase_test

import (
	"context"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

func newTestConfigService(t *testing.T) (*usecase.ConfigService, *usecase.ProdutoService, *repository.UtilizadorRepository) {
	t.Helper()
	// FinalizarWizard escreve config.json no diretório de configuração do
	// utilizador — sem isto, correr os testes apagava o da loja real.
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	db := newTestDB(t)

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	negocioRepo := repository.NewNegocioRepository(db)
	configRepo := repository.NewConfigRepository(db)
	utilizadorRepo := repository.NewUtilizadorRepository(db)
	categoriaRepo := repository.NewCategoriaRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	configSvc := usecase.NewConfigService(negocioRepo, configRepo, utilizadorRepo, categoriaRepo, armazemSvc, produtoSvc)

	return configSvc, produtoSvc, utilizadorRepo
}

func TestEstaConfigurado_FalsoAntesEVerdadeiroDepoisDoWizard(t *testing.T) {
	ctx := context.Background()
	configSvc, _, _ := newTestConfigService(t)

	antes, err := configSvc.EstaConfigurado(ctx)
	if err != nil {
		t.Fatalf("EstaConfigurado antes: %v", err)
	}
	if antes {
		t.Fatalf("esperava EstaConfigurado=false antes do wizard")
	}

	if err := configSvc.FinalizarWizard(ctx, usecase.FinalizarWizardParams{
		NegocioNome:          "Ferragem Central",
		TipoNegocio:          domain.TipoNegocioFerragem,
		ArmazemPrincipalNome: "Armazém Principal",
		AdminNome:            "Rui Bila",
		AdminUsername:        "rui",
		AdminPassword:        "segredo123",
	}); err != nil {
		t.Fatalf("finalizar wizard: %v", err)
	}

	depois, err := configSvc.EstaConfigurado(ctx)
	if err != nil {
		t.Fatalf("EstaConfigurado depois: %v", err)
	}
	if !depois {
		t.Fatalf("esperava EstaConfigurado=true depois do wizard")
	}
}

func TestFinalizarWizard_PresetAplicaModulosETemaCorretos(t *testing.T) {
	ctx := context.Background()
	configSvc, _, utilizadorRepo := newTestConfigService(t)

	if err := configSvc.FinalizarWizard(ctx, usecase.FinalizarWizardParams{
		NegocioNome:          "Bottlestore XPTO",
		TipoNegocio:          domain.TipoNegocioBottlestore,
		ArmazemPrincipalNome: "Balcão",
		AdminNome:            "Ana Cumbe",
		AdminUsername:        "ana",
		AdminPassword:        "segredo123",
	}); err != nil {
		t.Fatalf("finalizar wizard: %v", err)
	}

	estado, err := configSvc.ObterEstado(ctx)
	if err != nil {
		t.Fatalf("obter estado: %v", err)
	}
	if estado.Negocio.Nome != "Bottlestore XPTO" {
		t.Errorf("nome = %q, want %q", estado.Negocio.Nome, "Bottlestore XPTO")
	}
	if estado.Negocio.TipoNegocio != domain.TipoNegocioBottlestore {
		t.Errorf("tipo = %v, want bottlestore", estado.Negocio.TipoNegocio)
	}
	if estado.Tema != domain.TemaBottlestore {
		t.Errorf("tema = %v, want bottlestore", estado.Tema)
	}

	quer := map[domain.Modulo]bool{
		domain.ModuloContasAbertas:  true,
		domain.ModuloDividas:        true,
		domain.ModuloFichasTecnicas: true,
	}
	if len(estado.ModulosAtivos) != len(quer) {
		t.Fatalf("módulos ativos = %v, want exatamente %v", estado.ModulosAtivos, quer)
	}
	for _, m := range estado.ModulosAtivos {
		if !quer[m] {
			t.Errorf("módulo inesperado ativo: %v", m)
		}
	}
	// pos_direto não faz parte do preset bottlestore (SDD 9.1) — venda
	// direta é modelada como conta que abre/fecha na mesma transação (3.5).
	for _, m := range estado.ModulosAtivos {
		if m == domain.ModuloPosDireto {
			t.Errorf("pos_direto não devia estar ativo para bottlestore")
		}
	}

	// utilizador admin foi atualizado com os dados reais e um hash válido.
	admin, err := utilizadorRepo.ObterPorUsername(ctx, "ana")
	if err != nil {
		t.Fatalf("obter admin: %v", err)
	}
	if admin.Nome != "Ana Cumbe" {
		t.Errorf("nome admin = %q, want %q", admin.Nome, "Ana Cumbe")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte("segredo123")); err != nil {
		t.Errorf("password hash não corresponde à password fornecida: %v", err)
	}
}

func TestFinalizarWizard_PersonalizadoUsaModulosEscolhidosETemaAdmin(t *testing.T) {
	ctx := context.Background()
	configSvc, _, _ := newTestConfigService(t)

	escolhidos := []domain.Modulo{domain.ModuloPosDireto, domain.ModuloVendaGranel}
	if err := configSvc.FinalizarWizard(ctx, usecase.FinalizarWizardParams{
		NegocioNome:          "Loja Mista",
		TipoNegocio:          domain.TipoNegocioPersonalizado,
		ModulosAtivos:        escolhidos,
		ArmazemPrincipalNome: "Armazém",
		AdminNome:            "Admin",
		AdminUsername:        "admin2",
		AdminPassword:        "segredo123",
	}); err != nil {
		t.Fatalf("finalizar wizard: %v", err)
	}

	estado, err := configSvc.ObterEstado(ctx)
	if err != nil {
		t.Fatalf("obter estado: %v", err)
	}
	if estado.Tema != domain.TemaAdmin {
		t.Errorf("tema = %v, want admin (sem preset para personalizado)", estado.Tema)
	}
	if len(estado.ModulosAtivos) != 2 {
		t.Fatalf("módulos ativos = %v, want %v", estado.ModulosAtivos, escolhidos)
	}
}

func TestFinalizarWizard_AplicaSeedDoTipoEscolhido(t *testing.T) {
	ctx := context.Background()
	configSvc, produtoSvc, _ := newTestConfigService(t)

	if err := configSvc.FinalizarWizard(ctx, usecase.FinalizarWizardParams{
		NegocioNome:          "Bottlestore XPTO",
		TipoNegocio:          domain.TipoNegocioBottlestore,
		ArmazemPrincipalNome: "Balcão",
		AdminNome:            "Ana",
		AdminUsername:        "ana2",
		AdminPassword:        "segredo123",
	}); err != nil {
		t.Fatalf("finalizar wizard: %v", err)
	}

	produtos, err := produtoSvc.ListarTodos(ctx, domain.NegocioID)
	if err != nil {
		t.Fatalf("listar produtos: %v", err)
	}
	// seedsPorTipo[bottlestore] tem 2 categorias x 2 produtos = 4 (ver seed.go).
	if len(produtos) != 4 {
		t.Fatalf("produtos semeados = %d, want 4", len(produtos))
	}
	for _, p := range produtos {
		if p.CategoriaID == nil {
			t.Errorf("produto %q sem categoria atribuída pelo seed", p.Nome)
		}
	}
}

func TestFinalizarWizard_ValidaCamposObrigatorios(t *testing.T) {
	ctx := context.Background()
	configSvc, _, _ := newTestConfigService(t)

	casos := []struct {
		nome   string
		params usecase.FinalizarWizardParams
	}{
		{"sem nome do negócio", usecase.FinalizarWizardParams{
			TipoNegocio: domain.TipoNegocioMercearia, ArmazemPrincipalNome: "A",
			AdminNome: "X", AdminUsername: "x", AdminPassword: "segredo123",
		}},
		{"password curta", usecase.FinalizarWizardParams{
			NegocioNome: "Loja", TipoNegocio: domain.TipoNegocioMercearia, ArmazemPrincipalNome: "A",
			AdminNome: "X", AdminUsername: "x", AdminPassword: "123",
		}},
	}

	for _, c := range casos {
		if err := configSvc.FinalizarWizard(ctx, c.params); err == nil {
			t.Errorf("%s: esperava erro, obteve nil", c.nome)
		}
	}
}
