package usecase

import (
	"context"
	"fmt"
	"math"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// FichaTecnicaService cobre a receita de produtos compostos/semi-elaborados
// (SDD 3.3): definir/editar a receita, calcular o custo teórico, simular
// quanto dá para vender com o stock atual, e produzir semi-elaborados em
// lote (3.3.2).
type FichaTecnicaService struct {
	fichas  *repository.FichaTecnicaRepository
	produto *repository.ProdutoRepository
	stock   *repository.StockRepository
}

func NewFichaTecnicaService(
	fichas *repository.FichaTecnicaRepository,
	produto *repository.ProdutoRepository,
	stock *repository.StockRepository,
) *FichaTecnicaService {
	return &FichaTecnicaService{fichas: fichas, produto: produto, stock: stock}
}

func (s *FichaTecnicaService) ListarIngredientes(ctx context.Context, produtoCompostoID int64) ([]domain.FichaTecnica, error) {
	return s.fichas.ListarIngredientes(ctx, produtoCompostoID)
}

type IngredienteInput struct {
	ProdutoIngredienteID int64
	Quantidade           float64
	PerdaPercentual      float64
}

// DefinirReceita substitui a receita inteira do produto composto/
// semi-elaborado — mais simples e mais seguro para um ecrã de edição do
// que reconciliar add/remove/editar linha a linha (ver
// FichaTecnicaRepository.DefinirReceita).
func (s *FichaTecnicaService) DefinirReceita(ctx context.Context, produtoCompostoID int64, ingredientes []IngredienteInput) ([]domain.FichaTecnica, error) {
	produto, err := s.produto.Obter(ctx, produtoCompostoID)
	if err != nil {
		return nil, err
	}
	if produto.TipoProduto != domain.TipoProdutoComposto && produto.TipoProduto != domain.TipoProdutoSemiElaborado {
		return nil, fmt.Errorf("ficha técnica só se aplica a produtos compostos ou semi-elaborados")
	}
	if len(ingredientes) == 0 {
		return nil, fmt.Errorf("ficha técnica precisa de pelo menos um ingrediente")
	}
	if produto.ID == 0 {
		return nil, fmt.Errorf("produto inválido")
	}

	linhas := make([]domain.FichaTecnica, 0, len(ingredientes))
	for _, ing := range ingredientes {
		if ing.Quantidade <= 0 {
			return nil, fmt.Errorf("quantidade do ingrediente deve ser positiva")
		}
		if ing.PerdaPercentual < 0 || ing.PerdaPercentual >= 100 {
			return nil, fmt.Errorf("perda percentual deve estar entre 0 e 100")
		}
		if ing.ProdutoIngredienteID == produtoCompostoID {
			return nil, fmt.Errorf("um produto não pode ser ingrediente de si próprio")
		}

		// Só simples/semi-elaborado têm stock real para consumir — um
		// composto usado como ingrediente exigiria expandir duas fichas
		// técnicas em cascata na venda, o que o SDD resolve com
		// semi-elaborados produzidos em lote (3.3.2), não com aninhamento.
		ingrediente, err := s.produto.Obter(ctx, ing.ProdutoIngredienteID)
		if err != nil {
			return nil, err
		}
		if ingrediente.TipoProduto == domain.TipoProdutoComposto {
			return nil, fmt.Errorf("%q é composto — só simples/semi-elaborado podem ser ingredientes", ingrediente.Nome)
		}

		linhas = append(linhas, domain.FichaTecnica{
			ProdutoIngredienteID: ing.ProdutoIngredienteID,
			Quantidade:           ing.Quantidade,
			PerdaPercentual:      ing.PerdaPercentual,
		})
	}

	return s.fichas.DefinirReceita(ctx, produtoCompostoID, linhas)
}

// DuplicarReceita copia a ficha técnica de um produto para outro (SDD 4.2:
// "útil para variantes, ex. Caipirinha de Vodka vs. Caipirinha de Gin").
func (s *FichaTecnicaService) DuplicarReceita(ctx context.Context, produtoOrigemID, produtoDestinoID int64) ([]domain.FichaTecnica, error) {
	origem, err := s.fichas.ListarIngredientes(ctx, produtoOrigemID)
	if err != nil {
		return nil, err
	}
	if len(origem) == 0 {
		return nil, fmt.Errorf("produto de origem não tem ficha técnica definida")
	}

	inputs := make([]IngredienteInput, 0, len(origem))
	for _, ing := range origem {
		inputs = append(inputs, IngredienteInput{
			ProdutoIngredienteID: ing.ProdutoIngredienteID,
			Quantidade:           ing.Quantidade,
			PerdaPercentual:      ing.PerdaPercentual,
		})
	}
	return s.DefinirReceita(ctx, produtoDestinoID, inputs)
}

// CalcularCustoTeorico soma quantidade_receita × custo_médio_atual de cada
// ingrediente (SDD 4.2) — mostrado lado a lado com o preço de venda no
// ecrã de edição, para visibilidade imediata da margem.
func (s *FichaTecnicaService) CalcularCustoTeorico(ctx context.Context, produtoCompostoID, armazemID int64) (float64, error) {
	ingredientes, err := s.fichas.ListarIngredientes(ctx, produtoCompostoID)
	if err != nil {
		return 0, err
	}
	var total float64
	for _, ing := range ingredientes {
		stock, err := s.stock.ObterStock(ctx, ing.ProdutoIngredienteID, armazemID)
		if err != nil {
			return 0, err
		}
		total += ing.ConsumoReal() * stock.CustoMedioPonderado
	}
	return total, nil
}

// SimularDisponibilidade responde "quantas unidades deste composto dá
// para preparar com o stock atual" (SDD 4.2) — limitado pelo ingrediente
// mais restritivo, arredondado para baixo (não vendemos meia caipirinha).
func (s *FichaTecnicaService) SimularDisponibilidade(ctx context.Context, produtoCompostoID, armazemID int64) (float64, error) {
	ingredientes, err := s.fichas.ListarIngredientes(ctx, produtoCompostoID)
	if err != nil {
		return 0, err
	}
	if len(ingredientes) == 0 {
		return 0, nil
	}

	disponivel := math.Inf(1)
	for _, ing := range ingredientes {
		stock, err := s.stock.ObterStock(ctx, ing.ProdutoIngredienteID, armazemID)
		if err != nil {
			return 0, err
		}
		consumo := ing.ConsumoReal()
		possivel := stock.Quantidade / consumo
		if possivel < disponivel {
			disponivel = possivel
		}
	}
	if math.IsInf(disponivel, 1) || disponivel < 0 {
		return 0, nil
	}
	return math.Floor(disponivel), nil
}

// --- Produção em lote de semi-elaborados (SDD 3.3.2) ------------------------

type ProducaoParams struct {
	ProdutoSemiElaboradoID int64
	ArmazemID              int64
	Quantidade             float64
	UtilizadorID           int64
}

// RegistarProducao consome os ingredientes da receita do semi-elaborado e
// cria stock dele no armazém indicado — tudo numa transação (SDD 3.3.2:
// "consome os ingredientes... e cria stock do semi-elaborado").
func (s *FichaTecnicaService) RegistarProducao(ctx context.Context, p ProducaoParams) (domain.MovimentoStock, error) {
	if p.Quantidade <= 0 {
		return domain.MovimentoStock{}, fmt.Errorf("quantidade a produzir deve ser positiva")
	}

	produto, err := s.produto.Obter(ctx, p.ProdutoSemiElaboradoID)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	if produto.TipoProduto != domain.TipoProdutoSemiElaborado {
		return domain.MovimentoStock{}, fmt.Errorf("produção em lote só se aplica a produtos semi-elaborados")
	}

	ingredientes, err := s.fichas.ListarIngredientes(ctx, p.ProdutoSemiElaboradoID)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	if len(ingredientes) == 0 {
		return domain.MovimentoStock{}, fmt.Errorf("%q não tem ficha técnica definida", produto.Nome)
	}

	var movimentoProducao domain.MovimentoStock
	err = s.stock.WithTx(ctx, func(tx *repository.StockRepository) error {
		var custoTotalIngredientes float64

		for _, ing := range ingredientes {
			consumoTotal := ing.ConsumoReal() * p.Quantidade

			stockAtual, err := tx.ObterStock(ctx, ing.ProdutoIngredienteID, p.ArmazemID)
			if err != nil {
				return err
			}
			if stockAtual.Quantidade < consumoTotal {
				ingrediente, errNome := s.produto.Obter(ctx, ing.ProdutoIngredienteID)
				nome := ingrediente.Nome
				if errNome != nil {
					nome = fmt.Sprintf("#%d", ing.ProdutoIngredienteID)
				}
				return fmt.Errorf("%w: %s — disponível %.3f, pedido %.3f", domain.ErrStockInsuficiente, nome, stockAtual.Quantidade, consumoTotal)
			}
			custoTotalIngredientes += consumoTotal * stockAtual.CustoMedioPonderado

			armazemIDCopy := p.ArmazemID
			custoCopy := stockAtual.CustoMedioPonderado
			if _, err := tx.CriarMovimento(ctx, repository.NovoMovimento{
				ProdutoID:       ing.ProdutoIngredienteID,
				ArmazemOrigemID: &armazemIDCopy,
				Tipo:            domain.TipoMovimentoProducao,
				Quantidade:      consumoTotal,
				CustoUnitario:   &custoCopy,
				UtilizadorID:    p.UtilizadorID,
			}); err != nil {
				return fmt.Errorf("consumir ingrediente: %w", err)
			}
		}

		custoUnitarioProduzido := custoTotalIngredientes / p.Quantidade

		stockAtualProduzido, err := tx.ObterStock(ctx, p.ProdutoSemiElaboradoID, p.ArmazemID)
		if err != nil {
			return err
		}
		novoCusto := custoMedioPonderado(stockAtualProduzido.Quantidade, stockAtualProduzido.CustoMedioPonderado, p.Quantidade, custoUnitarioProduzido)

		armazemIDCopy := p.ArmazemID
		custoProduzidoCopy := custoUnitarioProduzido
		movimentoProducao, err = tx.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        p.ProdutoSemiElaboradoID,
			ArmazemDestinoID: &armazemIDCopy,
			Tipo:             domain.TipoMovimentoProducao,
			Quantidade:       p.Quantidade,
			CustoUnitario:    &custoProduzidoCopy,
			UtilizadorID:     p.UtilizadorID,
		})
		if err != nil {
			return fmt.Errorf("criar stock do produzido: %w", err)
		}

		return tx.DefinirCustoMedio(ctx, p.ProdutoSemiElaboradoID, p.ArmazemID, novoCusto)
	})
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	return movimentoProducao, nil
}
