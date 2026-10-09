package admin_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// lojaTeste é uma base de loja completa (schema + wizard concluído) num
// ficheiro em disco — ao contrário dos testes de usecase (SQLite em
// memória por teste), tem de ser um ficheiro real para admin.AbrirLoja
// conseguir abrir a sua própria ligação independente desta.
type lojaTeste struct {
	Caminho  string
	Produtos *usecase.ProdutoService
	Armazens *usecase.ArmazemService
	Stock    *usecase.StockService
	Vendas   *usecase.VendaService
	Contas   *usecase.ContaService
	Clientes *usecase.ClienteService
	Dividas  *usecase.DividaService
	Armazem  domain.Armazem
}

func novaLojaTeste(t *testing.T, nomeFicheiro, nomeNegocio string, tipo domain.TipoNegocio) lojaTeste {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), nomeFicheiro)

	db, err := repository.NewSQLiteConn(config.DBPath(caminho))
	if err != nil {
		t.Fatalf("abrir loja de teste: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	produtoRepo := repository.NewProdutoRepository(db)
	armazemRepo := repository.NewArmazemRepository(db)
	stockRepo := repository.NewStockRepository(db)
	invRepo := repository.NewInventarioRepository(db)
	negocioRepo := repository.NewNegocioRepository(db)
	configRepo := repository.NewConfigRepository(db)
	utilizadorRepo := repository.NewUtilizadorRepository(db)
	categoriaRepo := repository.NewCategoriaRepository(db)
	clienteRepo := repository.NewClienteRepository(db)
	loteRepo := repository.NewLoteRepository(db, stockRepo)
	vendaRepo := repository.NewVendaRepository(db, stockRepo, loteRepo)
	fichaRepo := repository.NewFichaTecnicaRepository(db)
	dividaRepo := repository.NewDividaRepository(db)

	produtoSvc := usecase.NewProdutoService(produtoRepo)
	armazemSvc := usecase.NewArmazemService(armazemRepo)
	stockSvc := usecase.NewStockService(stockRepo, produtoRepo, armazemRepo, invRepo, loteRepo)
	configSvc := usecase.NewConfigService(negocioRepo, configRepo, utilizadorRepo, categoriaRepo, armazemSvc, produtoSvc)
	vendaSvc := usecase.NewVendaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo)
	clienteSvc := usecase.NewClienteService(clienteRepo)
	dividaSvc := usecase.NewDividaService(dividaRepo, clienteRepo)
	contaSvc := usecase.NewContaService(vendaRepo, produtoRepo, armazemRepo, fichaRepo, dividaSvc)

	ctx := context.Background()
	if err := configSvc.FinalizarWizard(ctx, usecase.FinalizarWizardParams{
		NegocioNome:          nomeNegocio,
		TipoNegocio:          tipo,
		ArmazemPrincipalNome: "Balcão",
		AdminNome:            "Admin Teste",
		AdminUsername:        "admin",
		AdminPassword:        "password123",
	}); err != nil {
		t.Fatalf("finalizar wizard: %v", err)
	}

	armazens, err := armazemSvc.Listar(ctx, domain.NegocioID)
	if err != nil || len(armazens) == 0 {
		t.Fatalf("listar armazéns após wizard: %v", err)
	}

	return lojaTeste{
		Caminho:  caminho,
		Produtos: produtoSvc,
		Armazens: armazemSvc,
		Stock:    stockSvc,
		Vendas:   vendaSvc,
		Contas:   contaSvc,
		Clientes: clienteSvc,
		Dividas:  dividaSvc,
		Armazem:  armazens[0],
	}
}

// novoAdminDBTeste devolve um repository.LojaRepository sobre um admin.db
// em memória — mesmo padrão dos testes de usecase, só que a tabela `lojas`
// vive na migração própria do admin (migrations_admin), não na partilhada.
func novoLojaRepositoryTeste(t *testing.T) *repository.LojaRepository {
	t.Helper()
	return repository.NewLojaRepository(novoAdminDBTeste(t))
}

// novoAdministradorRepositoryTeste segue o mesmo padrão, para os testes de
// AuthService (login único, decisão pós-Fase 10).
func novoAdministradorRepositoryTeste(t *testing.T) *repository.AdministradorRepository {
	t.Helper()
	return repository.NewAdministradorRepository(novoAdminDBTeste(t))
}

func novoAdminDBTeste(t *testing.T) *sql.DB {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "admin.db")
	db, err := repository.NewAdminSQLiteConn(config.AdminDBPath(caminho))
	if err != nil {
		t.Fatalf("abrir admin.db de teste: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}
