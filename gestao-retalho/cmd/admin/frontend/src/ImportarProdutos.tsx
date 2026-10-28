import { useEffect, useState } from "react";
import {
  ArrowUpTrayIcon,
  DocumentArrowDownIcon,
  CheckIcon,
  XMarkIcon,
} from "@heroicons/react/24/outline";
import {
  EscolherFicheiroExcel,
  AnalisarExcelProdutos,
  PreVisualizarImportacao,
  ImportarProdutos,
  AbrirModeloImportacao,
  ListarArmazens,
} from "../wailsjs/go/admin/AdminApp";
import { domain, usecase } from "../wailsjs/go/models";
import { mt, num } from "./formato";

const lerNumero = (s: string) => Number(s.trim().replace(/\s/g, "").replace(",", "."));

const CAMPOS: { chave: string; rotulo: string; obrigatorio?: boolean }[] = [
  { chave: "nome", rotulo: "Nome do produto", obrigatorio: true },
  { chave: "categoria", rotulo: "Categoria" },
  { chave: "quantidade", rotulo: "Quantidade (stock)" },
  { chave: "custo", rotulo: "Custo" },
  { chave: "preco", rotulo: "Preço de venda" },
  { chave: "unidade", rotulo: "Unidade" },
  { chave: "codigo", rotulo: "Código de barras" },
];

const ESTADO: Record<string, { rotulo: string; classe: string }> = {
  novo: { rotulo: "Novo", classe: "paga" },
  existe: { rotulo: "Já existe", classe: "cancelada" },
  repetido: { rotulo: "Repetido", classe: "cancelada" },
  erro: { rotulo: "Erro", classe: "divida" },
};

/**
 * Importa produtos novos de uma folha de Excel: escolhe-se o ficheiro, a
 * folha e que coluna é cada campo (o programa sugere pelo cabeçalho),
 * revêem-se as linhas e escreve-se o preço de venda que faltar. Produtos
 * que já existem são ignorados. A quantidade entra como stock no armazém
 * escolhido, tudo numa só operação.
 */
