package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// ProdutosService gere produtos (e as suas unidades de venda/compra —
// "variações ou grupos" do mesmo produto, ex: unidade/caixa, garrafa/dose)
// da única loja que este cmd/admin serve — dados de negócio continuam
// inteiramente dentro da base da loja (SDD 8.5), este serviço só resolve a
// ligação (AbrirLoja) e delega no mesmo ProdutoService que o cmd/app usaria.
type ProdutosService struct {
	lojas *repository.LojaRepository
}

func NewProdutosService(lojas *repository.LojaRepository) *ProdutosService {
	return &ProdutosService{lojas: lojas}
}

func (s *ProdutosService) Listar(ctx context.Context) ([]domain.Produto, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarTodosProdutos()
}

func (s *ProdutosService) ListarUnidades(ctx context.Context, produtoID int64) ([]domain.UnidadeProduto, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarUnidadesProduto(produtoID)
}

func (s *ProdutosService) Criar(ctx context.Context, req api.CriarProdutoRequest) (api.CriarProdutoResponse, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return api.CriarProdutoResponse{}, err
	}
	defer fechar()

	return app.CriarProduto(req)
}

func (s *ProdutosService) Atualizar(ctx context.Context, p domain.Produto) (domain.Produto, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.Produto{}, err
	}
	defer fechar()

	return app.AtualizarProduto(p)
}

func (s *ProdutosService) DefinirAtivo(ctx context.Context, produtoID int64, ativo bool) error {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return err
	}
	defer fechar()

	if ativo {
		return app.AtivarProduto(produtoID)
	}
	return app.DesativarProduto(produtoID)
}

func (s *ProdutosService) Duplicar(ctx context.Context, produtoID int64) (api.CriarProdutoResponse, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return api.CriarProdutoResponse{}, err
	}
	defer fechar()

	return app.DuplicarProduto(produtoID)
}

func (s *ProdutosService) AdicionarUnidade(ctx context.Context, u domain.UnidadeProduto) (domain.UnidadeProduto, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.UnidadeProduto{}, err
	}
	defer fechar()

	return app.AdicionarUnidadeProduto(u)
}

func (s *ProdutosService) AtualizarUnidade(ctx context.Context, u domain.UnidadeProduto) (domain.UnidadeProduto, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return domain.UnidadeProduto{}, err
	}
	defer fechar()

	return app.AtualizarUnidadeProduto(u)
}

func (s *ProdutosService) ListarTodasUnidades(ctx context.Context) ([]domain.UnidadeProduto, error) {
	app, fechar, err := s.abrirLoja(ctx)
	if err != nil {
		return nil, err
	}
	defer fechar()

	return app.ListarUnidadesDoNegocio()
}

func (s *ProdutosService) abrirLoja(ctx context.Context) (*api.App, func(), error) {
	return abrirLojaUnica(ctx, s.lojas)
}
