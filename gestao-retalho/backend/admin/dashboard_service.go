package admin

import (
	"context"
	"fmt"
	"sort"
	"time"

	"gestao-retalho/backend/api"
	"gestao-retalho/backend/domain"
	"gestao-retalho/backend/repository"
)

// DashboardService monta o dashboard multi-loja da tela `1c` (SDD 7.3/8.5)
// — abre e fecha uma ligação por loja ativa a cada chamada, nunca mantém
// N ligações persistentes (ver AbrirLoja).
type DashboardService struct {
	lojas *repository.LojaRepository
}

func NewDashboardService(lojas *repository.LojaRepository) *DashboardService {
	return &DashboardService{lojas: lojas}
}

type CartaoLoja struct {
	LojaID               int64
	Nome                 string
	TipoNegocio          string
	Online               bool
	Erro                 string
	TotalVendido14Dias   float64
	PercentualDoGrupo    float64
	TemContasAbertas     bool
	ContasAbertas        int
	VendasHoje           int
	ProdutosAbaixoMinimo int
}

// PontoSerieDia é um dia do gráfico "vendas por loja, últimos 14 dias" da
// tela `1c` — agrupado por TIPO de negócio, não por loja individual, para
// bater com a legenda do mock (dourado/laranja/verde/teal por tipo); duas
// lojas do mesmo tipo somam-se no mesmo segmento.
type PontoSerieDia struct {
	Dia     string
	PorTipo map[string]float64
}

type Alerta struct {
	Severidade string // "alta" | "media"
	Titulo     string
	Detalhe    string
	LojaNome   string
}

type ItemMaisVendido struct {
	ProdutoNome string
	Quantidade  float64
	LojaNome    string
	TipoNegocio string
}

