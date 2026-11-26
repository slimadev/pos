import { useEffect, useMemo, useRef, useState } from "react";
import {
  PlusIcon,
  MinusIcon,
  ArrowLeftIcon,
  CheckCircleIcon,
  MagnifyingGlassIcon,
  EllipsisVerticalIcon,
  ClockIcon,
  PrinterIcon,
  PaperAirplaneIcon,
  UserGroupIcon,
  ArrowsRightLeftIcon,
  ScissorsIcon,
  RectangleGroupIcon,
  ExclamationTriangleIcon,
  ShieldCheckIcon,
  DocumentDuplicateIcon,
  ChevronUpIcon,
  TagIcon,
  DocumentPlusIcon,
} from "@heroicons/react/24/outline";
import {
  ListarMesas,
  ObterMesa,
  AbrirMesa,
  LancarItemMesa,
  EnviarPedido,
  AnularItemMesa,
  DividirMesa,
  TransferirMesa,
  JuntarMesas,
  DefinirPessoasMesa,
  NovaContaMesa,
  DefinirEtiquetaMesa,
  ImprimirContaMesa,
  ListarImpressoras,
  ListarPedidosMesa,
  ReimprimirPedido,
  ListarOperadoresAtivos,
  ListarArmazens,
  FecharMesa,
  EstadoCaixa,
  ListarClientes,
  DefinirClienteMesa,
} from "../../wailsjs/go/api/App";
import { domain, usecase } from "../../wailsjs/go/models";
import Catalogo from "./Catalogo";
import { FechoConta, ContaFechada, Folha } from "./Bottlestore";
import { tempoAberta, textoErro } from "./comum";
import ReciboModal from "./ReciboModal";
import { mt, qtd } from "./formato";
import { ponte } from "../plataforma";
import "./Bottlestore.css";
import "./Mesas.css";

const EPS = 1e-9;
const porEnviar = (i: domain.ItemMesa) => Math.max(0, i.Quantidade - i.Enviada);
// "5" e "5 (2)" (segunda conta da mesa 5) levam "Mesa" à frente; "Varanda" não.
const rotuloMesa = (m: string) => (/^\d+( \(\d+\))?$/.test(m) ? `Mesa ${m}` : m);
const baseMesa = (m: string) => m.replace(/\s*\(\d+\)$/, "");
const mesmaMesa = (a: string, b: string) => baseMesa(a).toLowerCase() === baseMesa(b).toLowerCase();
const ordemMesas = (a: domain.MesaAberta, b: domain.MesaAberta) => a.Mesa.localeCompare(b.Mesa, "pt", { numeric: true, sensitivity: "base" });
const MAX_ETIQUETA = 30;
const CLIENTE_ANONIMO = 1; // domain.ClienteAnonimoID

/**
 * Mesas do restaurante. No balcão (`balcao`) vê-se tudo por garçom e
 * fecha-se a conta; no aparelho do garçom vêem-se as suas mesas, lança-se o
 * pedido, envia-se para a cozinha e imprime-se a conta para o cliente.
 * Os outros aparelhos mexem nas mesas ao mesmo tempo — a lista atualiza-se
 * sozinha.
 */
