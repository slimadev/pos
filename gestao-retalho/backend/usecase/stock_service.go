package usecase

import (
	"context"
	"fmt"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// StockService é o único caminho para alterar stock (SDD secção 4.4-4.8):
// entrada, transferência, perda, consumo interno e inventário passam todos
// por aqui, nunca por INSERTs diretos noutro lugar.
type StockService struct {
	stock   *repository.StockRepository
	produto *repository.ProdutoRepository
	armazem *repository.ArmazemRepository
	inv     *repository.InventarioRepository
	lotes   *repository.LoteRepository
}

func NewStockService(
	stock *repository.StockRepository,
	produto *repository.ProdutoRepository,
	armazem *repository.ArmazemRepository,
	inv *repository.InventarioRepository,
	lotes *repository.LoteRepository,
) *StockService {
	return &StockService{stock: stock, produto: produto, armazem: armazem, inv: inv, lotes: lotes}
}

func (s *StockService) ObterStock(ctx context.Context, produtoID, armazemID int64) (domain.StockItem, error) {
	return s.stock.ObterStock(ctx, produtoID, armazemID)
}

func (s *StockService) ListarStockPorArmazem(ctx context.Context, armazemID int64) ([]domain.StockItem, error) {
	return s.stock.ListarPorArmazem(ctx, armazemID)
}

func (s *StockService) ListarMovimentos(ctx context.Context, produtoID int64) ([]domain.MovimentoStock, error) {
	return s.stock.ListarMovimentosPorProduto(ctx, produtoID)
}

// --- Entrada (SDD 4.4) -------------------------------------------------

type EntradaParams struct {
	ProdutoID        int64
	ArmazemDestinoID int64
	Quantidade       float64
	CustoUnitario    float64
	UnidadeProdutoID *int64
	ReferenciaID     *string
	UtilizadorID     int64
}

func (s *StockService) RegistarEntrada(ctx context.Context, p EntradaParams) (domain.MovimentoStock, error) {
	if p.Quantidade <= 0 {
		return domain.MovimentoStock{}, fmt.Errorf("quantidade deve ser positiva")
	}
	if p.CustoUnitario < 0 {
		return domain.MovimentoStock{}, fmt.Errorf("custo unitário não pode ser negativo")
	}
	produto, err := s.validarProdutoParaStock(ctx, p.ProdutoID)
	if err != nil {
		return domain.MovimentoStock{}, err
	}
	// Produtos com rastreio por lote (Fase 6, SDD 9.4) têm de entrar com
	// número de lote e validade — usar o fluxo genérico aqui deixaria
	// stock_atual e lotes_produto divergirem, e o FEFO da venda leria
	// lotes vazios. FichaTecnicaService/FarmaciaService.RegistarEntradaComLote
	// é o caminho certo para estes.
	if produto.LoteControlado {
		return domain.MovimentoStock{}, fmt.Errorf("%q controla lotes — registe a entrada com número de lote e validade", produto.Nome)
	}
	if _, err := s.validarArmazemAtivo(ctx, p.ArmazemDestinoID); err != nil {
		return domain.MovimentoStock{}, err
	}

	var movimento domain.MovimentoStock
	err = s.stock.WithTx(ctx, func(tx *repository.StockRepository) error {
		atual, err := tx.ObterStock(ctx, p.ProdutoID, p.ArmazemDestinoID)
		if err != nil {
			return err
		}
		novoCusto := custoMedioPonderado(atual.Quantidade, atual.CustoMedioPonderado, p.Quantidade, p.CustoUnitario)

		destino := p.ArmazemDestinoID
		custo := p.CustoUnitario
		movimento, err = tx.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        p.ProdutoID,
			ArmazemDestinoID: &destino,
			Tipo:             domain.TipoMovimentoEntrada,
			Quantidade:       p.Quantidade,
			CustoUnitario:    &custo,
			UnidadeProdutoID: p.UnidadeProdutoID,
			ReferenciaID:     p.ReferenciaID,
			UtilizadorID:     p.UtilizadorID,
		})
		if err != nil {
			return err
		}
		return tx.DefinirCustoMedio(ctx, p.ProdutoID, p.ArmazemDestinoID, novoCusto)
	})
	return movimento, err
}

// --- Transferência (SDD 4.5) -------------------------------------------

type TransferenciaParams struct {
	ProdutoID        int64
	ArmazemOrigemID  int64
	ArmazemDestinoID int64
	Quantidade       float64
	UnidadeProdutoID *int64
	UtilizadorID     int64
}

