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

// --- Saídas e transferência de um lote escolhido -------------------------
//
// Produtos LoteControlado não podem sair pelo StockService genérico (ver
// validarProdutoSemLotes): cada perda, consumo ou transferência aponta o
// lote concreto, para lotes_produto e stock_atual andarem sempre juntos.

type SaidaLoteParams struct {
	LoteID       int64
	Quantidade   float64
	Motivo       string
	UtilizadorID int64
}

func (s *FarmaciaService) RegistarPerdaDeLote(ctx context.Context, p SaidaLoteParams) (domain.MovimentoStock, error) {
	return s.registarSaidaDeLote(ctx, domain.TipoMovimentoPerda, p)
}

func (s *FarmaciaService) RegistarConsumoInternoDeLote(ctx context.Context, p SaidaLoteParams) (domain.MovimentoStock, error) {
	return s.registarSaidaDeLote(ctx, domain.TipoMovimentoConsumoInterno, p)
}

func (s *FarmaciaService) registarSaidaDeLote(ctx context.Context, tipo domain.TipoMovimento, p SaidaLoteParams) (domain.MovimentoStock, error) {
	if p.Quantidade <= 0 {
		return domain.MovimentoStock{}, fmt.Errorf("quantidade deve ser positiva")
	}
	if p.Motivo == "" {
		return domain.MovimentoStock{}, domain.ErrMotivoObrigatorio
	}

	var movimento domain.MovimentoStock
	err := s.lotes.WithTx(ctx, func(tx *repository.LoteRepository) error {
		lote, err := s.loteParaSaida(ctx, tx, p.LoteID, p.Quantidade)
		if err != nil {
			return err
		}
		if err := tx.AtualizarQuantidade(ctx, lote.ID, lote.Quantidade-p.Quantidade); err != nil {
			return fmt.Errorf("atualizar lote %s: %w", lote.NumeroLote, err)
		}

		origemID := lote.ArmazemID
		custo := lote.CustoUnitario
		motivo := p.Motivo
		referencia := "lote:" + lote.NumeroLote
		movimento, err = tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:       lote.ProdutoID,
			ArmazemOrigemID: &origemID,
			Tipo:            tipo,
			Quantidade:      p.Quantidade,
			CustoUnitario:   &custo,
			Motivo:          &motivo,
			ReferenciaID:    &referencia,
			UtilizadorID:    p.UtilizadorID,
		})
		return err
	})
	return movimento, err
}

type TransferenciaLoteParams struct {
	LoteID           int64
	ArmazemDestinoID int64
	Quantidade       float64
	UtilizadorID     int64
}

// TransferirLote move unidades de um lote para outro armazém mantendo o
// número de lote, a validade e o custo — no destino soma a um lote com o
// mesmo número se já existir, senão cria-o.
func (s *FarmaciaService) TransferirLote(ctx context.Context, p TransferenciaLoteParams) (domain.MovimentoStock, error) {
	if p.Quantidade <= 0 {
		return domain.MovimentoStock{}, fmt.Errorf("quantidade deve ser positiva")
	}
	destino, err := s.armazem.Obter(ctx, p.ArmazemDestinoID)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	if !destino.Ativo {
		return domain.MovimentoStock{}, domain.ErrArmazemInativo
	}

	var movimento domain.MovimentoStock
	err = s.lotes.WithTx(ctx, func(tx *repository.LoteRepository) error {
		lote, err := s.loteParaSaida(ctx, tx, p.LoteID, p.Quantidade)
		if err != nil {
			return err
		}
		if lote.ArmazemID == p.ArmazemDestinoID {
			return domain.ErrArmazemIgual
		}

		destinoAtual, err := tx.Stock.ObterStock(ctx, lote.ProdutoID, p.ArmazemDestinoID)
		if err != nil {
			return err
		}
		novoCustoDestino := custoMedioPonderado(destinoAtual.Quantidade, destinoAtual.CustoMedioPonderado, p.Quantidade, lote.CustoUnitario)

		if err := tx.AtualizarQuantidade(ctx, lote.ID, lote.Quantidade-p.Quantidade); err != nil {
			return fmt.Errorf("atualizar lote %s: %w", lote.NumeroLote, err)
		}

		loteDestino, existe, err := tx.ObterPorNumero(ctx, lote.ProdutoID, p.ArmazemDestinoID, lote.NumeroLote)
		if err != nil {
			return err
		}
		if existe {
			if !loteDestino.DataValidade.Equal(lote.DataValidade) {
				return fmt.Errorf("o lote %s já existe no destino com outra validade (%s)", lote.NumeroLote, loteDestino.DataValidade.Format("2006-01-02"))
			}
			if err := tx.AtualizarQuantidade(ctx, loteDestino.ID, loteDestino.Quantidade+p.Quantidade); err != nil {
				return fmt.Errorf("atualizar lote %s no destino: %w", lote.NumeroLote, err)
			}
		} else {
			if _, err := tx.CriarLote(ctx, domain.Lote{
				ProdutoID:     lote.ProdutoID,
				ArmazemID:     p.ArmazemDestinoID,
				NumeroLote:    lote.NumeroLote,
				DataValidade:  lote.DataValidade,
				Quantidade:    p.Quantidade,
				CustoUnitario: lote.CustoUnitario,
			}); err != nil {
				return fmt.Errorf("criar lote %s no destino: %w", lote.NumeroLote, err)
			}
		}

		origemID, destinoID := lote.ArmazemID, p.ArmazemDestinoID
		custo := lote.CustoUnitario
		referencia := "lote:" + lote.NumeroLote
		movimento, err = tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        lote.ProdutoID,
			ArmazemOrigemID:  &origemID,
			ArmazemDestinoID: &destinoID,
			Tipo:             domain.TipoMovimentoTransferencia,
			Quantidade:       p.Quantidade,
			CustoUnitario:    &custo,
			ReferenciaID:     &referencia,
			UtilizadorID:     p.UtilizadorID,
		})
		if err != nil {
			return err
		}
		return tx.Stock.DefinirCustoMedio(ctx, lote.ProdutoID, p.ArmazemDestinoID, novoCustoDestino)
	})
	return movimento, err
}

// loteParaSaida carrega o lote dentro da transação e valida que o produto
// e o armazém continuam ativos e que o lote tem a quantidade pedida.
func (s *FarmaciaService) loteParaSaida(ctx context.Context, tx *repository.LoteRepository, loteID int64, quantidade float64) (domain.Lote, error) {
	lote, err := tx.Obter(ctx, loteID)
	if err != nil {
		return domain.Lote{}, err
	}
	produto, err := s.produto.Obter(ctx, lote.ProdutoID)
	if err != nil {
		return domain.Lote{}, err
	}
	if !produto.Ativo {
		return domain.Lote{}, domain.ErrProdutoInativo
	}
	armazem, err := s.armazem.Obter(ctx, lote.ArmazemID)
	if err != nil {
		return domain.Lote{}, err
	}
	if !armazem.Ativo {
		return domain.Lote{}, domain.ErrArmazemInativo
	}
	if lote.Quantidade < quantidade {
		return domain.Lote{}, fmt.Errorf("%w: lote %s tem %s, pedido %s", domain.ErrStockInsuficiente, lote.NumeroLote, qtd(lote.Quantidade), qtd(quantidade))
	}
	return lote, nil
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
