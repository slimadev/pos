package usecase

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// VendaService implementa o fluxo de venda direta (SDD 4.10, ferragem/
// mercearia/farmácia): internamente abre e fecha uma `conta` na mesma
// transação (SDD 3.5). O mecanismo de resolver linhas e gerar itens_venda
// + movimentos de stock (linhaResolvida / criarItensVendaEMovimentos, mais
// abaixo) é o mesmo que ContaService.FecharConta usa para a bottlestore —
// só a origem das linhas muda (parâmetros frescos vs. itens_conta já
// persistidos ao longo da noite).
type VendaService struct {
	vendas   *repository.VendaRepository
	produto  *repository.ProdutoRepository
	armazem  *repository.ArmazemRepository
	fichas   *repository.FichaTecnicaRepository
	clientes *repository.ClienteRepository
}

func NewVendaService(
	vendas *repository.VendaRepository,
	produto *repository.ProdutoRepository,
	armazem *repository.ArmazemRepository,
	fichas *repository.FichaTecnicaRepository,
	clientes *repository.ClienteRepository,
) *VendaService {
	return &VendaService{vendas: vendas, produto: produto, armazem: armazem, fichas: fichas, clientes: clientes}
}

type ItemVendaDiretaParams struct {
	ProdutoID        int64
	UnidadeProdutoID int64
	Quantidade       float64
	Desconto         float64
}

type PagamentoVendaParams struct {
	Metodo domain.MetodoPagamento
	Valor  float64
}

type VendaDiretaParams struct {
	ArmazemID     int64
	ClienteID     *int64 // nil = anónimo (SDD 3.2)
	Itens         []ItemVendaDiretaParams
	DescontoTotal float64
	Pagamentos    []PagamentoVendaParams
	UtilizadorID  int64
}

type VendaCompleta struct {
	Venda      domain.Venda
	Itens      []domain.ItemVenda
	Pagamentos []domain.Pagamento
}

// toleranciaPagamento absorve erros de arredondamento de ponto flutuante
// ao comparar soma de pagamentos com o total — não é uma margem de
// crédito, é só folga para 0.1 + 0.2 != 0.3 não bloquear uma venda certa.
const toleranciaPagamento = 0.01

func (s *VendaService) RegistarVendaDireta(ctx context.Context, p VendaDiretaParams) (VendaCompleta, error) {
	if len(p.Pagamentos) == 0 {
		return VendaCompleta{}, fmt.Errorf("venda sem forma de pagamento")
	}
	return s.registarVenda(ctx, p, false)
}

// RegistarVendaACredito é a venda direta em que o cliente leva os produtos
// e paga depois, no todo ou em parte: o que falta fica como dívida do
// cliente (mesmas regras do fecho de uma conta em dívida). Só o cmd/admin
// a expõe — o PDV continua a exigir o pagamento total.
func (s *VendaService) RegistarVendaACredito(ctx context.Context, p VendaDiretaParams) (VendaCompleta, error) {
	if p.ClienteID == nil || *p.ClienteID == domain.ClienteAnonimoID {
		return VendaCompleta{}, domain.ErrClienteAnonimoNaoPodeDivida
	}
	cliente, err := s.clientes.Obter(ctx, *p.ClienteID)
	if err != nil {
		return VendaCompleta{}, fmt.Errorf("obter cliente: %w", err)
	}
	if !cliente.Ativo {
		return VendaCompleta{}, fmt.Errorf("%q está inativo — ative o cliente antes de lhe vender a crédito", cliente.Nome)
	}
	return s.registarVenda(ctx, p, true)
}