func (s *StockService) TransferirStock(ctx context.Context, p TransferenciaParams) (domain.MovimentoStock, error) {
	if p.Quantidade <= 0 {
		return domain.MovimentoStock{}, fmt.Errorf("quantidade deve ser positiva")
	}
	if p.ArmazemOrigemID == p.ArmazemDestinoID {
		return domain.MovimentoStock{}, domain.ErrArmazemIgual
	}
	if _, err := s.validarProdutoSemLotes(ctx, p.ProdutoID); err != nil {
		return domain.MovimentoStock{}, err
	}
	if _, err := s.validarArmazemAtivo(ctx, p.ArmazemOrigemID); err != nil {
		return domain.MovimentoStock{}, err
	}
	if _, err := s.validarArmazemAtivo(ctx, p.ArmazemDestinoID); err != nil {
		return domain.MovimentoStock{}, err
	}

	var movimento domain.MovimentoStock
	err := s.stock.WithTx(ctx, func(tx *repository.StockRepository) error {
		origemAtual, err := tx.ObterStock(ctx, p.ProdutoID, p.ArmazemOrigemID)
		if err != nil {
			return err
		}
		if origemAtual.Quantidade < p.Quantidade {
			return fmt.Errorf("%w: disponível %.3f, pedido %.3f", domain.ErrStockInsuficiente, origemAtual.Quantidade, p.Quantidade)
		}

		destinoAtual, err := tx.ObterStock(ctx, p.ProdutoID, p.ArmazemDestinoID)
		if err != nil {
			return err
		}
		// O lote transferido entra no destino ao custo médio da origem —
		// transferir não cria nem apaga custo, só o move.
		novoCustoDestino := custoMedioPonderado(destinoAtual.Quantidade, destinoAtual.CustoMedioPonderado, p.Quantidade, origemAtual.CustoMedioPonderado)

		origemID, destinoID := p.ArmazemOrigemID, p.ArmazemDestinoID
		movimento, err = tx.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        p.ProdutoID,
			ArmazemOrigemID:  &origemID,
			ArmazemDestinoID: &destinoID,
			Tipo:             domain.TipoMovimentoTransferencia,
			Quantidade:       p.Quantidade,
			UnidadeProdutoID: p.UnidadeProdutoID,
			UtilizadorID:     p.UtilizadorID,
		})
		if err != nil {
			return err
		}
		return tx.DefinirCustoMedio(ctx, p.ProdutoID, p.ArmazemDestinoID, novoCustoDestino)
	})
	return movimento, err
}

// --- Perda e consumo interno (SDD 4.6/4.7) ------------------------------

type SaidaParams struct {
	ProdutoID        int64
	ArmazemOrigemID  int64
	Quantidade       float64
	Motivo           string
	UnidadeProdutoID *int64
	UtilizadorID     int64
}

func (s *StockService) RegistarPerda(ctx context.Context, p SaidaParams) (domain.MovimentoStock, error) {
	return s.registarSaida(ctx, domain.TipoMovimentoPerda, p)
}

func (s *StockService) RegistarConsumoInterno(ctx context.Context, p SaidaParams) (domain.MovimentoStock, error) {
	return s.registarSaida(ctx, domain.TipoMovimentoConsumoInterno, p)
}

func (s *StockService) registarSaida(ctx context.Context, tipo domain.TipoMovimento, p SaidaParams) (domain.MovimentoStock, error) {
	if p.Quantidade <= 0 {
		return domain.MovimentoStock{}, fmt.Errorf("quantidade deve ser positiva")
	}
	if p.Motivo == "" {
		return domain.MovimentoStock{}, domain.ErrMotivoObrigatorio
	}
	if _, err := s.validarProdutoSemLotes(ctx, p.ProdutoID); err != nil {
		return domain.MovimentoStock{}, err
	}
	if _, err := s.validarArmazemAtivo(ctx, p.ArmazemOrigemID); err != nil {
		return domain.MovimentoStock{}, err
	}

	var movimento domain.MovimentoStock
	err := s.stock.WithTx(ctx, func(tx *repository.StockRepository) error {
		atual, err := tx.ObterStock(ctx, p.ProdutoID, p.ArmazemOrigemID)
		if err != nil {
			return err
		}
		if atual.Quantidade < p.Quantidade {
			return fmt.Errorf("%w: disponível %.3f, pedido %.3f", domain.ErrStockInsuficiente, atual.Quantidade, p.Quantidade)
		}

		origemID := p.ArmazemOrigemID
		motivo := p.Motivo
		custo := atual.CustoMedioPonderado
		movimento, err = tx.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        p.ProdutoID,
			ArmazemOrigemID:  &origemID,
			Tipo:             tipo,
			Quantidade:       p.Quantidade,
			CustoUnitario:    &custo,
			UnidadeProdutoID: p.UnidadeProdutoID,
			Motivo:           &motivo,
			UtilizadorID:     p.UtilizadorID,
		})
		return err
	})
	return movimento, err
}