type Dashboard struct {
	VendasHoje            float64
	VendasHojeVariacaoPct *float64
	LucroEstimadoHoje     float64
	MargemHojePct         float64
	ContasAbertasAgora    int
	ContasAbertasValor    float64
	DividasEmAbertoValor  float64
	DividasClientes       int
	DividasAntigasCount   int
	SerieVendasPorLoja    []PontoSerieDia
	TiposNegocio          []string
	Lojas                 []CartaoLoja
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
	lojasAtivas, err := s.lojas.ListarAtivas(ctx)
	if err != nil {
		return Dashboard{}, err
	}

	agora := time.Now()
	inicioHoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, agora.Location())
	fimHoje := inicioHoje.Add(24 * time.Hour)
	inicioOntem := inicioHoje.Add(-24 * time.Hour)
	inicio14Dias := inicioHoje.AddDate(0, 0, -13)

	var d Dashboard
	var totalOntem, totalVendido14DiasGrupo float64
	seriePorDia := map[string]map[string]float64{}

	for _, loja := range lojasAtivas {
		cartao := CartaoLoja{LojaID: loja.ID, Nome: loja.Nome}

		app, fechar, err := AbrirLoja(loja.CaminhoDB)
		if err != nil {
			cartao.Erro = err.Error()
			d.Lojas = append(d.Lojas, cartao)
			continue
		}

		func() {
			defer fechar()
			cartao.Online = true

			if estado, err := app.ObterEstadoConfig(); err == nil {
				cartao.Nome = estado.Negocio.Nome
				cartao.TipoNegocio = string(estado.Negocio.TipoNegocio)
				cartao.TemContasAbertas = temModulo(estado.ModulosAtivos, domain.ModuloContasAbertas)
			}

			if r, err := app.RelatorioVendasPorPeriodo(api.FiltroRelatorioRequest{Desde: inicioHoje, Ate: fimHoje}); err == nil {
				d.VendasHoje += r.Resumo.TotalVendido
				cartao.VendasHoje = int(r.Resumo.NumeroVendas)
			}
			if r, err := app.RelatorioVendasPorPeriodo(api.FiltroRelatorioRequest{Desde: inicioOntem, Ate: inicioHoje}); err == nil {
				totalOntem += r.Resumo.TotalVendido
			}
			if r, err := app.RelatorioVendasPorPeriodo(api.FiltroRelatorioRequest{Desde: inicio14Dias, Ate: fimHoje}); err == nil {
				cartao.TotalVendido14Dias = r.Resumo.TotalVendido
				totalVendido14DiasGrupo += r.Resumo.TotalVendido
				tipo := cartao.TipoNegocio
				if tipo == "" {
					tipo = "outro"
				}
				for _, linha := range r.PorDia {
					if seriePorDia[linha.Dia] == nil {
						seriePorDia[linha.Dia] = map[string]float64{}
					}
					seriePorDia[linha.Dia][tipo] += linha.TotalVendido
				}
			}

			if lucro, err := app.RelatorioLucroPorPeriodo(api.FiltroRelatorioRequest{Desde: inicioHoje, Ate: fimHoje}); err == nil {
				d.LucroEstimadoHoje += lucro.Lucro
			}

			if contas, err := app.ListarContasAbertas(); err == nil {
				cartao.ContasAbertas = len(contas)
				d.ContasAbertasAgora += len(contas)
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
							LojaNome:   cartao.Nome,
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
							LojaNome:   cartao.Nome,
						})
					}
				}
				d.DividasClientes += len(clientesComDivida)
			}

			if stock, err := app.RelatorioStockAtual(nil); err == nil {
				for _, l := range stock.Linhas {
					if l.AbaixoDoMinimo {
						cartao.ProdutosAbaixoMinimo++
					}
				}
				if cartao.ProdutosAbaixoMinimo > 0 {
					d.Alertas = append(d.Alertas, Alerta{
						Severidade: "alta",
						Titulo:     fmt.Sprintf("%d produto(s) abaixo do mínimo", cartao.ProdutosAbaixoMinimo),
						Detalhe:    cartao.Nome,
						LojaNome:   cartao.Nome,
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
							LojaNome:   cartao.Nome,
						})
					}
				}
			}

			if porProduto, err := app.RelatorioVendasPorProduto(api.FiltroRelatorioRequest{Desde: inicioHoje, Ate: fimHoje}); err == nil {
				for _, l := range porProduto {
					d.MaisVendidosHoje = append(d.MaisVendidosHoje, ItemMaisVendido{
						ProdutoNome: l.ProdutoNome,
						Quantidade:  l.QuantidadeTotal,
						LojaNome:    cartao.Nome,
						TipoNegocio: cartao.TipoNegocio,
					})
				}
			}
		}()

		d.Lojas = append(d.Lojas, cartao)
	}

	tiposVistos := map[string]bool{}
	for _, c := range d.Lojas {
		if c.Online && c.TipoNegocio != "" && !tiposVistos[c.TipoNegocio] {
			tiposVistos[c.TipoNegocio] = true
			d.TiposNegocio = append(d.TiposNegocio, c.TipoNegocio)
		}
	}
	sort.Strings(d.TiposNegocio)

	if totalOntem > 0 {
		variacao := (d.VendasHoje - totalOntem) / totalOntem * 100
		d.VendasHojeVariacaoPct = &variacao
	}
	if d.VendasHoje > 0 {
		d.MargemHojePct = d.LucroEstimadoHoje / d.VendasHoje * 100
	}
	if totalVendido14DiasGrupo > 0 {
		for i := range d.Lojas {
			d.Lojas[i].PercentualDoGrupo = d.Lojas[i].TotalVendido14Dias / totalVendido14DiasGrupo * 100
		}
	}

	dias := make([]string, 0, len(seriePorDia))
	for dia := range seriePorDia {
		dias = append(dias, dia)
	}
	sort.Strings(dias)
	for _, dia := range dias {
		d.SerieVendasPorLoja = append(d.SerieVendasPorLoja, PontoSerieDia{Dia: dia, PorTipo: seriePorDia[dia]})
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

func temModulo(modulos []domain.Modulo, alvo domain.Modulo) bool {
	for _, m := range modulos {
		if m == alvo {
			return true
		}
	}
	return false
}

func formatarDuracao(d time.Duration) string {
	horas := int(d.Hours())
	minutos := int(d.Minutes()) % 60
	if horas > 0 {
		return fmt.Sprintf("%dh%02dm", horas, minutos)
	}
	return fmt.Sprintf("%dm", minutos)
}