// registarVenda é o fluxo comum da venda direta: `credito` permite que os
// pagamentos fiquem abaixo do total, deixando a diferença em dívida.
func (s *VendaService) registarVenda(ctx context.Context, p VendaDiretaParams, credito bool) (VendaCompleta, error) {
	if len(p.Itens) == 0 {
		return VendaCompleta{}, fmt.Errorf("venda sem itens")
	}

	armazem, err := s.armazem.Obter(ctx, p.ArmazemID)
	if err != nil {
		return VendaCompleta{}, err
	}
	if !armazem.Ativo {
		return VendaCompleta{}, domain.ErrArmazemInativo
	}

	clienteID := domain.ClienteAnonimoID
	if p.ClienteID != nil {
		clienteID = *p.ClienteID
	}

	linhas, subtotal, descontoLinhas, err := s.resolverLinhas(ctx, p.Itens)
	if err != nil {
		return VendaCompleta{}, err
	}

	descontoTotal := descontoLinhas + p.DescontoTotal
	total := subtotal - descontoTotal
	if total < 0 {
		return VendaCompleta{}, fmt.Errorf("desconto maior que o total da venda")
	}

	var somaPagamentos float64
	for _, pag := range p.Pagamentos {
		if pag.Valor <= 0 {
			return VendaCompleta{}, fmt.Errorf("valor de pagamento deve ser positivo")
		}
		somaPagamentos += pag.Valor
	}
	if somaPagamentos > total+toleranciaPagamento {
		return VendaCompleta{}, fmt.Errorf("pagamentos (%s) excedem o total da venda (%s)", qtd(somaPagamentos), qtd(total))
	}
	faltaPagar := total - somaPagamentos
	temDivida := faltaPagar > toleranciaPagamento
	if temDivida && !credito {
		return VendaCompleta{}, fmt.Errorf("pagamentos (%s) não cobrem o total da venda (%s)", qtd(somaPagamentos), qtd(total))
	}
	status := domain.StatusVendaPaga
	if temDivida {
		status = domain.StatusVendaDivida
		if somaPagamentos > toleranciaPagamento {
			status = domain.StatusVendaParcial
		}
	}
	identificadorConta := "Venda direta"
	if credito {
		identificadorConta = "Venda a crédito"
	}

	// Pagamentos nunca nil: uma venda a crédito sem entrada não tem nenhum,
	// e nil chega ao frontend como null.
	resultado := VendaCompleta{Pagamentos: []domain.Pagamento{}}

	err = s.vendas.WithTx(ctx, func(tx *repository.VendaRepository) error {
		conta, err := tx.CriarConta(ctx, domain.NegocioID, clienteID, identificadorConta, p.UtilizadorID)
		if err != nil {
			return fmt.Errorf("abrir conta: %w", err)
		}

		for _, l := range linhas {
			if _, err := tx.CriarItemConta(ctx, domain.ItemConta{
				ContaID:               conta.ID,
				ProdutoID:             l.ProdutoID,
				UnidadeProdutoID:      l.UnidadeProdutoID,
				Quantidade:            l.Quantidade,
				PrecoUnitarioAplicado: l.PrecoUnitario,
				Desconto:              l.Desconto,
				UtilizadorID:          p.UtilizadorID,
			}); err != nil {
				return fmt.Errorf("lançar item na conta: %w", err)
			}
		}

		if _, err := tx.FecharConta(ctx, conta.ID, p.UtilizadorID); err != nil {
			return fmt.Errorf("fechar conta: %w", err)
		}

		contaID := conta.ID
		venda, err := tx.CriarVenda(ctx, domain.Venda{
			NegocioID:     domain.NegocioID,
			ArmazemID:     p.ArmazemID,
			ClienteID:     clienteID,
			ContaID:       &contaID,
			TipoVenda:     domain.TipoVendaDireta,
			Subtotal:      subtotal,
			DescontoTotal: descontoTotal,
			Total:         total,
			Status:        status,
			UtilizadorID:  p.UtilizadorID,
		})
		if err != nil {
			return fmt.Errorf("criar venda: %w", err)
		}
		resultado.Venda = venda

		itens, err := criarItensVendaEMovimentos(ctx, tx, s.fichas, s.produto, venda.ID, p.ArmazemID, p.UtilizadorID, linhas)
		if err != nil {
			return err
		}
		resultado.Itens = itens

		for _, pag := range p.Pagamentos {
			pagamentoGravado, err := tx.CriarPagamento(ctx, domain.Pagamento{
				VendaID: venda.ID,
				Metodo:  pag.Metodo,
				Valor:   pag.Valor,
			})
			if err != nil {
				return fmt.Errorf("gravar pagamento: %w", err)
			}
			resultado.Pagamentos = append(resultado.Pagamentos, pagamentoGravado)
		}

		if temDivida {
			dividaStatus := domain.StatusDividaAberta
			if somaPagamentos > toleranciaPagamento {
				dividaStatus = domain.StatusDividaParcial
			}
			if _, err := tx.CriarDivida(ctx, domain.Divida{
				ClienteID:     clienteID,
				VendaID:       venda.ID,
				ValorOriginal: total,
				ValorPago:     somaPagamentos,
				ValorEmAberto: faltaPagar,
				Status:        dividaStatus,
			}); err != nil {
				return fmt.Errorf("registar dívida: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		return VendaCompleta{}, err
	}
	return resultado, nil
}

// linhaResolvida é uma linha de venda já validada e pronta a virar
// itens_venda + movimento(s) de stock — venha ela de parâmetros frescos
// (venda direta) ou de itens_conta já persistidos (fecho de conta aberta).
// Composto=true significa que QtdEmUnidadeBase/QtdBase não se aplicam — o
// consumo real vem da ficha técnica, expandida em criarItensVendaEMovimentos.
// LoteControlado=true desvia o consumo para FEFO (Fase 6) em vez do custo
// médio ponderado normal — as duas flags são mutuamente exclusivas na
// prática (um composto nunca tem LoteControlado=true, ver
// ProdutoService.Criar).
type linhaResolvida struct {
	ProdutoID        int64
	ProdutoNome      string
	Composto         bool
	LoteControlado   bool
	UnidadeProdutoID int64
	QtdEmUnidadeBase float64
	PrecoUnitario    float64
	Quantidade       float64
	Desconto         float64
	QtdBase          float64
}

func (s *VendaService) resolverLinhas(ctx context.Context, itens []ItemVendaDiretaParams) ([]linhaResolvida, float64, float64, error) {
	linhas := make([]linhaResolvida, 0, len(itens))
	var subtotal, descontoLinhas float64

	for _, item := range itens {
		if item.Quantidade <= 0 {
			return nil, 0, 0, fmt.Errorf("quantidade deve ser positiva")
		}
		if item.Desconto < 0 {
			return nil, 0, 0, fmt.Errorf("desconto não pode ser negativo")
		}

		produto, err := s.produto.Obter(ctx, item.ProdutoID)
		if err != nil {
			return nil, 0, 0, err
		}
		if !produto.Ativo {
			return nil, 0, 0, domain.ErrProdutoInativo
		}

		unidade, err := s.produto.ObterUnidade(ctx, item.UnidadeProdutoID)
		if err != nil {
			return nil, 0, 0, err
		}
		if unidade.ProdutoID != produto.ID {
			return nil, 0, 0, fmt.Errorf("unidade %d não pertence ao produto %q", unidade.ID, produto.Nome)
		}

		subtotal += item.Quantidade * unidade.PrecoVenda
		descontoLinhas += item.Desconto

		composto := produto.TipoProduto == domain.TipoProdutoComposto
		linha := linhaResolvida{
			ProdutoID:        produto.ID,
			ProdutoNome:      produto.Nome,
			Composto:         composto,
			LoteControlado:   produto.LoteControlado,
			UnidadeProdutoID: unidade.ID,
			PrecoUnitario:    unidade.PrecoVenda,
			Quantidade:       item.Quantidade,
			Desconto:         item.Desconto,
		}
		if !composto {
			linha.QtdEmUnidadeBase = unidade.QtdEmUnidadeBase
			linha.QtdBase = item.Quantidade * unidade.QtdEmUnidadeBase
		}
		linhas = append(linhas, linha)
	}

	return linhas, subtotal, descontoLinhas, nil
}

// criarItensVendaEMovimentos grava um itens_venda por linha, dentro da
// transação corrente. Um produto simples/semi-elaborado gera um único
// movimento de stock; um produto composto expande a ficha técnica e gera
// um movimento por ingrediente (SDD 3.3.1) — o cliente só vê "1x
// Caipirinha de Vodka" no recibo, a expansão é inteiramente interna.
// Falha (e aborta tudo, incluindo linhas já processadas nesta chamada — o
// chamador está sempre dentro de um WithTx) se alguma linha, ou algum
// ingrediente de uma linha composta, não tiver stock suficiente.
func criarItensVendaEMovimentos(
	ctx context.Context,
	tx *repository.VendaRepository,
	fichas *repository.FichaTecnicaRepository,
	produtos *repository.ProdutoRepository,
	vendaID, armazemID, utilizadorID int64,
	linhas []linhaResolvida,
) ([]domain.ItemVenda, error) {
	itens := make([]domain.ItemVenda, 0, len(linhas))
	referenciaID := strconv.FormatInt(vendaID, 10)

	for _, l := range linhas {
		var custoPorUnidadeVendida float64

		if l.Composto {
			custo, err := consumirFichaTecnica(ctx, tx, fichas, produtos, l.ProdutoID, l.ProdutoNome, l.Quantidade, armazemID, utilizadorID, referenciaID)
			if err != nil {
				return nil, err
			}
			custoPorUnidadeVendida = custo
		} else if l.LoteControlado {
			custoTotal, _, err := consumirLoteFEFO(ctx, tx.Lotes, l.ProdutoID, l.ProdutoNome, armazemID, l.QtdBase, utilizadorID, domain.TipoMovimentoVenda, &referenciaID, false)
			if err != nil {
				return nil, err
			}
			// Custo por unidade vendida (não por unidade base) — mesma
			// convenção do ramo simples/composto, mas com o custo real dos
			// lotes efetivamente consumidos (mais preciso que o custo
			// médio ponderado quando há lotes a preços diferentes).
			custoPorUnidadeVendida = custoTotal / l.Quantidade
		} else {
			stockAtual, err := tx.Stock.ObterStock(ctx, l.ProdutoID, armazemID)
			if err != nil {
				return nil, err
			}
			if stockAtual.Quantidade < l.QtdBase {
				return nil, fmt.Errorf("%w: %s — disponível %s, pedido %s", domain.ErrStockInsuficiente, l.ProdutoNome, qtd(stockAtual.Quantidade), qtd(l.QtdBase))
			}
			// Custo por unidade vendida (não por unidade base) — ver nota
			// em domain.ItemVenda.
			custoPorUnidadeVendida = stockAtual.CustoMedioPonderado * l.QtdEmUnidadeBase

			armazemIDCopy := armazemID
			unidadeIDCopy := l.UnidadeProdutoID
			custoCopy := stockAtual.CustoMedioPonderado
			refCopy := referenciaID
			if _, err := tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
				ProdutoID:        l.ProdutoID,
				ArmazemOrigemID:  &armazemIDCopy,
				Tipo:             domain.TipoMovimentoVenda,
				Quantidade:       l.QtdBase,
				CustoUnitario:    &custoCopy,
				UnidadeProdutoID: &unidadeIDCopy,
				ReferenciaID:     &refCopy,
				UtilizadorID:     utilizadorID,
			}); err != nil {
				return nil, fmt.Errorf("baixar stock: %w", err)
			}
		}

		subtotalLinha := l.Quantidade*l.PrecoUnitario - l.Desconto
		itemGravado, err := tx.CriarItemVenda(ctx, domain.ItemVenda{
			VendaID:                vendaID,
			ProdutoID:              l.ProdutoID,
			UnidadeProdutoID:       l.UnidadeProdutoID,
			Quantidade:             l.Quantidade,
			PrecoUnitario:          l.PrecoUnitario,
			CustoUnitarioNoMomento: custoPorUnidadeVendida,
			Desconto:               l.Desconto,
			Subtotal:               subtotalLinha,
		})
		if err != nil {
			return nil, fmt.Errorf("gravar item de venda: %w", err)
		}
		itens = append(itens, itemGravado)
	}

	return itens, nil
}

// consumirFichaTecnica expande a receita de um produto composto (SDD
// 3.3.1): gera um movimento de stock tipo `venda` por ingrediente
// (quantidade já ajustada por perda_percentual e escalada pela quantidade
// vendida), e devolve o custo total dividido pela quantidade vendida —
// isto é, o custo por unidade do composto, no mesmo formato que
// CustoUnitarioNoMomento usa para produtos simples.
func consumirFichaTecnica(
	ctx context.Context,
	tx *repository.VendaRepository,
	fichas *repository.FichaTecnicaRepository,
	produtos *repository.ProdutoRepository,
	produtoCompostoID int64,
	produtoCompostoNome string,
	quantidadeVendida float64,
	armazemID, utilizadorID int64,
	referenciaID string,
) (custoPorUnidade float64, err error) {
	ingredientes, err := fichas.ListarIngredientes(ctx, produtoCompostoID)
	if err != nil {
		return 0, err
	}
	if len(ingredientes) == 0 {
		return 0, fmt.Errorf("%q não tem ficha técnica definida", produtoCompostoNome)
	}

	var custoTotal float64
	for _, ing := range ingredientes {
		consumoTotal := ing.ConsumoReal() * quantidadeVendida

		ingrediente, err := produtos.Obter(ctx, ing.ProdutoIngredienteID)
		if err != nil {
			return 0, err
		}
		refCopy := referenciaID

		// Um ingrediente com lotes sai pelos lotes (FEFO, sem vencidos), como
		// se fosse vendido sozinho — senão stock_atual e lotes_produto
		// ficavam diferentes.
		if ingrediente.LoteControlado {
			custo, _, err := consumirLoteFEFO(ctx, tx.Lotes, ing.ProdutoIngredienteID,
				fmt.Sprintf("%s (ingrediente de %s)", ingrediente.Nome, produtoCompostoNome),
				armazemID, consumoTotal, utilizadorID, domain.TipoMovimentoVenda, &refCopy, false)
			if err != nil {
				return 0, err
			}
			custoTotal += custo
			continue
		}

		stockAtual, err := tx.Stock.ObterStock(ctx, ing.ProdutoIngredienteID, armazemID)
		if err != nil {
			return 0, err
		}
		if stockAtual.Quantidade < consumoTotal {
			return 0, fmt.Errorf("%w: %s (ingrediente de %s) — disponível %s, pedido %s", domain.ErrStockInsuficiente, ingrediente.Nome, produtoCompostoNome, qtd(stockAtual.Quantidade), qtd(consumoTotal))
		}
		custoTotal += consumoTotal * stockAtual.CustoMedioPonderado

		armazemIDCopy := armazemID
		custoCopy := stockAtual.CustoMedioPonderado
		if _, err := tx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:       ing.ProdutoIngredienteID,
			ArmazemOrigemID: &armazemIDCopy,
			Tipo:            domain.TipoMovimentoVenda,
			Quantidade:      consumoTotal,
			CustoUnitario:   &custoCopy,
			ReferenciaID:    &refCopy,
			UtilizadorID:    utilizadorID,
		}); err != nil {
			return 0, fmt.Errorf("baixar stock do ingrediente: %w", err)
		}
	}

	return custoTotal / quantidadeVendida, nil
}

