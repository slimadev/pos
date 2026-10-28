import { useEffect, useState } from "react";
import {
  ExclamationTriangleIcon,
  TrashIcon,
  BoltIcon,
  DocumentDuplicateIcon,
  PlusIcon,
  XMarkIcon,
  CheckIcon,
} from "@heroicons/react/24/outline";
import {
  ListarArmazens,
  ListarProdutos,
  ListarStockPorArmazem,
  ListarFichasTecnicas,
  DefinirReceita,
  DuplicarReceita,
  RegistarProducao,
} from "../wailsjs/go/admin/AdminApp";
import { admin, domain, usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";
import { mt, num } from "./formato";

const pct = (n: number) => `${n.toLocaleString("pt-PT", { maximumFractionDigits: 1 })}%`;
const lerNumero = (s: string) => Number(s.replace(",", "."));
const consumoReal = (quantidade: number, perda: number) => (perda > 0 && perda < 100 ? quantidade / (1 - perda / 100) : quantidade);

const ROTULO_TIPO: Record<string, string> = { composto: "Composto", semi_elaborado: "Semi Produzido" };

type Vista = { tipo: "lista" } | { tipo: "editar"; ficha: admin.FichaResumo };
type ModalLista = { tipo: "nenhum" } | { tipo: "produzir"; ficha: admin.FichaResumo } | { tipo: "duplicar"; ficha: admin.FichaResumo };

/**
 * Fichas técnicas (receitas) dos produtos compostos e semi-produzidos: o
 * que cada um leva, quanto custa a fazer face ao preço de venda e quanto dá
 * para fazer com o stock do armazém. Um composto sem receita não se
 * consegue vender. Os semi-produzidos produzem-se aqui em lote.
 */
export default function FichasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [vista, setVista] = useState<Vista>({ tipo: "lista" });
  const [versao, setVersao] = useState(0);

  useEffect(() => {
    ListarArmazens()
      .then((lista) => {
        const ativos = lista.filter((a) => a.Ativo);
        setArmazens(ativos);
        if (ativos.length > 0) setArmazemID(ativos[0].ID);
      })
      .catch((e) => onErro(String(e)));
    ListarProdutos()
      .then(setProdutos)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const voltar = () => {
    setVista({ tipo: "lista" });
    setVersao((v) => v + 1);
  };

  const seletorArmazem = (
    <label className="admin-stock-armazem-seletor">
      Armazém:{" "}
      <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
        {armazens.map((a) => (
          <option key={a.ID} value={a.ID}>
            {a.Nome}
          </option>
        ))}
      </select>
    </label>
  );

  if (vista.tipo === "editar" && armazemID !== "") {
    return (
      <div className="admin-lojas-screen">
        <Breadcrumb itens={[{ rotulo: "Fichas técnicas", onClick: voltar }, { rotulo: vista.ficha.Produto.Nome }]} />
        <EditorReceita
          key={armazemID}
          ficha={vista.ficha}
          armazemID={armazemID}
          produtos={produtos}
          seletorArmazem={seletorArmazem}
          onGuardada={voltar}
          onErro={onErro}
        />
      </div>
    );
  }

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Fichas técnicas" }]} />
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>Fichas técnicas</h2>
          <div style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
            Receitas dos compostos e semi-produzidos (criados em "Produtos").
          </div>
        </div>
        {seletorArmazem}
      </div>
      {armazemID !== "" && (
        <ListaFichas
          key={`${armazemID}-${versao}`}
          armazemID={armazemID}
          onEditar={(ficha) => setVista({ tipo: "editar", ficha })}
          onErro={onErro}
        />
      )}
    </div>
  );
}

function ListaFichas({
  armazemID,
  onEditar,
  onErro,
}: {
  armazemID: number;
  onEditar: (f: admin.FichaResumo) => void;
  onErro: (e: string) => void;
}) {
  const [fichas, setFichas] = useState<admin.FichaResumo[] | null>(null);
  const [modal, setModal] = useState<ModalLista>({ tipo: "nenhum" });
  const [n, setN] = useState(0);

  useEffect(() => {
    ListarFichasTecnicas(armazemID)
      .then(setFichas)
      .catch((e) => {
        setFichas([]);
        onErro(String(e));
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- recarrega quando `n` muda (depois de produzir/duplicar)
  }, [n]);

  const lista = [...(fichas ?? [])]
    .filter((f) => f.Produto.Ativo)
    .sort((a, b) => a.Produto.Nome.localeCompare(b.Produto.Nome, "pt"));
  const semReceita = lista.filter((f) => f.Ingredientes.length === 0);
  const fechar = () => setModal({ tipo: "nenhum" });
  const recarregar = () => {
    fechar();
    setN((x) => x + 1);
  };

  return (
    <>
      <div className="admin-inventario-resumo">
        <span>
          <strong>{lista.length - semReceita.length}</strong> com receita
        </span>
        {semReceita.length > 0 && (
          <span className="admin-inventario-dif falta">
            <ExclamationTriangleIcon className="admin-relatorios-alerta" aria-hidden="true" />
            {semReceita.length} sem receita — os compostos não se conseguem vender assim
          </span>
        )}
      </div>

      <table className="admin-tabela admin-fichas-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th>Tipo</th>
            <th style={{ textAlign: "right" }}>Ingredientes</th>
            <th style={{ textAlign: "right" }}>Custo</th>
            <th style={{ textAlign: "right" }}>Preço</th>
            <th style={{ textAlign: "right" }}>Margem</th>
            <th style={{ textAlign: "right" }}>Dá para fazer</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {fichas === null && (
            <tr>
              <td colSpan={8}>A carregar…</td>
            </tr>
          )}
          {fichas !== null && lista.length === 0 && (
            <tr>
              <td colSpan={8}>Não há produtos compostos nem semi-produzidos. Crie-os em "Produtos".</td>
            </tr>
          )}
          {lista.map((f) => {
            const temReceita = f.Ingredientes.length > 0;
            const composto = f.Produto.TipoProduto === "composto";
            const margem = f.PrecoVenda > 0 ? ((f.PrecoVenda - f.CustoTeorico) / f.PrecoVenda) * 100 : null;
            return (
              <tr key={f.Produto.ID}>
                <td>{f.Produto.Nome}</td>
                <td>{ROTULO_TIPO[f.Produto.TipoProduto] ?? f.Produto.TipoProduto}</td>
                <td style={{ textAlign: "right" }}>
                  {temReceita ? f.Ingredientes.length : <span className="admin-inventario-dif falta">sem receita</span>}
                </td>
                <td style={{ textAlign: "right" }}>{temReceita ? mt(f.CustoTeorico) : "—"}</td>
                <td style={{ textAlign: "right" }}>{composto ? mt(f.PrecoVenda) : "—"}</td>
                <td style={{ textAlign: "right" }}>
                  {composto && temReceita && margem != null ? (
                    <span className={margem < 0 ? "admin-inventario-dif falta" : undefined}>{pct(margem)}</span>
                  ) : (
                    "—"
                  )}
                </td>
                <td style={{ textAlign: "right" }}>
                  {temReceita ? `${num(f.Disponivel)} ${f.Produto.UnidadeBase}` : "—"}
                </td>
                <td className="admin-fichas-acoes">
                  <button className="admin-btn-secundario" onClick={() => onEditar(f)}>
                    <CheckIcon className="btn-icone" aria-hidden="true" />
                    {temReceita ? "Editar receita" : "Criar receita"}
                  </button>
                  {!composto && temReceita && (
                    <button className="admin-btn" onClick={() => setModal({ tipo: "produzir", ficha: f })}>
                      <BoltIcon className="btn-icone" aria-hidden="true" />
                      Produzir
                    </button>
                  )}
                  {temReceita && (
                    <button className="admin-btn-secundario" onClick={() => setModal({ tipo: "duplicar", ficha: f })}>
                      <DocumentDuplicateIcon className="btn-icone" aria-hidden="true" />
                      Duplicar
                    </button>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {modal.tipo === "produzir" && (
        <Modal titulo={`Produzir ${modal.ficha.Produto.Nome}`} onFechar={fechar}>
          <Produzir ficha={modal.ficha} armazemID={armazemID} onProduzido={recarregar} onCancelar={fechar} onErro={onErro} />
        </Modal>
      )}
      {modal.tipo === "duplicar" && (
        <Modal titulo={`Duplicar a receita de ${modal.ficha.Produto.Nome}`} onFechar={fechar}>
          <Duplicar origem={modal.ficha} fichas={lista} onDuplicada={recarregar} onCancelar={fechar} onErro={onErro} />
        </Modal>
      )}
    </>
  );
}

type LinhaReceita = { chave: number; ingredienteID: number | ""; quantidade: string; perda: string };
let proximaChave = 0;

function EditorReceita({
  ficha,
  armazemID,
  produtos,
  seletorArmazem,
  onGuardada,
  onErro,
}: {
  ficha: admin.FichaResumo;
  armazemID: number;
  produtos: domain.Produto[];
  seletorArmazem: React.ReactNode;
  onGuardada: () => void;
  onErro: (e: string) => void;
}) {
  const [custos, setCustos] = useState<Record<number, number>>({});
  const [linhas, setLinhas] = useState<LinhaReceita[]>(() =>
    ficha.Ingredientes.length > 0
      ? ficha.Ingredientes.map((i) => ({
          chave: ++proximaChave,
          ingredienteID: i.ProdutoIngredienteID,
          quantidade: String(i.Quantidade).replace(".", ","),
          perda: i.PerdaPercentual ? String(i.PerdaPercentual).replace(".", ",") : "",
        }))
      : [{ chave: ++proximaChave, ingredienteID: "", quantidade: "", perda: "" }],
  );
  const [aGuardar, setAGuardar] = useState(false);

  useEffect(() => {
    ListarStockPorArmazem(armazemID)
      .then((itens) => setCustos(Object.fromEntries(itens.map((i) => [i.ProdutoID, i.CustoMedioPonderado]))))
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- armazém fixo por instância (key)
  }, []);

  // Só simples e semi-produzidos têm stock próprio para servir de ingrediente.
  const candidatos = produtos
    .filter((p) => p.Ativo && p.TipoProduto !== "composto" && p.ID !== ficha.Produto.ID)
    .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"));
  const produtoDe = (id: number | "") => produtos.find((p) => p.ID === id);

  const custoLinha = (l: LinhaReceita) => {
    const q = lerNumero(l.quantidade);
    const perda = l.perda.trim() === "" ? 0 : lerNumero(l.perda);
    if (l.ingredienteID === "" || Number.isNaN(q) || Number.isNaN(perda)) return 0;
    return consumoReal(q, perda) * (custos[l.ingredienteID] ?? 0);
  };
  const custoTotal = linhas.reduce((s, l) => s + custoLinha(l), 0);
  const composto = ficha.Produto.TipoProduto === "composto";
  const margem = ficha.PrecoVenda > 0 ? ((ficha.PrecoVenda - custoTotal) / ficha.PrecoVenda) * 100 : null;

  const linhaValida = (l: LinhaReceita) => {
    const q = lerNumero(l.quantidade);
    const perda = l.perda.trim() === "" ? 0 : lerNumero(l.perda);
    return l.ingredienteID !== "" && q > 0 && perda >= 0 && perda < 100;
  };
  const repetidos = new Set(
    linhas.map((l) => l.ingredienteID).filter((id, i, arr) => id !== "" && arr.indexOf(id) !== i),
  );
  const podeGuardar = linhas.length > 0 && linhas.every(linhaValida) && repetidos.size === 0 && !aGuardar;

  function atualizar(chave: number, alteracoes: Partial<LinhaReceita>) {
    setLinhas((ls) => ls.map((l) => (l.chave === chave ? { ...l, ...alteracoes } : l)));
  }

  function guardar() {
    if (!podeGuardar) return;
    const ingredientes = linhas.map((l) => {
      const i = new usecase.IngredienteInput();
      i.ProdutoIngredienteID = l.ingredienteID as number;
      i.Quantidade = lerNumero(l.quantidade);
      i.PerdaPercentual = l.perda.trim() === "" ? 0 : lerNumero(l.perda);
      return i;
    });
    setAGuardar(true);
    DefinirReceita(ficha.Produto.ID, ingredientes)
      .then(() => onGuardada())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <>
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>{ficha.Produto.Nome}</h2>
          <div style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
            {ROTULO_TIPO[ficha.Produto.TipoProduto]} · receita para 1 {ficha.Produto.UnidadeBase}
          </div>
        </div>
        {seletorArmazem}
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Ingrediente</th>
            <th style={{ textAlign: "right" }}>Quantidade</th>
            <th style={{ textAlign: "right" }}>Perda %</th>
            <th style={{ textAlign: "right" }}>Sai do stock</th>
            <th style={{ textAlign: "right" }}>Custo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {linhas.map((l) => {
            const ingrediente = produtoDe(l.ingredienteID);
            const q = lerNumero(l.quantidade);
            const perda = l.perda.trim() === "" ? 0 : lerNumero(l.perda);
            return (
              <tr key={l.chave}>
                <td>
                  <select
                    value={l.ingredienteID}
                    onChange={(e) => atualizar(l.chave, { ingredienteID: e.target.value ? Number(e.target.value) : "" })}
                  >
                    <option value="">Escolha…</option>
                    {candidatos.map((p) => (
                      <option key={p.ID} value={p.ID}>
                        {p.Nome} ({p.UnidadeBase})
                      </option>
                    ))}
                  </select>
                  {repetidos.has(l.ingredienteID) && <div className="admin-inventario-dif falta">ingrediente repetido</div>}
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  <input
                    className="admin-credito-qtd"
                    inputMode="decimal"
                    value={l.quantidade}
                    onChange={(e) => atualizar(l.chave, { quantidade: e.target.value })}
                  />{" "}
                  {ingrediente?.UnidadeBase}
                </td>
                <td style={{ textAlign: "right" }}>
                  <input
                    className="admin-credito-qtd"
                    inputMode="decimal"
                    placeholder="0"
                    value={l.perda}
                    onChange={(e) => atualizar(l.chave, { perda: e.target.value })}
                  />
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  {q > 0 && !Number.isNaN(perda) ? `${num(consumoReal(q, perda))} ${ingrediente?.UnidadeBase ?? ""}` : "—"}
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  {l.ingredienteID !== "" && custos[l.ingredienteID] == null ? (
                    <span style={{ color: "var(--text-secondary)" }}>sem custo</span>
                  ) : (
                    mt(custoLinha(l))
                  )}
                </td>
                <td>
                  <button
                    className="admin-btn-secundario admin-credito-remover"
                    aria-label="Remover ingrediente"
                    disabled={linhas.length === 1}
                    onClick={() => setLinhas((ls) => ls.filter((x) => x.chave !== l.chave))}
                  >
                    <TrashIcon className="admin-btn-icon-inline" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
      <div className="admin-form">
        <button
          className="admin-btn-secundario"
          onClick={() => setLinhas((ls) => [...ls, { chave: ++proximaChave, ingredienteID: "", quantidade: "", perda: "" }])}
        >
          <PlusIcon className="btn-icone" aria-hidden="true" />
          Adicionar ingrediente
        </button>
      </div>
      <p style={{ margin: 0, color: "var(--text-secondary)", fontSize: "0.85rem" }}>
        A quantidade é na unidade base do ingrediente. A perda (ex: o que fica no fundo da garrafa) aumenta o que sai do
        stock. "Sem custo" quer dizer que o ingrediente ainda não teve entradas neste armazém.
      </p>

      <div className="admin-venda-totais" style={{ marginLeft: 0, maxWidth: 360 }}>
        <div className="total">
          <span>Custo teórico</span>
          <span>{mt(custoTotal)}</span>
        </div>
        {composto && (
          <>
            <div>
              <span>Preço de venda</span>
              <span>{mt(ficha.PrecoVenda)}</span>
            </div>
            <div className={margem != null && margem < 0 ? "falta" : undefined}>
              <span>Margem</span>
              <span>{margem != null ? `${mt(ficha.PrecoVenda - custoTotal)} (${pct(margem)})` : "—"}</span>
            </div>
          </>
        )}
      </div>

      <div className="admin-form">
        <button className="admin-btn" disabled={!podeGuardar} onClick={guardar}>
          <CheckIcon className="btn-icone" aria-hidden="true" />
          {aGuardar ? "A guardar…" : "Guardar receita"}
        </button>
        <button className="admin-btn-secundario" onClick={onGuardada}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
      </div>
    </>
  );
}

function Produzir({
  ficha,
  armazemID,
  onProduzido,
  onCancelar,
  onErro,
}: {
  ficha: admin.FichaResumo;
  armazemID: number;
  onProduzido: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [quantidade, setQuantidade] = useState("1");
  const [aGuardar, setAGuardar] = useState(false);
  const q = lerNumero(quantidade);
  const valido = q > 0 && q <= ficha.Disponivel + 1e-9;

  function produzir() {
    if (!valido || aGuardar) return;
    const p = new usecase.ProducaoParams();
    p.ProdutoSemiElaboradoID = ficha.Produto.ID;
    p.ArmazemID = armazemID;
    p.Quantidade = q;
    setAGuardar(true);
    RegistarProducao(p)
      .then(() => onProduzido())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      <p style={{ margin: 0 }}>
        Consome os ingredientes da receita e cria stock de {ficha.Produto.Nome} neste armazém. Com o stock atual dá para
        produzir até <strong>{num(ficha.Disponivel)} {ficha.Produto.UnidadeBase}</strong>.
      </p>
      <label className="admin-campo" style={{ maxWidth: 220 }}>
        Quantidade a produzir ({ficha.Produto.UnidadeBase})
        <input inputMode="decimal" value={quantidade} onChange={(e) => setQuantidade(e.target.value)} autoFocus />
      </label>
      {!valido && quantidade.trim() !== "" && (
        <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
          A quantidade tem de ser maior que zero e no máximo {num(ficha.Disponivel)}.
        </p>
      )}
      <div className="admin-form">
        <button className="admin-btn" disabled={!valido || aGuardar} onClick={produzir}>
          <BoltIcon className="btn-icone" aria-hidden="true" />
          {aGuardar ? "A produzir…" : "Produzir"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
      </div>
    </div>
  );
}

function Duplicar({
  origem,
  fichas,
  onDuplicada,
  onCancelar,
  onErro,
}: {
  origem: admin.FichaResumo;
  fichas: admin.FichaResumo[];
  onDuplicada: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [destinoID, setDestinoID] = useState<number | "">("");
  const [aGuardar, setAGuardar] = useState(false);
  const destinos = fichas.filter((f) => f.Produto.ID !== origem.Produto.ID);
  const destino = destinos.find((f) => f.Produto.ID === destinoID);

  function duplicar() {
    if (destinoID === "" || aGuardar) return;
    setAGuardar(true);
    DuplicarReceita(origem.Produto.ID, destinoID)
      .then(() => onDuplicada())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      <p style={{ margin: 0 }}>
        Copia os {origem.Ingredientes.length} ingrediente(s) de {origem.Produto.Nome} para outro produto — útil para
        variantes (ex: a mesma bebida com outra base). Depois pode ajustar a cópia.
      </p>
      <label className="admin-campo">
        Copiar para
        <select value={destinoID} onChange={(e) => setDestinoID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Escolha o produto…</option>
          {destinos.map((f) => (
            <option key={f.Produto.ID} value={f.Produto.ID}>
              {f.Produto.Nome}
              {f.Ingredientes.length > 0 ? " (já tem receita)" : ""}
            </option>
          ))}
        </select>
      </label>
      {destino && destino.Ingredientes.length > 0 && (
        <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
          A receita atual de {destino.Produto.Nome} vai ser substituída.
        </p>
      )}
      <div className="admin-form">
        <button className="admin-btn" disabled={destinoID === "" || aGuardar} onClick={duplicar}>
          <DocumentDuplicateIcon className="btn-icone" aria-hidden="true" />
          {aGuardar ? "A copiar…" : "Duplicar receita"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
      </div>
    </div>
  );
}
