package usecase

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

func novoOperadorServiceTeste(t *testing.T) *OperadorService {
	t.Helper()
	db, err := repository.NewSQLiteConn(config.DBPath(filepath.Join(t.TempDir(), "op.db")))
	if err != nil {
		t.Fatalf("abrir base: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return NewOperadorService(repository.NewUtilizadorRepository(db))
}

func TestOperador_CriarListarEAutenticar(t *testing.T) {
	ctx := context.Background()
	s := novoOperadorServiceTeste(t)

	joao, err := s.Criar(ctx, "João Mabunda", "1234")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	if _, err := s.Criar(ctx, "João Mabunda", "5678"); err != nil {
		t.Fatalf("segundo operador com o mesmo nome devia ter outro username: %v", err)
	}
	if _, err := s.Criar(ctx, "Sem PIN", "12a4"); !errors.Is(err, ErrPINFormato) {
		t.Errorf("PIN com letras: esperava ErrPINFormato, obteve %v", err)
	}
	if _, err := s.Criar(ctx, "Curto", "123"); !errors.Is(err, ErrPINFormato) {
		t.Errorf("PIN de 3 algarismos: esperava ErrPINFormato, obteve %v", err)
	}

	lista, err := s.Listar(ctx)
	if err != nil || len(lista) != 2 {
		t.Fatalf("listar = %+v, err %v; want 2 operadores (o administrador fixo fica de fora)", lista, err)
	}

	if _, err := s.Autenticar(ctx, joao.ID, "9999"); !errors.Is(err, ErrPINErrado) {
		t.Errorf("PIN errado: esperava ErrPINErrado, obteve %v", err)
	}
	op, err := s.Autenticar(ctx, joao.ID, "1234")
	if err != nil || op.Nome != "João Mabunda" {
		t.Fatalf("autenticar = %+v, err %v", op, err)
	}

	if err := s.DefinirAtivo(ctx, joao.ID, false); err != nil {
		t.Fatalf("desativar: %v", err)
	}
	if _, err := s.Autenticar(ctx, joao.ID, "1234"); !errors.Is(err, ErrOperadorInativo) {
		t.Errorf("operador inativo: esperava ErrOperadorInativo, obteve %v", err)
	}
	if ativos, _ := s.ListarAtivos(ctx); len(ativos) != 1 {
		t.Errorf("ativos = %d, want 1", len(ativos))
	}

	if err := s.Renomear(ctx, domain.UtilizadorAdminSeedID, "Outro"); !errors.Is(err, ErrNaoEOperador) {
		t.Errorf("renomear o administrador: esperava ErrNaoEOperador, obteve %v", err)
	}
}

func TestOperador_BloqueiaDepoisDeCincoPINsErrados(t *testing.T) {
	ctx := context.Background()
	s := novoOperadorServiceTeste(t)
	agora := time.Date(2026, 10, 9, 10, 0, 0, 0, time.UTC)
	s.agora = func() time.Time { return agora }

	op, err := s.Criar(ctx, "Rita", "4321")
	if err != nil {
		t.Fatalf("criar: %v", err)
	}
	for i := 0; i < maxTentativasPIN; i++ {
		if _, err := s.Autenticar(ctx, op.ID, "0000"); !errors.Is(err, ErrPINErrado) {
			t.Fatalf("tentativa %d: esperava ErrPINErrado, obteve %v", i+1, err)
		}
	}
	if _, err := s.Autenticar(ctx, op.ID, "4321"); err == nil {
		t.Fatal("depois de 5 erros, até o PIN certo devia ser recusado durante o bloqueio")
	}

	agora = agora.Add(bloqueioPIN + time.Second)
	if _, err := s.Autenticar(ctx, op.ID, "4321"); err != nil {
		t.Fatalf("depois do bloqueio, o PIN certo devia entrar: %v", err)
	}
}