// consumirLoteFEFO desconta `quantidadeBase` do produto lote-controlado
// (Fase 6, SDD 9.4) consumindo os lotes por ordem de validade — o que vence
// primeiro sai primeiro. Uma venda pode atravessar vários lotes (ex: fecha
// as últimas 3 unidades de um lote a vencer e completa com o lote
// seguinte); gera um movimento de stock por lote tocado, cada um ao custo
// desse lote específico — mais preciso que um custo médio global.
//
// incluirVencidos=false (venda) salta os lotes já fora de validade: um
// medicamento vencido nunca pode sair numa venda. O ajuste de inventário
// passa true — uma falta física começa precisamente pelos vencidos.
func consumirLoteFEFO(
	ctx context.Context,
	lotesTx *repository.LoteRepository,
	produtoID int64,
	produtoNome string,
	armazemID int64,
	quantidadeBase float64,
	utilizadorID int64,
	tipo domain.TipoMovimento,
	referenciaID *string,
	incluirVencidos bool,
) (custoTotal float64, movimentos []domain.MovimentoStock, err error) {
	lotes, err := lotesTx.ListarFEFO(ctx, produtoID, armazemID)
	if err != nil {
		return 0, nil, err
	}

	agora := time.Now()
	restante := quantidadeBase
	var qtdVencida float64
	for _, lote := range lotes {
		if restante <= 0 {
			break
		}
		if !incluirVencidos && lote.VencidoEm(agora) {
			qtdVencida += lote.Quantidade
			continue
		}
		consumoDoLote := restante
		if lote.Quantidade < consumoDoLote {
			consumoDoLote = lote.Quantidade
		}
		if consumoDoLote <= 0 {
			continue
		}

		if err := lotesTx.AtualizarQuantidade(ctx, lote.ID, lote.Quantidade-consumoDoLote); err != nil {
			return 0, nil, fmt.Errorf("atualizar lote %s: %w", lote.NumeroLote, err)
		}
		custoTotal += consumoDoLote * lote.CustoUnitario

		armazemIDCopy := armazemID
		custoCopy := lote.CustoUnitario
		// O lote fica na referência ("<ref>;lote:<id>") para que anular a
		// venda devolva as unidades ao lote de onde saíram.
		refLote := fmt.Sprintf("lote:%d", lote.ID)
		if referenciaID != nil {
			refLote = *referenciaID + ";" + refLote
		}
		movimento, err := lotesTx.Stock.CriarMovimento(ctx, repository.NovoMovimento{
			ProdutoID:       produtoID,
			ArmazemOrigemID: &armazemIDCopy,
			Tipo:            tipo,
			Quantidade:      consumoDoLote,
			CustoUnitario:   &custoCopy,
			ReferenciaID:    &refLote,
			UtilizadorID:    utilizadorID,
		})
		if err != nil {
			return 0, nil, fmt.Errorf("baixar stock do lote %s: %w", lote.NumeroLote, err)
		}
		movimentos = append(movimentos, movimento)

		restante -= consumoDoLote
	}

	if restante > 0.0001 {
		if qtdVencida > 0 {
			return 0, nil, fmt.Errorf("%w: %s — falta %s (%s em lotes vencidos, que não podem ser vendidos)", domain.ErrStockInsuficiente, produtoNome, qtd(restante), qtd(qtdVencida))
		}
		return 0, nil, fmt.Errorf("%w: %s — falta %s (lotes disponíveis esgotados)", domain.ErrStockInsuficiente, produtoNome, qtd(restante))
	}

	return custoTotal, movimentos, nil
}