export default function Mesas({
  balcao,
  operador,
  onIrParaCaixa,
}: {
  balcao: boolean;
  operador: usecase.Operador;
  onIrParaCaixa?: () => void;
}) {
  const [mesas, setMesas] = useState<domain.MesaAberta[] | null>(null);
  const [garcom, setGarcom] = useState<number | "todos" | "meus">(balcao ? "todos" : "meus");
  const [selecionada, setSelecionada] = useState<number | null>(null);
  const [abrir, setAbrir] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [fechada, setFechada] = useState<{ id: number; identificador: string; total: number; troco: number; divida: number } | null>(
    null,
  );
  const [recibo, setRecibo] = useState<{ id: number; troco: number } | null>(null);
  const [, setRelogio] = useState(0);

  const recarregar = () => {
    ListarMesas(false)
      .then(setMesas)
      .catch((e) => setErro(textoErro(e)));
  };

  useEffect(() => {
    recarregar();
    // Outros aparelhos lançam e enviam ao mesmo tempo.
    const t = window.setInterval(() => {
      recarregar();
      setRelogio((n) => n + 1);
    }, 15000);
    window.addEventListener("focus", recarregar);
    return () => {
      window.clearInterval(t);
      window.removeEventListener("focus", recarregar);
    };
  }, []);

  const sobreposicoes = (
    <>
      {recibo && <ReciboModal vendaId={recibo.id} troco={recibo.troco} onFechar={() => setRecibo(null)} />}
      {fechada && !recibo && (
        <ContaFechada
          {...fechada}
          onRecibo={() => setRecibo({ id: fechada.id, troco: fechada.troco })}
          onFechar={() => setFechada(null)}
        />
      )}
    </>
  );

  if (selecionada !== null) {
    return (
      <>
        {sobreposicoes}
        <MesaDetalhe
          key={selecionada}
          contaID={selecionada}
          balcao={balcao}
          outrasMesas={(mesas ?? []).filter((m) => m.ContaID !== selecionada)}
          onAbrirOutra={setSelecionada}
          onIrParaCaixa={onIrParaCaixa}
          onVoltar={() => {
            setSelecionada(null);
            recarregar();
          }}
          onFechada={(r) => {
            setSelecionada(null);
            recarregar();
            setFechada(r);
          }}
        />
      </>
    );
  }

  const lista = mesas ?? [];
  const garcons = Array.from(new Map(lista.map((m) => [m.GarcomID, m.GarcomNome])).entries()).sort((a, b) =>
    a[1].localeCompare(b[1], "pt"),
  );
  const visiveis = lista
    .filter((m) => (garcom === "todos" ? true : garcom === "meus" ? m.GarcomID === operador.ID : m.GarcomID === garcom))
    .sort(ordemMesas);
  const total = visiveis.reduce((s, m) => s + m.Total, 0);

  return (
    <div className="bs mesas">
      {sobreposicoes}
      {erro && (
        <div className="bs-erro" role="alert">
          {erro}
          <button onClick={() => setErro(null)} aria-label="Fechar">
            ×
          </button>
        </div>
      )}

      <div className="bs-topo">
        <div className="bs-kpis">
          <div>
            <span className="bs-kpi-valor">{visiveis.length}</span>
            <span className="bs-kpi-label">{visiveis.length === 1 ? "mesa aberta" : "mesas abertas"}</span>
          </div>
          <div>
            <span className="bs-kpi-valor">{mt(total)}</span>
            <span className="bs-kpi-label">em consumo</span>
          </div>
        </div>
        <button className="bs-abrir" onClick={() => setAbrir(true)}>
          <PlusIcon className="bs-icon" aria-hidden="true" />
          Abrir mesa
        </button>
      </div>

      {/* No balcão escolhe-se o garçom; no aparelho, as minhas ou todas. */}
      <div className="mesas-garcons" role="tablist" aria-label="Garçom">
        {!balcao && (
          <button role="tab" aria-selected={garcom === "meus"} className={garcom === "meus" ? "ativo" : ""} onClick={() => setGarcom("meus")}>
            As minhas
            <span className="mesas-n">{lista.filter((m) => m.GarcomID === operador.ID).length}</span>
          </button>
        )}
        <button role="tab" aria-selected={garcom === "todos"} className={garcom === "todos" ? "ativo" : ""} onClick={() => setGarcom("todos")}>
          Todas
          <span className="mesas-n">{lista.length}</span>
        </button>
        {balcao &&
          garcons.map(([id, nome]) => (
            <button key={id} role="tab" aria-selected={garcom === id} className={garcom === id ? "ativo" : ""} onClick={() => setGarcom(id)}>
              {nome}
              <span className="mesas-n">{lista.filter((m) => m.GarcomID === id).length}</span>
            </button>
          ))}
      </div>

      <div className="bs-grid">
        {visiveis.map((m) => {
          const tempo = tempoAberta(m.AbertaEm);
          return (
            <button key={m.ContaID} className={`bs-card${tempo.longa ? " longa" : ""}`} onClick={() => setSelecionada(m.ContaID)}>
              <div className="bs-card-cabeca">
                <span className="bs-card-id">{rotuloMesa(m.Mesa)}</span>
                {tempo.texto && (
                  <span className="bs-card-tempo">
                    <ClockIcon className="bs-icon-mini" aria-hidden="true" />
                    {tempo.texto}
                  </span>
                )}
              </div>
              {m.Etiqueta && (
                <div className="mesas-etiqueta">
                  <TagIcon className="bs-icon-mini" aria-hidden="true" />
                  {m.Etiqueta}
                </div>
              )}
              <div className="bs-card-cliente">
                {m.GarcomNome} · {m.Pessoas} {m.Pessoas === 1 ? "pessoa" : "pessoas"}
              </div>
              <div className="bs-card-bottom">
                <span className="bs-card-total">{mt(m.Total)}</span>
                {m.PorEnviar > EPS && <span className="mesas-por-enviar">{qtd(m.PorEnviar)} por enviar</span>}
              </div>
            </button>
          );
        })}
        {mesas !== null && visiveis.length === 0 && (
          <div className="bs-vazio">
            <RectangleGroupIcon className="bs-vazio-icone" aria-hidden="true" />
            <strong>{garcom === "meus" ? "Não tem mesas abertas" : "Nenhuma mesa aberta"}</strong>
            <span>Abra a mesa, lance o pedido e carregue em Enviar: sai logo impresso no balcão e na cozinha.</span>
            <button className="bs-abrir" onClick={() => setAbrir(true)}>
              <PlusIcon className="bs-icon" aria-hidden="true" />
              Abrir mesa
            </button>
          </div>
        )}
      </div>

      {abrir && (
        <AbrirMesaFolha
          abertas={lista}
          onCancelar={() => setAbrir(false)}
          onAberta={(m) => {
            setAbrir(false);
            recarregar();
            setSelecionada(m.ContaID);
          }}
        />
      )}
    </div>
  );
}

// --- Abrir mesa ---------------------------------------------------------------

/**
 * Abrir mesa. Uma mesa já aberta não se abre outra vez: escolhê-la abre mais
 * uma conta nela ("Mesa 5 (2)"), para quem quer pagar à parte desde o início.
 */
function AbrirMesaFolha({
  abertas,
  onCancelar,
  onAberta,
}: {
  abertas: domain.MesaAberta[];
  onCancelar: () => void;
  onAberta: (m: domain.MesaAberta) => void;
}) {
  const [mesa, setMesa] = useState("");
  const [etiqueta, setEtiqueta] = useState("");
  const [pessoas, setPessoas] = useState(2);
  const [erro, setErro] = useState<string | null>(null);
  const [aAbrir, setAAbrir] = useState(false);
  const numeros = Array.from({ length: 20 }, (_, i) => String(i + 1));
  const nome = mesa.trim();
  // A primeira conta da mesa (a "5", não a "5 (2)"), se já estiver aberta.
  const jaAberta = nome ? abertas.find((m) => m.Mesa.toLowerCase() === nome.toLowerCase().replace(/\s+/g, " ")) : undefined;
  const contasDaMesa = nome ? abertas.filter((m) => mesmaMesa(m.Mesa, nome)).length : 0;

  function abrir() {
    if (!nome || aAbrir) return;
    setAAbrir(true);
    setErro(null);
    (jaAberta ? NovaContaMesa(jaAberta.ContaID, etiqueta) : AbrirMesa(nome, etiqueta, pessoas))
      .then(onAberta)
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setAAbrir(false));
  }

  return (
    <Folha titulo="Abrir mesa" onFechar={onCancelar}>
      <div className="bs-rapidos mesas-numeros">
        {numeros.map((n) => {
          const ocupada = abertas.some((m) => m.Mesa === n);
          return (
            <button
              key={n}
              disabled={aAbrir}
              className={`${mesa === n ? "ativo" : ""}${ocupada ? " ocupada" : ""}`}
              title={ocupada ? "Já aberta — escolher abre mais uma conta nesta mesa" : undefined}
              onClick={() => setMesa(n)}
            >
              {n}
            </button>
          );
        })}
      </div>
      <label className="bs-campo">
        <span>Ou escreva (ex: Varanda 2)</span>
        <input value={mesa} onChange={(e) => setMesa(e.target.value)} onKeyDown={(e) => e.key === "Enter" && abrir()} />
      </label>
      {jaAberta && (
        <div className="bs-aviso">
          {rotuloMesa(jaAberta.Mesa)} já está aberta ({jaAberta.GarcomNome}
          {contasDaMesa > 1 ? `, ${contasDaMesa} contas` : ""}). Abre-se mais uma conta nela, que se paga à parte.
        </div>
      )}
      <CampoEtiqueta valor={etiqueta} onMudar={setEtiqueta} onEnter={abrir} />
      {!jaAberta && <Pessoas valor={pessoas} onMudar={setPessoas} />}
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn" disabled={!nome || aAbrir} onClick={abrir}>
          {aAbrir ? "A abrir…" : jaAberta ? "Abrir conta separada" : `Abrir ${nome ? rotuloMesa(nome) : "mesa"}`}
        </button>
      </div>
    </Folha>
  );
}

