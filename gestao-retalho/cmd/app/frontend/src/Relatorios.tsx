import { useEffect, useState } from "react";
import {
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
  ExportarTabelaExcel,
  ExportarTabelaPDF,
} from "../wailsjs/go/api/App";
import { api, domain, repository, usecase } from "../wailsjs/go/models";

type Relatorio =
  | "vendas"
  | "lucro"
  | "produtos"
  | "consumo"
  | "metodos"
  | "stock"
  | "movimentos"
  | "perdas"
  | "contas"
  | "dividas"
  | "caixa";

const RELATORIOS: { chave: Relatorio; nome: string }[] = [
  { chave: "vendas", nome: "Vendas por período" },
  { chave: "lucro", nome: "Lucro por período" },
  { chave: "produtos", nome: "Vendas por produto" },
  { chave: "consumo", nome: "Consumo teórico vs real" },
  { chave: "metodos", nome: "Vendas por método de pagamento" },
  { chave: "stock", nome: "Stock atual" },
  { chave: "movimentos", nome: "Movimentos de stock" },
  { chave: "perdas", nome: "Perdas e consumo interno" },
  { chave: "contas", nome: "Contas abertas / fechadas" },
  { chave: "dividas", nome: "Dívidas em aberto" },
  { chave: "caixa", nome: "Fecho de caixa diário" },
];

function hoje(): string {
  return new Date().toISOString().slice(0, 10);
}

function haUmMes(): string {
  const d = new Date();
  d.setUTCDate(d.getUTCDate() - 30);
  return d.toISOString().slice(0, 10);
}

// diaParaISOInicio/FimExclusivo convertem um <input type="date"> (string
// "AAAA-MM-DD", sem fuso) para os limites [Desde, Até) em UTC que o
// backend espera — Até é sempre meia-noite do dia seguinte, para incluir o
// dia selecionado inteiro (RelatorioService compara `criado_em < ate`).
function diaParaISOInicio(dia: string): string {
  return new Date(dia + "T00:00:00Z").toISOString();
}
function diaParaISOFimExclusivo(dia: string): string {
  const d = new Date(dia + "T00:00:00Z");
  d.setUTCDate(d.getUTCDate() + 1);
  return d.toISOString();
}

/**
 * Relatórios (SDD secção 6, Fase 8) — mesma família de scaffolding
 * funcional do resto de FaseUm.tsx, sem tema/polimento dedicado: um
 * seletor de relatório, filtro de período/armazém partilhado onde faz
 * sentido, e exportação Excel/PDF genérica (TabelaRelatorio) em todos.
 * "Comparativo entre negócios" (última linha da tabela do SDD 6) fica
 * para a Fase 9 — não há o que comparar dentro de uma única loja.
 */
