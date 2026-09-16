package usecase

import (
	"context"
	"fmt"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// FarmaciaService cobre o módulo de nicho da farmácia (SDD 9.4): entrada
// de stock por lote/validade (consumido por FEFO nas vendas — ver
// consumirLoteFEFO em venda_service.go), alertas de validade próxima, e
// registo de receitas médicas associadas a uma venda.
type FarmaciaService struct {
	lotes    *repository.LoteRepository
	produto  *repository.ProdutoRepository
	armazem  *repository.ArmazemRepository
	receitas *repository.ReceitaMedicaRepository
}

func NewFarmaciaService(
	lotes *repository.LoteRepository,
	produto *repository.ProdutoRepository,
	armazem *repository.ArmazemRepository,
	receitas *repository.ReceitaMedicaRepository,
) *FarmaciaService {
	return &FarmaciaService{lotes: lotes, produto: produto, armazem: armazem, receitas: receitas}
}

type EntradaComLoteParams struct {
	ProdutoID     int64
	ArmazemID     int64
	NumeroLote    string
	DataValidade  time.Time
	Quantidade    float64
	CustoUnitario float64
	UtilizadorID  int64
}

// RegistarEntradaComLote é o único caminho de entrada para produtos
// LoteControlado — StockService.RegistarEntrada rejeita-os explicitamente
// (ver stock_service.go) porque uma entrada sem lote deixaria stock_atual
// e lotes_produto divergentes, e o FEFO da venda leria lotes vazios.
func (s *FarmaciaService) RegistarEntradaComLote(ctx context.Context, p EntradaComLoteParams) (domain.Lote, domain.MovimentoStock, error) {
	if p.Quantidade <= 0 {
		return domain.Lote{}, domain.MovimentoStock{}, fmt.Errorf("quantidade deve ser positiva")
	}
	if p.CustoUnitario < 0 {
		return domain.Lote{}, domain.MovimentoStock{}, fmt.Errorf("custo unitário não pode ser negativo")
	}
	if p.NumeroLote == "" {
		return domain.Lote{}, domain.MovimentoStock{}, fmt.Errorf("número de lote é obrigatório")
	}
	if p.DataValidade.IsZero() {
		return domain.Lote{}, domain.MovimentoStock{}, fmt.Errorf("data de validade é obrigatória")
	}

	produto, err := s.produto.Obter(ctx, p.ProdutoID)
	if err != nil {
		return domain.Lote{}, domain.MovimentoStock{}, err
	}
	if !produto.Ativo {
		return domain.Lote{}, domain.MovimentoStock{}, domain.ErrProdutoInativo
	}
	if !produto.LoteControlado {
		return domain.Lote{}, domain.MovimentoStock{}, fmt.Errorf("%q não controla lotes — use a entrada normal de stock", produto.Nome)
	}

	armazem, err := s.armazem.Obter(ctx, p.ArmazemID)
	if err != nil {
		return domain.Lote{}, domain.MovimentoStock{}, err
	}
	if !armazem.Ativo {
		return domain.Lote{}, domain.MovimentoStock{}, domain.ErrArmazemInativo
	}

	var lote domain.Lote
	var movimento domain.MovimentoStock
	err = s.lotes.WithTx(ctx, func(tx *repository.LoteRepository) error {
		atual, err := tx.Stock.ObterStock(ctx, p.ProdutoID, p.ArmazemID)
		if err != nil {
			return err
		}
		novoCusto := custoMedioPonderado(atual.Quantidade, atual.CustoMedioPonderado, p.Quantidade, p.CustoUnitario)

		armazemID := p.ArmazemID
		custo := p.CustoUnitario
		movimento, err = tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        p.ProdutoID,
			ArmazemDestinoID: &armazemID,
			Tipo:             domain.TipoMovimentoEntrada,
			Quantidade:       p.Quantidade,
			CustoUnitario:    &custo,
			UtilizadorID:     p.UtilizadorID,
		})
		if err != nil {
			return fmt.Errorf("registar movimento de entrada: %w", err)
		}

		if err := tx.Stock.DefinirCustoMedio(ctx, p.ProdutoID, p.ArmazemID, novoCusto); err != nil {
			return fmt.Errorf("atualizar custo médio: %w", err)
		}

		lote, err = tx.CriarLote(ctx, domain.Lote{
			ProdutoID:     p.ProdutoID,
			ArmazemID:     p.ArmazemID,
			NumeroLote:    p.NumeroLote,
			DataValidade:  p.DataValidade,
			Quantidade:    p.Quantidade,
			CustoUnitario: p.CustoUnitario,
		})
		if err != nil {
			return fmt.Errorf("criar lote: %w", err)
		}
		return nil
	})
	if err != nil {
		return domain.Lote{}, domain.MovimentoStock{}, err
	}
	return lote, movimento, nil
}

func (s *FarmaciaService) ListarLotes(ctx context.Context, produtoID, armazemID int64) ([]domain.Lote, error) {
	return s.lotes.ListarFEFO(ctx, produtoID, armazemID)
}

// ListarLotesAVencer cobre os "alertas de validade próxima" do SDD 9.4 —
// lotes com stock que vencem dentro de `dentroDeDias` dias a partir de hoje.
func (s *FarmaciaService) ListarLotesAVencer(ctx context.Context, negocioID int64, dentroDeDias int) ([]domain.Lote, error) {
	ate := time.Now().AddDate(0, 0, dentroDeDias)
	return s.lotes.ListarAVencer(ctx, negocioID, ate)
}

type ReceitaMedicaInput struct {
	VendaID       int64
	ClienteID     int64
	Medico        string
	NumeroReceita string
	ValidadoPor   int64
}

func (s *FarmaciaService) RegistarReceitaMedica(ctx context.Context, p ReceitaMedicaInput) (domain.ReceitaMedica, error) {
	if p.Medico == "" {
		return domain.ReceitaMedica{}, fmt.Errorf("médico é obrigatório")
	}
	if p.NumeroReceita == "" {
		return domain.ReceitaMedica{}, fmt.Errorf("número da receita é obrigatório")
	}
	return s.receitas.Criar(ctx, domain.ReceitaMedica{
		VendaID:       p.VendaID,
		ClienteID:     p.ClienteID,
		Medico:        p.Medico,
		NumeroReceita: p.NumeroReceita,
		ValidadoPor:   p.ValidadoPor,
	})
}

func (s *FarmaciaService) ListarReceitasPorVenda(ctx context.Context, vendaID int64) ([]domain.ReceitaMedica, error) {
	return s.receitas.ListarPorVenda(ctx, vendaID)
}
