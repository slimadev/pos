package admin

import (
	"context"
	"fmt"
	"os/exec"
	"sync"

	"gestao-retalho/backend/config"
	"gestao-retalho/backend/repository"
)

// PDVService lança o cmd/app (o ecrã de venda) como processo separado,
// apontado para o mesmo ficheiro .db da única loja que o cmd/admin serve
// — cmd/app deixou de abrir sozinho, só arranca a partir do botão "PDV"
// (decisão pós-Fase 10, a pedido explícito). O caminho -db garante que os
// dois processos usam sempre a mesma base, mesmo que a loja não esteja no
// caminho por omissão que o cmd/app resolveria sozinho.
type PDVService struct {
	lojas              *repository.LojaRepository
	resolverExecutavel func() (string, error)

	mu     sync.Mutex
	aberto bool
}

func NewPDVService(lojas *repository.LojaRepository) *PDVService {
	return NewPDVServiceComExecutavel(lojas, config.ResolveAppExecutavel)
}

// NewPDVServiceComExecutavel permite injetar o resolvedor do executável —
// usado pelos testes para não depender do binário real do cmd/app estar
// compilado.
func NewPDVServiceComExecutavel(lojas *repository.LojaRepository, resolverExecutavel func() (string, error)) *PDVService {
	return &PDVService{lojas: lojas, resolverExecutavel: resolverExecutavel}
}

// Abrir arranca o cmd/app se ainda não estiver aberto — clicar "PDV" outra
// vez enquanto já está a correr não abre uma segunda janela.
func (s *PDVService) Abrir(ctx context.Context) error {
	s.mu.Lock()
	if s.aberto {
		s.mu.Unlock()
		return nil
	}
	s.mu.Unlock()

	loja, existe, err := s.lojas.ObterUnica(ctx)
	if err != nil {
		return err
	}
	if !existe {
		return fmt.Errorf("nenhuma loja configurada")
	}

	appExe, err := s.resolverExecutavel()
	if err != nil {
		return err
	}

	cmd := exec.Command(appExe, "-db", loja.CaminhoDB)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("abrir PDV: %w", err)
	}

	s.mu.Lock()
	s.aberto = true
	s.mu.Unlock()

	go func() {
		_ = cmd.Wait()
		s.mu.Lock()
		s.aberto = false
		s.mu.Unlock()
	}()

	return nil
}
