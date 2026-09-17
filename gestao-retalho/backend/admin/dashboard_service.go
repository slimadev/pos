package admin

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/repository"
)

// DashboardService monta o dashboard da única loja que este cmd/admin
// serve (decisão pós-Fase 10, a pedido explícito) — abre e fecha uma
// ligação por chamada, nunca mantém uma ligação persistente (ver
// AbrirLoja).
type DashboardService struct {
	lojas *repository.LojaRepository
}

func NewDashboardService(lojas *repository.LojaRepository) *DashboardService {
	return &DashboardService{lojas: lojas}
}

// PontoSerieDia é um dia do gráfico "vendas, últimos 14 dias".
type PontoSerieDia struct {
	Dia   string
	Total float64
}

type Alerta struct {
	Severidade string // "alta" | "media"
	Titulo     string
	Detalhe    string
}

type ItemMaisVendido struct {
	ProdutoNome string
	Quantidade  float64
}

type Dashboard struct {
	TemLoja               bool
	NegocioNome           string
	TipoNegocio           string
	VendasHoje            float64
	VendasHojeVariacaoPct *float64
	LucroEstimadoHoje     float64
	MargemHojePct         float64
	ContasAbertasAgora    int
	ContasAbertasValor    float64
	DividasEmAbertoValor  float64
	DividasClientes       int
	DividasAntigasCount   int
	ProdutosAbaixoMinimo  int
	SerieVendas14Dias     []PontoSerieDia
	Alertas               []Alerta
	MaisVendidosHoje      []ItemMaisVendido
}

// diasJanelaConsumo reaproveita a mesma janela dos 30 dias de "dívida
// antiga" (SDD 4.11) para decidir se um desvio de consumo teórico vs real
// (SDD 6) já é dados suficientes para valer um alerta.
const diasJanelaConsumo = 30

// limiarDesvioConsumoPct só alerta desvios de consumo genuinamente fora do
// normal — ruído de arredondamento em poucas vendas não deve gerar alerta.
const limiarDesvioConsumoPct = 10.0

// horasContaAbertaAlerta é o limiar "conta aberta há muito tempo" do
// painel de alertas (SDD, tela `1c`) — sem número fixo no SDD, 4h cobre um
// turno de bar sem soar para contas normais de uma noite.
const horasContaAbertaAlerta = 4 * time.Hour