/** Etiqueta livre do garçom para não confundir as contas ("camisa azul"). */
function CampoEtiqueta({ valor, onMudar, onEnter }: { valor: string; onMudar: (v: string) => void; onEnter?: () => void }) {
  return (
    <label className="bs-campo">
      <span>Etiqueta para não confundir (opcional — ex: camisa azul, aniversariante)</span>
      <input
        value={valor}
        maxLength={MAX_ETIQUETA}
        onChange={(e) => onMudar(e.target.value)}
        onKeyDown={(e) => e.key === "Enter" && onEnter?.()}
      />
    </label>
  );
}

function Pessoas({ valor, onMudar }: { valor: number; onMudar: (n: number) => void }) {
  return (
    <div className="mesas-pessoas">
      <span>
        <UserGroupIcon className="bs-icon" aria-hidden="true" /> Pessoas
      </span>
      <div className="bs-stepper">
        <button onClick={() => onMudar(Math.max(1, valor - 1))} aria-label="Menos uma pessoa">
          <MinusIcon className="bs-icon" aria-hidden="true" />
        </button>
        <span>{valor}</span>
        <button onClick={() => onMudar(valor + 1)} aria-label="Mais uma pessoa">
          <PlusIcon className="bs-icon" aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}

// --- Mesa ----------------------------------------------------------------------

type FolhaAberta =
  | { tipo: "enviado"; res: usecase.ResultadoPedido }
  | { tipo: "conta" }
  | { tipo: "anular"; item: domain.ItemMesa }
  | { tipo: "dividir" }
  | { tipo: "transferir" }
  | { tipo: "juntar" }
  | { tipo: "pedidos" }
  | { tipo: "pessoas" }
  | { tipo: "nova-conta" }
  | { tipo: "etiqueta" }
  | { tipo: "fechar"; armazemID: number }
  | { tipo: "caixa-fechado" };

function MesaDetalhe({
  contaID,
  balcao,
  outrasMesas,
  onAbrirOutra,
  onVoltar,
  onFechada,
  onIrParaCaixa,
}: {
  contaID: number;
  balcao: boolean;
  outrasMesas: domain.MesaAberta[];
  onAbrirOutra: (contaID: number) => void;
  onVoltar: () => void;
  onFechada: (r: { id: number; identificador: string; total: number; troco: number; divida: number }) => void;
  onIrParaCaixa?: () => void;
}) {
  const [detalhe, setDetalhe] = useState<usecase.MesaDetalhe | null>(null);
  const [armazemID, setArmazemID] = useState<number | null>(null);
  const [folha, setFolha] = useState<FolhaAberta | null>(null);
  const [menu, setMenu] = useState(false);
  const [comandaAberta, setComandaAberta] = useState(false);
  const [aEnviar, setAEnviar] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [ultimo, setUltimo] = useState<{ nome: string; n: number } | null>(null);
  const [clientes, setClientes] = useState<domain.Cliente[]>([]);
  const toastRef = useRef<number | undefined>(undefined);

  const recarregar = () =>
    ObterMesa(contaID)
      .then(setDetalhe)
      .catch((e) => {
        setErro(textoErro(e));
        // Fechada ou junta noutro aparelho: volta à lista.
        if (String(e).includes("não encontrada")) onVoltar();
      });

  useEffect(() => {
    recarregar();
    ListarArmazens().then((lista) => {
      const ativos = lista.filter((a) => a.Ativo && !a.ForaDeVenda);
      const inicial = ativos.find((a) => a.Tipo === "principal") ?? ativos[0];
      if (inicial) setArmazemID(inicial.ID);
    });
    const t = window.setInterval(recarregar, 15000);
    return () => {
      window.clearInterval(t);
      window.clearTimeout(toastRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só ao abrir a mesa
  }, []);

  const itens = useMemo(() => detalhe?.Itens ?? [], [detalhe]);
  const mesa = detalhe?.Mesa;
  const total = itens.reduce((s, i) => s + i.Subtotal, 0);
  const pendente = itens.reduce((s, i) => s + porEnviar(i), 0);

  function adicionar(item: usecase.ItemCatalogo, unidade: domain.UnidadeProduto) {
    lancar(item.Produto.ID, unidade.ID, item.Produto.Nome);
  }

  function lancar(produtoID: number, unidadeID: number, nome?: string) {
    const p = new usecase.AdicionarItemContaParams();
    p.ContaID = contaID;
    p.ProdutoID = produtoID;
    p.UnidadeProdutoID = unidadeID;
    p.Quantidade = 1;
    LancarItemMesa(p)
      .then(() => {
        recarregar();
        if (nome) {
          setUltimo((u) => ({ nome, n: u?.nome === nome ? u.n + 1 : 1 }));
          window.clearTimeout(toastRef.current);
          toastRef.current = window.setTimeout(() => setUltimo(null), 1800);
        }
      })
      .catch((e) => setErro(textoErro(e)));
  }

  function menos(i: domain.ItemMesa) {
    // O que ainda não foi enviado sai logo; o enviado pede o gerente.
    if (porEnviar(i) > EPS) {
      anular(i, 1, "", "");
    } else {
      setFolha({ tipo: "anular", item: i });
    }
  }

  function anular(i: domain.ItemMesa, quantidade: number, pin: string, motivo: string) {
    const p = new usecase.AnularItemParams();
    p.ItemID = i.ID;
    p.Quantidade = quantidade;
    p.PINGerente = pin;
    p.Motivo = motivo;
    return AnularItemMesa(p)
      .then((r) => {
        if (r.PrecisaAutorizacao) {
          setFolha({ tipo: "anular", item: i });
          return;
        }
        setFolha(r.Anulacao ? { tipo: "enviado", res: r.Anulacao } : null);
        recarregar();
      })
      .catch((e) => {
        setErro(textoErro(e));
        throw e;
      });
  }

  function enviar() {
    if (aEnviar) return;
    setAEnviar(true);
    EnviarPedido(contaID)
      .then((res) => {
        setFolha({ tipo: "enviado", res });
        recarregar();
      })
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setAEnviar(false));
  }

  function fechar() {
    ListarClientes()
      .then(setClientes)
      .catch(() => {});
    EstadoCaixa()
      .then((c) => {
        if (!c.Aberto) setFolha({ tipo: "caixa-fechado" });
        else if (armazemID !== null) setFolha({ tipo: "fechar", armazemID });
      })
      .catch((e) => setErro(textoErro(e)));
  }

  if (!mesa) {
    return (
      <div className="bs bs-detalhe">
        {erro && <div className="bs-erro">{erro}</div>}
      </div>
    );
  }
  const tempo = tempoAberta(mesa.AbertaEm);
  const titulo = rotuloMesa(mesa.Mesa);
  // As outras contas desta mesa, para saltar entre elas.
  const irmas = outrasMesas.filter((m) => mesmaMesa(m.Mesa, mesa.Mesa));

  const acoesComanda = (
    <>
      <button className="bs-fechar mesas-enviar" disabled={pendente <= EPS || aEnviar} onClick={enviar}>
        <PaperAirplaneIcon className="bs-icon" aria-hidden="true" />
        {aEnviar ? "A enviar…" : pendente > EPS ? `Enviar pedido (${qtd(pendente)})` : "Tudo enviado"}
      </button>
      <div className="mesas-acoes-2">
        <button className="bs-btn-sec" disabled={itens.length === 0} onClick={() => setFolha({ tipo: "conta" })}>
          <PrinterIcon className="bs-icon" aria-hidden="true" />
          Conta
        </button>
        {balcao && (
          <button className="bs-btn" disabled={itens.length === 0 || armazemID === null} onClick={fechar}>
            <CheckCircleIcon className="bs-icon" aria-hidden="true" />
            Fechar
          </button>
        )}
      </div>
    </>
  );

  return (
    <div className="bs bs-detalhe">
      <div className="bs-detalhe-header">
        <button className="bs-voltar" onClick={onVoltar} aria-label="Voltar às mesas">
          <ArrowLeftIcon className="bs-icon-grande" aria-hidden="true" />
        </button>
        <div className="bs-detalhe-titulo">
          <div className="bs-detalhe-nome">{titulo}</div>
          <div className="bs-detalhe-sub">
            {mesa.Etiqueta && (
              <button className="mesas-etiqueta" onClick={() => setFolha({ tipo: "etiqueta" })} title="Mudar a etiqueta">
                <TagIcon className="bs-icon-mini" aria-hidden="true" />
                {mesa.Etiqueta}
              </button>
            )}
            {mesa.GarcomNome} · {mesa.Pessoas} {mesa.Pessoas === 1 ? "pessoa" : "pessoas"}
            {tempo.texto && ` · há ${tempo.texto}`}
          </div>
        </div>
        <div className="bs-menu">
          <button className="bs-menu-botao" onClick={() => setMenu((v) => !v)} aria-label="Mais opções">
            <EllipsisVerticalIcon className="bs-icon-grande" aria-hidden="true" />
          </button>
          {menu && (
            <>
              <div className="bs-menu-fundo" onClick={() => setMenu(false)} />
              <div className="bs-menu-lista">
                {(
                  [
                    ["nova-conta", DocumentPlusIcon, "Nova conta nesta mesa"],
                    ["etiqueta", TagIcon, mesa.Etiqueta ? "Mudar a etiqueta" : "Pôr uma etiqueta"],
                    ["pessoas", UserGroupIcon, "Número de pessoas"],
                    ["dividir", ScissorsIcon, "Dividir a conta"],
                    ["transferir", ArrowsRightLeftIcon, "Passar a outro garçom"],
                    ["juntar", RectangleGroupIcon, "Juntar a outra mesa"],
                    ["pedidos", DocumentDuplicateIcon, "Pedidos enviados"],
                  ] as const
                ).map(([tipo, Icone, rotulo]) => (
                  <button
                    key={tipo}
                    className="bs-menu-item"
                    onClick={() => {
                      setMenu(false);
                      setFolha({ tipo });
                    }}
                  >
                    <Icone className="bs-icon" aria-hidden="true" />
                    {rotulo}
                  </button>
                ))}
              </div>
            </>
          )}
        </div>
      </div>

      {erro && (
        <div className="bs-erro" role="alert">
          {erro}
          <button onClick={() => setErro(null)} aria-label="Fechar">
            ×
          </button>
        </div>
      )}

      {irmas.length > 0 && (
        <div className="mesas-contas" role="tablist" aria-label="Contas desta mesa">
          {[mesa, ...irmas].sort(ordemMesas).map((m) => (
            <button
              key={m.ContaID}
              role="tab"
              aria-selected={m.ContaID === contaID}
              className={m.ContaID === contaID ? "ativo" : ""}
              onClick={() => m.ContaID !== contaID && onAbrirOutra(m.ContaID)}
            >
              <strong>{rotuloMesa(m.Mesa)}</strong>
              {m.Etiqueta && <span>{m.Etiqueta}</span>}
              <small>{m.ContaID === contaID ? mt(total) : mt(m.Total)}</small>
            </button>
          ))}
        </div>
      )}

      <div className="bs-detalhe-body">
        <div className="bs-catalogo">
          {armazemID !== null && (
            <Catalogo key={armazemID} armazemID={armazemID} IconeBusca={MagnifyingGlassIcon} onEscolher={adicionar} onErro={setErro} />
          )}
          {ultimo && (
            <div className="bs-toast" role="status">
              <CheckCircleIcon className="bs-icon" aria-hidden="true" />+{ultimo.n} {ultimo.nome}
            </div>
          )}
        </div>

        <div className="bs-resumo-movel">
          <button className="bs-resumo-ver" onClick={() => setComandaAberta(true)} disabled={itens.length === 0}>
            <ChevronUpIcon className="bs-icon" aria-hidden="true" />
            <span>
              <span className="bs-resumo-total">{mt(total)}</span>
              <span className="bs-resumo-qtd">
                {itens.length === 0 ? "Toque nos produtos para lançar" : pendente > EPS ? `${qtd(pendente)} por enviar · ver` : "ver o pedido"}
              </span>
            </span>
          </button>
          <button className="bs-fechar mesas-enviar" disabled={pendente <= EPS || aEnviar} onClick={enviar}>
            <PaperAirplaneIcon className="bs-icon" aria-hidden="true" />
            {aEnviar ? "A enviar…" : "Enviar"}
          </button>
        </div>

        <aside className={`bs-comanda${comandaAberta ? " aberta" : ""}`}>
          <div className="bs-comanda-header">
            <button className="bs-comanda-fechar-movel" onClick={() => setComandaAberta(false)} aria-label="Voltar aos produtos">
              <ArrowLeftIcon className="bs-icon-grande" aria-hidden="true" />
            </button>
            <span className="bs-comanda-titulo">Pedido · {titulo}</span>
          </div>
          <div className="bs-comanda-itens">
            {itens.map((i) => {
              const pend = porEnviar(i);
              return (
                <div key={i.ID} className="bs-linha">
                  <div className="bs-linha-topo">
                    <div className="bs-linha-nome">{i.Produto}</div>
                    <div className="bs-linha-total">{mt(i.Subtotal)}</div>
                  </div>
                  <div className="bs-linha-baixo">
                    <div className="bs-linha-preco">
                      {mt(i.PrecoUnitario)}
                      {i.Unidade && i.Unidade.toLowerCase() !== "unidade" ? ` · ${i.Unidade}` : ""}
                      {pend > EPS ? (
                        <span className="mesas-estado por-enviar">{i.Enviada > EPS ? `${qtd(pend)} por enviar` : "por enviar"}</span>
                      ) : (
                        <span className="mesas-estado enviado">enviado</span>
                      )}
                    </div>
                    <div className="bs-stepper">
                      <button onClick={() => menos(i)} aria-label={`Tirar uma ${i.Produto}`}>
                        <MinusIcon className="bs-icon" aria-hidden="true" />
                      </button>
                      <span>{qtd(i.Quantidade)}</span>
                      <button onClick={() => lancar(i.ProdutoID, i.UnidadeProdutoID)} aria-label={`Mais uma ${i.Produto}`}>
                        <PlusIcon className="bs-icon" aria-hidden="true" />
                      </button>
                    </div>
                  </div>
                </div>
              );
            })}
            {itens.length === 0 && <div className="bs-comanda-vazia">Mesa vazia — toque nos produtos para lançar o pedido.</div>}
          </div>
          <div className="bs-comanda-footer">
            <div className="bs-comanda-total">
              <span>Total</span>
              <span>{mt(total)}</span>
            </div>
            {mesa.Pessoas > 1 && total > 0 && (
              <div className="mesas-por-pessoa">
                {mesa.Pessoas} pessoas · {mt(total / mesa.Pessoas)} cada
              </div>
            )}
            {acoesComanda}
          </div>
        </aside>
      </div>

      {folha?.tipo === "enviado" && <ResultadoEnvio res={folha.res} onFechar={() => setFolha(null)} />}
      {folha?.tipo === "conta" && <ImprimirConta mesa={mesa} total={total} onFechar={() => setFolha(null)} onMudou={recarregar} />}
      {folha?.tipo === "anular" && (
        <AnularFolha
          item={folha.item}
          onCancelar={() => setFolha(null)}
          onAnular={(q, pin, motivo) => anular(folha.item, q, pin, motivo)}
        />
      )}
      {folha?.tipo === "dividir" && (
        <DividirFolha
          mesa={mesa}
          itens={itens}
          onCancelar={() => setFolha(null)}
          onDividida={(nova) => {
            setFolha(null);
            onAbrirOutra(nova.ContaID);
          }}
        />
      )}
      {folha?.tipo === "transferir" && (
        <TransferirFolha
          mesa={mesa}
          onCancelar={() => setFolha(null)}
          onFeito={(saiDeMim) => {
            setFolha(null);
            if (saiDeMim && !balcao) onVoltar();
            else recarregar();
          }}
        />
      )}
      {folha?.tipo === "juntar" && (
        <Folha titulo={`Juntar ${titulo} a…`} onFechar={() => setFolha(null)}>
          <p className="bs-folha-texto">Os itens passam para a outra mesa (o que já foi enviado continua enviado) e esta fecha.</p>
          <div className="bs-destinos">
            {outrasMesas.map((m) => (
              <button
                key={m.ContaID}
                onClick={() =>
                  JuntarMesas(contaID, m.ContaID)
                    .then(() => onAbrirOutra(m.ContaID))
                    .catch((e) => setErro(textoErro(e)))
                }
              >
                {rotuloMesa(m.Mesa)}
                {m.Etiqueta && ` (${m.Etiqueta})`} · {m.GarcomNome}
              </button>
            ))}
            {outrasMesas.length === 0 && <p className="bs-folha-texto">Não há outras mesas abertas.</p>}
          </div>
        </Folha>
      )}
      {folha?.tipo === "pedidos" && <PedidosFolha contaID={contaID} onFechar={() => setFolha(null)} onReimpresso={(res) => setFolha({ tipo: "enviado", res })} />}
      {folha?.tipo === "pessoas" && (
        <PessoasFolha
          atual={mesa.Pessoas}
          onCancelar={() => setFolha(null)}
          onGuardar={(n) =>
            DefinirPessoasMesa(contaID, n)
              .then(() => {
                setFolha(null);
                recarregar();
              })
              .catch((e) => setErro(textoErro(e)))
          }
        />
      )}
      {folha?.tipo === "nova-conta" && (
        <EtiquetaFolha
          titulo={`Nova conta na ${rotuloMesa(baseMesa(mesa.Mesa))}`}
          texto="Abre mais uma conta nesta mesa, que se lança e se paga à parte. A etiqueta ajuda a não confundir quem é quem."
          inicial=""
          botao="Abrir conta"
          onCancelar={() => setFolha(null)}
          onGuardar={(e) =>
            NovaContaMesa(contaID, e).then((nova) => {
              setFolha(null);
              onAbrirOutra(nova.ContaID);
            })
          }
        />
      )}
      {folha?.tipo === "etiqueta" && (
        <EtiquetaFolha
          titulo={`Etiqueta · ${titulo}`}
          texto="Uma palavra para não confundir esta conta (sai também nos talões). Deixe vazio para tirar."
          inicial={mesa.Etiqueta}
          botao="Guardar"
          onCancelar={() => setFolha(null)}
          onGuardar={(e) =>
            DefinirEtiquetaMesa(contaID, e).then(() => {
              setFolha(null);
              recarregar();
            })
          }
        />
      )}
      {folha?.tipo === "caixa-fechado" && (
        <Folha titulo="O caixa está fechado" onFechar={() => setFolha(null)}>
          <p className="bs-folha-texto">Para receber o dinheiro das mesas, abra primeiro o caixa com o fundo de troco.</p>
          <div className="bs-folha-acoes">
            <button className="bs-btn-sec" onClick={() => setFolha(null)}>
              Voltar
            </button>
            {onIrParaCaixa && (
              <button className="bs-btn" onClick={onIrParaCaixa}>
                Abrir o caixa
              </button>
            )}
          </div>
        </Folha>
      )}
      {folha?.tipo === "fechar" && (
        <FechoConta
          conta={{ ID: contaID, Identificador: titulo }}
          total={total}
          armazemId={folha.armazemID}
          clienteIdentificado={mesa.ClienteID !== CLIENTE_ANONIMO}
          nomeCliente={clientes.find((c) => c.ID === mesa.ClienteID)?.Nome}
          cliente={
            <label className="bs-campo">
              <span>Cliente (para poder ficar em dívida / fiado)</span>
              <select
                value={mesa.ClienteID === CLIENTE_ANONIMO ? "" : mesa.ClienteID}
                onChange={(e) =>
                  DefinirClienteMesa(contaID, e.target.value ? Number(e.target.value) : null)
                    .then(recarregar)
                    .catch((err) => setErro(textoErro(err)))
                }
              >
                <option value="">Anónimo (paga agora)</option>
                {clientes
                  .filter((c) => c.Ativo || c.ID === mesa.ClienteID)
                  .map((c) => (
                    <option key={c.ID} value={c.ID}>
                      {c.Nome}
                    </option>
                  ))}
              </select>
            </label>
          }
          fechar={FecharMesa}
          onCancelar={() => setFolha(null)}
          onFechada={(r) => {
            setFolha(null);
            onFechada({ ...r, identificador: titulo });
          }}
        />
      )}
    </div>
  );
}

// --- Resultado do envio --------------------------------------------------------

/** Onde saiu o pedido; se uma impressora falhou, dá para imprimir no aparelho. */
function ResultadoEnvio({ res, onFechar }: { res: usecase.ResultadoPedido; onFechar: () => void }) {
  const [aImprimir, setAImprimir] = useState(false);
  const [aviso, setAviso] = useState<string | null>(null);
  const falhas = res.Impressoes.filter((i) => i.Erro);
  const anulacao = res.Pedido.Tipo === "anulacao";

  function imprimirAqui() {
    if (!ponte) return;
    setAImprimir(true);
    ponte
      .ImprimirNoAparelho(res.Texto)
      .then(() => setAviso("Impresso neste aparelho."))
      .catch((e) => setAviso(textoErro(e)))
      .finally(() => setAImprimir(false));
  }

  return (
    <Folha titulo={anulacao ? "Anulação enviada" : `Pedido #${res.Pedido.ID} enviado`} onFechar={onFechar}>
      <ul className="mesas-impressoes">
        {res.Impressoes.map((i) => (
          <li key={i.ImpressoraID} className={i.Erro ? "erro" : "ok"}>
            {i.Erro ? <ExclamationTriangleIcon className="bs-icon" aria-hidden="true" /> : <CheckCircleIcon className="bs-icon" aria-hidden="true" />}
            <span>
              <strong>{i.Nome}</strong>
              {i.Erro ? ` — ${i.Erro}` : " — impresso"}
            </span>
          </li>
        ))}
        {res.Impressoes.length === 0 && <li className="erro">Não há impressoras de rede configuradas.</li>}
      </ul>
      <pre className="mesas-talao">{res.Texto}</pre>
      {aviso && <div className="bs-aviso">{aviso}</div>}
      <div className="bs-folha-acoes">
        {ponte && (
          <button className={falhas.length > 0 || res.Impressoes.length === 0 ? "bs-btn" : "bs-btn-sec"} disabled={aImprimir} onClick={imprimirAqui}>
            <PrinterIcon className="bs-icon" aria-hidden="true" />
            {aImprimir ? "A imprimir…" : "Imprimir neste aparelho"}
          </button>
        )}
        <button className={ponte && falhas.length > 0 ? "bs-btn-sec" : "bs-btn"} onClick={onFechar}>
          OK
        </button>
      </div>
    </Folha>
  );
}

// --- Imprimir a conta ------------------------------------------------------------

const ESTE_APARELHO = -1;

function ImprimirConta({
  mesa,
  total,
  onFechar,
  onMudou,
}: {
  mesa: domain.MesaAberta;
  total: number;
  onFechar: () => void;
  onMudou: () => void;
}) {
  const [pessoas, setPessoas] = useState(mesa.Pessoas);
  const [impressoras, setImpressoras] = useState<domain.Impressora[]>([]);
  const [destino, setDestino] = useState<number | null>(ponte ? ESTE_APARELHO : null);
  const [aImprimir, setAImprimir] = useState(false);
  const [resultado, setResultado] = useState<{ ok: boolean; texto: string } | null>(null);
  const [previa, setPrevia] = useState<string | null>(null);

  useEffect(() => {
    ListarImpressoras().then((l) => {
      const ativas = l.filter((i) => i.Ativa);
      setImpressoras(ativas);
      if (!ponte) {
        const balcao = ativas.find((i) => i.Funcao === "balcao") ?? ativas[0];
        if (balcao) setDestino(balcao.ID);
      }
    });
  }, []);

  function imprimir() {
    if (destino === null || aImprimir) return;
    setAImprimir(true);
    setResultado(null);
    const rede = destino === ESTE_APARELHO ? null : destino;
    ImprimirContaMesa(mesa.ContaID, pessoas, rede)
      .then(async (c) => {
        setPrevia(c.Texto);
        onMudou();
        if (rede === null) {
          await ponte!.ImprimirNoAparelho(c.Texto);
          setResultado({ ok: true, texto: "Conta impressa neste aparelho." });
        } else if (c.Impressao?.Erro) {
          setResultado({ ok: false, texto: `${c.Impressao.Nome}: ${c.Impressao.Erro}` });
        } else {
          setResultado({ ok: true, texto: `Conta impressa em ${c.Impressao?.Nome}.` });
        }
      })
      .catch((e) => setResultado({ ok: false, texto: textoErro(e) }))
      .finally(() => setAImprimir(false));
  }

  return (
    <Folha titulo={`Conta · ${rotuloMesa(mesa.Mesa)}`} onFechar={onFechar}>
      <div className="bs-fecho-total">
        <span>Total</span>
        <strong>{mt(total)}</strong>
        {pessoas > 1 && (
          <small>
            {pessoas} pessoas · <strong>{mt(total / pessoas)}</strong> cada
          </small>
        )}
      </div>
      <Pessoas valor={pessoas} onMudar={setPessoas} />

      <div className="bs-metodos mesas-destinos" role="radiogroup" aria-label="Impressora">
        {ponte && (
          <button role="radio" aria-checked={destino === ESTE_APARELHO} className={destino === ESTE_APARELHO ? "ativo" : ""} onClick={() => setDestino(ESTE_APARELHO)}>
            <PrinterIcon className="bs-icon" aria-hidden="true" />
            Este aparelho
          </button>
        )}
        {impressoras.map((i) => (
          <button key={i.ID} role="radio" aria-checked={destino === i.ID} className={destino === i.ID ? "ativo" : ""} onClick={() => setDestino(i.ID)}>
            <PrinterIcon className="bs-icon" aria-hidden="true" />
            {i.Nome}
          </button>
        ))}
        {!ponte && impressoras.length === 0 && <p className="bs-folha-texto">Não há impressoras configuradas (Administração → Restaurante).</p>}
      </div>

      {resultado && <div className={`bs-aviso${resultado.ok ? "" : " erro"}`}>{resultado.texto}</div>}
      {previa && <pre className="mesas-talao">{previa}</pre>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onFechar}>
          Voltar
        </button>
        <button className="bs-btn" disabled={destino === null || aImprimir} onClick={imprimir}>
          <PrinterIcon className="bs-icon" aria-hidden="true" />
          {aImprimir ? "A imprimir…" : "Imprimir conta"}
        </button>
      </div>
    </Folha>
  );
}

function PessoasFolha({ atual, onCancelar, onGuardar }: { atual: number; onCancelar: () => void; onGuardar: (n: number) => void }) {
  const [n, setN] = useState(atual);
  return (
    <Folha titulo="Número de pessoas" onFechar={onCancelar}>
      <Pessoas valor={n} onMudar={setN} />
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn" onClick={() => onGuardar(n)}>
          Guardar
        </button>
      </div>
    </Folha>
  );
}

function EtiquetaFolha({
  titulo,
  texto,
  inicial,
  botao,
  onCancelar,
  onGuardar,
}: {
  titulo: string;
  texto: string;
  inicial: string;
  botao: string;
  onCancelar: () => void;
  onGuardar: (etiqueta: string) => Promise<unknown>;
}) {
  const [etiqueta, setEtiqueta] = useState(inicial);
  const [erro, setErro] = useState<string | null>(null);
  const [aGuardar, setAGuardar] = useState(false);

  function guardar() {
    if (aGuardar) return;
    setAGuardar(true);
    setErro(null);
    onGuardar(etiqueta.trim())
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <Folha titulo={titulo} onFechar={onCancelar}>
      <p className="bs-folha-texto">{texto}</p>
      <CampoEtiqueta valor={etiqueta} onMudar={setEtiqueta} onEnter={guardar} />
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn" disabled={aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : botao}
        </button>
      </div>
    </Folha>
  );
}

// --- Anular (com o gerente) ---------------------------------------------------------

const MOTIVOS_ANULACAO = ["Cliente desistiu", "Engano no pedido", "Demorou demasiado", "Produto acabou"];

function AnularFolha({
  item,
  onCancelar,
  onAnular,
}: {
  item: domain.ItemMesa;
  onCancelar: () => void;
  onAnular: (quantidade: number, pin: string, motivo: string) => Promise<void>;
}) {
  const [quantidade, setQuantidade] = useState(1);
  const [motivo, setMotivo] = useState("");
  const [pin, setPin] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aAnular, setAAnular] = useState(false);

  function confirmar() {
    if (aAnular) return;
    setAAnular(true);
    setErro(null);
    onAnular(quantidade, pin, motivo)
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setAAnular(false));
  }

  return (
    <Folha titulo={`Anular ${item.Produto}`} onFechar={onCancelar}>
      <p className="bs-folha-texto">
        Já foi enviado para preparação: precisa do <strong>PIN de um gerente</strong> e do motivo. Sai um talão "ANULADO" onde o pedido foi
        impresso.
      </p>
      {item.Quantidade > 1 && (
        <div className="mesas-pessoas">
          <span>Quantas anular</span>
          <div className="bs-stepper">
            <button onClick={() => setQuantidade((q) => Math.max(1, q - 1))} aria-label="Menos">
              <MinusIcon className="bs-icon" aria-hidden="true" />
            </button>
            <span>{qtd(quantidade)}</span>
            <button onClick={() => setQuantidade((q) => Math.min(item.Quantidade, q + 1))} aria-label="Mais">
              <PlusIcon className="bs-icon" aria-hidden="true" />
            </button>
          </div>
        </div>
      )}
      <div className="bs-rapidos motivos">
        {MOTIVOS_ANULACAO.map((m) => (
          <button key={m} className={motivo === m ? "ativo" : ""} onClick={() => setMotivo(m)}>
            {m}
          </button>
        ))}
      </div>
      <label className="bs-campo">
        <span>Motivo</span>
        <input value={motivo} onChange={(e) => setMotivo(e.target.value)} />
      </label>
      <label className="bs-campo">
        <span>
          <ShieldCheckIcon className="bs-icon" aria-hidden="true" /> PIN do gerente
        </span>
        <input type="password" inputMode="numeric" autoComplete="off" value={pin} onChange={(e) => setPin(e.target.value.replace(/\D/g, ""))} />
      </label>
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn perigo" disabled={!motivo.trim() || pin.length < 4 || aAnular} onClick={confirmar}>
          {aAnular ? "A anular…" : `Anular ${qtd(quantidade)}`}
        </button>
      </div>
    </Folha>
  );
}

