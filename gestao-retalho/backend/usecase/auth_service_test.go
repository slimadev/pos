package usecase_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/usecase"
)

func TestAuthService_LoginComCredenciaisCorretas(t *testing.T) {
	ctx := context.Background()
	configSvc, _, utilizadorRepo := newTestConfigService(t)
	authSvc := usecase.NewAuthService(utilizadorRepo)

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

	u, err := authSvc.Login(ctx, "rui", "segredo123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if u.Username != "rui" || u.Nome != "Rui Bila" {
		t.Errorf("utilizador = %+v, want username=rui nome=Rui Bila", u)
	}
}

func TestAuthService_LoginRejeitaPasswordErrada(t *testing.T) {
	ctx := context.Background()
	configSvc, _, utilizadorRepo := newTestConfigService(t)
	authSvc := usecase.NewAuthService(utilizadorRepo)

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

	if _, err := authSvc.Login(ctx, "rui", "password-errada"); !errors.Is(err, usecase.ErrCredenciaisInvalidas) {
		t.Errorf("erro = %v, want ErrCredenciaisInvalidas", err)
	}
}

func TestAuthService_LoginRejeitaUsernameInexistente(t *testing.T) {
	ctx := context.Background()
	_, _, utilizadorRepo := newTestConfigService(t)
	authSvc := usecase.NewAuthService(utilizadorRepo)

	// Nem sequer é preciso correr o wizard: este username não existe em
	// lado nenhum, antes ou depois do wizard.
	if _, err := authSvc.Login(ctx, "ninguem", "qualquer-coisa"); !errors.Is(err, usecase.ErrCredenciaisInvalidas) {
		t.Errorf("erro = %v, want ErrCredenciaisInvalidas", err)
	}
}

// TestAuthService_LoginRejeitaAntesDoWizard cobre a linha semente da
// migration core (id=1, username "admin", password_hash vazio) — o wizard
// ainda não substituiu essas credenciais por umas reais, por isso o login
// tem de falhar mesmo com o username certo.
func TestAuthService_LoginRejeitaAntesDoWizard(t *testing.T) {
	ctx := context.Background()
	_, _, utilizadorRepo := newTestConfigService(t)
	authSvc := usecase.NewAuthService(utilizadorRepo)

	if _, err := authSvc.Login(ctx, "admin", "qualquer-coisa"); !errors.Is(err, usecase.ErrCredenciaisInvalidas) {
		t.Errorf("erro = %v, want ErrCredenciaisInvalidas", err)
	}
}

func TestAuthService_LoginRejeitaCamposVazios(t *testing.T) {
	ctx := context.Background()
	_, _, utilizadorRepo := newTestConfigService(t)
	authSvc := usecase.NewAuthService(utilizadorRepo)

	if _, err := authSvc.Login(ctx, "", ""); !errors.Is(err, usecase.ErrCredenciaisInvalidas) {
		t.Errorf("erro = %v, want ErrCredenciaisInvalidas", err)
	}
}