// --- Inventário (SDD 4.8) -----------------------------------------------

func (s *StockService) IniciarInventario(ctx context.Context, negocioID, armazemID, utilizadorID int64) (domain.Inventario, []domain.InventarioItem, error) {
	if _, err := s.validarArmazemAtivo(ctx, armazemID); err != nil {
		return domain.Inventario{}, nil, err
	}

	snapshot, err := s.stock.ListarParaInventario(ctx, negocioID, armazemID)
	if err != nil {
		return domain.Inventario{}, nil, err
	}

	var inv domain.Inventario
	var itens []domain.InventarioItem
	err = s.inv.WithTx(ctx, func(tx *repository.InventarioRepository) error {
		var err error
		inv, err = tx.Criar(ctx, negocioID, armazemID, utilizadorID)
		if err != nil {
			return err
		}
		for _, item := range snapshot {
			if err := tx.CriarItem(ctx, inv.ID, item.ProdutoID, item.QuantidadeSistema); err != nil {
				return err
			}
		}
		itens, err = tx.ListarItens(ctx, inv.ID)
		return err
	})
	return inv, itens, err
}

func (s *StockService) ListarInventarioItens(ctx context.Context, inventarioID int64) ([]domain.InventarioItem, error) {
	return s.inv.ListarItens(ctx, inventarioID)
}

// RegistarContagem grava a quantidade contada fisicamente para um item de
// inventário e calcula a diferença face ao que o sistema esperava.
func (s *StockService) RegistarContagem(ctx context.Context, itemID int64, quantidadeContada float64) (domain.InventarioItem, error) {
	if quantidadeContada < 0 {
		return domain.InventarioItem{}, fmt.Errorf("quantidade contada não pode ser negativa")
	}
	item, err := s.inv.ObterItem(ctx, itemID)
	if err != nil {
		return domain.InventarioItem{}, err
	}
	diferenca := quantidadeContada - item.QuantidadeSistema
	if err := s.inv.RegistarContagem(ctx, itemID, quantidadeContada, diferenca); err != nil {
		return domain.InventarioItem{}, err
	}
	return s.inv.ObterItem(ctx, itemID)
}

// FecharInventario gera um movimento `ajuste_inventario` por cada item com
// diferença não nula e fecha o inventário. Itens nunca contados (ainda sem
// quantidade_contada) são ignorados — fica registado no histórico do
// inventário para o operador perceber que ficaram por contar.
//
// Produtos LoteControlado não podem ajustar só stock_atual (lotes_produto
// ficaria a mostrar unidades que não existem): uma falta sai dos lotes por
// FEFO, vencidos primeiro; uma sobra não tem número de lote nem validade,
// por isso bloqueia o fecho até o operador a registar com "Entrada c/ lote".
func (s *StockService) FecharInventario(ctx context.Context, inventarioID, utilizadorID int64) ([]domain.MovimentoStock, error) {
	inv, err := s.inv.Obter(ctx, inventarioID)
	if err != nil {
		return nil, err
	}
	if inv.Status == domain.StatusInventarioFechado {
		return nil, domain.ErrInventarioFechado
	}

	itens, err := s.inv.ListarItens(ctx, inventarioID)
	if err != nil {
		return nil, err
	}

	// Valida tudo antes de escrever o primeiro ajuste, para uma sobra num
	// produto com lotes não deixar o inventário meio ajustado.
	produtos := make(map[int64]domain.Produto)
	for _, item := range itens {
		if item.Diferenca == nil || *item.Diferenca == 0 {
			continue
		}
		p, err := s.produto.Obter(ctx, item.ProdutoID)
		if err != nil {
			return nil, err
		}
		produtos[item.ProdutoID] = p
		if p.LoteControlado && *item.Diferenca > 0 {
			return nil, fmt.Errorf(
				"%q tem %.3f a mais do que o sistema e controla lotes — registe essas unidades com \"Entrada c/ lote\" (número e validade) e corrija a contagem para %.3f antes de fechar",
				p.Nome, *item.Diferenca, item.QuantidadeSistema)
		}
	}

	var movimentos []domain.MovimentoStock
	for _, item := range itens {
		if item.Diferenca == nil || *item.Diferenca == 0 {
			continue
		}
		diferenca := *item.Diferenca
		produto := produtos[item.ProdutoID]

		var novos []domain.MovimentoStock
		if produto.LoteControlado {
			novos, err = s.ajustarFaltaPorLotes(ctx, produto, inv, -diferenca, utilizadorID)
		} else {
			novos, err = s.ajustarSemLotes(ctx, item.ProdutoID, inv.ArmazemID, diferenca, utilizadorID)
		}
		if err != nil {
			return movimentos, err
		}
		if err := s.inv.MarcarAjustado(ctx, item.ID); err != nil {
			return movimentos, err
		}
		movimentos = append(movimentos, novos...)
	}

	if err := s.inv.Fechar(ctx, inventarioID); err != nil {
		return movimentos, err
	}
	return movimentos, nil
}