func (s *DashboardService) ObterDashboard(ctx context.Context) (Dashboard, error) {
	// Campos que só se enchem por append começam como slice vazio, nunca
	// nil — um "sem alertas, tudo em ordem" é um estado normal, e um nil
	// marshalla para `null` em JSON em vez de `[]`; o frontend faz
	// `d.Alertas.length` sem guarda nenhuma, o que rebenta o ecrã inteiro
	// (sem error boundary) assim que não há nenhum alerta.
	d := Dashboard{
		SerieVendas14Dias: []PontoSerieDia{},
		Alertas:           []Alerta{},
		MaisVendidosHoje:  []ItemMaisVendido{},
	}

	loja, existe, err := s.lojas.ObterUnica(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	if !existe {
		return d, nil
	}

	app, fechar, err := AbrirLoja(loja.CaminhoDB)
	if err != nil {
		return Dashboard{}, fmt.Errorf("ligar à loja: %w", err)
	}
	defer fechar()
	d.TemLoja = true

	if estado, err := app.ObterEstadoConfig(); err == nil {
		d.NegocioNome = estado.Negocio.Nome
		d.TipoNegocio = string(estado.Negocio.TipoNegocio)
	}

	agora := time.Now()
	inicioHoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, agora.Location())
	fimHoje := inicioHoje.Add(24 * time.Hour)
	inicioOntem := inicioHoje.Add(-24 * time.Hour)
	inicio14Dias := inicioHoje.AddDate(0, 0, -13)

	var totalOntem float64

	if r, err := app.RelatorioVendasPorPeriodo(api.FiltroRelatorioRequest{Desde: inicioHoje, Ate: fimHoje}); err == nil {
		d.VendasHoje = r.Resumo.TotalVendido
	}
	if r, err := app.RelatorioVendasPorPeriodo(api.FiltroRelatorioRequest{Desde: inicioOntem, Ate: inicioHoje}); err == nil {
		totalOntem = r.Resumo.TotalVendido
	}
	if r, err := app.RelatorioVendasPorPeriodo(api.FiltroRelatorioRequest{Desde: inicio14Dias, Ate: fimHoje}); err == nil {
		porDia := make(map[string]float64, len(r.PorDia))
		for _, linha := range r.PorDia {
			porDia[linha.Dia] = linha.TotalVendido
		}
		dias := make([]string, 0, len(porDia))
		for dia := range porDia {
			dias = append(dias, dia)
		}
		sort.Strings(dias)
		for _, dia := range dias {
			d.SerieVendas14Dias = append(d.SerieVendas14Dias, PontoSerieDia{Dia: dia, Total: porDia[dia]})
		}
	}

	if lucro, err := app.RelatorioLucroPorPeriodo(api.FiltroRelatorioRequest{Desde: inicioHoje, Ate: fimHoje}); err == nil {
		d.LucroEstimadoHoje = lucro.Lucro
	}

	if contas, err := app.ListarContasAbertas(); err == nil {
		d.ContasAbertasAgora = len(contas)
		for _, c := range contas {
			if itens, err := app.ListarItensConta(c.ID); err == nil {
				var total float64
				for _, i := range itens {
					total += i.Quantidade*i.PrecoUnitarioAplicado - i.Desconto
				}
				d.ContasAbertasValor += total
			}
			if time.Since(c.AbertaEm) > horasContaAbertaAlerta {
				d.Alertas = append(d.Alertas, Alerta{
					Severidade: "media",
					Titulo:     "Conta aberta há " + formatarDuracao(time.Since(c.AbertaEm)),
					Detalhe:    c.Identificador,
				})
			}
		}
	}

	if dividas, err := app.RelatorioDividasEmAberto(); err == nil {
		clientesComDivida := map[int64]bool{}
		for _, dv := range dividas {
			d.DividasEmAbertoValor += dv.ValorEmAberto
			clientesComDivida[dv.ClienteID] = true
			if dv.IdadeDias > 30 {
				d.DividasAntigasCount++
				d.Alertas = append(d.Alertas, Alerta{
					Severidade: "alta",
					Titulo:     fmt.Sprintf("Dívida de %d dias", dv.IdadeDias),
					Detalhe:    fmt.Sprintf("%s · %.2f MT", dv.ClienteNome, dv.ValorEmAberto),
				})
			}
		}
		d.DividasClientes = len(clientesComDivida)
	}

	if stock, err := app.RelatorioStockAtual(nil); err == nil {
		for _, l := range stock.Linhas {
			if l.AbaixoDoMinimo {
				d.ProdutosAbaixoMinimo++
			}
		}
		if d.ProdutosAbaixoMinimo > 0 {
			d.Alertas = append(d.Alertas, Alerta{
				Severidade: "alta",
				Titulo:     fmt.Sprintf("%d produto(s) abaixo do mínimo", d.ProdutosAbaixoMinimo),
				Detalhe:    "Ver Stock",
			})
		}
	}

	if consumo, err := app.RelatorioConsumoTeoricoVsReal(api.FiltroRelatorioRequest{
		Desde: inicioHoje.AddDate(0, 0, -diasJanelaConsumo), Ate: fimHoje,
	}); err == nil {
		for _, l := range consumo {
			if l.DesvioPercentual > limiarDesvioConsumoPct {
				d.Alertas = append(d.Alertas, Alerta{
					Severidade: "media",
					Titulo:     "Consumo real acima da receita",
					Detalhe:    fmt.Sprintf("%s · +%.0f%% vs teórico", l.ProdutoNome, l.DesvioPercentual),
				})
			}
		}
	}

	if porProduto, err := app.RelatorioVendasPorProduto(api.FiltroRelatorioRequest{Desde: inicioHoje, Ate: fimHoje}); err == nil {
		for _, l := range porProduto {
			d.MaisVendidosHoje = append(d.MaisVendidosHoje, ItemMaisVendido{
				ProdutoNome: l.ProdutoNome,
				Quantidade:  l.QuantidadeTotal,
			})
		}
	}

	if totalOntem > 0 {
		variacao := (d.VendasHoje - totalOntem) / totalOntem * 100
		d.VendasHojeVariacaoPct = &variacao
	}
	if d.VendasHoje > 0 {
		d.MargemHojePct = d.LucroEstimadoHoje / d.VendasHoje * 100
	}

	sort.SliceStable(d.MaisVendidosHoje, func(i, j int) bool {
		return d.MaisVendidosHoje[i].Quantidade > d.MaisVendidosHoje[j].Quantidade
	})
	if len(d.MaisVendidosHoje) > 8 {
		d.MaisVendidosHoje = d.MaisVendidosHoje[:8]
	}

	sort.SliceStable(d.Alertas, func(i, j int) bool {
		return d.Alertas[i].Severidade == "alta" && d.Alertas[j].Severidade != "alta"
	})

	return d, nil
}

func formatarDuracao(d time.Duration) string {
	horas := int(d.Hours())
	minutos := int(d.Minutes()) % 60
	if horas > 0 {
		return fmt.Sprintf("%dh%02dm", horas, minutos)
	}
	return fmt.Sprintf("%dm", minutos)
}
