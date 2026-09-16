package admin_test

import (
	"context"
	"errors"
	"testing"

	"gestao-retalho/backend/admin"
)

func TestAuthService_PrecisaConfigurarAntesDeExistirAdministrador(t *testing.T) {
	ctx := context.Background()
	repo := novoAdministradorRepositoryTeste(t)
	svc := admin.NewAuthService(repo)

	precisa, err := svc.PrecisaConfigurar(ctx)
	if err != nil {
		t.Fatalf("precisa configurar: %v", err)
	}
	if !precisa {
		t.Fatal("esperava PrecisaConfigurar=true com admin.db vazio")
	}

	if _, err := svc.CriarAdministrador(ctx, "Nelson Matola", "nelson", "segredo123"); err != nil {
		t.Fatalf("criar administrador: %v", err)
	}

	precisa, err = svc.PrecisaConfigurar(ctx)
	if err != nil {
		t.Fatalf("precisa configurar depois: %v", err)
	}
	if precisa {
		t.Fatal("esperava PrecisaConfigurar=false depois de criar o administrador")
	}
}

func TestAuthService_CriarAdministradorRejeitaSegundo(t *testing.T) {
	ctx := context.Background()
	repo := novoAdministradorRepositoryTeste(t)
	svc := admin.NewAuthService(repo)

	if _, err := svc.CriarAdministrador(ctx, "Nelson Matola", "nelson", "segredo123"); err != nil {
		t.Fatalf("criar primeiro administrador: %v", err)
	}

	// "Só deve existir um" (decisão pós-Fase 10) não é só UI — o service
	// rejeita um segundo mesmo que alguém contorne o ecrã de bootstrap.
	if _, err := svc.CriarAdministrador(ctx, "Outra Pessoa", "outra", "segredo123"); err == nil {
		t.Fatal("esperava erro ao criar um segundo administrador")
	}
}

func TestAuthService_CriarAdministradorValidaCampos(t *testing.T) {
	ctx := context.Background()
	repo := novoAdministradorRepositoryTeste(t)
	svc := admin.NewAuthService(repo)

	if _, err := svc.CriarAdministrador(ctx, "", "nelson", "segredo123"); err == nil {
		t.Error("esperava erro sem nome")
	}
	if _, err := svc.CriarAdministrador(ctx, "Nelson", "", "segredo123"); err == nil {
		t.Error("esperava erro sem username")
	}
	if _, err := svc.CriarAdministrador(ctx, "Nelson", "nelson", "curta"); err == nil {
		t.Error("esperava erro com password curta")
	}
}

func TestAuthService_LoginComCredenciaisCorretas(t *testing.T) {
	ctx := context.Background()
	repo := novoAdministradorRepositoryTeste(t)
	svc := admin.NewAuthService(repo)

	if _, err := svc.CriarAdministrador(ctx, "Nelson Matola", "nelson", "segredo123"); err != nil {
		t.Fatalf("criar administrador: %v", err)
	}

	a, err := svc.Login(ctx, "nelson", "segredo123")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if a.Username != "nelson" || a.Nome != "Nelson Matola" {
		t.Errorf("administrador = %+v, want username=nelson nome=Nelson Matola", a)
	}
}

func TestAuthService_LoginRejeitaPasswordErrada(t *testing.T) {
	ctx := context.Background()
	repo := novoAdministradorRepositoryTeste(t)
	svc := admin.NewAuthService(repo)

	if _, err := svc.CriarAdministrador(ctx, "Nelson Matola", "nelson", "segredo123"); err != nil {
		t.Fatalf("criar administrador: %v", err)
	}

	if _, err := svc.Login(ctx, "nelson", "password-errada"); !errors.Is(err, admin.ErrCredenciaisInvalidas) {
		t.Errorf("erro = %v, want ErrCredenciaisInvalidas", err)
	}
}

func TestAuthService_LoginRejeitaAntesDeExistirAdministrador(t *testing.T) {
	ctx := context.Background()
	repo := novoAdministradorRepositoryTeste(t)
	svc := admin.NewAuthService(repo)

	if _, err := svc.Login(ctx, "nelson", "qualquer-coisa"); !errors.Is(err, admin.ErrCredenciaisInvalidas) {
		t.Errorf("erro = %v, want ErrCredenciaisInvalidas", err)
	}
}