export default function ImportarProdutosAssistente({
  categorias,
  categoriaInicial,
  onConcluido,
  onCancelar,
  onErro,
}: {
  categorias: domain.Categoria[];
  categoriaInicial?: number;
  onConcluido: (r: usecase.ResultadoImportacao) => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [analise, setAnalise] = useState<usecase.AnaliseExcel | null>(null);
  const [folha, setFolha] = useState("");
  const [mapa, setMapa] = useState<Record<string, number>>({});
  const [categoriaPadrao, setCategoriaPadrao] = useState<number | "">(categoriaInicial ?? "");
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [lidas, setLinhas] = useState<usecase.LinhaImportacao[] | null>(null);
  const [precos, setPrecos] = useState<Record<number, string>>({});
  const [excluidas, setExcluidas] = useState<Set<number>>(new Set());
  const [margem, setMargem] = useState("30");
  const [aCarregar, setACarregar] = useState(false);
  const [aImportar, setAImportar] = useState(false);

  useEffect(() => {
    ListarArmazens()
      .then((l) => {
        const ativos = l.filter((a) => a.Ativo);
        setArmazens(ativos);
        const inicial = ativos.find((a) => a.Tipo === "principal") ?? ativos[0];
        if (inicial) setArmazemID(inicial.ID);
      })
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  function escolherFicheiro() {
    EscolherFicheiroExcel()
      .then((caminho) => {
        if (!caminho) return;
        setACarregar(true);
        return AnalisarExcelProdutos(caminho).then((a) => {
          setAnalise(a);
          const f = a.Folhas.find((x) => x.LinhaCabecalho > 0) ?? a.Folhas[0];
          if (f) escolherFolha(a, f.Nome);
        });
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  function escolherFolha(a: usecase.AnaliseExcel, nome: string) {
    const f = a.Folhas.find((x) => x.Nome === nome);
    setFolha(nome);
    setMapa({ ...(f?.Mapeamento ?? {}) });
  }

  const folhaAtual = analise?.Folhas.find((f) => f.Nome === folha);

  // A pré-visualização acompanha a folha e as colunas escolhidas.
  useEffect(() => {
    if (!analise || !folha || (mapa.nome ?? -1) < 0) return;
    const p = new usecase.PedidoPreVisualizacao();
    p.Ficheiro = analise.Ficheiro;
    p.Folha = folha;
    p.Mapeamento = mapa;
    PreVisualizarImportacao(p)
      .then((ls) => {
        setLinhas(ls);
        setPrecos(Object.fromEntries(ls.map((l) => [l.Linha, l.Preco > 0 ? String(l.Preco).replace(".", ",") : ""])));
        setExcluidas(new Set());
      })
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, [analise, folha, mapa]);

  // Sem coluna do nome não há pré-visualização (a anterior deixa de valer).
  const linhas = analise && folha && (mapa.nome ?? -1) >= 0 ? lidas : null;
  const novas = (linhas ?? []).filter((l) => l.Estado === "novo");
  const aImportarLinhas = novas.filter((l) => !excluidas.has(l.Linha));
  const precoDe = (l: usecase.LinhaImportacao) => lerNumero(precos[l.Linha] ?? "");
  const semPreco = aImportarLinhas.filter((l) => !(precoDe(l) > 0));
  const contagem = (estado: string) => (linhas ?? []).filter((l) => l.Estado === estado).length;
  const nomeCategoria = (id: number | "") => categorias.find((c) => c.ID === id)?.Nome;

  function aplicarMargem() {
    const m = lerNumero(margem);
    if (!Number.isFinite(m) || m < 0) return;
    setPrecos((p) => {
      const n = { ...p };
      for (const l of aImportarLinhas) {
        if (!(lerNumero(n[l.Linha] ?? "") > 0) && l.Custo > 0) {
          n[l.Linha] = (Math.round(l.Custo * (1 + m / 100) * 100) / 100).toFixed(2).replace(".", ",");
        }
      }
      return n;
    });
  }

  function importar() {
    if (aImportarLinhas.length === 0 || semPreco.length > 0 || armazemID === "" || !analise || aImportar) return;
    const p = new usecase.ImportarProdutosParams();
    p.ArmazemID = Number(armazemID);
    p.CategoriaPadraoID = categoriaPadrao === "" ? undefined : categoriaPadrao;
    p.Referencia = `${analise.Ficheiro.split(/[\\/]/).pop()} — ${folha}`;
    p.Linhas = aImportarLinhas.map((l) => ({ ...l, Preco: precoDe(l) }) as usecase.LinhaImportacao);
    setAImportar(true);
    ImportarProdutos(p)
      .then(onConcluido)
      .catch((e) => onErro(String(e)))
      .finally(() => setAImportar(false));
  }

  return (
    <div className="admin-produto-form">
      <div className="admin-importar-topo">
        <button className="admin-btn" onClick={escolherFicheiro} disabled={aCarregar}>
          <ArrowUpTrayIcon className="admin-btn-icon-inline" aria-hidden="true" />
          {analise ? "Escolher outro ficheiro…" : "Escolher ficheiro Excel…"}
        </button>
        {analise ? (
          <span className="admin-importar-ficheiro">{analise.Ficheiro.split(/[\\/]/).pop()}</span>
        ) : (
          <span className="admin-texto-fraco">
            Uma folha com uma linha de títulos (Descrição, Quantidade, Custo, Preço…). Não precisa de estar na primeira
            linha.
          </span>
        )}
        <button className="admin-btn-secundario" style={{ marginLeft: "auto" }} onClick={() => AbrirModeloImportacao().catch((e) => onErro(String(e)))}>
          <DocumentArrowDownIcon className="admin-btn-icon-inline" aria-hidden="true" />
          Folha modelo
        </button>
      </div>

      {analise && (
        <>
          <div className="admin-config-grelha admin-importar-opcoes">
            <label className="admin-campo">
              Folha
              <select value={folha} onChange={(e) => escolherFolha(analise, e.target.value)}>
                {analise.Folhas.map((f) => (
                  <option key={f.Nome} value={f.Nome}>
                    {f.Nome}
                    {f.LinhaCabecalho > 0 ? ` (${f.LinhasDados} linhas)` : " (sem cabeçalho reconhecido)"}
                  </option>
                ))}
              </select>
            </label>
            <label className="admin-campo">
              Categoria {mapa.categoria >= 0 ? "(para as linhas sem categoria)" : ""}
              <select value={categoriaPadrao} onChange={(e) => setCategoriaPadrao(e.target.value ? Number(e.target.value) : "")}>
                <option value="">Sem categoria</option>
                {categorias
                  .filter((c) => c.Ativo)
                  .map((c) => (
                    <option key={c.ID} value={c.ID}>
                      {c.Nome}
                    </option>
                  ))}
              </select>
            </label>
            <label className="admin-campo">
              Stock entra no armazém
              <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
                {armazens.map((a) => (
                  <option key={a.ID} value={a.ID}>
                    {a.Nome}
                  </option>
                ))}
              </select>
            </label>
          </div>

          {folhaAtual && folhaAtual.LinhaCabecalho > 0 && (
            <div>
              <div className="admin-unidades-titulo">
                Colunas (cabeçalho na linha {folhaAtual.LinhaCabecalho} — corrija se alguma estiver errada)
              </div>
              <div className="admin-importar-colunas">
                {CAMPOS.map((c) => (
                  <label key={c.chave} className="admin-campo">
                    {c.rotulo}
                    {c.obrigatorio ? " *" : ""}
                    <select
                      value={mapa[c.chave] ?? -1}
                      onChange={(e) => setMapa((m) => ({ ...m, [c.chave]: Number(e.target.value) }))}
                    >
                      <option value={-1}>— nenhuma —</option>
                      {folhaAtual.Colunas.map((col, i) =>
                        col.trim() ? (
                          <option key={i} value={i}>
                            {col.trim()}
                          </option>
                        ) : null,
                      )}
                    </select>
                  </label>
                ))}
              </div>
            </div>
          )}
          {folhaAtual && folhaAtual.LinhaCabecalho === 0 && (
            <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
              Nesta folha não se encontrou uma linha de títulos com o nome do produto. Escolha outra folha ou use a folha
              modelo.
            </p>
          )}
        </>
      )}

      {linhas && (
        <>
          <div className="admin-importar-resumo">
            <span>
              <strong>{aImportarLinhas.length}</strong> a importar
            </span>
            {contagem("existe") > 0 && <span>{contagem("existe")} já existem (ignorados)</span>}
            {contagem("repetido") > 0 && <span>{contagem("repetido")} repetidos no ficheiro</span>}
            {contagem("erro") > 0 && <span className="admin-campo-erro">{contagem("erro")} com erro</span>}
            <span className="admin-importar-margem">
              Preço = custo +
              <input inputMode="decimal" value={margem} onChange={(e) => setMargem(e.target.value)} />%
              <button className="admin-btn-secundario" onClick={aplicarMargem}>
                <CheckIcon className="btn-icone" aria-hidden="true" />
                Aplicar aos sem preço
              </button>
            </span>
          </div>

          <div className="admin-importar-tabela">
            <table className="admin-tabela admin-unidades-tabela">
              <thead>
                <tr>
                  <th style={{ width: 32 }}></th>
                  <th style={{ width: 48 }}>Linha</th>
                  <th>Produto</th>
                  <th>Categoria</th>
                  <th style={{ textAlign: "right" }}>Qtd</th>
                  <th style={{ textAlign: "right" }}>Custo</th>
                  <th style={{ width: 130 }}>Preço de venda</th>
                  <th>Estado</th>
                </tr>
              </thead>
              <tbody>
                {linhas.map((l) => {
                  const nova = l.Estado === "novo";
                  const incluida = nova && !excluidas.has(l.Linha);
                  const preco = precoDe(l);
                  return (
                    <tr key={l.Linha} className={incluida ? undefined : "admin-linha-inativa"}>
                      <td>
                        {nova && (
                          <input
                            type="checkbox"
                            aria-label={`Importar ${l.Nome}`}
                            checked={incluida}
                            onChange={(e) =>
                              setExcluidas((s) => {
                                const n = new Set(s);
                                if (e.target.checked) n.delete(l.Linha);
                                else n.add(l.Linha);
                                return n;
                              })
                            }
                          />
                        )}
                      </td>
                      <td>{l.Linha}</td>
                      <td>
                        {l.Nome}
                        {l.Unidade && <span className="admin-texto-fraco"> · {l.Unidade}</span>}
                      </td>
                      <td>
                        {l.Categoria || nomeCategoria(categoriaPadrao) || "—"}
                        {l.CategoriaNova && nova && <span className="admin-etiqueta">nova</span>}
                      </td>
                      <td style={{ textAlign: "right" }}>{num(l.Quantidade)}</td>
                      <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{l.Custo > 0 ? mt(l.Custo) : "—"}</td>
                      <td>
                        {nova ? (
                          <input
                            aria-label={`Preço de ${l.Nome}`}
                            inputMode="decimal"
                            placeholder="obrigatório"
                            className={incluida && !(preco > 0) ? "admin-importar-falta" : undefined}
                            value={precos[l.Linha] ?? ""}
                            disabled={!incluida}
                            onChange={(e) => setPrecos((p) => ({ ...p, [l.Linha]: e.target.value }))}
                          />
                        ) : l.Preco > 0 ? (
                          mt(l.Preco)
                        ) : (
                          "—"
                        )}
                        {incluida && preco > 0 && l.Custo > 0 && preco < l.Custo && (
                          <div className="admin-campo-erro">abaixo do custo</div>
                        )}
                      </td>
                      <td>
                        <span className={`admin-venda-estado ${ESTADO[l.Estado]?.classe ?? ""}`}>
                          {ESTADO[l.Estado]?.rotulo ?? l.Estado}
                        </span>
                        {l.Motivo && !nova && <div className="admin-texto-fraco">{l.Motivo}</div>}
                      </td>
                    </tr>
                  );
                })}
                {linhas.length === 0 && (
                  <tr>
                    <td colSpan={8}>Nenhuma linha com nome de produto nesta folha.</td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </>
      )}

      <div className="admin-form">
        <button
          className="admin-btn"
          disabled={aImportarLinhas.length === 0 || semPreco.length > 0 || armazemID === "" || aImportar}
          onClick={importar}
        >
          {aImportar ? "A importar…" : `Importar ${aImportarLinhas.length} produto(s)`}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
        {semPreco.length > 0 && (
          <span className="admin-campo-erro">Falta o preço de venda em {semPreco.length} produto(s).</span>
        )}
      </div>
    </div>
  );
}