// --- Dividir a conta -----------------------------------------------------------------

function DividirFolha({
  mesa,
  itens,
  onCancelar,
  onDividida,
}: {
  mesa: domain.MesaAberta;
  itens: domain.ItemMesa[];
  onCancelar: () => void;
  onDividida: (nova: domain.MesaAberta) => void;
}) {
  const [partes, setPartes] = useState<Record<number, number>>({});
  const [etiqueta, setEtiqueta] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aDividir, setADividir] = useState(false);
  const totalNova = itens.reduce((s, i) => s + (partes[i.ID] ?? 0) * i.PrecoUnitario, 0);
  const algum = Object.values(partes).some((q) => q > 0);

  function mudar(i: domain.ItemMesa, delta: number) {
    setPartes((p) => ({ ...p, [i.ID]: Math.min(i.Quantidade, Math.max(0, (p[i.ID] ?? 0) + delta)) }));
  }

  function dividir() {
    const lista = itens
      .filter((i) => (partes[i.ID] ?? 0) > 0)
      .map((i) => {
        const p = new usecase.ParteDivisao();
        p.ItemID = i.ID;
        p.Quantidade = partes[i.ID];
        return p;
      });
    setADividir(true);
    DividirMesa(mesa.ContaID, lista, etiqueta.trim())
      .then(onDividida)
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setADividir(false));
  }

  return (
    <Folha titulo={`Dividir ${rotuloMesa(mesa.Mesa)}`} onFechar={onCancelar} larga>
      <p className="bs-folha-texto">
        Escolha o que passa para uma conta à parte (paga-se separada). Para dividir em partes iguais, imprima a conta com o número de pessoas.
      </p>
      <div className="mesas-dividir">
        {itens.map((i) => (
          <div key={i.ID} className="mesas-dividir-linha">
            <span>
              {i.Produto}
              <small> · {qtd(i.Quantidade)} na mesa</small>
            </span>
            <div className="bs-stepper">
              <button onClick={() => mudar(i, -1)} aria-label={`Menos ${i.Produto}`}>
                <MinusIcon className="bs-icon" aria-hidden="true" />
              </button>
              <span>{qtd(partes[i.ID] ?? 0)}</span>
              <button onClick={() => mudar(i, 1)} aria-label={`Mais ${i.Produto}`}>
                <PlusIcon className="bs-icon" aria-hidden="true" />
              </button>
            </div>
          </div>
        ))}
      </div>
      <div className="bs-fecho-total">
        <span>Nova conta</span>
        <strong>{mt(totalNova)}</strong>
      </div>
      <CampoEtiqueta valor={etiqueta} onMudar={setEtiqueta} />
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn" disabled={!algum || aDividir} onClick={dividir}>
          <ScissorsIcon className="bs-icon" aria-hidden="true" />
          {aDividir ? "A dividir…" : "Criar conta separada"}
        </button>
      </div>
    </Folha>
  );
}

