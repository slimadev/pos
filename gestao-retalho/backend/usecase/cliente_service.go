package usecase

import (
	"context"
	"fmt"
	"strings"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

type ClienteService struct {
	repo *repository.ClienteRepository
}

func NewClienteService(repo *repository.ClienteRepository) *ClienteService {
	return &ClienteService{repo: repo}
}

func (s *ClienteService) Listar(ctx context.Context, negocioID int64) ([]domain.Cliente, error) {
	return s.repo.Listar(ctx, negocioID)
}

func (s *ClienteService) Obter(ctx context.Context, id int64) (domain.Cliente, error) {
	return s.repo.Obter(ctx, id)
}

// validarContactos aplica ao cliente as mesmas regras de NUIT e email
// da loja — vão parar à fatura.
func validarContactos(c *domain.Cliente) error {
	if c.NIF != nil {
		nuit := strings.ReplaceAll(strings.TrimSpace(*c.NIF), " ", "")
		if nuit != "" && !nuitValido(nuit) {
			return fmt.Errorf("o NUIT do cliente tem de ter 9 algarismos")
		}
		c.NIF = &nuit
		if nuit == "" {
			c.NIF = nil
		}
	}
	if c.Email != nil {
		email := strings.TrimSpace(*c.Email)
		if email != "" && !emailValido(email) {
			return fmt.Errorf("email inválido: %q", email)
		}
		c.Email = &email
		if email == "" {
			c.Email = nil
		}
	}
	return nil
}

func (s *ClienteService) Criar(ctx context.Context, c domain.Cliente) (domain.Cliente, error) {
	if c.Nome == "" {
		return domain.Cliente{}, fmt.Errorf("nome do cliente é obrigatório")
	}
	if err := validarContactos(&c); err != nil {
		return domain.Cliente{}, err
	}
	c.NegocioID = domain.NegocioID
	c.Ativo = true
	return s.repo.Criar(ctx, c)
}

func (s *ClienteService) Atualizar(ctx context.Context, c domain.Cliente) error {
	if c.Nome == "" {
		return fmt.Errorf("nome do cliente é obrigatório")
	}
	if c.ID == domain.ClienteAnonimoID {
		return domain.ErrClienteAnonimoFixo
	}
	if err := validarContactos(&c); err != nil {
		return err
	}
	return s.repo.Atualizar(ctx, c)
}

func (s *ClienteService) Ativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtivo(ctx, id, true)
}

// Desativar recusa o cliente anónimo: é o registo fixo de todas as vendas
// sem cliente identificado.
func (s *ClienteService) Desativar(ctx context.Context, id int64) error {
	if id == domain.ClienteAnonimoID {
		return domain.ErrClienteAnonimoFixo
	}
	return s.repo.DefinirAtivo(ctx, id, false)
}