func (s *StockService) ajustarSemLotes(ctx context.Context, produtoID, armazemID int64, diferenca float64, utilizadorID int64) ([]domain.MovimentoStock, error) {
	var origemID, destinoID *int64
	quantidade := diferenca
	if diferenca > 0 {
		destinoID = &armazemID
	} else {
		origemID = &armazemID
		quantidade = -diferenca
	}

	var movimento domain.MovimentoStock
	err := s.stock.WithTx(ctx, func(tx *repository.StockRepository) error {
		var err error
		movimento, err = tx.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:        produtoID,
			ArmazemOrigemID:  origemID,
			ArmazemDestinoID: destinoID,
			Tipo:             domain.TipoMovimentoAjusteInventario,
			Quantidade:       quantidade,
			UtilizadorID:     utilizadorID,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	return []domain.MovimentoStock{movimento}, nil
}

func (s *StockService) ajustarFaltaPorLotes(ctx context.Context, produto domain.Produto, inv domain.Inventario, falta float64, utilizadorID int64) ([]domain.MovimentoStock, error) {
	referencia := fmt.Sprintf("inventario:%d", inv.ID)
	var movimentos []domain.MovimentoStock
	err := s.lotes.WithTx(ctx, func(tx *repository.LoteRepository) error {
		var err error
		_, movimentos, err = consumirLoteFEFO(ctx, tx, produto.ID, produto.Nome, inv.ArmazemID, falta, utilizadorID,
			domain.TipoMovimentoAjusteInventario, &referencia, true)
		return err
	})
	return movimentos, err
}

// --- validações partilhadas ---------------------------------------------

func (s *StockService) validarProdutoParaStock(ctx context.Context, produtoID int64) (domain.Produto, error) {
	p, err := s.produto.Obter(ctx, produtoID)
	if err != nil {
		return domain.Produto{}, err
	}
	if !p.Ativo {
		return domain.Produto{}, domain.ErrProdutoInativo
	}
	if !p.ControlaStock {
		return domain.Produto{}, domain.ErrProdutoComposto
	}
	return p, nil
}

// validarProdutoSemLotes protege as saídas/transferências genéricas: um
// produto LoteControlado que saísse por aqui baixaria stock_atual sem
// baixar lotes_produto, e o FEFO da venda passaria a ver unidades que já
// não existem. Esses produtos passam pelo FarmaciaService, com o lote
// escolhido explicitamente.
func (s *StockService) validarProdutoSemLotes(ctx context.Context, produtoID int64) (domain.Produto, error) {
	p, err := s.validarProdutoParaStock(ctx, produtoID)
	if err != nil {
		return domain.Produto{}, err
	}
	if p.LoteControlado {
		return domain.Produto{}, fmt.Errorf("%q: %w", p.Nome, domain.ErrLoteObrigatorio)
	}
	return p, nil
}

func (s *StockService) validarArmazemAtivo(ctx context.Context, armazemID int64) (domain.Armazem, error) {
	a, err := s.armazem.Obter(ctx, armazemID)
	if err != nil {
		return domain.Armazem{}, err
	}
	if !a.Ativo {
		return domain.Armazem{}, domain.ErrArmazemInativo
	}
	return a, nil
}

// custoMedioPonderado combina o que já existia com uma nova entrada de
// quantidade/custo (SDD 4.4). Saídas (perda, venda, transferência) nunca
// chamam isto — remover unidades ao custo médio atual não muda a média do
// que resta, só entradas (e transferências, na perspetiva do destino) mudam.
func custoMedioPonderado(qtdAtual, custoAtual, qtdNova, custoNovo float64) float64 {
	totalQtd := qtdAtual + qtdNova
	if totalQtd <= 0 {
		return custoNovo
	}
	return (qtdAtual*custoAtual + qtdNova*custoNovo) / totalQtd
}