export default function RelatoriosSection({
  armazens,
  onErro,
}: {
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const [relatorio, setRelatorio] = useState<Relatorio>("vendas");
  const [desde, setDesde] = useState(haUmMes());
  const [ate, setAte] = useState(hoje());
  const [armazemId, setArmazemId] = useState<number | "">("");

  const filtro = () => {
    const f = new api.FiltroRelatorioRequest();
    f.Desde = diaParaISOInicio(desde);
    f.Ate = diaParaISOFimExclusivo(ate);
    f.ArmazemID = armazemId === "" ? undefined : Number(armazemId);
    return f;
  };

  return (
    <section>
      <h2>Relatórios</h2>

      <div className="fase1-form">
        <select value={relatorio} onChange={(e) => setRelatorio(e.target.value as Relatorio)}>
          {RELATORIOS.map((r) => (
            <option key={r.chave} value={r.chave}>
              {r.nome}
            </option>
          ))}
        </select>
      </div>

      {relatorio !== "stock" && relatorio !== "dividas" && relatorio !== "caixa" && (
        <div className="fase1-form">
          <label>
            Desde <input type="date" value={desde} onChange={(e) => setDesde(e.target.value)} />
          </label>
          <label>
            Até <input type="date" value={ate} onChange={(e) => setAte(e.target.value)} />
          </label>
          <select value={armazemId} onChange={(e) => setArmazemId(e.target.value ? Number(e.target.value) : "")}>
            <option value="">Todos os armazéns</option>
            {armazens.map((a) => (
              <option key={a.ID} value={a.ID}>
                {a.Nome}
              </option>
            ))}
          </select>
        </div>
      )}

      {relatorio === "vendas" && <RelatorioVendas filtro={filtro()} onErro={onErro} />}
      {relatorio === "lucro" && <RelatorioLucro filtro={filtro()} onErro={onErro} />}
      {relatorio === "produtos" && <RelatorioProdutos filtro={filtro()} onErro={onErro} />}
      {relatorio === "consumo" && <RelatorioConsumo filtro={filtro()} onErro={onErro} />}
      {relatorio === "metodos" && <RelatorioMetodos filtro={filtro()} onErro={onErro} />}
      {relatorio === "stock" && <RelatorioStock armazens={armazens} onErro={onErro} />}
      {relatorio === "movimentos" && <RelatorioMovimentos filtro={filtro()} onErro={onErro} />}
      {relatorio === "perdas" && <RelatorioPerdas filtro={filtro()} onErro={onErro} />}
      {relatorio === "contas" && <RelatorioContas desde={desde} ate={ate} onErro={onErro} />}
      {relatorio === "dividas" && <RelatorioDividas onErro={onErro} />}
      {relatorio === "caixa" && <RelatorioCaixa onErro={onErro} />}
    </section>
  );
}

// --- Tabela + exportação genérica -------------------------------------------

function TabelaRelatorio({
  titulo,
  colunas,
  linhas,
  totais,
  onErro,
}: {
  titulo: string;
  colunas: string[];
  linhas: (string | number)[][];
  totais?: (string | number)[];
  onErro: (e: string) => void;
}) {
  const linhasStr = linhas.map((l) => l.map(String));
  const totaisStr = totais?.map(String);

  function exportar(formato: "excel" | "pdf") {
    const fn = formato === "excel" ? ExportarTabelaExcel : ExportarTabelaPDF;
    fn(titulo, colunas, linhasStr, totaisStr ?? []).catch((e) => onErro(String(e)));
  }

  return (
    <div>
      <div className="fase1-form">
        <button onClick={() => exportar("excel")}>Exportar Excel</button>
        <button onClick={() => exportar("pdf")}>Exportar PDF</button>
      </div>
      <table>
        <thead>
          <tr>
            {colunas.map((c) => (
              <th key={c}>{c}</th>
            ))}
          </tr>
        </thead>
        <tbody>
          {linhasStr.map((linha, i) => (
            <tr key={i}>
              {linha.map((v, j) => (
                <td key={j}>{v}</td>
              ))}
            </tr>
          ))}
          {linhasStr.length === 0 && (
            <tr>
              <td colSpan={colunas.length}>Sem dados no período selecionado</td>
            </tr>
          )}
        </tbody>
        {totaisStr && (
          <tfoot>
            <tr>
              {totaisStr.map((v, j) => (
                <td key={j}>
                  <strong>{v}</strong>
                </td>
              ))}
            </tr>
          </tfoot>
        )}
      </table>
    </div>
  );
}

const mt = (n: number) => `${n.toFixed(2)} MT`;

// --- 1. Vendas por período ---------------------------------------------------

function RelatorioVendas({ filtro, onErro }: { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void }) {
  const [resposta, setResposta] = useState<api.RelatorioVendasResponse | null>(null);

  useEffect(() => {
    RelatorioVendasPorPeriodo(filtro).then(setResposta).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- filtro é recriado a cada render; comparar por JSON evitaria isto mas o custo aqui é 1 pedido extra, não um loop
  }, [JSON.stringify(filtro)]);

  if (!resposta) return null;
  return (
    <div>
      <p className="fase1-stock-atual">
        {resposta.Resumo.NumeroVendas} venda(s) · total {mt(resposta.Resumo.TotalVendido)} · ticket médio{" "}
        {mt(resposta.Resumo.TicketMedio)}
      </p>
      <TabelaRelatorio
        titulo="Vendas por período"
        colunas={["Dia", "Nº vendas", "Total vendido"]}
        linhas={resposta.PorDia.map((l) => [l.Dia, l.NumeroVendas, mt(l.TotalVendido)])}
        totais={["TOTAL", resposta.Resumo.NumeroVendas, mt(resposta.Resumo.TotalVendido)]}
        onErro={onErro}
      />
    </div>
  );
}

// --- 2. Lucro por período -----------------------------------------------------

function RelatorioLucro({ filtro, onErro }: { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void }) {
  const [r, setR] = useState<usecase.RelatorioLucro | null>(null);

  useEffect(() => {
    RelatorioLucroPorPeriodo(filtro).then(setR).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(filtro)]);

  if (!r) return null;
  return (
    <TabelaRelatorio
      titulo="Lucro por período"
      colunas={["Receita", "Custo", "Lucro", "Margem %"]}
      linhas={[[mt(r.Receita), mt(r.Custo), mt(r.Lucro), `${r.MargemPercentual.toFixed(1)}%`]]}
      onErro={onErro}
    />
  );
}

// --- 3. Vendas por produto ----------------------------------------------------

function RelatorioProdutos({ filtro, onErro }: { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void }) {
  const [linhas, setLinhas] = useState<repository.LinhaVendasPorProduto[]>([]);

  useEffect(() => {
    RelatorioVendasPorProduto(filtro).then(setLinhas).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(filtro)]);

  return (
    <TabelaRelatorio
      titulo="Vendas por produto"
      colunas={["Produto", "Quantidade", "Valor total"]}
      linhas={linhas.map((l) => [l.ProdutoNome, l.QuantidadeTotal, mt(l.ValorTotal)])}
      onErro={onErro}
    />
  );
}

// --- 4. Consumo teórico vs real -----------------------------------------------

function RelatorioConsumo({ filtro, onErro }: { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void }) {
  const [linhas, setLinhas] = useState<usecase.LinhaConsumoIngrediente[]>([]);

  useEffect(() => {
    RelatorioConsumoTeoricoVsReal(filtro).then(setLinhas).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(filtro)]);

  return (
    <TabelaRelatorio
      titulo="Consumo teórico vs real de ingredientes"
      colunas={["Ingrediente", "Consumo teórico", "Consumo real", "Desvio", "Desvio %"]}
      linhas={linhas.map((l) => [
        l.ProdutoNome,
        l.ConsumoTeorico.toFixed(3),
        l.ConsumoReal.toFixed(3),
        l.Desvio.toFixed(3),
        `${l.DesvioPercentual.toFixed(1)}%`,
      ])}
      onErro={onErro}
    />
  );
}

// --- 5. Vendas por método de pagamento ----------------------------------------

function RelatorioMetodos({ filtro, onErro }: { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void }) {
  const [linhas, setLinhas] = useState<repository.LinhaVendasPorMetodo[]>([]);

  useEffect(() => {
    RelatorioVendasPorMetodoPagamento(filtro).then(setLinhas).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(filtro)]);

  return (
    <TabelaRelatorio
      titulo="Vendas por método de pagamento"
      colunas={["Método", "Total"]}
      linhas={linhas.map((l) => [l.Metodo, mt(l.Total)])}
      onErro={onErro}
    />
  );
}

// --- 6. Stock atual ------------------------------------------------------------

function RelatorioStock({ armazens, onErro }: { armazens: domain.Armazem[]; onErro: (e: string) => void }) {
  const [armazemId, setArmazemId] = useState<number | "">("");
  const [resposta, setResposta] = useState<api.RelatorioStockAtualResponse | null>(null);

  useEffect(() => {
    RelatorioStockAtual(armazemId === "" ? undefined : Number(armazemId))
      .then(setResposta)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [armazemId]);

  const nomeArmazem = (id: number) => armazens.find((a) => a.ID === id)?.Nome ?? `#${id}`;

  return (
    <div>
      <div className="fase1-form">
        <select value={armazemId} onChange={(e) => setArmazemId(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Todos os armazéns</option>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
      </div>
      {resposta && (
        <>
          <p className="fase1-stock-atual">Valor total de stock: {mt(resposta.ValorTotal)}</p>
          <TabelaRelatorio
            titulo="Stock atual"
            colunas={["Produto", "Armazém", "Quantidade", "Custo médio", "Valor stock", "Abaixo do mínimo"]}
            linhas={resposta.Linhas.map((l) => [
              l.ProdutoNome,
              nomeArmazem(l.ArmazemID),
              l.QuantidadeUnidadeBase,
              mt(l.CustoMedioPonderado),
              mt(l.ValorStock),
              l.AbaixoDoMinimo ? "⚠ sim" : "não",
            ])}
            totais={["TOTAL", "", "", "", mt(resposta.ValorTotal), ""]}
            onErro={onErro}
          />
        </>
      )}
    </div>
  );
}

// --- 7. Movimentos de stock ----------------------------------------------------

function RelatorioMovimentos({ filtro, onErro }: { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void }) {
  const [linhas, setLinhas] = useState<repository.LinhaMovimento[]>([]);

  useEffect(() => {
    RelatorioMovimentosPorPeriodo(filtro).then(setLinhas).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(filtro)]);

  return (
    <TabelaRelatorio
      titulo="Movimentos de stock"
      colunas={["Data", "Produto", "Tipo", "Quantidade", "Custo unitário", "Motivo"]}
      linhas={linhas.map((l) => [
        new Date(String(l.CriadoEm)).toLocaleString(),
        l.ProdutoNome,
        l.Tipo,
        l.QuantidadeUnidadeBase,
        l.CustoUnitario != null ? mt(l.CustoUnitario) : "—",
        l.Motivo ?? "—",
      ])}
      onErro={onErro}
    />
  );
}

// --- 8. Perdas e consumo interno ------------------------------------------------

function RelatorioPerdas({ filtro, onErro }: { filtro: api.FiltroRelatorioRequest; onErro: (e: string) => void }) {
  const [resposta, setResposta] = useState<api.RelatorioPerdasConsumoInternoResponse | null>(null);

  useEffect(() => {
    RelatorioPerdasConsumoInterno(filtro).then(setResposta).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(filtro)]);

  if (!resposta) return null;
  return (
    <TabelaRelatorio
      titulo="Perdas e consumo interno"
      colunas={["Tipo", "Produto", "Quantidade", "Valor"]}
      linhas={resposta.Linhas.map((l) => [l.Tipo, l.ProdutoNome, l.QuantidadeTotal, mt(l.ValorTotal)])}
      totais={["TOTAL", "", "", mt(resposta.ValorTotal)]}
      onErro={onErro}
    />
  );
}

// --- 9. Contas abertas / fechadas -----------------------------------------------

function RelatorioContas({ desde, ate, onErro }: { desde: string; ate: string; onErro: (e: string) => void }) {
  const [resposta, setResposta] = useState<api.RelatorioContasResponse | null>(null);

  useEffect(() => {
    RelatorioContasPorPeriodo(diaParaISOInicio(desde), diaParaISOFimExclusivo(ate))
      .then(setResposta)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [desde, ate]);

  if (!resposta) return null;
  return (
    <div>
      <p className="fase1-stock-atual">
        {resposta.Resumo.NumeroContas} conta(s) · tempo médio aberto {resposta.Resumo.TempoMedioAbertoHoras.toFixed(1)}h ·
        valor médio {mt(resposta.Resumo.ValorMedio)}
      </p>
      <TabelaRelatorio
        titulo="Contas abertas e fechadas"
        colunas={["Identificador", "Status", "Aberta em", "Fechada em", "Valor total", "Tempo aberto (h)"]}
        linhas={resposta.Linhas.map((l) => [
          l.Identificador,
          l.Status,
          new Date(String(l.AbertaEm)).toLocaleString(),
          l.FechadaEm ? new Date(String(l.FechadaEm)).toLocaleString() : "—",
          mt(l.ValorTotal),
          l.TempoAbertoHoras != null ? l.TempoAbertoHoras.toFixed(1) : "—",
        ])}
        onErro={onErro}
      />
    </div>
  );
}

// --- 10. Dívidas em aberto -------------------------------------------------------

function RelatorioDividas({ onErro }: { onErro: (e: string) => void }) {
  const [linhas, setLinhas] = useState<usecase.LinhaDividaRelatorio[]>([]);

  useEffect(() => {
    RelatorioDividasEmAberto().then(setLinhas).catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <TabelaRelatorio
      titulo="Dívidas em aberto"
      colunas={["Cliente", "Valor original", "Pago", "Em aberto", "Idade (dias)", "Status"]}
      linhas={linhas.map((l) => [
        l.ClienteNome,
        mt(l.ValorOriginal),
        mt(l.ValorPago),
        mt(l.ValorEmAberto),
        l.IdadeDias,
        l.IdadeDias > 30 ? `${l.Status} ⚠ antiga` : l.Status,
      ])}
      onErro={onErro}
    />
  );
}

// --- 11. Fecho de caixa diário ----------------------------------------------------

function RelatorioCaixa({ onErro }: { onErro: (e: string) => void }) {
  const [dia, setDia] = useState(hoje());
  const [fecho, setFecho] = useState<usecase.FechoCaixa | null>(null);
  const [contados, setContados] = useState<Record<string, string>>({});

  useEffect(() => {
    RelatorioFechoCaixaDiario(diaParaISOInicio(dia))
      .then((f) => {
        setFecho(f);
        setContados({});
      })
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [dia]);

  return (
    <div>
      <div className="fase1-form">
        <label>
          Dia <input type="date" value={dia} onChange={(e) => setDia(e.target.value)} />
        </label>
      </div>
      {fecho && (
        <>
          <p className="fase1-stock-atual">Total vendido em {fecho.Dia}: {mt(fecho.TotalVendido)}</p>
          <table>
            <thead>
              <tr>
                <th>Método</th>
                <th>Esperado</th>
                <th>Contado</th>
                <th>Diferença</th>
              </tr>
            </thead>
            <tbody>
              {fecho.PorMetodo.map((m) => {
                const contado = contados[m.Metodo] ?? "";
                const diferenca = contado !== "" ? Number(contado) - m.Total : null;
                return (
                  <tr key={m.Metodo}>
                    <td>{m.Metodo}</td>
                    <td>{mt(m.Total)}</td>
                    <td>
                      <input
                        type="number"
                        value={contado}
                        onChange={(e) => setContados((prev) => ({ ...prev, [m.Metodo]: e.target.value }))}
                        style={{ width: "8em" }}
                      />
                    </td>
                    <td>{diferenca != null ? mt(diferenca) : "—"}</td>
                  </tr>
                );
              })}
              {fecho.PorMetodo.length === 0 && (
                <tr>
                  <td colSpan={4}>Sem vendas neste dia</td>
                </tr>
              )}
            </tbody>
          </table>
        </>
      )}
    </div>
  );
}
