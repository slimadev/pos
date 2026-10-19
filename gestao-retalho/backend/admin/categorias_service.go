package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// CategoriasService gere categorias da única loja que este cmd/admin serve
// — dados de negócio continuam inteiramente dentro da base da loja (SDD
// 8.5), este serviço só resolve a ligação (AbrirLoja) e delega no mesmo
// CategoriaService que o cmd/app usaria.
type CategoriasService struct {
	lojas *repository.LojaRepository
}

func NewCategoriasService(lojas *repository.LojaRepository) *CategoriasService {
	return &CategoriasService{lojas: lojas}
}

func (s *CategoriasService) Listar(ctx context.Context) ([]domain.Categoria, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarCategorias()
}

func (s *CategoriasService) Criar(ctx context.Context, c domain.Categoria) (domain.Categoria, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.Categoria{}, err
	}
	defer fechar()

	return app.CriarCategoria(c)
}

func (s *CategoriasService) Atualizar(ctx context.Context, c domain.Categoria) error {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return err
	}
	defer fechar()

	return app.AtualizarCategoria(c)
}

func (s *CategoriasService) DefinirAtiva(ctx context.Context, categoriaID int64, ativo bool) error {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return err
	}
	defer fechar()

	if ativo {
		return app.AtivarCategoria(categoriaID)
	}
	return app.DesativarCategoria(categoriaID)
}

func (s *CategoriasService) Juntar(ctx context.Context, origemID, destinoID int64) (int64, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return 0, err
	}
	defer fechar()

	return app.JuntarCategorias(origemID, destinoID)
}

func (s *CategoriasService) abrirLoja(ctx context.Context) (*api.App, func(), error) {
	return abrirLojaUnica(ctx, s.lojas)
}
