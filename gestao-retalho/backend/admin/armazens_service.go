package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// ArmazensService gere os armazéns da única loja que este cmd/admin serve
// — dados de negócio continuam inteiramente dentro da base da loja (SDD
// 8.5), este serviço só resolve a ligação (AbrirLoja) e delega no mesmo
// ArmazemService que o cmd/app usaria.
type ArmazensService struct {
	lojas *repository.LojaRepository
}

func NewArmazensService(lojas *repository.LojaRepository) *ArmazensService {
	return &ArmazensService{lojas: lojas}
}

func (s *ArmazensService) Listar(ctx context.Context) ([]domain.Armazem, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarArmazens()
}

func (s *ArmazensService) Criar(ctx context.Context, a domain.Armazem) (domain.Armazem, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.Armazem{}, err
	}
	defer fechar()

	return app.CriarArmazem(a)
}

func (s *ArmazensService) Atualizar(ctx context.Context, a domain.Armazem) error {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return err
	}
	defer fechar()

	return app.AtualizarArmazem(a)
}

func (s *ArmazensService) DefinirAtivo(ctx context.Context, armazemID int64, ativo bool) error {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return err
	}
	defer fechar()

	if ativo {
		return app.AtivarArmazem(armazemID)
	}
	return app.DesativarArmazem(armazemID)
}

func (s *ArmazensService) abrirLoja(ctx context.Context) (*api.App, func(), error) {
	return abrirLojaUnica(ctx, s.lojas)
}
