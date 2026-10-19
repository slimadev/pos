import { useEffect, useState } from "react";
import { ArrowDownTrayIcon, ExclamationTriangleIcon } from "@heroicons/react/24/outline";
import {
  ListarArmazens,
  RelatorioVendasPorPeriodo,
  RelatorioLucroPorPeriodo,
  RelatorioVendasPorProduto,
  RelatorioConsumoTeoricoVsReal,
  RelatorioVendasPorMetodoPagamento,
  RelatorioStockAtual,
  RelatorioMovimentosPorPeriodo,
  RelatorioPerdasConsumoInterno,
  RelatorioContasPorPeriodo,
  RelatorioDividasEmAberto,
  RelatorioFechoCaixaDiario,
  RelatorioVendasPorOperador,
  ExportarRelatorio,
} from "../wailsjs/go/admin/AdminApp";
import { admin, api, domain, repository, usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";

type Relatorio =
  | "vendas"
  | "lucro"
  | "produtos"
  | "metodos"
  | "operadores"
  | "caixa"
  | "stock"
  | "movimentos"
  | "perdas"
  | "dividas"
  | "contas"
  | "consumo";

type Filtro = "periodo" | "armazem" | "dia" | "nenhum";

const GRUPOS: { nome: string; relatorios: { chave: Relatorio; nome: string; filtro: Filtro; soBar?: boolean }[] }[] = [
  {
    nome: "Vendas",
    relatorios: [
      { chave: "vendas", nome: "Vendas por período", filtro: "periodo" },
      { chave: "lucro", nome: "Lucro por período", filtro: "periodo" },
      { chave: "produtos", nome: "Vendas por produto", filtro: "periodo" },
      { chave: "metodos", nome: "Vendas por método de pagamento", filtro: "periodo" },
      { chave: "operadores", nome: "Vendas por operador", filtro: "periodo" },
      { chave: "caixa", nome: "Fecho de caixa diário", filtro: "dia" },
    ],
  },
  {
    nome: "Stock",
    relatorios: [
      { chave: "stock", nome: "Stock atual", filtro: "armazem" },
      { chave: "movimentos", nome: "Movimentos de stock", filtro: "periodo" },
      { chave: "perdas", nome: "Perdas e consumo interno", filtro: "periodo" },
    ],
  },
  {
    nome: "Clientes",
    relatorios: [{ chave: "dividas", nome: "Dívidas em aberto", filtro: "nenhum" }],
  },
  {
    nome: "Bar",
    relatorios: [
      { chave: "contas", nome: "Contas abertas e fechadas", filtro: "periodo", soBar: true },
      { chave: "consumo", nome: "Consumo teórico vs real", filtro: "periodo", soBar: true },
    ],
  },
];
const TODOS = GRUPOS.flatMap((g) => g.relatorios);

const ROTULO_METODO: Record<string, string> = {
  dinheiro: "Dinheiro",
  mpesa: "M-Pesa",
  emola: "e-Mola",
  cartao: "Cartão",
  transferencia: "Transferência",
};
const ROTULO_MOVIMENTO: Record<string, string> = {
  entrada: "Entrada",
  transferencia: "Transferência",
  perda: "Perda",
  consumo_interno: "Consumo interno",
  ajuste_inventario: "Ajuste de inventário",
  venda: "Venda",
  producao: "Produção",
};
const ROTULO_ESTADO: Record<string, string> = {
  aberta: "Aberta",
  fechada: "Fechada",
  cancelada: "Cancelada",
  parcial: "Parcial",
  paga: "Paga",
};

const mt = (n: number) => `${n.toLocaleString("pt-PT", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} MT`;
const num = (n: number, casas = 3) => n.toLocaleString("pt-PT", { maximumFractionDigits: casas });
const pct = (n: number) => `${n.toLocaleString("pt-PT", { maximumFractionDigits: 1 })}%`;
const dataHora = (d: unknown) => new Date(String(d)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" });

function diaLocal(d: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}
function diasAtras(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return diaLocal(d);
}
function porExtenso(dia: string): string {
  return new Date(`${dia}T00:00:00`).toLocaleDateString("pt-PT", { day: "numeric", month: "long", year: "numeric" });
}

type Consulta = { relatorio: Relatorio; desde: string; ate: string; armazemID: number | ""; dia: string; n: number };

/**
 * Relatórios da loja (os mesmos que o cmd/app calcula), agrupados por
 * área. Os limites do período são a meia-noite do relógio local —
 * "AAAA-MM-DD" como meia-noite UTC cortaria as vendas da madrugada.
 * Cada relatório exporta para Excel ou PDF exatamente as linhas mostradas.
 */
export default function RelatoriosScreen({
  tipoNegocio,
  onErro,
}: {
  tipoNegocio?: string;
  onErro: (e: string) => void;
}) {
  const temBar = tipoNegocio === "bottlestore" || tipoNegocio === "personalizado";
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [relatorio, setRelatorio] = useState<Relatorio>("vendas");
  const [desde, setDesde] = useState(diasAtras(29));
  const [ate, setAte] = useState(diaLocal(new Date()));
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [dia, setDia] = useState(diaLocal(new Date()));
  const [consulta, setConsulta] = useState<Consulta>({ relatorio: "vendas", desde, ate, armazemID: "", dia, n: 0 });

  useEffect(() => {
    ListarArmazens()
      .then(setArmazens)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const info = TODOS.find((r) => r.chave === relatorio)!;

  function consultar(alteracoes: Partial<Consulta> = {}) {
    setConsulta((c) => ({ relatorio, desde, ate, armazemID, dia, ...alteracoes, n: c.n + 1 }));
  }

  function escolher(r: Relatorio) {
    setRelatorio(r);
    consultar({ relatorio: r });
  }

  function atalho(dias: number) {
    const d = diasAtras(dias);
    const a = diaLocal(new Date());
    setDesde(d);
    setAte(a);
    consultar({ desde: d, ate: a });
  }

  const filtro = (): api.FiltroRelatorioRequest => {
    const f = new api.FiltroRelatorioRequest();
    f.Desde = new Date(`${consulta.desde}T00:00:00`).toISOString();
    const fim = new Date(`${consulta.ate}T00:00:00`);
    fim.setDate(fim.getDate() + 1);
    f.Ate = fim.toISOString();
    f.ArmazemID = consulta.armazemID === "" ? undefined : consulta.armazemID;
    return f;
  };
  const props = { filtro: filtro(), onErro };
  const chave = JSON.stringify(consulta);

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Relatórios" }]} />

      <div className="admin-relatorios">
        <nav className="admin-relatorios-lista" aria-label="Relatórios">
          {GRUPOS.filter((g) => g.relatorios.some((r) => !r.soBar || temBar)).map((g) => (
            <div key={g.nome}>
              <div className="admin-relatorios-grupo">{g.nome}</div>
              {g.relatorios
                .filter((r) => !r.soBar || temBar)
                .map((r) => (
                  <button
                    key={r.chave}
                    className={`admin-relatorios-item${relatorio === r.chave ? " ativo" : ""}`}
                    onClick={() => escolher(r.chave)}
                  >
                    {r.nome}
                  </button>
                ))}
            </div>
          ))}
        </nav>

        <div className="admin-relatorios-conteudo">
          <h2 style={{ margin: 0 }}>{info.nome}</h2>

          {info.filtro === "periodo" && (
            <div className="admin-form">
              <label>
                De <input type="date" value={desde} max={ate} onChange={(e) => setDesde(e.target.value)} />
              </label>
              <label>
                até <input type="date" value={ate} min={desde} onChange={(e) => setAte(e.target.value)} />
              </label>
              {relatorio !== "contas" && relatorio !== "operadores" && (
                <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
                  <option value="">Todos os armazéns</option>
                  {armazens.map((a) => (
                    <option key={a.ID} value={a.ID}>
                      {a.Nome}
                    </option>
                  ))}
                </select>
              )}
              <button className="admin-btn" disabled={!desde || !ate} onClick={() => consultar()}>
                Consultar
              </button>
              <button className="admin-btn-secundario" onClick={() => atalho(0)}>
                Hoje
              </button>
              <button className="admin-btn-secundario" onClick={() => atalho(6)}>
                7 dias
              </button>
              <button className="admin-btn-secundario" onClick={() => atalho(29)}>
                30 dias
              </button>
            </div>
          )}
          {info.filtro === "armazem" && (
            <div className="admin-form">
              <select
                value={armazemID}
                onChange={(e) => {
                  const id = e.target.value ? Number(e.target.value) : "";
                  setArmazemID(id);
                  consultar({ armazemID: id });
                }}
              >
                <option value="">Todos os armazéns</option>
                {armazens.map((a) => (
                  <option key={a.ID} value={a.ID}>
                    {a.Nome}
                  </option>
                ))}
              </select>
            </div>
          )}
          {info.filtro === "dia" && (
            <div className="admin-form">
              <label>
                Dia{" "}
                <input
                  type="date"
                  value={dia}
                  onChange={(e) => {
                    setDia(e.target.value);
                    if (e.target.value) consultar({ dia: e.target.value });
                  }}
                />
              </label>
            </div>
          )}

          {info.filtro === "periodo" && consulta.relatorio === relatorio && (
            <div style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
              {consulta.desde === consulta.ate
                ? porExtenso(consulta.desde)
                : `${porExtenso(consulta.desde)} a ${porExtenso(consulta.ate)}`}
            </div>
          )}

          {consulta.relatorio === "vendas" && <RelVendas key={chave} {...props} />}
          {consulta.relatorio === "lucro" && <RelLucro key={chave} {...props} />}
          {consulta.relatorio === "produtos" && <RelProdutos key={chave} {...props} />}
          {consulta.relatorio === "metodos" && <RelMetodos key={chave} {...props} />}
          {consulta.relatorio === "operadores" && (
            <RelOperadores key={chave} desde={consulta.desde} ate={consulta.ate} onErro={onErro} />
          )}
          {consulta.relatorio === "caixa" && <RelCaixa key={chave} dia={consulta.dia} onErro={onErro} />}
          {consulta.relatorio === "stock" && (
            <RelStock key={chave} armazemID={consulta.armazemID} armazens={armazens} onErro={onErro} />
          )}
          {consulta.relatorio === "movimentos" && <RelMovimentos key={chave} {...props} />}
          {consulta.relatorio === "perdas" && <RelPerdas key={chave} {...props} />}
          {consulta.relatorio === "dividas" && <RelDividas key={chave} onErro={onErro} />}
          {consulta.relatorio === "contas" && (
            <RelContas key={chave} desde={consulta.desde} ate={consulta.ate} onErro={onErro} />
          )}
          {consulta.relatorio === "consumo" && <RelConsumo key={chave} {...props} />}
        </div>
      </div>
    </div>
  );
}

// --- Tabela + exportação ---------------------------------------------------

type Celula = string | { texto: string; alerta?: boolean };

function TabelaRelatorio({
  titulo,
  colunas,
  numericas = [],
  linhas,
  totais,
  onErro,
}: {
  titulo: string;
  colunas: string[];
  numericas?: number[];
  linhas: Celula[][] | null;
  totais?: string[];
  onErro: (e: string) => void;
}) {
  const [aExportar, setAExportar] = useState(false);
  const texto = (c: Celula) => (typeof c === "string" ? c : c.texto);
  const alinhar = (j: number) => (numericas.includes(j) ? { textAlign: "right" as const } : undefined);

  function exportar(formato: "xlsx" | "pdf") {
    if (!linhas) return;
    setAExportar(true);
    ExportarRelatorio(titulo, colunas, linhas.map((l) => l.map(texto)), totais ?? [], formato)
      .catch((e) => onErro(String(e)))
      .finally(() => setAExportar(false));
  }

  return (
    <>
      <div className="admin-form" style={{ justifyContent: "flex-end" }}>
        <button
          className="admin-btn-secundario"
          disabled={!linhas || linhas.length === 0 || aExportar}
          onClick={() => exportar("xlsx")}
        >
          <ArrowDownTrayIcon className="admin-btn-icon-inline" aria-hidden="true" />
          Excel
        </button>
        <button
          className="admin-btn-secundario"
          disabled={!linhas || linhas.length === 0 || aExportar}
          onClick={() => exportar("pdf")}
        >
          <ArrowDownTrayIcon className="admin-btn-icon-inline" aria-hidden="true" />
          PDF
        </button>
      </div>
      <table className="admin-tabela">
        <thead>
          <tr>
            {colunas.map((c, j) => (
              <th key={c} style={alinhar(j)}>
                {c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {linhas === null && (
            <tr>
              <td colSpan={colunas.length}>A carregar…</td>
            </tr>
          )}
          {linhas?.length === 0 && (
            <tr>
              <td colSpan={colunas.length}>Sem dados para este filtro.</td>
            </tr>
          )}
          {linhas?.map((linha, i) => (
            <tr key={i}>
              {linha.map((c, j) => (
                <td key={j} style={{ ...alinhar(j), fontVariantNumeric: "tabular-nums" }}>
                  {typeof c !== "string" && c.alerta && (
                    <ExclamationTriangleIcon className="admin-relatorios-alerta" aria-hidden="true" />
                  )}
                  {texto(c)}
                </td>
              ))}
            </tr>
          ))}
        </tbody>
        {totais && linhas && linhas.length > 0 && (
          <tfoot>
            <tr className="admin-relatorios-totais">
              {totais.map((v, j) => (
                <td key={j} style={alinhar(j)}>
                  {v}
                </td>
              ))}
            </tr>
          </tfoot>
        )}
      </table>
    </>
  );
}

function Resumo({ itens }: { itens: [string, string][] }) {
  return (
    <div className="admin-inventario-resumo">
      {itens.map(([valor, rotulo]) => (
        <span key={rotulo}>
          <strong>{valor}</strong> {rotulo}
        </span>
      ))}
    </div>
  );
}

// Carrega uma vez por instância — a instância muda (key) a cada consulta.
function useCarregar<T>(pedido: () => Promise<T>, onErro: (e: string) => void): T | null {
  const [dados, setDados] = useState<T | null>(null);
  useEffect(() => {
    pedido()
      .then(setDados)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- a consulta é fixa por instância
  }, []);
  return dados;
}

type PropsPeriodo = { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void };

function RelVendas({ filtro, onErro }: PropsPeriodo) {
  const r = useCarregar(() => RelatorioVendasPorPeriodo(filtro), onErro);
  return (
    <>
      {r && (
        <Resumo
          itens={[
            [String(r.Resumo.NumeroVendas), "venda(s)"],
            [mt(r.Resumo.TotalVendido), "vendido"],
            [mt(r.Resumo.TicketMedio), "por venda"],
          ]}
        />
      )}
      <TabelaRelatorio
        titulo="Vendas por período"
        colunas={["Dia", "N.º vendas", "Total vendido"]}
        numericas={[1, 2]}
        linhas={r ? r.PorDia.map((l) => [porExtenso(l.Dia), String(l.NumeroVendas), mt(l.TotalVendido)]) : null}
        totais={r ? ["Total", String(r.Resumo.NumeroVendas), mt(r.Resumo.TotalVendido)] : undefined}
        onErro={onErro}
      />
    </>
  );
}

function RelLucro({ filtro, onErro }: PropsPeriodo) {
  const r = useCarregar<usecase.RelatorioLucro>(() => RelatorioLucroPorPeriodo(filtro), onErro);
  return (
    <TabelaRelatorio
      titulo="Lucro por período"
      colunas={["Receita", "Custo", "Lucro", "Margem"]}
      numericas={[0, 1, 2, 3]}
      linhas={r ? [[mt(r.Receita), mt(r.Custo), mt(r.Lucro), pct(r.MargemPercentual)]] : null}
      onErro={onErro}
    />
  );
}

function RelProdutos({ filtro, onErro }: PropsPeriodo) {
  const linhas = useCarregar<repository.LinhaVendasPorProduto[]>(() => RelatorioVendasPorProduto(filtro), onErro);
  const total = (linhas ?? []).reduce((s, l) => s + l.ValorTotal, 0);
  return (
    <TabelaRelatorio
      titulo="Vendas por produto"
      colunas={["Produto", "Quantidade", "Valor total", "% do total"]}
      numericas={[1, 2, 3]}
      linhas={
        linhas
          ? linhas.map((l) => [
              l.ProdutoNome,
              num(l.QuantidadeTotal),
              mt(l.ValorTotal),
              total > 0 ? pct((l.ValorTotal / total) * 100) : "—",
            ])
          : null
      }
      totais={linhas ? ["Total", "", mt(total), ""] : undefined}
      onErro={onErro}
    />
  );
}

function RelMetodos({ filtro, onErro }: PropsPeriodo) {
  const linhas = useCarregar<repository.LinhaVendasPorMetodo[]>(() => RelatorioVendasPorMetodoPagamento(filtro), onErro);
  const total = (linhas ?? []).reduce((s, l) => s + l.Total, 0);
  return (
    <TabelaRelatorio
      titulo="Vendas por método de pagamento"
      colunas={["Método", "Total", "% do total"]}
      numericas={[1, 2]}
      linhas={
        linhas
          ? linhas.map((l) => [
              ROTULO_METODO[l.Metodo] ?? l.Metodo,
              mt(l.Total),
              total > 0 ? pct((l.Total / total) * 100) : "—",
            ])
          : null
      }
      totais={linhas ? ["Total", mt(total), ""] : undefined}
      onErro={onErro}
    />
  );
}

function RelOperadores({ desde, ate, onErro }: { desde: string; ate: string; onErro: (e: string) => void }) {
  const linhas = useCarregar<admin.LinhaVendasPorOperador[]>(() => RelatorioVendasPorOperador(desde, ate), onErro);
  const total = (linhas ?? []).reduce((s, l) => s + l.Total, 0);
  const vendas = (linhas ?? []).reduce((s, l) => s + l.NumeroVendas, 0);
  return (
    <TabelaRelatorio
      titulo="Vendas por operador"
      colunas={["Operador", "N.º vendas", "Total", "Por venda", "% do total"]}
      numericas={[1, 2, 3, 4]}
      linhas={
        linhas
          ? linhas.map((l) => [
              l.OperadorNome,
              String(l.NumeroVendas),
              mt(l.Total),
              mt(l.NumeroVendas > 0 ? l.Total / l.NumeroVendas : 0),
              total > 0 ? pct((l.Total / total) * 100) : "—",
            ])
          : null
      }
      totais={linhas ? ["Total", String(vendas), mt(total), "", ""] : undefined}
      onErro={onErro}
    />
  );
}

function RelCaixa({ dia, onErro }: { dia: string; onErro: (e: string) => void }) {
  const fecho = useCarregar<usecase.FechoCaixa>(() => RelatorioFechoCaixaDiario(dia), onErro);
  const [contados, setContados] = useState<Record<string, string>>({});

  if (!fecho) return <p>A carregar…</p>;
  const diferencaTotal = fecho.PorMetodo.reduce((s, m) => {
    const c = contados[m.Metodo];
    return c ? s + (Number(c.replace(",", ".")) - m.Total) : s;
  }, 0);
  const algumContado = Object.values(contados).some((c) => c !== "");

  return (
    <>
      <Resumo
        itens={[
          [mt(fecho.TotalRecebido), `recebido a ${porExtenso(fecho.Dia)}`],
          [mt(fecho.TotalVendido), "de vendas"],
          [mt(fecho.TotalDividasRecebidas), "de dívidas pagas"],
        ]}
      />
      <p style={{ color: "var(--text-secondary)", fontSize: "0.85rem", margin: 0 }}>
        Escreva o que contou para cada método (gaveta, M-Pesa, etc.) e veja a diferença face ao que devia lá estar.
        Uma venda a crédito só conta com a entrada que foi paga.
      </p>
      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Método</th>
            <th style={{ textAlign: "right" }}>Vendas</th>
            <th style={{ textAlign: "right" }}>Dívidas pagas</th>
            <th style={{ textAlign: "right" }}>Esperado</th>
            <th style={{ textAlign: "right" }}>Contado</th>
            <th style={{ textAlign: "right" }}>Diferença</th>
          </tr>
        </thead>
        <tbody>
          {fecho.PorMetodo.length === 0 && (
            <tr>
              <td colSpan={6}>Nada recebido neste dia.</td>
            </tr>
          )}
          {fecho.PorMetodo.map((m) => {
            const contado = contados[m.Metodo] ?? "";
            const diferenca = contado !== "" ? Number(contado.replace(",", ".")) - m.Total : null;
            return (
              <tr key={m.Metodo}>
                <td>{ROTULO_METODO[m.Metodo] ?? m.Metodo}</td>
                <td style={{ textAlign: "right" }}>{m.Vendas ? mt(m.Vendas) : "—"}</td>
                <td style={{ textAlign: "right" }}>{m.Dividas ? mt(m.Dividas) : "—"}</td>
                <td style={{ textAlign: "right", fontWeight: 600 }}>{mt(m.Total)}</td>
                <td style={{ textAlign: "right" }}>
                  <input
                    className="admin-inventario-contagem"
                    inputMode="decimal"
                    placeholder="—"
                    value={contado}
                    onChange={(e) => setContados((prev) => ({ ...prev, [m.Metodo]: e.target.value }))}
                  />
                </td>
                <td style={{ textAlign: "right" }}>
                  <DiferencaCaixa valor={diferenca} />
                </td>
              </tr>
            );
          })}
        </tbody>
        {algumContado && (
          <tfoot>
            <tr className="admin-relatorios-totais">
              <td>Diferença total</td>
              <td></td>
              <td></td>
              <td></td>
              <td></td>
              <td style={{ textAlign: "right" }}>
                <DiferencaCaixa valor={diferencaTotal} />
              </td>
            </tr>
          </tfoot>
        )}
      </table>
    </>
  );
}

function DiferencaCaixa({ valor }: { valor: number | null }) {
  if (valor == null || Number.isNaN(valor)) return <span style={{ color: "var(--text-secondary)" }}>—</span>;
  if (Math.abs(valor) < 0.005) return <span className="admin-inventario-dif zero">certo</span>;
  return (
    <span className={`admin-inventario-dif ${valor > 0 ? "sobra" : "falta"}`}>
      {valor > 0 ? "+" : "−"}
      {mt(Math.abs(valor))}
    </span>
  );
}

function RelStock({
  armazemID,
  armazens,
  onErro,
}: {
  armazemID: number | "";
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const r = useCarregar<api.RelatorioStockAtualResponse>(
    () => RelatorioStockAtual(armazemID === "" ? undefined : armazemID),
    onErro,
  );
  const nomeArmazem = (id: number) => armazens.find((a) => a.ID === id)?.Nome ?? `#${id}`;
  const abaixo = (r?.Linhas ?? []).filter((l) => l.AbaixoDoMinimo).length;
  return (
    <>
      {r && (
        <Resumo
          itens={[
            [mt(r.ValorTotal), "em stock (ao custo)"],
            [String(abaixo), "abaixo do mínimo"],
          ]}
        />
      )}
      <TabelaRelatorio
        titulo="Stock atual"
        colunas={["Produto", "Armazém", "Quantidade", "Custo médio", "Valor"]}
        numericas={[2, 3, 4]}
        linhas={
          r
            ? r.Linhas.map((l) => [
                { texto: l.ProdutoNome + (l.AbaixoDoMinimo ? " (abaixo do mínimo)" : ""), alerta: l.AbaixoDoMinimo },
                nomeArmazem(l.ArmazemID),
                num(l.QuantidadeUnidadeBase),
                mt(l.CustoMedioPonderado),
                mt(l.ValorStock),
              ])
            : null
        }
        totais={r ? ["Total", "", "", "", mt(r.ValorTotal)] : undefined}
        onErro={onErro}
      />
    </>
  );
}

function RelMovimentos({ filtro, onErro }: PropsPeriodo) {
  const linhas = useCarregar<repository.LinhaMovimento[]>(() => RelatorioMovimentosPorPeriodo(filtro), onErro);
  return (
    <TabelaRelatorio
      titulo="Movimentos de stock"
      colunas={["Data", "Produto", "Tipo", "Quantidade", "Custo unitário", "Motivo"]}
      numericas={[3, 4]}
      linhas={
        linhas
          ? linhas.map((l) => [
              dataHora(l.CriadoEm),
              l.ProdutoNome,
              ROTULO_MOVIMENTO[l.Tipo] ?? l.Tipo,
              num(l.QuantidadeUnidadeBase),
              l.CustoUnitario != null ? mt(l.CustoUnitario) : "—",
              l.Motivo ?? "—",
            ])
          : null
      }
      onErro={onErro}
    />
  );
}

function RelPerdas({ filtro, onErro }: PropsPeriodo) {
  const r = useCarregar<api.RelatorioPerdasConsumoInternoResponse>(() => RelatorioPerdasConsumoInterno(filtro), onErro);
  return (
    <TabelaRelatorio
      titulo="Perdas e consumo interno"
      colunas={["Tipo", "Produto", "Quantidade", "Valor"]}
      numericas={[2, 3]}
      linhas={
        r
          ? r.Linhas.map((l) => [ROTULO_MOVIMENTO[l.Tipo] ?? l.Tipo, l.ProdutoNome, num(l.QuantidadeTotal), mt(l.ValorTotal)])
          : null
      }
      totais={r ? ["Total", "", "", mt(r.ValorTotal)] : undefined}
      onErro={onErro}
    />
  );
}

function RelDividas({ onErro }: { onErro: (e: string) => void }) {
  const linhas = useCarregar<usecase.LinhaDividaRelatorio[]>(() => RelatorioDividasEmAberto(), onErro);
  const total = (linhas ?? []).reduce((s, l) => s + l.ValorEmAberto, 0);
  return (
    <>
      {linhas && (
        <Resumo
          itens={[
            [mt(total), "em aberto"],
            [String(linhas.filter((l) => l.IdadeDias > 30).length), "com mais de 30 dias"],
          ]}
        />
      )}
      <TabelaRelatorio
        titulo="Dívidas em aberto"
        colunas={["Cliente", "Valor original", "Pago", "Em aberto", "Idade (dias)", "Estado"]}
        numericas={[1, 2, 3, 4]}
        linhas={
          linhas
            ? linhas.map((l) => [
                l.ClienteNome,
                mt(l.ValorOriginal),
                mt(l.ValorPago),
                mt(l.ValorEmAberto),
                { texto: String(l.IdadeDias), alerta: l.IdadeDias > 30 },
                ROTULO_ESTADO[l.Status] ?? l.Status,
              ])
            : null
        }
        totais={linhas ? ["Total", "", "", mt(total), "", ""] : undefined}
        onErro={onErro}
      />
    </>
  );
}

function RelContas({ desde, ate, onErro }: { desde: string; ate: string; onErro: (e: string) => void }) {
  const r = useCarregar<api.RelatorioContasResponse>(() => RelatorioContasPorPeriodo(desde, ate), onErro);
  return (
    <>
      {r && (
        <Resumo
          itens={[
            [String(r.Resumo.NumeroContas), "conta(s)"],
            [mt(r.Resumo.ValorMedio), "valor médio"],
            [`${num(r.Resumo.TempoMedioAbertoHoras, 1)} h`, "aberta em média"],
          ]}
        />
      )}
      <TabelaRelatorio
        titulo="Contas abertas e fechadas"
        colunas={["Identificador", "Estado", "Aberta em", "Fechada em", "Valor", "Tempo aberta"]}
        numericas={[4, 5]}
        linhas={
          r
            ? r.Linhas.map((l) => [
                l.Identificador,
                ROTULO_ESTADO[l.Status] ?? l.Status,
                dataHora(l.AbertaEm),
                l.FechadaEm ? dataHora(l.FechadaEm) : "—",
                mt(l.ValorTotal),
                l.TempoAbertoHoras != null ? `${num(l.TempoAbertoHoras, 1)} h` : "—",
              ])
            : null
        }
        onErro={onErro}
      />
    </>
  );
}

function RelConsumo({ filtro, onErro }: PropsPeriodo) {
  const linhas = useCarregar<usecase.LinhaConsumoIngrediente[]>(() => RelatorioConsumoTeoricoVsReal(filtro), onErro);
  return (
    <TabelaRelatorio
      titulo="Consumo teórico vs real de ingredientes"
      colunas={["Ingrediente", "Teórico", "Real", "Desvio", "Desvio %"]}
      numericas={[1, 2, 3, 4]}
      linhas={
        linhas
          ? linhas.map((l) => [
              l.ProdutoNome,
              num(l.ConsumoTeorico),
              num(l.ConsumoReal),
              { texto: num(l.Desvio), alerta: Math.abs(l.DesvioPercentual) > 10 },
              pct(l.DesvioPercentual),
            ])
          : null
      }
      onErro={onErro}
    />
  );
}
