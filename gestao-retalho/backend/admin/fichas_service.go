package admin

import (
	"context"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
	"gestao-retalho/backend/usecase"
)

// FichasService serve o ecrã "Fichas técnicas" do cmd/admin: a receita de
// cada produto composto/semi-elaborado com o custo teórico e quanto dá para
// fazer com o stock de um armazém, a edição da receita e a produção em
// lote de semi-elaborados.
type FichasService struct {
	lojas *repository.LojaRepository
}

func NewFichasService(lojas *repository.LojaRepository) *FichasService {
	return &FichasService{lojas: lojas}
}

type FichaResumo struct {
	Produto      domain.Produto
	PrecoVenda   float64
	Ingredientes []domain.FichaTecnica
	CustoTeorico float64
	// Disponivel é quantas unidades dá para preparar (compostos) ou
	// produzir (semi-elaborados) com o stock atual do armazém.
	Disponivel float64
}

func (s *FichasService) Listar(ctx context.Context, armazemID int64) ([]FichaResumo, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]FichaResumo, error) {
		produtos, err := a.ListarProdutos()
		if err != nil {
			return nil, err
		}
		out := make([]FichaResumo, 0)
		for _, p := range produtos {
			if p.TipoProduto != domain.TipoProdutoComposto && p.TipoProduto != domain.TipoProdutoSemiElaborado {
				continue
			}
			ingredientes, err := a.ListarIngredientes(p.ID)
			if err != nil {
				return nil, err
			}
			custo, err := a.CalcularCustoTeorico(p.ID, armazemID)
			if err != nil {
				return nil, err
			}
			disponivel, err := a.SimularDisponibilidade(p.ID, armazemID)
			if err != nil {
				return nil, err
			}
			unidades, err := a.ListarUnidadesProduto(p.ID)
			if err != nil {
				return nil, err
			}
			var preco float64
			for i, u := range unidades {
				if i == 0 || u.IsPadraoVenda {
					preco = u.PrecoVenda
				}
				if u.IsPadraoVenda {
					break
				}
			}
			out = append(out, FichaResumo{Produto: p, PrecoVenda: preco, Ingredientes: ingredientes, CustoTeorico: custo, Disponivel: disponivel})
		}
		return out, nil
	})
}

func (s *FichasService) DefinirReceita(ctx context.Context, produtoID int64, ingredientes []usecase.IngredienteInput) ([]domain.FichaTecnica, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]domain.FichaTecnica, error) { return a.DefinirReceita(produtoID, ingredientes) })
}

func (s *FichasService) DuplicarReceita(ctx context.Context, origemID, destinoID int64) ([]domain.FichaTecnica, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) ([]domain.FichaTecnica, error) { return a.DuplicarReceita(origemID, destinoID) })
}

func (s *FichasService) RegistarProducao(ctx context.Context, p usecase.ProducaoParams) (domain.MovimentoStock, error) {
	return comLoja(ctx, s.lojas, func(a *api.App) (domain.MovimentoStock, error) { return a.RegistarProducao(p) })
}