// --- Passar a outro garçom -------------------------------------------------------------

function TransferirFolha({
  mesa,
  onCancelar,
  onFeito,
}: {
  mesa: domain.MesaAberta;
  onCancelar: () => void;
  onFeito: (saiDeMim: boolean) => void;
}) {
  const [garcons, setGarcons] = useState<usecase.Operador[]>([]);
  const [erro, setErro] = useState<string | null>(null);
  useEffect(() => {
    ListarOperadoresAtivos().then((l) => setGarcons(l.filter((o) => o.ID !== mesa.GarcomID)));
  }, [mesa.GarcomID]);
  return (
    <Folha titulo={`Passar ${rotuloMesa(mesa.Mesa)} para…`} onFechar={onCancelar}>
      <p className="bs-folha-texto">A mesa passa com tudo o que já tem para o outro garçom.</p>
      <div className="bs-destinos">
        {garcons.map((g) => (
          <button
            key={g.ID}
            onClick={() =>
              TransferirMesa(mesa.ContaID, g.ID)
                .then(() => onFeito(true))
                .catch((e) => setErro(textoErro(e)))
            }
          >
            {g.Nome}
          </button>
        ))}
      </div>
      {erro && <div className="bs-aviso erro">{erro}</div>}
    </Folha>
  );
}

