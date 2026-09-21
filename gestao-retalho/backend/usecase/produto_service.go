package usecase

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

type ProdutoService struct {
	repo *repository.ProdutoRepository
}

func NewProdutoService(repo *repository.ProdutoRepository) *ProdutoService {
	return &ProdutoService{repo: repo}
}

func (s *ProdutoService) Listar(ctx context.Context, negocioID int64) ([]domain.Produto, error) {
	return s.repo.ListarAtivos(ctx, negocioID)
}

func (s *ProdutoService) ListarTodos(ctx context.Context, negocioID int64) ([]domain.Produto, error) {
	return s.repo.Listar(ctx, negocioID)
}

func (s *ProdutoService) Obter(ctx context.Context, id int64) (domain.Produto, error) {
	return s.repo.Obter(ctx, id)
}

func (s *ProdutoService) ListarUnidades(ctx context.Context, produtoID int64) ([]domain.UnidadeProduto, error) {
	return s.repo.ListarUnidades(ctx, produtoID)
}

// NovoProduto agrupa produto e unidades: um produto sem unidade de
// venda/compra não é vendável (SDD 4.1), por isso a criação exige as duas
// coisas juntas em vez de deixar um produto "incompleto" gravável.
type NovoProduto struct {
	Produto  domain.Produto
	Unidades []domain.UnidadeProduto
}

func (s *ProdutoService) Criar(ctx context.Context, np NovoProduto) (domain.Produto, []domain.UnidadeProduto, error) {
	if np.Produto.Nome == "" {
		return domain.Produto{}, nil, fmt.Errorf("nome do produto é obrigatório")
	}
	if len(np.Unidades) == 0 {
		return domain.Produto{}, nil, fmt.Errorf("produto precisa de pelo menos uma unidade de venda/compra")
	}
	for _, u := range np.Unidades {
		if u.QtdEmUnidadeBase <= 0 {
			return domain.Produto{}, nil, fmt.Errorf("unidade %q: quantidade em unidade base deve ser positiva", u.NomeUnidade)
		}
	}

	// Sempre a mesma loja (SDD 2.1) — forçado aqui, não confiado a cada
	// chamador, para nunca haver um produto órfão por esquecimento.
	np.Produto.NegocioID = domain.NegocioID
	// SDD 3.2: produto composto nunca guarda stock próprio — é sempre
	// inferido via ficha técnica (Fase 5); simples/semi_elaborado controlam.
	np.Produto.ControlaStock = np.Produto.TipoProduto != domain.TipoProdutoComposto
	// Rastreio por lote (Fase 6, SDD 9.4) só faz sentido para quem tem
	// stock físico próprio — um composto nunca guarda o seu, ver acima.
	if np.Produto.TipoProduto == domain.TipoProdutoComposto {
		np.Produto.LoteControlado = false
	}
	np.Produto.Ativo = true

	return s.repo.Criar(ctx, np.Produto, np.Unidades)
}

func (s *ProdutoService) Atualizar(ctx context.Context, p domain.Produto) (domain.Produto, error) {
	if p.Nome == "" {
		return domain.Produto{}, fmt.Errorf("nome do produto é obrigatório")
	}
	atual, err := s.repo.Obter(ctx, p.ID)
	if err != nil {
		return domain.Produto{}, err
	}
	if atual.TipoProduto == domain.TipoProdutoComposto {
		p.LoteControlado = false
	}
	return s.repo.Atualizar(ctx, p)
}

func (s *ProdutoService) Ativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtivo(ctx, id, true)
}

func (s *ProdutoService) Desativar(ctx context.Context, id int64) error {
	return s.repo.DefinirAtivo(ctx, id, false)
}

// AdicionarUnidade acrescenta mais uma unidade de venda/compra a um
// produto já existente (ex: o mesmo produto passa a vender-se também em
// caixa, além de unidade) — Criar só aceitava unidades no momento da
// criação, sem forma nenhuma de acrescentar depois.
func (s *ProdutoService) AdicionarUnidade(ctx context.Context, u domain.UnidadeProduto) (domain.UnidadeProduto, error) {
	if u.NomeUnidade == "" {
		return domain.UnidadeProduto{}, fmt.Errorf("nome da unidade é obrigatório")
	}
	if u.QtdEmUnidadeBase <= 0 {
		return domain.UnidadeProduto{}, fmt.Errorf("quantidade em unidade base deve ser positiva")
	}
	return s.repo.CriarUnidade(ctx, u)
}

// Duplicar cria uma cópia do produto e das suas unidades — útil para
// variantes (SDD 4.1: "mesma cerveja, tamanhos diferentes").
func (s *ProdutoService) Duplicar(ctx context.Context, produtoID int64) (domain.Produto, []domain.UnidadeProduto, error) {
	original, err := s.repo.Obter(ctx, produtoID)
	if err != nil {
		return domain.Produto{}, nil, err
	}
	unidadesOriginais, err := s.repo.ListarUnidades(ctx, produtoID)
	if err != nil {
		return domain.Produto{}, nil, err
	}

	copia := original
	copia.Nome = original.Nome + " (cópia)"

	unidades := make([]domain.UnidadeProduto, 0, len(unidadesOriginais))
	for _, u := range unidadesOriginais {
		// código de barras é único (SDD 3.2) — a cópia não pode herdar o
		// mesmo, tem de ser preenchido à parte antes de vender.
		u.CodigoBarras = nil
		unidades = append(unidades, u)
	}

	return s.repo.Criar(ctx, copia, unidades)
}
