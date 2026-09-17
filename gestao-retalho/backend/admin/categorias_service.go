package admin

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// CategoriasService gere categorias de uma loja específica a partir do
// cmd/admin — dados de negócio continuam inteiramente dentro da base da
// loja (SDD 8.5), este serviço só resolve qual ligação abrir (AbrirLoja),
// delega no mesmo CategoriaService que o cmd/app usaria e fecha a seguir,
// tal como LojasService.Criar já faz para validar uma loja nova.
type CategoriasService struct {
	lojas *repository.LojaRepository
}

func NewCategoriasService(lojas *repository.LojaRepository) *CategoriasService {
	return &CategoriasService{lojas: lojas}
}

func (s *CategoriasService) Listar(ctx context.Context, lojaID int64) ([]domain.Categoria, error) {
	loja, err := s.lojas.Obter(ctx, lojaID)
	if err != nil {
		return nil, fmt.Errorf("loja não encontrada: %w", err)
	}

	app, fechar, err := AbrirLoja(loja.CaminhoDB)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarCategorias()
}

func (s *CategoriasService) Criar(ctx context.Context, lojaID int64, c domain.Categoria) (domain.Categoria, error) {
	loja, err := s.lojas.Obter(ctx, lojaID)
	if err != nil {
		return domain.Categoria{}, fmt.Errorf("loja não encontrada: %w", err)
	}

	app, fechar, err := AbrirLoja(loja.CaminhoDB)
	if err != nil {
		return domain.Categoria{}, err
	}
	defer fechar()

	return app.CriarCategoria(c)
}

func (s *CategoriasService) Atualizar(ctx context.Context, lojaID int64, c domain.Categoria) error {
	loja, err := s.lojas.Obter(ctx, lojaID)
	if err != nil {
		return fmt.Errorf("loja não encontrada: %w", err)
	}

	app, fechar, err := AbrirLoja(loja.CaminhoDB)
	if err != nil {
		return err
	}
	defer fechar()

	return app.AtualizarCategoria(c)
}

func (s *CategoriasService) DefinirAtiva(ctx context.Context, lojaID, categoriaID int64, ativo bool) error {
	loja, err := s.lojas.Obter(ctx, lojaID)
	if err != nil {
		return fmt.Errorf("loja não encontrada: %w", err)
	}

	app, fechar, err := AbrirLoja(loja.CaminhoDB)
	if err != nil {
		return err
	}
	defer fechar()

	if ativo {
		return app.AtivarCategoria(categoriaID)
	}
	return app.DesativarCategoria(categoriaID)
}
