import { useEffect, useState } from "react";
import {
  ListarArmazens,
  ListarProdutos,
  ListarInventarios,
  IniciarInventario,
  ListarInventarioItens,
  RegistarContagemInventario,
  FecharInventario,
} from "../wailsjs/go/admin/AdminApp";
import { domain } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";
import { num } from "./formato";

const dataHora = (d: unknown) => new Date(String(d)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" });

type Vista = { tipo: "lista" } | { tipo: "contagem"; inventario: domain.Inventario };

/**
 * Inventário por armazém: iniciar tira uma fotografia do stock de todos os
 * produtos com controlo de stock; conta-se produto a produto (cada valor
 * grava logo); fechar gera os ajustes. Só pode haver um inventário aberto
 * por armazém, e um inventário fechado fica só de leitura.
 */
export default function InventarioScreen({ onErro }: { onErro: (e: string) => void }) {
  const [vista, setVista] = useState<Vista>({ tipo: "lista" });
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);

  useEffect(() => {
    ListarArmazens()
      .then(setArmazens)
      .catch((e) => onErro(String(e)));
    ListarProdutos()
      .then(setProdutos)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const nomeArmazem = (id: number) => armazens.find((a) => a.ID === id)?.Nome ?? `#${id}`;
  const voltarALista = () => setVista({ tipo: "lista" });

  if (vista.tipo === "contagem") {
    return (
      <div className="admin-lojas-screen">
        <Breadcrumb
          itens={[
            { rotulo: "Inventário", onClick: voltarALista },
            { rotulo: `#${vista.inventario.ID} — ${nomeArmazem(vista.inventario.ArmazemID)}` },
          ]}
        />
        <Contagem
          inventario={vista.inventario}
          nomeArmazem={nomeArmazem(vista.inventario.ArmazemID)}
          produtos={produtos}
          onFechado={(inv) => setVista({ tipo: "contagem", inventario: inv })}
          onErro={onErro}
        />
      </div>
    );
  }

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Inventário" }]} />
      <ListaInventarios
        armazens={armazens}
        nomeArmazem={nomeArmazem}
        onAbrir={(inventario) => setVista({ tipo: "contagem", inventario })}
        onErro={onErro}
      />
    </div>
  );
}

function ListaInventarios({
  armazens,
  nomeArmazem,
  onAbrir,
  onErro,
}: {
  armazens: domain.Armazem[];
  nomeArmazem: (id: number) => string;
  onAbrir: (inv: domain.Inventario) => void;
  onErro: (e: string) => void;
}) {
  const [inventarios, setInventarios] = useState<domain.Inventario[] | null>(null);
  const [armazemEscolhido, setArmazemEscolhido] = useState<number | "">("");
  const [aIniciar, setAIniciar] = useState(false);

  useEffect(() => {
    ListarInventarios()
      .then(setInventarios)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const armazensAtivos = armazens.filter((a) => a.Ativo);
  const armazemID: number | "" =
    armazemEscolhido !== "" ? armazemEscolhido : armazensAtivos.length > 0 ? armazensAtivos[0].ID : "";
  const abertoNoArmazem = inventarios?.find((i) => i.ArmazemID === armazemID && i.Status === "aberto");

  function iniciar() {
    if (armazemID === "" || aIniciar) return;
    setAIniciar(true);
    IniciarInventario(armazemID)
      .then((r) => onAbrir(r.Inventario))
      .catch((e) => onErro(String(e)))
      .finally(() => setAIniciar(false));
  }

  return (
    <>
      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Inventário</h2>
        <div className="admin-form">
          <label className="admin-stock-armazem-seletor">
            Armazém:{" "}
            <select
              value={armazemID}
              onChange={(e) => setArmazemEscolhido(e.target.value ? Number(e.target.value) : "")}
            >
              {armazensAtivos.map((a) => (
                <option key={a.ID} value={a.ID}>
                  {a.Nome}
                </option>
              ))}
            </select>
          </label>
          {abertoNoArmazem ? (
            <button className="admin-btn" onClick={() => onAbrir(abertoNoArmazem)}>
              Continuar inventário em curso
            </button>
          ) : (
            <button className="admin-btn" disabled={armazemID === "" || aIniciar} onClick={iniciar}>
              {aIniciar ? "A iniciar…" : "+ Iniciar inventário"}
            </button>
          )}
        </div>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>N.º</th>
            <th>Data</th>
            <th>Armazém</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {inventarios === null && (
            <tr>
              <td colSpan={5}>A carregar…</td>
            </tr>
          )}
          {inventarios?.length === 0 && (
            <tr>
              <td colSpan={5}>Ainda não foi feito nenhum inventário.</td>
            </tr>
          )}
          {inventarios?.map((inv) => (
            <tr key={inv.ID}>
              <td>#{inv.ID}</td>
              <td>{dataHora(inv.Data)}</td>
              <td>{nomeArmazem(inv.ArmazemID)}</td>
              <td>
                <span className={`admin-inventario-estado ${inv.Status}`}>
                  {inv.Status === "aberto" ? "Em curso" : "Fechado"}
                </span>
              </td>
              <td className="admin-form">
                <button className="admin-btn-secundario" onClick={() => onAbrir(inv)}>
                  {inv.Status === "aberto" ? "Continuar" : "Ver"}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}

type Filtro = "todos" | "por_contar" | "com_diferenca";

function Contagem({
  inventario,
  nomeArmazem,
  produtos,
  onFechado,
  onErro,
}: {
  inventario: domain.Inventario;
  nomeArmazem: string;
  produtos: domain.Produto[];
  onFechado: (inv: domain.Inventario) => void;
  onErro: (e: string) => void;
}) {
  const [itens, setItens] = useState<domain.InventarioItem[] | null>(null);
  const [filtro, setFiltro] = useState<Filtro>("todos");
  const [pesquisa, setPesquisa] = useState("");
  const [confirmarFecho, setConfirmarFecho] = useState(false);
  const [aFechar, setAFechar] = useState(false);
  const [ajustesGerados, setAjustesGerados] = useState<number | null>(null);

  useEffect(() => {
    ListarInventarioItens(inventario.ID)
      .then(setItens)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- inventário fixo por instância
  }, []);

  const aberto = inventario.Status === "aberto";
  const produtoDe = (id: number) => produtos.find((p) => p.ID === id);

  const lista = itens ?? [];
  const contados = lista.filter((i) => i.QuantidadeContada != null).length;
  const comDiferenca = lista.filter((i) => i.Diferenca != null && i.Diferenca !== 0).length;
  const porContar = lista.length - contados;

  const termo = pesquisa.trim().toLowerCase();
  const visiveis = lista
    .filter((i) => {
      if (filtro === "por_contar") return i.QuantidadeContada == null;
      if (filtro === "com_diferenca") return i.Diferenca != null && i.Diferenca !== 0;
      return true;
    })
    .filter((i) => !termo || (produtoDe(i.ProdutoID)?.Nome ?? "").toLowerCase().includes(termo))
    .sort((a, b) => (produtoDe(a.ProdutoID)?.Nome ?? "").localeCompare(produtoDe(b.ProdutoID)?.Nome ?? "", "pt"));

  function atualizarItem(novo: domain.InventarioItem) {
    setItens((atual) => (atual ?? []).map((i) => (i.ID === novo.ID ? novo : i)));
  }

  function fechar() {
    setAFechar(true);
    FecharInventario(inventario.ID)
      .then((movimentos) => {
        setAjustesGerados(movimentos.length);
        setConfirmarFecho(false);
        onFechado({ ...inventario, Status: "fechado" } as domain.Inventario);
        return ListarInventarioItens(inventario.ID).then(setItens);
      })
      .catch((e) => {
        setConfirmarFecho(false);
        onErro(String(e));
      })
      .finally(() => setAFechar(false));
  }

  return (
    <>
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>
            Inventário #{inventario.ID} — {nomeArmazem}
          </h2>
          <div style={{ color: "var(--text-secondary)", fontSize: "0.85rem" }}>
            Iniciado a {dataHora(inventario.Data)} · {aberto ? "em curso" : "fechado"}
          </div>
        </div>
        {aberto && (
          <button className="admin-btn" disabled={itens === null || contados === 0} onClick={() => setConfirmarFecho(true)}>
            Fechar inventário
          </button>
        )}
      </div>

      {ajustesGerados !== null && (
        <div className="admin-inventario-aviso ok">
          Inventário fechado — {ajustesGerados === 0 ? "sem diferenças a ajustar" : `${ajustesGerados} movimento(s) de ajuste gerado(s)`}.
        </div>
      )}

      <div className="admin-inventario-resumo">
        <span>
          <strong>{contados}</strong> de {lista.length} contados
        </span>
        <span>
          <strong>{comDiferenca}</strong> com diferença
        </span>
        {aberto && (
          <span>
            <strong>{porContar}</strong> por contar
          </span>
        )}
      </div>

      <div className="admin-form">
        <input placeholder="Pesquisar produto…" value={pesquisa} onChange={(e) => setPesquisa(e.target.value)} />
        <select value={filtro} onChange={(e) => setFiltro(e.target.value as Filtro)}>
          <option value="todos">Todos</option>
          <option value="por_contar">Por contar</option>
          <option value="com_diferenca">Com diferença</option>
        </select>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th>No sistema</th>
            <th>Contado</th>
            <th>Diferença</th>
          </tr>
        </thead>
        <tbody>
          {itens === null && (
            <tr>
              <td colSpan={4}>A carregar…</td>
            </tr>
          )}
          {itens !== null && visiveis.length === 0 && (
            <tr>
              <td colSpan={4}>Nenhum produto neste filtro.</td>
            </tr>
          )}
          {visiveis.map((item) => {
            const produto = produtoDe(item.ProdutoID);
            return (
              <tr key={item.ID}>
                <td>
                  {produto?.Nome ?? `#${item.ProdutoID}`}
                  {produto?.LoteControlado && <span className="admin-inventario-tag">lotes</span>}
                </td>
                <td>
                  {num(item.QuantidadeSistema)} {produto?.UnidadeBase}
                </td>
                <td>
                  {aberto ? (
                    <CampoContagem item={item} onGuardado={atualizarItem} onErro={onErro} />
                  ) : item.QuantidadeContada != null ? (
                    num(item.QuantidadeContada)
                  ) : (
                    <span style={{ color: "var(--text-secondary)" }}>não contado</span>
                  )}
                </td>
                <td>
                  <Diferenca valor={item.Diferenca} />
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      {confirmarFecho && (
        <Modal titulo={`Fechar inventário #${inventario.ID}`} onFechar={() => setConfirmarFecho(false)}>
          <div className="admin-armazem-form">
            <p style={{ margin: 0 }}>
              Vão ser gerados ajustes de stock para os <strong>{comDiferenca}</strong> produto(s) com diferença. Depois de
              fechado, o inventário já não pode ser alterado.
            </p>
            {porContar > 0 && (
              <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
                {porContar} produto(s) ficaram por contar — serão ignorados e o stock deles não muda.
              </p>
            )}
            <div className="admin-form">
              <button className="admin-btn" disabled={aFechar} onClick={fechar}>
                {aFechar ? "A fechar…" : "Fechar e ajustar stock"}
              </button>
              <button className="admin-btn-secundario" onClick={() => setConfirmarFecho(false)}>
                Voltar à contagem
              </button>
            </div>
          </div>
        </Modal>
      )}
    </>
  );
}

/**
 * Grava ao sair do campo (ou Enter), só se o valor mudou — contar um
 * armazém inteiro é muitas linhas seguidas, um botão por linha seria lento.
 */
function CampoContagem({
  item,
  onGuardado,
  onErro,
}: {
  item: domain.InventarioItem;
  onGuardado: (i: domain.InventarioItem) => void;
  onErro: (e: string) => void;
}) {
  const inicial = item.QuantidadeContada != null ? String(item.QuantidadeContada) : "";
  const [valor, setValor] = useState(inicial);
  const [aGuardar, setAGuardar] = useState(false);

  function guardar() {
    if (valor === inicial || valor.trim() === "") return;
    const quantidade = Number(valor.replace(",", "."));
    if (Number.isNaN(quantidade) || quantidade < 0) {
      onErro("Quantidade contada inválida");
      return;
    }
    setAGuardar(true);
    RegistarContagemInventario(item.ID, quantidade)
      .then(onGuardado)
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <input
      className="admin-inventario-contagem"
      inputMode="decimal"
      placeholder="—"
      value={valor}
      disabled={aGuardar}
      onChange={(e) => setValor(e.target.value)}
      onBlur={guardar}
      onKeyDown={(e) => {
        if (e.key === "Enter") (e.target as HTMLInputElement).blur();
      }}
    />
  );
}

function Diferenca({ valor }: { valor?: number }) {
  if (valor == null) return <span style={{ color: "var(--text-secondary)" }}>—</span>;
  if (valor === 0) return <span className="admin-inventario-dif zero">0</span>;
  return (
    <span className={`admin-inventario-dif ${valor > 0 ? "sobra" : "falta"}`}>
      {valor > 0 ? "+" : ""}
      {num(valor)}
    </span>
  );
}