// --- Pedidos enviados (reimprimir) -----------------------------------------------------

function PedidosFolha({
  contaID,
  onFechar,
  onReimpresso,
}: {
  contaID: number;
  onFechar: () => void;
  onReimpresso: (res: usecase.ResultadoPedido) => void;
}) {
  const [pedidos, setPedidos] = useState<domain.Pedido[] | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  useEffect(() => {
    ListarPedidosMesa(contaID)
      .then(setPedidos)
      .catch((e) => setErro(textoErro(e)));
  }, [contaID]);
  return (
    <Folha titulo="Pedidos enviados" onFechar={onFechar}>
      <div className="bs-destinos">
        {(pedidos ?? []).map((p) => (
          <button
            key={p.ID}
            onClick={() =>
              ReimprimirPedido(p.ID, null)
                .then(onReimpresso)
                .catch((e) => setErro(textoErro(e)))
            }
          >
            {p.Tipo === "anulacao" ? "Anulação" : "Pedido"} #{p.ID} ·{" "}
            {new Date(String(p.CriadoEm)).toLocaleTimeString("pt-PT", { hour: "2-digit", minute: "2-digit" })} — reimprimir
          </button>
        ))}
        {pedidos !== null && pedidos.length === 0 && <p className="bs-folha-texto">Ainda não foi enviado nenhum pedido.</p>}
      </div>
      {erro && <div className="bs-aviso erro">{erro}</div>}
    </Folha>
  );
}
