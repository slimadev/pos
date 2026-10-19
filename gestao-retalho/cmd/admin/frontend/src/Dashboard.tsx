import { useEffect, useState } from "react";
import {
  BanknotesIcon,
  ArrowTrendingUpIcon,
  ClipboardDocumentListIcon,
  ExclamationTriangleIcon,
  ArrowPathIcon,
} from "@heroicons/react/24/outline";
import { ObterDashboard } from "../wailsjs/go/admin/AdminApp";
import { admin } from "../wailsjs/go/models";

const mt = (n: number) => `${Math.round(n).toLocaleString("pt-PT")} MT`;

function formatarDia(dia: string): string {
  const d = new Date(dia + "T00:00:00Z");
  return d.toLocaleDateString("pt-PT", { day: "2-digit", month: "2-digit" });
}

/**
 * Dashboard da única loja que este cmd/admin serve (decisão pós-Fase 10, a
 * pedido explícito) — KPIs do dia, gráfico "vendas, últimos 14 dias",
 * painel de alertas e "mais vendidos hoje". Sem cartões por loja nem
 * comparação entre negócios — só existe uma loja.
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

  if (!d.TemLoja) {
    return (
      <div className="admin-content">
        <p style={{ color: "var(--text-secondary)", maxWidth: 480 }}>
          Nenhuma loja configurada ainda — conclua o assistente de configuração para começar a ver dados aqui.
        </p>
      </div>
    );
  }

  const maxDiaTotal = Math.max(1, ...d.SerieVendas14Dias.map((p) => p.Total));
  const maxVendido = Math.max(1, ...d.MaisVendidosHoje.map((i) => i.Quantidade));

  return (
    <>
      <div className="admin-topbar">
        <div style={{ display: "flex", alignItems: "baseline" }}>
          <div className="admin-topbar-titulo">Dashboard</div>
          <div className="admin-topbar-sub">
            {new Date().toLocaleDateString("pt-PT", { weekday: "long", day: "2-digit", month: "long" })}
          </div>
        </div>
        <button className="admin-btn-secundario admin-btn-com-icone" onClick={recarregar} disabled={aCarregar}>
          <ArrowPathIcon className={`admin-btn-icon${aCarregar ? " a-girar" : ""}`} aria-hidden="true" />
          {aCarregar ? "A atualizar…" : "Atualizar"}
        </button>
      </div>

      <div className="admin-content">
        <div className="admin-col">
          <div className="admin-kpis">
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">
                <BanknotesIcon className="admin-kpi-icon" aria-hidden="true" />
                Vendas hoje
              </div>
              <div className="admin-kpi-valor">{mt(d.VendasHoje)}</div>
              {d.VendasHojeVariacaoPct != null && (
                <div className={`admin-kpi-nota ${d.VendasHojeVariacaoPct >= 0 ? "ok" : "danger"}`}>
                  {d.VendasHojeVariacaoPct >= 0 ? "+" : ""}
                  {d.VendasHojeVariacaoPct.toFixed(1)}% vs ontem
                </div>
              )}
            </div>
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">
                <ArrowTrendingUpIcon className="admin-kpi-icon" aria-hidden="true" />
                Lucro estimado
              </div>
              <div className="admin-kpi-valor">{mt(d.LucroEstimadoHoje)}</div>
              <div className="admin-kpi-nota">margem {d.MargemHojePct.toFixed(1)}%</div>
            </div>
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">
                <ClipboardDocumentListIcon className="admin-kpi-icon" aria-hidden="true" />
                Contas abertas agora
              </div>
              <div className="admin-kpi-valor">{d.ContasAbertasAgora}</div>
              <div className="admin-kpi-nota warn">{mt(d.ContasAbertasValor)} em consumo</div>
            </div>
            <div className="admin-card admin-kpi">
              <div className="admin-kpi-label">
                <ExclamationTriangleIcon className="admin-kpi-icon" aria-hidden="true" />
                Dívidas em aberto
              </div>
              <div className="admin-kpi-valor">{mt(d.DividasEmAbertoValor)}</div>
              <div className="admin-kpi-nota danger">
                {d.DividasClientes} cliente(s) · {d.DividasAntigasCount} &gt; 30 dias
              </div>
            </div>
          </div>

          <div className="admin-card admin-chart-card">
            <div className="admin-chart-header">
              <div className="admin-chart-titulo">Vendas, últimos 14 dias</div>
            </div>
            {d.SerieVendas14Dias.every((p) => p.Total === 0) ? (
              <div className="admin-chart-vazio">Sem vendas nos últimos 14 dias.</div>
            ) : (
              <div className="admin-chart">
                {d.SerieVendas14Dias.map((ponto, i) => {
                  const hoje = i === d.SerieVendas14Dias.length - 1;
                  return (
                    <div
                      key={ponto.Dia}
                      className={`admin-chart-col${hoje ? " hoje" : ""}`}
                      title={`${formatarDia(ponto.Dia)}: ${mt(ponto.Total)}`}
                    >
                      <div className="admin-chart-area">
                        <div
                          className="admin-chart-seg"
                          style={{ height: `${ponto.Total > 0 ? Math.max(2, (ponto.Total / maxDiaTotal) * 100) : 0}%` }}
                        />
                      </div>
                      <div className="admin-chart-dia">{hoje ? "Hoje" : formatarDia(ponto.Dia)}</div>
                    </div>
                  );
                })}
              </div>
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
                    <div className="admin-alerta-detalhe">{a.Detalhe}</div>
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
                      style={{ width: `${(item.Quantidade / maxVendido) * 100}%`, background: "var(--accent)" }}
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
