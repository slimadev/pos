package usecase

import (
	"context"
	"time"

	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// RelatorioService cobre a tabela de relatórios do SDD secção 6 (Fase 8):
// todos com filtro por período e armazém (filtro por negócio só passa a
// fazer sentido na Fase 9, quando o Admin universal agrega várias bases de
// dados — aqui negocio_id é sempre domain.NegocioID). O relatório
// "Comparativo entre negócios" (SDD 6, última linha) fica inteiramente
// para a Fase 9 — não há o que comparar dentro de uma única loja.
type RelatorioService struct {
	relatorios *repository.RelatorioRepository
	fichas     *repository.FichaTecnicaRepository
	produtos   *repository.ProdutoRepository
	clientes   *repository.ClienteRepository
	dividas    *DividaService
}

func NewRelatorioService(
	relatorios *repository.RelatorioRepository,
	fichas *repository.FichaTecnicaRepository,
	produtos *repository.ProdutoRepository,
	clientes *repository.ClienteRepository,
	dividas *DividaService,
) *RelatorioService {
	return &RelatorioService{relatorios: relatorios, fichas: fichas, produtos: produtos, clientes: clientes, dividas: dividas}
}

// FiltroPeriodo é o par (Desde, Até) + armazém opcional partilhado por
// praticamente todos os relatórios (SDD 6: "todos devem suportar filtro
// por período... e por armazém").
type FiltroPeriodo struct {
	Desde     time.Time
	Ate       time.Time
	ArmazemID *int64
}

func (f FiltroPeriodo) periodoFiltro() repository.PeriodoFiltro {
	return repository.PeriodoFiltro{
		NegocioID: domain.NegocioID,
		Desde:     f.Desde,
		Ate:       f.Ate,
		ArmazemID: f.ArmazemID,
	}
}

func (f FiltroPeriodo) movimentosFiltro() repository.MovimentosFiltro {
	return repository.MovimentosFiltro{
		Desde:     f.Desde,
		Ate:       f.Ate,
		ArmazemID: f.ArmazemID,
	}
}

// --- 1. Vendas por período ---------------------------------------------------

type ResumoVendas struct {
	NumeroVendas int64
	TotalVendido float64
	TicketMedio  float64
}

func (s *RelatorioService) VendasPorPeriodo(ctx context.Context, f FiltroPeriodo) (ResumoVendas, []repository.LinhaVendasPorDia, error) {
	linhas, err := s.relatorios.VendasPorDia(ctx, f.periodoFiltro())
	if err != nil {
		return ResumoVendas{}, nil, err
	}
	var resumo ResumoVendas
	for _, l := range linhas {
		resumo.NumeroVendas += l.NumeroVendas
		resumo.TotalVendido += l.TotalVendido
	}
	if resumo.NumeroVendas > 0 {
		resumo.TicketMedio = resumo.TotalVendido / float64(resumo.NumeroVendas)
	}
	return resumo, linhas, nil
}

// --- 2. Lucro por período -----------------------------------------------------

type RelatorioLucro struct {
	Receita          float64
	Custo            float64
	Lucro            float64
	MargemPercentual float64
}

func (s *RelatorioService) LucroPorPeriodo(ctx context.Context, f FiltroPeriodo) (RelatorioLucro, error) {
	lp, err := s.relatorios.LucroPorPeriodo(ctx, f.periodoFiltro())
	if err != nil {
		return RelatorioLucro{}, err
	}
	lucro := lp.Receita - lp.Custo
	var margem float64
	if lp.Receita > 0 {
		margem = lucro / lp.Receita * 100
	}
	return RelatorioLucro{Receita: lp.Receita, Custo: lp.Custo, Lucro: lucro, MargemPercentual: margem}, nil
}

// --- 3. Vendas por produto ----------------------------------------------------

func (s *RelatorioService) VendasPorProduto(ctx context.Context, f FiltroPeriodo) ([]repository.LinhaVendasPorProduto, error) {
	return s.relatorios.VendasPorProduto(ctx, f.periodoFiltro())
}

// --- 4. Consumo teórico vs real de ingredientes -------------------------------

type LinhaConsumoIngrediente struct {
	ProdutoID        int64
	ProdutoNome      string
	ConsumoTeorico   float64
	ConsumoReal      float64
	Desvio           float64
	DesvioPercentual float64
}

// ConsumoTeoricoVsReal fecha o ciclo de reconciliação da SDD 3.3: para cada
// venda de um produto composto no período, expande a ficha técnica para
// saber quanto DEVIA ter saído de cada ingrediente (teórico), e cruza com o
// que realmente saiu via movimentos_stock tipo=venda desse ingrediente
// (real) — um desvio positivo revela perdas de corte/porção acima do
// previsto na receita.
func (s *RelatorioService) ConsumoTeoricoVsReal(ctx context.Context, f FiltroPeriodo) ([]LinhaConsumoIngrediente, error) {
	itens, err := s.relatorios.ItensVendaPorPeriodo(ctx, f.periodoFiltro())
	if err != nil {
		return nil, err
	}

	produtos, err := s.produtos.Listar(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	produtoPorID := make(map[int64]domain.Produto, len(produtos))
	for _, p := range produtos {
		produtoPorID[p.ID] = p
	}

	ingredientesPorComposto := make(map[int64][]domain.FichaTecnica)
	teorico := make(map[int64]float64)
	for _, item := range itens {
		produto, ok := produtoPorID[item.ProdutoID]
		if !ok || produto.TipoProduto != domain.TipoProdutoComposto {
			continue
		}
		ingredientes, ok := ingredientesPorComposto[item.ProdutoID]
		if !ok {
			ingredientes, err = s.fichas.ListarIngredientes(ctx, item.ProdutoID)
			if err != nil {
				return nil, err
			}
			ingredientesPorComposto[item.ProdutoID] = ingredientes
		}
		for _, ing := range ingredientes {
			teorico[ing.ProdutoIngredienteID] += ing.ConsumoReal() * item.Quantidade
		}
	}
	if len(teorico) == 0 {
		return nil, nil
	}

	movimentos, err := s.relatorios.MovimentosPorPeriodo(ctx, f.movimentosFiltro())
	if err != nil {
		return nil, err
	}
	real := make(map[int64]float64)
	for _, m := range movimentos {
		if m.Tipo != domain.TipoMovimentoVenda {
			continue
		}
		if _, relevante := teorico[m.ProdutoID]; !relevante {
			continue
		}
		real[m.ProdutoID] += m.QuantidadeUnidadeBase
	}

	out := make([]LinhaConsumoIngrediente, 0, len(teorico))
	for produtoID, teoricoTotal := range teorico {
		realTotal := real[produtoID]
		desvio := realTotal - teoricoTotal
		var desvioPct float64
		if teoricoTotal > 0 {
			desvioPct = desvio / teoricoTotal * 100
		}
		out = append(out, LinhaConsumoIngrediente{
			ProdutoID:        produtoID,
			ProdutoNome:      produtoPorID[produtoID].Nome,
			ConsumoTeorico:   teoricoTotal,
			ConsumoReal:      realTotal,
			Desvio:           desvio,
			DesvioPercentual: desvioPct,
		})
	}
	return out, nil
}

// --- 5. Vendas por método de pagamento ----------------------------------------

func (s *RelatorioService) VendasPorMetodoPagamento(ctx context.Context, f FiltroPeriodo) ([]repository.LinhaVendasPorMetodo, error) {
	return s.relatorios.VendasPorMetodoPagamento(ctx, f.periodoFiltro())
}

// --- 6. Stock atual ------------------------------------------------------------

type LinhaStockAtualRelatorio struct {
	repository.LinhaStockAtual
	ValorStock     float64
	AbaixoDoMinimo bool
}

func (s *RelatorioService) StockAtual(ctx context.Context, armazemID *int64) ([]LinhaStockAtualRelatorio, float64, error) {
	linhas, err := s.relatorios.StockAtual(ctx, domain.NegocioID, armazemID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]LinhaStockAtualRelatorio, 0, len(linhas))
	var valorTotal float64
	for _, l := range linhas {
		valorStock := l.QuantidadeUnidadeBase * l.CustoMedioPonderado
		valorTotal += valorStock
		out = append(out, LinhaStockAtualRelatorio{
			LinhaStockAtual: l,
			ValorStock:      valorStock,
			AbaixoDoMinimo:  l.StockMinimo != nil && l.QuantidadeUnidadeBase < *l.StockMinimo,
		})
	}
	return out, valorTotal, nil
}

// --- 7. Movimentos de stock ----------------------------------------------------

func (s *RelatorioService) MovimentosPorPeriodo(ctx context.Context, f FiltroPeriodo) ([]repository.LinhaMovimento, error) {
	return s.relatorios.MovimentosPorPeriodo(ctx, f.movimentosFiltro())
}

// --- 8. Perdas e consumo interno ------------------------------------------------

func (s *RelatorioService) PerdasConsumoInterno(ctx context.Context, f FiltroPeriodo) ([]repository.LinhaPerdaConsumo, float64, error) {
	linhas, err := s.relatorios.PerdasConsumoInterno(ctx, f.movimentosFiltro())
	if err != nil {
		return nil, 0, err
	}
	var total float64
	for _, l := range linhas {
		total += l.ValorTotal
	}
	return linhas, total, nil
}

// --- 9. Contas abertas / fechadas -----------------------------------------------

type LinhaConta struct {
	domain.Conta
	ValorTotal       float64
	TempoAbertoHoras *float64
}

type ResumoContas struct {
	NumeroContas          int64
	TempoMedioAbertoHoras float64
	ValorMedio            float64
}

func (s *RelatorioService) ContasPorPeriodo(ctx context.Context, desde, ate time.Time) (ResumoContas, []LinhaConta, error) {
	contas, err := s.relatorios.ContasPorPeriodo(ctx, domain.NegocioID, desde, ate)
	if err != nil {
		return ResumoContas{}, nil, err
	}
	valores, err := s.relatorios.ValorPorConta(ctx)
	if err != nil {
		return ResumoContas{}, nil, err
	}

	out := make([]LinhaConta, 0, len(contas))
	var somaTempoHoras float64
	var numeroComTempo int64
	var somaValor float64
	for _, c := range contas {
		var tempoHoras *float64
		if c.FechadaEm != nil {
			h := c.FechadaEm.Sub(c.AbertaEm).Hours()
			tempoHoras = &h
			somaTempoHoras += h
			numeroComTempo++
		}
		valor := valores[c.ID]
		somaValor += valor
		out = append(out, LinhaConta{Conta: c, ValorTotal: valor, TempoAbertoHoras: tempoHoras})
	}

	resumo := ResumoContas{NumeroContas: int64(len(contas))}
	if numeroComTempo > 0 {
		resumo.TempoMedioAbertoHoras = somaTempoHoras / float64(numeroComTempo)
	}
	if len(contas) > 0 {
		resumo.ValorMedio = somaValor / float64(len(contas))
	}
	return resumo, out, nil
}

// --- 10. Dívidas em aberto -------------------------------------------------------

type LinhaDividaRelatorio struct {
	domain.Divida
	ClienteNome string
	IdadeDias   int
}

func (s *RelatorioService) DividasEmAberto(ctx context.Context) ([]LinhaDividaRelatorio, error) {
	dividas, err := s.dividas.ListarAbertas(ctx, domain.NegocioID)
	if err != nil {
		return nil, err
	}
	agora := time.Now()
	out := make([]LinhaDividaRelatorio, 0, len(dividas))
	for _, d := range dividas {
		nome := ""
		if cliente, err := s.clientes.Obter(ctx, d.ClienteID); err == nil {
			nome = cliente.Nome
		}
		out = append(out, LinhaDividaRelatorio{
			Divida:      d,
			ClienteNome: nome,
			IdadeDias:   int(agora.Sub(d.CriadoEm).Hours() / 24),
		})
	}
	return out, nil
}

// --- 11. Fecho de caixa diário ----------------------------------------------------

type FechoCaixa struct {
	Dia          string
	TotalVendido float64
	PorMetodo    []repository.LinhaVendasPorMetodo
}

// FechoCaixaDiario reduz-se a "vendas por método de pagamento" com a janela
// fechada num único dia — a comparação "esperado vs contado" do SDD 6 fica
// só no ecrã (o operador introduz o valor contado na gaveta; este método
// só devolve o valor esperado, não há tabela de "caixa" no modelo de
// dados para guardar uma contagem física).
func (s *RelatorioService) FechoCaixaDiario(ctx context.Context, dia time.Time) (FechoCaixa, error) {
	// O dia é o do relógio da loja: um instante vindo do ecrã em UTC (a
	// meia-noite local é 22:00 UTC da véspera em Moçambique) cairia na véspera.
	dia = dia.Local()
	inicio := time.Date(dia.Year(), dia.Month(), dia.Day(), 0, 0, 0, 0, time.Local)
	fim := inicio.Add(24 * time.Hour)
	porMetodo, err := s.VendasPorMetodoPagamento(ctx, FiltroPeriodo{Desde: inicio, Ate: fim})
	if err != nil {
		return FechoCaixa{}, err
	}
	var total float64
	for _, m := range porMetodo {
		total += m.Total
	}
	return FechoCaixa{Dia: inicio.Format("2006-01-02"), TotalVendido: total, PorMetodo: porMetodo}, nil
}
