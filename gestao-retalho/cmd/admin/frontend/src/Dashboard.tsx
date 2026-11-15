import { useEffect, useState } from "react";
import {
  BanknotesIcon,
  ArrowTrendingUpIcon,
  ClipboardDocumentListIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
  ArrowUpIcon,
  ArrowDownIcon,
  CheckCircleIcon,
  InformationCircleIcon,
  SparklesIcon,
  ChartBarIcon,
} from "@heroicons/react/24/outline";
import { ObterDashboard } from "../wailsjs/go/admin/AdminApp";
import { admin } from "../wailsjs/go/models";
import { mtInteiro, pct } from "./formato";

const mt = mtInteiro;

function formatarDia(dia: string): string {
  const d = new Date(dia + "T00:00:00Z");
  return d.toLocaleDateString("pt-PT", { day: "2-digit", month: "2-digit" });
}

function formatarDiaSemana(dia: string): string {
  const d = new Date(dia + "T00:00:00Z");
  return d.toLocaleDateString("pt-PT", { weekday: "short" }).replace(".", "");
}

/**
 * Dashboard moderna, responsiva e com suporte a glassmorphism.
 * Inclui KPIs com indicadores de tendência, gráfico SVG de área/barras interativo
 * e visualização otimizada para ecrãs mobile e desktop.
 */
