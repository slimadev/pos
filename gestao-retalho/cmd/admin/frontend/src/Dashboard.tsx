import { useEffect, useState } from "react";
import { ObterDashboard } from "../wailsjs/go/admin/AdminApp";
import { admin } from "../wailsjs/go/models";

type CoresTipo = { barra: string; bg: string; texto: string; nome: string };

const CORES: Record<string, CoresTipo> = {
  bottlestore: { barra: "var(--bottlestore)", bg: "var(--bottlestore-bg)", texto: "var(--bottlestore-text)", nome: "Bottlestore" },
  ferragem: { barra: "var(--ferragem)", bg: "var(--ferragem-bg)", texto: "var(--ferragem-text)", nome: "Ferragem" },
  mercearia: { barra: "var(--mercearia)", bg: "var(--mercearia-bg)", texto: "var(--mercearia-text)", nome: "Mercearia" },
  farmacia: { barra: "var(--farmacia)", bg: "var(--farmacia-bg)", texto: "var(--farmacia-text)", nome: "Farmácia" },
};

function coresDoTipo(tipo: string): CoresTipo {
  return CORES[tipo] ?? { barra: "var(--personalizado)", bg: "var(--personalizado-bg)", texto: "var(--personalizado-text)", nome: tipo || "—" };
}

const mt = (n: number) => `${Math.round(n).toLocaleString("pt-PT")} MT`;

function formatarDia(dia: string): string {
  const d = new Date(dia + "T00:00:00Z");
  return d.toLocaleDateString("pt-PT", { day: "2-digit", month: "2-digit" });
}

/**
 * Dashboard multi-loja (SDD 7.3/8.5, tela `1c`, Fase 9) — KPIs do grupo,
 * gráfico de barras empilhadas "vendas por loja, últimos 14 dias"
 * (agrupado por tipo de negócio para bater com a legenda do mock), cartões
 * por loja com tag de tipo colorida, painel de alertas e "mais vendidos
 * hoje". "Comparativo entre negócios" (SDD 6) é este próprio ecrã — não há
 * um relatório à parte para isso.
 */
