package admin_test

import (
	"context"
	"fmt"
	"testing"

	"gestao-retalho/backend/admin"
	"gestao-retalho/backend/domain"
)

func TestPDVService_AbrirRejeitaSemLojaConfigurada(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)
	svc := admin.NewPDVServiceComExecutavel(repo, func() (string, error) { return "/bin/echo", nil })

	if err := svc.Abrir(ctx); err == nil {
		t.Fatal("esperava erro sem nenhuma loja registada")
	}
}

func TestPDVService_AbrirPropagaErroDoResolverExecutavel(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)

	loja := novaLojaTeste(t, "loja-pdv.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	svc := admin.NewPDVServiceComExecutavel(repo, func() (string, error) {
		return "", fmt.Errorf("executável não encontrado")
	})
	if err := svc.Abrir(ctx); err == nil {
		t.Fatal("esperava erro do resolverExecutavel")
	}
}

func TestPDVService_AbrirLancaOProcessoComOCaminhoDaLoja(t *testing.T) {
	ctx := context.Background()
	repo := novoLojaRepositoryTeste(t)

	loja := novaLojaTeste(t, "loja-pdv-ok.db", "Ferragem Central", domain.TipoNegocioFerragem)
	if _, err := repo.Criar(ctx, "Ferragem Central", loja.Caminho); err != nil {
		t.Fatalf("registar loja: %v", err)
	}

	svc := admin.NewPDVServiceComExecutavel(repo, func() (string, error) { return "/bin/echo", nil })
	if err := svc.Abrir(ctx); err != nil {
		t.Fatalf("abrir PDV: %v", err)
	}
}