export default function Dashboard({ onErro }: { onErro: (e: string) => void }) {
  const [d, setD] = useState<admin.Dashboard | null>(null);
  const [aCarregar, setACarregar] = useState(true);
  const [pontoAtivo, setPontoAtivo] = useState<admin.PontoSerieDia | null>(null);
  const [tipoGrafico, setTipoGrafico] = useState<"area" | "barras">("area");

  function carregar() {
    ObterDashboard()
      .then(setD)
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  function recarregar() {
    setACarregar(true);
    carregar();
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(carregar, []);

  if (!d) {
    return (
      <div className="admin-content">
        <div className="admin-card" style={{ textAlign: "center", padding: "48px 24px" }}>
          <ArrowPathIcon className="admin-btn-icon a-girar" style={{ width: 32, height: 32, margin: "0 auto 12px", color: "var(--accent)" }} />
          <p style={{ color: "var(--text-secondary)", margin: 0 }}>
            {aCarregar ? "A carregar indicadores da loja..." : "Sem dados disponíveis."}
          </p>
        </div>
      </div>
    );
  }

  if (!d.TemLoja) {
    return (
      <div className="admin-content">
        <div className="admin-card" style={{ maxWidth: 520, margin: "40px auto", textAlign: "center", padding: "36px 24px" }}>
          <SparklesIcon style={{ width: 40, height: 40, margin: "0 auto 16px", color: "var(--accent)" }} />
          <h3 style={{ margin: "0 0 10px", fontSize: 18 }}>Nenhuma Loja Configurada</h3>
          <p style={{ color: "var(--text-secondary)", margin: "0 0 20px" }}>
            Conclua o assistente de configuração inicial para começar a visualizar vendas, lucros e relatórios em tempo real.
          </p>
        </div>
      </div>
    );
  }

  const pontos = d.SerieVendas14Dias;
  const maxDiaTotal = Math.max(1, ...pontos.map((p) => p.Total));
  const total14Dias = pontos.reduce((acc, p) => acc + p.Total, 0);
  const mediaDiaria = total14Dias / (pontos.length || 1);
  const melhorDia = pontos.reduce((max, p) => (p.Total > max.Total ? p : max), pontos[0] || { Dia: "", Total: 0 });
  const maxVendido = Math.max(1, ...d.MaisVendidosHoje.map((i) => i.Quantidade));

  // Geração de coordenadas SVG para gráfico de área suave
  const svgWidth = 600;
  const svgHeight = 160;
  const paddingX = 20;
  const paddingY = 20;
  const usableWidth = svgWidth - paddingX * 2;
  const usableHeight = svgHeight - paddingY * 2;

  const coords = pontos.map((p, i) => {
    const x = paddingX + (i / Math.max(1, pontos.length - 1)) * usableWidth;
    const y = svgHeight - paddingY - (p.Total / maxDiaTotal) * usableHeight;
    return { x, y, ponto: p };
  });

  const pathD = coords.length > 0
    ? `M ${coords[0].x} ${coords[0].y} ` +
      coords.slice(1).map((c) => `L ${c.x} ${c.y}`).join(" ")
    : "";

  const areaD = coords.length > 0
    ? `${pathD} L ${coords[coords.length - 1].x} ${svgHeight - paddingY} L ${coords[0].x} ${svgHeight - paddingY} Z`
    : "";

  return (
    <>
      <div className="admin-topbar">
        <div style={{ display: "flex", alignItems: "baseline", gap: 12 }}>
          <div className="admin-topbar-titulo">Painel de Controlo</div>
          <div className="admin-topbar-sub">
            {new Date().toLocaleDateString("pt-PT", { weekday: "long", day: "2-digit", month: "long" })}
          </div>
        </div>
        <button className="admin-btn-secundario admin-btn-com-icone" onClick={recarregar} disabled={aCarregar}>
          <ArrowPathIcon className={`admin-btn-icon${aCarregar ? " a-girar" : ""}`} aria-hidden="true" />
          <span>{aCarregar ? "A atualizar..." : "Atualizar"}</span>
        </button>
      </div>

      <div className="admin-content">
        <div className="admin-col">
          {/* 4 KPIs em estilo Glass */}
          <div className="admin-kpis">
            <div className="admin-card admin-kpi admin-glass">
              <div className="admin-kpi-label">
                <span className="admin-kpi-icon-wrap" style={{ background: "var(--accent-bg)", color: "var(--accent)" }}>
                  <BanknotesIcon className="admin-kpi-icon" aria-hidden="true" />
                </span>
                <span>Vendas Hoje</span>
              </div>
              <div className="admin-kpi-valor">{mt(d.VendasHoje)}</div>
              {d.VendasHojeVariacaoPct != null ? (
                <div className={`admin-kpi-nota ${d.VendasHojeVariacaoPct >= 0 ? "ok" : "danger"}`}>
                  {d.VendasHojeVariacaoPct >= 0 ? (
                    <ArrowUpIcon className="admin-trend-icon" aria-hidden="true" />
                  ) : (
                    <ArrowDownIcon className="admin-trend-icon" aria-hidden="true" />
                  )}
                  <span>{pct(Math.abs(d.VendasHojeVariacaoPct))} vs ontem</span>
                </div>
              ) : (
                <div className="admin-kpi-nota" style={{ color: "var(--text-secondary)" }}>Primeiro registo</div>
              )}
            </div>

            <div className="admin-card admin-kpi admin-glass">
              <div className="admin-kpi-label">
                <span className="admin-kpi-icon-wrap" style={{ background: "var(--ok-bg)", color: "var(--ok)" }}>
                  <ArrowTrendingUpIcon className="admin-kpi-icon" aria-hidden="true" />
                </span>
                <span>Lucro Estimado</span>
              </div>
              <div className="admin-kpi-valor">{mt(d.LucroEstimadoHoje)}</div>
              <div className="admin-kpi-nota ok">
                <span>Margem {pct(d.MargemHojePct)}</span>
              </div>
            </div>

            <div className="admin-card admin-kpi admin-glass">
              <div className="admin-kpi-label">
                <span className="admin-kpi-icon-wrap" style={{ background: "var(--warn-bg)", color: "var(--warn)" }}>
                  <ClipboardDocumentListIcon className="admin-kpi-icon" aria-hidden="true" />
                </span>
                <span>Contas Abertas</span>
              </div>
              <div className="admin-kpi-valor">{d.ContasAbertasAgora}</div>
              <div className="admin-kpi-nota warn">
                <span>{mt(d.ContasAbertasValor)} em consumo</span>
              </div>
            </div>

            <div className="admin-card admin-kpi admin-glass">
              <div className="admin-kpi-label">
                <span className="admin-kpi-icon-wrap" style={{ background: "var(--danger-bg)", color: "var(--danger)" }}>
                  <ExclamationTriangleIcon className="admin-kpi-icon" aria-hidden="true" />
                </span>
                <span>Dívidas a Receber</span>
              </div>
              <div className="admin-kpi-valor">{mt(d.DividasEmAbertoValor)}</div>
              <div className="admin-kpi-nota danger">
                <span>{d.DividasClientes} cliente(s)</span>
                {d.DividasAntigasCount > 0 && <span> · {d.DividasAntigasCount} &gt; 30d</span>}
              </div>
            </div>
          </div>

          {/* Gráfico Visual de Vendas com Glassmorphism */}
          <div className="admin-card admin-chart-card admin-glass">
            <div className="admin-chart-header">
              <div>
                <div className="admin-chart-titulo">Evolução de Vendas (Últimos 14 Dias)</div>
                <div className="admin-chart-sub">
                  Total: <strong>{mt(total14Dias)}</strong> · Média diária: <strong>{mt(mediaDiaria)}</strong>
                </div>
              </div>
              <div className="admin-chart-actions">
                <button
                  className={`admin-chart-tab${tipoGrafico === "area" ? " active" : ""}`}
                  onClick={() => setTipoGrafico("area")}
                >
                  Curva
                </button>
                <button
                  className={`admin-chart-tab${tipoGrafico === "barras" ? " active" : ""}`}
                  onClick={() => setTipoGrafico("barras")}
                >
                  Barras
                </button>
              </div>
            </div>

            {pontos.every((p) => p.Total === 0) ? (
              <div className="admin-chart-vazio">
                <ChartBarIcon style={{ width: 32, height: 32, margin: "0 auto 8px", opacity: 0.5 }} />
                Sem vendas registadas nos últimos 14 dias.
              </div>
            ) : (
              <div className="admin-chart-wrapper">
                {tipoGrafico === "area" ? (
                  <div className="admin-svg-chart-container">
                    <svg viewBox={`0 0 ${svgWidth} ${svgHeight}`} className="admin-svg-chart" preserveAspectRatio="none">
                      <defs>
                        <linearGradient id="areaGradient" x1="0" y1="0" x2="0" y2="1">
                          <stop offset="0%" stopColor="var(--accent)" stopOpacity="0.45" />
                          <stop offset="100%" stopColor="var(--accent)" stopOpacity="0.02" />
                        </linearGradient>
                      </defs>

                      {/* Linhas guia horizontais */}
                      <line x1={paddingX} y1={paddingY} x2={svgWidth - paddingX} y2={paddingY} stroke="var(--border)" strokeDasharray="3 3" opacity="0.6" />
                      <line x1={paddingX} y1={svgHeight / 2} x2={svgWidth - paddingX} y2={svgHeight / 2} stroke="var(--border)" strokeDasharray="3 3" opacity="0.6" />
                      <line x1={paddingX} y1={svgHeight - paddingY} x2={svgWidth - paddingX} y2={svgHeight - paddingY} stroke="var(--border)" />

                      {/* Área preenchida */}
                      <path d={areaD} fill="url(#areaGradient)" />

                      {/* Linha principal */}
                      <path d={pathD} fill="none" stroke="var(--accent)" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" />

                      {/* Pontos de dados */}
                      {coords.map((c, i) => {
                        const ehHoje = i === coords.length - 1;
                        const ativo = pontoAtivo?.Dia === c.ponto.Dia;
                        return (
                          <g key={c.ponto.Dia} className="admin-chart-point-group">
                            <circle
                              cx={c.x}
                              cy={c.y}
                              r={ativo ? 5.5 : ehHoje ? 4.5 : 3.5}
                              fill={ehHoje ? "var(--accent)" : "var(--surface)"}
                              stroke="var(--accent)"
                              strokeWidth={ehHoje ? 2 : 1.5}
                              className="admin-chart-point"
                              onMouseEnter={() => setPontoAtivo(c.ponto)}
                              onMouseLeave={() => setPontoAtivo(null)}
                            />
                          </g>
                        );
                      })}
                    </svg>

                    {/* Datas no eixo X */}
                    <div className="admin-svg-labels">
                      {pontos.map((p, i) => {
                        const ehHoje = i === pontos.length - 1;
                        return (
                          <div key={p.Dia} className={`admin-svg-label${ehHoje ? " hoje" : ""}`}>
                            <span className="admin-svg-label-sem">{formatarDiaSemana(p.Dia)}</span>
                            <span className="admin-svg-label-dia">{ehHoje ? "Hoje" : formatarDia(p.Dia)}</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                ) : (
                  <div className="admin-chart">
                    {pontos.map((ponto, i) => {
                      const hoje = i === pontos.length - 1;
                      return (
                        <div
                          key={ponto.Dia}
                          className={`admin-chart-col${hoje ? " hoje" : ""}`}
                          onMouseEnter={() => setPontoAtivo(ponto)}
                          onMouseLeave={() => setPontoAtivo(null)}
                        >
                          <div className="admin-chart-area">
                            <div
                              className="admin-chart-seg"
                              style={{
                                height: `${ponto.Total > 0 ? Math.max(4, (ponto.Total / maxDiaTotal) * 100) : 0}%`,
                              }}
                            />
                          </div>
                          <div className="admin-chart-dia">{hoje ? "Hoje" : formatarDia(ponto.Dia)}</div>
                        </div>
                      );
                    })}
                  </div>
                )}

                {/* Banner de detalhe ao passar o cursor ou tocar num dia */}
                <div className="admin-chart-detalhe-flutuante">
                  {pontoAtivo ? (
                    <div>
                      Dia <strong>{formatarDia(pontoAtivo.Dia)}</strong>: Vendas de{" "}
                      <span className="admin-destaque-valor">{mt(pontoAtivo.Total)}</span>
                    </div>
                  ) : (
                    <div>
                      Melhor dia: <strong>{formatarDia(melhorDia.Dia)}</strong> ({mt(melhorDia.Total)})
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Barra Lateral com Alertas e Produtos Mais Vendidos */}
        <div className="admin-side">
          <div className="admin-card admin-alertas admin-glass">
            <div className="admin-alertas-header">
              <div className="admin-alertas-titulo">
                <ExclamationTriangleIcon className="admin-btn-icon" style={{ color: "var(--warn)" }} />
                <span>Alertas da Loja</span>
              </div>
              {d.Alertas.length > 0 && <span className="admin-badge-alerta">{d.Alertas.length}</span>}
            </div>
            <div className="admin-alertas-lista">
              {d.Alertas.slice(0, 8).map((a, i) => (
                <div key={i} className="admin-alerta-card">
                  <div
                    className="admin-alerta-barra"
                    style={{ background: a.Severidade === "alta" ? "var(--danger)" : "var(--warn)" }}
                  />
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div className="admin-alerta-titulo">{a.Titulo}</div>
                    <div className="admin-alerta-detalhe">{a.Detalhe}</div>
                  </div>
                </div>
              ))}
              {d.Alertas.length === 0 && (
                <div className="admin-alerta-vazio">
                  <CheckCircleIcon className="admin-icon-vazio" style={{ color: "var(--ok)" }} />
                  <span>Sem alertas operacionais — tudo em ordem.</span>
                </div>
              )}
            </div>
          </div>

          <div className="admin-card admin-vendidos admin-glass">
            <div className="admin-vendidos-header">
              <div className="admin-vendidos-titulo">
                <SparklesIcon className="admin-btn-icon" style={{ color: "var(--accent)" }} />
                <span>Mais Vendidos Hoje</span>
              </div>
            </div>
            <div className="admin-vendidos-lista">
              {d.MaisVendidosHoje.map((item, i) => (
                <div key={i} className="admin-vendido-linha">
                  <div className="admin-vendido-topo">
                    <div style={{ display: "flex", alignItems: "center", gap: 8, minWidth: 0 }}>
                      <span className={`admin-rank-badge rank-${i + 1}`}>#{i + 1}</span>
                      <span className="admin-vendido-nome" title={item.ProdutoNome}>{item.ProdutoNome}</span>
                    </div>
                    <span className="admin-vendido-qtd">{item.Quantidade} un</span>
                  </div>
                  <div className="admin-barra-fundo">
                    <div
                      className="admin-barra-valor"
                      style={{
                        width: `${(item.Quantidade / maxVendido) * 100}%`,
                        background: "var(--accent)",
                      }}
                    />
                  </div>
                </div>
              ))}
              {d.MaisVendidosHoje.length === 0 && (
                <div className="admin-alerta-vazio">
                  <InformationCircleIcon className="admin-icon-vazio" />
                  <span>Nenhuma venda concluída hoje ainda.</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