export default function Dashboard({ onErro }: { onErro: (e: string) => void }) {
  const [d, setD] = useState<admin.Dashboard | null>(null);
  const [aCarregar, setACarregar] = useState(true);

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

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(carregar, []);

  if (!d) {
    return (
      <div className="admin-content">
        <p style={{ color: "var(--text-secondary)" }}>{aCarregar ? "A carregar dashboard…" : "Sem dados."}</p>
      </div>
    );
  }

  const maxDiaTotal = Math.max(
    1,
    ...d.SerieVendasPorLoja.map((p) => Object.values(p.PorTipo).reduce((acc, v) => acc + v, 0)),
  );
  const maxVendido = Math.max(1, ...d.MaisVendidosHoje.map((i) => i.Quantidade));

  return (
    <>
      <div className="admin-topbar">
        <div style={{ display: "flex", alignItems: "baseline" }}>
          <div className="admin-topbar-titulo">Dashboard</div>
          <div className="admin-topbar-sub">
            {new Date().toLocaleDateString("pt-PT", { weekday: "long", day: "2-digit", month: "long" })} ·{" "}
            {d.Lojas.filter((l) => l.Online).length} loja(s) ligada(s)
          </div>
        </div>
        <button className="admin-btn-secundario" onClick={recarregar} disabled={aCarregar}>
          {aCarregar ? "A atualizar…" : "Atualizar"}
        </button>
      </div>

      <div className="admin-content">
        <div className="admin-col">
          <div className="admin-kpis">
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">Vendas hoje</div>
              <div className="admin-kpi-valor">{mt(d.VendasHoje)}</div>
              {d.VendasHojeVariacaoPct != null && (
                <div className={`admin-kpi-nota ${d.VendasHojeVariacaoPct >= 0 ? "ok" : "danger"}`}>
                  {d.VendasHojeVariacaoPct >= 0 ? "+" : ""}
                  {d.VendasHojeVariacaoPct.toFixed(1)}% vs ontem
                </div>
              )}
            </div>
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">Lucro estimado</div>
              <div className="admin-kpi-valor">{mt(d.LucroEstimadoHoje)}</div>
              <div className="admin-kpi-nota">margem {d.MargemHojePct.toFixed(1)}%</div>
            </div>
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">Contas abertas agora</div>
              <div className="admin-kpi-valor">{d.ContasAbertasAgora}</div>
              <div className="admin-kpi-nota warn">{mt(d.ContasAbertasValor)} em consumo</div>
            </div>
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">Dívidas em aberto</div>
              <div className="admin-kpi-valor">{mt(d.DividasEmAbertoValor)}</div>
              <div className="admin-kpi-nota danger">
                {d.DividasClientes} cliente(s) · {d.DividasAntigasCount} &gt; 30 dias
              </div>
            </div>
          </div>

          <div className="admin-card admin-chart-card">
            <div className="admin-chart-header">
              <div className="admin-chart-titulo">Vendas por loja, últimos 14 dias</div>
              <div className="admin-legenda">
                {d.TiposNegocio.map((tipo) => (
                  <div key={tipo} className="admin-legenda-item">
                    <div className="admin-legenda-dot" style={{ background: coresDoTipo(tipo).barra }} />
                    <span>{coresDoTipo(tipo).nome}</span>
                  </div>
                ))}
              </div>
            </div>
            {d.SerieVendasPorLoja.length === 0 ? (
              <div className="admin-chart-vazio">Sem vendas nos últimos 14 dias.</div>
            ) : (
              <div className="admin-chart">
                {d.SerieVendasPorLoja.map((ponto) => (
                  <div key={ponto.Dia} className="admin-chart-col" title={formatarDia(ponto.Dia)}>
                    {Object.entries(ponto.PorTipo).map(([tipo, valor]) => (
                      <div
                        key={tipo}
                        className="admin-chart-seg"
                        style={{ height: `${(valor / maxDiaTotal) * 150}px`, background: coresDoTipo(tipo).barra }}
                      />
                    ))}
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="admin-lojas-grid">
            {d.Lojas.map((loja) => {
              const cores = coresDoTipo(loja.TipoNegocio);
              return (
                <div key={loja.LojaID} className="admin-loja-card" style={{ borderTopColor: cores.barra }}>
                  <div className="admin-loja-card-header">
                    <div className="admin-loja-nome">{loja.Nome}</div>
                    {loja.Online && (
                      <div className="admin-tag" style={{ background: cores.bg, color: cores.texto }}>
                        {loja.TipoNegocio || "—"}
                      </div>
                    )}
                  </div>
                  {!loja.Online ? (
                    <div className="admin-loja-offline">Offline — {loja.Erro}</div>
                  ) : (
                    <>
                      <div className="admin-loja-total">
                        <div className="admin-loja-total-valor">{mt(loja.TotalVendido14Dias)}</div>
                        <div className="admin-loja-total-pct">{loja.PercentualDoGrupo.toFixed(0)}% do total do grupo</div>
                      </div>
                      <div className="admin-loja-metrics">
                        {loja.TemContasAbertas ? (
                          <div className="admin-loja-metric">
                            <span className="admin-loja-metric-label">Contas abertas</span>
                            <span className="admin-loja-metric-valor">{loja.ContasAbertas}</span>
                          </div>
                        ) : (
                          <div className="admin-loja-metric">
                            <span className="admin-loja-metric-label">Vendas hoje</span>
                            <span className="admin-loja-metric-valor">{loja.VendasHoje}</span>
                          </div>
                        )}
                        <div className="admin-loja-metric">
                          <span className="admin-loja-metric-label">Stock mínimo</span>
                          <span
                            className="admin-loja-metric-valor"
                            style={loja.ProdutosAbaixoMinimo > 0 ? { color: "var(--danger)" } : undefined}
                          >
                            {loja.ProdutosAbaixoMinimo} produto(s)
                          </span>
                        </div>
                      </div>
                    </>
                  )}
                </div>
              );
            })}
            {d.Lojas.length === 0 && (
              <p style={{ color: "var(--text-secondary)" }}>Nenhuma loja ativa registada — adicione uma em "Lojas".</p>
            )}
          </div>
        </div>

        <div className="admin-side">
          <div className="admin-card admin-alertas">
            <div className="admin-alertas-header">
              <div className="admin-alertas-titulo">Alertas</div>
              {d.Alertas.length > 0 && <div className="admin-alertas-contagem">{d.Alertas.length}</div>}
            </div>
            <div className="admin-alertas-lista">
              {d.Alertas.slice(0, 8).map((a, i) => (
                <div key={i} className="admin-alerta">
                  <div
                    className="admin-alerta-barra"
                    style={{ background: a.Severidade === "alta" ? "var(--danger)" : "var(--warn)" }}
                  />
                  <div>
                    <div className="admin-alerta-titulo">{a.Titulo}</div>
                    <div className="admin-alerta-detalhe">
                      {a.LojaNome} · {a.Detalhe}
                    </div>
                  </div>
                </div>
              ))}
              {d.Alertas.length === 0 && <div className="admin-alertas-vazio">Sem alertas — tudo em ordem.</div>}
            </div>
          </div>

          <div className="admin-card admin-vendidos">
            <div className="admin-vendidos-titulo">Mais vendidos hoje</div>
            <div className="admin-vendidos-lista">
              {d.MaisVendidosHoje.map((item, i) => (
                <div key={i} className="admin-vendido-linha">
                  <div className="admin-vendido-topo">
                    <span className="admin-vendido-nome">{item.ProdutoNome}</span>
                    <span className="admin-vendido-qtd">{item.Quantidade}</span>
                  </div>
                  <div className="admin-barra-fundo">
                    <div
                      className="admin-barra-valor"
                      style={{ width: `${(item.Quantidade / maxVendido) * 100}%`, background: coresDoTipo(item.TipoNegocio).barra }}
                    />
                  </div>
                </div>
              ))}
              {d.MaisVendidosHoje.length === 0 && <div className="admin-alertas-vazio">Sem vendas hoje.</div>}
            </div>
          </div>
        </div>
      </div>
    </>
  );
}