// ListarVendasPorCliente cobre o "histórico de compras" por cliente (SDD 4.9).
func (s *VendaService) ListarVendasPorCliente(ctx context.Context, clienteID int64) ([]domain.Venda, error) {
	return s.vendas.ListarPorCliente(ctx, clienteID)
}

// ObterVendaCompleta é usado pelo recibo e pelo histórico do dia (SDD 5.3).
func (s *VendaService) ObterVendaCompleta(ctx context.Context, vendaID int64) (VendaCompleta, error) {
	venda, err := s.vendas.ObterVenda(ctx, vendaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	itens, err := s.vendas.ListarItensVenda(ctx, vendaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	pagamentos, err := s.vendas.ListarPagamentos(ctx, vendaID)
	if err != nil {
		return VendaCompleta{}, err
	}
	return VendaCompleta{Venda: venda, Itens: itens, Pagamentos: pagamentos}, nil
}

// ListarVendasDoDia cobre o "Histórico de vendas do dia" da tela de
// ferragem (SDD 5.3) — usa o relógio local do computador da loja, que é
// o que faz sentido para um "dia de operação" de um balcão físico.
// ListarVendasEntreDatas devolve as vendas de `desde` a `ate`, ambos
// inclusive, em dias do relógio local da loja (o mesmo que ListarVendasDoDia).
func (s *VendaService) ListarVendasEntreDatas(ctx context.Context, negocioID int64, desde, ate time.Time) ([]domain.Venda, error) {
	inicio := time.Date(desde.Year(), desde.Month(), desde.Day(), 0, 0, 0, 0, time.Local)
	fim := time.Date(ate.Year(), ate.Month(), ate.Day(), 0, 0, 0, 0, time.Local).AddDate(0, 0, 1)
	if !fim.After(inicio) {
		return nil, fmt.Errorf("a data final não pode ser anterior à inicial")
	}
	return s.vendas.ListarPorPeriodo(ctx, negocioID, inicio, fim)
}

func (s *VendaService) ListarVendasDoDia(ctx context.Context, negocioID int64) ([]domain.Venda, error) {
	agora := time.Now()
	inicio := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, agora.Location())
	fim := inicio.Add(24 * time.Hour)
	return s.vendas.ListarPorPeriodo(ctx, negocioID, inicio, fim)
}
