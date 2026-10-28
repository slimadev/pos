import { useEffect, useMemo, useRef, useState } from "react";
import {
  PlusIcon,
  MinusIcon,
  ArrowLeftIcon,
  XCircleIcon,
  CheckCircleIcon,
  MagnifyingGlassIcon,
  XMarkIcon,
  ArrowsRightLeftIcon,
  EllipsisVerticalIcon,
  ClockIcon,
  PrinterIcon,
  ChevronUpIcon,
  BoltIcon,
  RectangleStackIcon,
} from "@heroicons/react/24/outline";
import {
  ListarContasAbertas,
  ListarItensConta,
  AbrirConta,
  AdicionarItemConta,
  AlterarQuantidadeItemConta,
  TransferirItemConta,
  CancelarConta,
  FecharContaAberta,
  ListarProdutos,
  ListarArmazens,
  ListarClientes,
} from "../../wailsjs/go/api/App";
import { api, domain, usecase } from "../../wailsjs/go/models";
import Catalogo from "./Catalogo";
import VendaDireta from "./VendaDireta";
import { METODOS, ROTULO_METODO, lerNumero, mt, qtd, normalizar } from "./formato";
import { calcularTroco } from "./troco";
import ReciboModal from "./ReciboModal";
import "./Bottlestore.css";

// --- Utilitários -------------------------------------------------------------

/** Há quanto tempo a conta está aberta ("2h 14m"); em aviso a partir das 3h. */
function tempoAberta(abertaEm: unknown): { texto: string; longa: boolean } {
  const inicio = new Date(String(abertaEm)).getTime();
  if (!inicio) return { texto: "", longa: false };
  const minutos = Math.max(0, Math.round((Date.now() - inicio) / 60000));
  const horas = Math.floor(minutos / 60);
  return {
    texto: horas > 0 ? `${horas}h ${String(minutos % 60).padStart(2, "0")}m` : `${minutos}m`,
    longa: horas >= 3,
  };
}

const totalItem = (i: domain.ItemConta) => i.Quantidade * i.PrecoUnitarioAplicado - i.Desconto;
const somaTotal = (itens: domain.ItemConta[]) => itens.reduce((s, i) => s + totalItem(i), 0);
const somaQtd = (itens: domain.ItemConta[]) => itens.reduce((s, i) => s + i.Quantidade, 0);
const arred = (n: number) => Math.round(n * 100) / 100;
const valorTexto = (n: number) => n.toFixed(2).replace(".", ",");

/** Uma linha da comanda: a mesma bebida (produto, unidade, preço) junta. */
interface Linha {
  chave: string;
  produtoId: number;
  unidadeId: number;
  preco: number;
  quantidade: number;
  total: number;
  itens: domain.ItemConta[];
}

function agrupar(itens: domain.ItemConta[]): Linha[] {
  const mapa = new Map<string, Linha>();
  for (const i of itens) {
    const chave = `${i.ProdutoID}-${i.UnidadeProdutoID}-${i.PrecoUnitarioAplicado}`;
    const l = mapa.get(chave) ?? {
      chave,
      produtoId: i.ProdutoID,
      unidadeId: i.UnidadeProdutoID,
      preco: i.PrecoUnitarioAplicado,
      quantidade: 0,
      total: 0,
      itens: [],
    };
    l.quantidade += i.Quantidade;
    l.total += totalItem(i);
    l.itens.push(i);
    mapa.set(chave, l);
  }
  return Array.from(mapa.values());
}

/** Notas para "o cliente deu…": o valor exato e os arredondamentos às notas. */
function valoresSugeridos(total: number): number[] {
  const out = new Set<number>([arred(total)]);
  for (const nota of [100, 200, 500, 1000, 2000]) {
    const v = Math.ceil(total / nota) * nota;
    if (v > total + 0.005) out.add(v);
    if (out.size >= 4) break;
  }
  return Array.from(out).sort((a, b) => a - b);
}

const NOMES_RAPIDOS = ["Balcão", ...Array.from({ length: 12 }, (_, i) => `Mesa ${i + 1}`), "Esplanada 1", "Esplanada 2"];

const MOTIVOS_CANCELAMENTO = ["Cliente desistiu", "Aberta por engano", "Conta repetida", "Não pagou"];

interface ContaComItens {
  conta: domain.Conta;
  itens: domain.ItemConta[];
}

type Modo = "contas" | "balcao";

function lerModo(): Modo {
  try {
    return sessionStorage.getItem("bs-modo") === "balcao" ? "balcao" : "contas";
  } catch {
    return "contas";
  }
}

const textoErro = (e: unknown) => String(e).replace(/^Error:\s*/, "");

// --- Ecrã principal ----------------------------------------------------------

/**
 * PDV do bar (SDD 4.10, telas `2a`/`2b`). Dois modos:
 * - Contas: mesas e grupos que consomem ao longo da noite — abrir, lançar
 *   rondas, corrigir, transferir, fechar com troco, fiado ou pagamento
 *   dividido.
 * - Balcão: venda rápida para levar (a mesma venda direta da mercearia).
 */
export default function Bottlestore() {
  const [modo, setModoEstado] = useState<Modo>(lerModo);
  const [contas, setContas] = useState<ContaComItens[]>([]);
  const [clientes, setClientes] = useState<domain.Cliente[]>([]);
  const [contaSelecionadaId, setContaSelecionadaId] = useState<number | null>(null);
  const [novaAberta, setNovaAberta] = useState(false);
  const [pesquisa, setPesquisa] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [fechada, setFechada] = useState<{ id: number; identificador: string; total: number; troco: number; divida: number } | null>(
    null,
  );
  const [recibo, setRecibo] = useState<{ id: number; troco: number } | null>(null);
  // Re-desenha os tempos ("2h 14m") a cada minuto.
  const [, setRelogio] = useState(0);

  function setModo(m: Modo) {
    setModoEstado(m);
    try {
      sessionStorage.setItem("bs-modo", m);
    } catch {
      /* sem armazenamento: fica só nesta sessão */
    }
  }

  const recarregarContas = () => {
    ListarContasAbertas()
      .then((lista) => Promise.all(lista.map(async (conta) => ({ conta, itens: await ListarItensConta(conta.ID) }))))
      .then(setContas)
      .catch((e) => setErro(textoErro(e)));
  };

  useEffect(() => {
    recarregarContas();
    ListarClientes().then(setClientes);
    const t = window.setInterval(() => setRelogio((n) => n + 1), 60000);
    // Voltar à app (ex: depois da administração) mostra contas atualizadas.
    window.addEventListener("focus", recarregarContas);
    return () => {
      window.clearInterval(t);
      window.removeEventListener("focus", recarregarContas);
    };
  }, []);

  const nomeCliente = (id: number) => clientes.find((c) => c.ID === id)?.Nome;
  const contaSelecionada = contas.find((c) => c.conta.ID === contaSelecionadaId) ?? null;

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

  if (contaSelecionada && modo === "contas") {
    return (
      <>
        {sobreposicoes}
        <ContaDetalhe
          key={contaSelecionada.conta.ID}
          contaComItens={contaSelecionada}
          nomeCliente={nomeCliente(contaSelecionada.conta.ClienteID)}
          outrasContas={contas.filter((c) => c.conta.ID !== contaSelecionadaId).map((c) => c.conta)}
          onVoltar={() => {
            setContaSelecionadaId(null);
            recarregarContas();
          }}
          onItens={(itens) =>
            setContas((prev) => prev.map((c) => (c.conta.ID === contaSelecionadaId ? { ...c, itens } : c)))
          }
          onMudou={recarregarContas}
          onFechada={(r) => {
            setContaSelecionadaId(null);
            recarregarContas();
            setFechada(r);
          }}
        />
      </>
    );
  }

  const totalEmConsumo = contas.reduce((acc, c) => acc + somaTotal(c.itens), 0);
  const termo = normalizar(pesquisa);
  const visiveis = termo
    ? contas.filter(
        (c) => normalizar(c.conta.Identificador).includes(termo) || normalizar(nomeCliente(c.conta.ClienteID) ?? "").includes(termo),
      )
    : contas;

  return (
    <div className={`bs-raiz${modo === "balcao" ? " modo-balcao" : ""}`}>
      {sobreposicoes}
      <div className="bs-modos" role="tablist">
        <button role="tab" aria-selected={modo === "contas"} className={modo === "contas" ? "ativo" : ""} onClick={() => setModo("contas")}>
          <RectangleStackIcon className="bs-icon" aria-hidden="true" />
          Contas
          {contas.length > 0 && <span className="bs-modos-badge">{contas.length}</span>}
        </button>
        <button role="tab" aria-selected={modo === "balcao"} className={modo === "balcao" ? "ativo" : ""} onClick={() => setModo("balcao")}>
          <BoltIcon className="bs-icon" aria-hidden="true" />
          Balcão
        </button>
      </div>

      {modo === "balcao" ? (
        <VendaDireta tipoNegocio="bottlestore" />
      ) : (
        <div className="bs">
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
                <span className="bs-kpi-valor">{contas.length}</span>
                <span className="bs-kpi-label">{contas.length === 1 ? "conta aberta" : "contas abertas"}</span>
              </div>
              <div>
                <span className="bs-kpi-valor">{mt(totalEmConsumo)}</span>
                <span className="bs-kpi-label">em consumo</span>
              </div>
            </div>
            <button className="bs-abrir" onClick={() => setNovaAberta(true)}>
              <PlusIcon className="bs-icon" aria-hidden="true" />
              Abrir conta
            </button>
          </div>

          {contas.length > 6 && (
            <label className="bs-pesquisa">
              <MagnifyingGlassIcon className="bs-icon" aria-hidden="true" />
              <input placeholder="Procurar mesa ou cliente" value={pesquisa} onChange={(e) => setPesquisa(e.target.value)} />
            </label>
          )}

          <div className="bs-grid">
            {visiveis.map(({ conta, itens }) => {
              const tempo = tempoAberta(conta.AbertaEm);
              const cliente = nomeCliente(conta.ClienteID);
              return (
                <button key={conta.ID} className={`bs-card${tempo.longa ? " longa" : ""}`} onClick={() => setContaSelecionadaId(conta.ID)}>
                  <div className="bs-card-cabeca">
                    <span className="bs-card-id">{conta.Identificador}</span>
                    {tempo.texto && (
                      <span className="bs-card-tempo">
                        <ClockIcon className="bs-icon-mini" aria-hidden="true" />
                        {tempo.texto}
                      </span>
                    )}
                  </div>
                  <div className="bs-card-cliente">{cliente ?? "Anónimo"}</div>
                  <div className="bs-card-bottom">
                    <span className="bs-card-total">{mt(somaTotal(itens))}</span>
                    <span className="bs-card-itens">
                      {itens.length === 0 ? "vazia" : `${qtd(somaQtd(itens))} ${somaQtd(itens) === 1 ? "bebida" : "bebidas"}`}
                    </span>
                  </div>
                </button>
              );
            })}
            {contas.length === 0 && (
              <div className="bs-vazio">
                <RectangleStackIcon className="bs-vazio-icone" aria-hidden="true" />
                <strong>Nenhuma conta aberta</strong>
                <span>Abra uma conta por mesa ou grupo e vá lançando as rondas. Para quem leva e paga logo, use o Balcão.</span>
                <button className="bs-abrir" onClick={() => setNovaAberta(true)}>
                  <PlusIcon className="bs-icon" aria-hidden="true" />
                  Abrir conta
                </button>
              </div>
            )}
            {contas.length > 0 && visiveis.length === 0 && <div className="bs-vazio">Nenhuma conta com "{pesquisa}".</div>}
          </div>
        </div>
      )}

      {novaAberta && (
        <NovaConta
          clientes={clientes}
          ocupados={contas.map((c) => c.conta.Identificador)}
          onCancelar={() => setNovaAberta(false)}
          onAberta={(conta) => {
            setNovaAberta(false);
            setContas((prev) => [...prev, { conta, itens: [] }]);
            setContaSelecionadaId(conta.ID);
          }}
        />
      )}
    </div>
  );
}

// --- Abrir conta -------------------------------------------------------------

function NovaConta({
  clientes,
  ocupados,
  onCancelar,
  onAberta,
}: {
  clientes: domain.Cliente[];
  ocupados: string[];
  onCancelar: () => void;
  onAberta: (c: domain.Conta) => void;
}) {
  const [nome, setNome] = useState("");
  const [clienteId, setClienteId] = useState<number | "">("");
  const [aAbrir, setAAbrir] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const livres = NOMES_RAPIDOS.filter((n) => !ocupados.some((o) => normalizar(o) === normalizar(n)));
  const repetido = nome.trim() !== "" && ocupados.some((o) => normalizar(o) === normalizar(nome));

  function abrir(identificador = nome) {
    if (!identificador.trim() || aAbrir) return;
    const req = new api.AbrirContaRequest();
    req.Identificador = identificador.trim();
    req.ClienteID = clienteId === "" ? undefined : Number(clienteId);
    setAAbrir(true);
    AbrirConta(req)
      .then(onAberta)
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setAAbrir(false));
  }

  return (
    <Folha titulo="Abrir conta" onFechar={onCancelar}>
      <label className="bs-campo">
        <span>Mesa ou nome do grupo</span>
        <input
          value={nome}
          placeholder="ex: Mesa 3, Grupo do João"
          onChange={(e) => setNome(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && abrir()}
        />
      </label>
      {repetido && <div className="bs-aviso">Já há uma conta aberta com este nome.</div>}
      <div className="bs-rapidos" aria-label="Escolha rápida">
        {livres.map((n) => (
          <button key={n} className={normalizar(n) === normalizar(nome) ? "ativo" : ""} onClick={() => setNome(n)}>
            {n}
          </button>
        ))}
      </div>
      <label className="bs-campo">
        <span>Cliente (para poder ficar em dívida)</span>
        <select value={clienteId} onChange={(e) => setClienteId(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Anónimo</option>
          {clientes.map((c) => (
            <option key={c.ID} value={c.ID}>
              {c.Nome}
            </option>
          ))}
        </select>
      </label>
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Cancelar
        </button>
        <button className="bs-btn" disabled={!nome.trim() || aAbrir} onClick={() => abrir()}>
          <PlusIcon className="bs-icon" aria-hidden="true" />
          {aAbrir ? "A abrir…" : "Abrir conta"}
        </button>
      </div>
    </Folha>
  );
}

// --- Conta aberta ------------------------------------------------------------

function ContaDetalhe({
  contaComItens,
  nomeCliente,
  outrasContas,
  onVoltar,
  onItens,
  onMudou,
  onFechada,
}: {
  contaComItens: ContaComItens;
  nomeCliente?: string;
  outrasContas: domain.Conta[];
  onVoltar: () => void;
  onItens: (itens: domain.ItemConta[]) => void;
  onMudou: () => void;
  onFechada: (r: { id: number; identificador: string; total: number; troco: number; divida: number }) => void;
}) {
  const { conta } = contaComItens;
  const [itens, setItensEstado] = useState(contaComItens.itens);
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [nomesUnidade, setNomesUnidade] = useState<Record<number, string>>({});
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [armazemId, setArmazemId] = useState<number | "">("");
  const [comandaAberta, setComandaAberta] = useState(false);
  const [menuAberto, setMenuAberto] = useState(false);
  const [aCancelar, setACancelar] = useState(false);
  const [aTransferir, setATransferir] = useState<Linha | null>(null);
  const [mostrarFecho, setMostrarFecho] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const [ultimo, setUltimo] = useState<{ nome: string; n: number } | null>(null);
  const toastRef = useRef<number | undefined>(undefined);

  const setItens = (f: (prev: domain.ItemConta[]) => domain.ItemConta[]) =>
    setItensEstado((prev) => {
      const novo = f(prev);
      onItens(novo);
      return novo;
    });

  useEffect(() => {
    ListarProdutos().then(setProdutos);
    ListarArmazens().then((lista) => {
      const ativos = lista.filter((a) => a.Ativo && !a.ForaDeVenda);
      setArmazens(ativos);
      const inicial = ativos.find((a) => a.Tipo === "principal") ?? ativos[0];
      if (inicial) setArmazemId(inicial.ID);
    });
    return () => window.clearTimeout(toastRef.current);
  }, []);

  const nomeProduto = (id: number) => produtos.find((p) => p.ID === id)?.Nome ?? "…";
  const linhas = useMemo(() => agrupar(itens), [itens]);
  const total = somaTotal(itens);
  const nBebidas = somaQtd(itens);
  const tempo = tempoAberta(conta.AbertaEm);

  const juntar = (item: domain.ItemConta) =>
    setItens((prev) =>
      item.Quantidade <= 0
        ? prev.filter((i) => i.ID !== item.ID)
        : prev.some((i) => i.ID === item.ID)
          ? prev.map((i) => (i.ID === item.ID ? item : i))
          : [...prev, item],
    );

  function lancar(produtoId: number, unidadeId: number, nome?: string) {
    const p = new usecase.AdicionarItemContaParams();
    p.ContaID = conta.ID;
    p.ProdutoID = produtoId;
    p.UnidadeProdutoID = unidadeId;
    p.Quantidade = 1;
    p.Desconto = 0;
    AdicionarItemConta(p)
      .then((item) => {
        juntar(item);
        if (nome) {
          setUltimo((u) => ({ nome, n: u?.nome === nome ? u.n + 1 : 1 }));
          window.clearTimeout(toastRef.current);
          toastRef.current = window.setTimeout(() => setUltimo(null), 1800);
        }
      })
      .catch((e) => setErro(textoErro(e)));
  }

  function adicionar(item: usecase.ItemCatalogo, unidade: domain.UnidadeProduto) {
    setNomesUnidade((m) => ({ ...m, [unidade.ID]: unidade.NomeUnidade }));
    lancar(item.Produto.ID, unidade.ID, item.Produto.Nome);
  }

  // "−": tira uma da última linha do grupo (a 0, a linha sai).
  function menos(l: Linha) {
    const ultimaLinha = l.itens[l.itens.length - 1];
    AlterarQuantidadeItemConta(ultimaLinha.ID, Math.max(0, ultimaLinha.Quantidade - 1))
      .then(juntar)
      .catch((e) => setErro(textoErro(e)));
  }

  function transferir(l: Linha, destino: domain.Conta) {
    Promise.all(l.itens.map((i) => TransferirItemConta(i.ID, destino.ID)))
      .then(() => {
        setItens((prev) => prev.filter((i) => !l.itens.some((x) => x.ID === i.ID)));
        onMudou();
      })
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setATransferir(null));
  }

  const nomeArmazem = armazens.find((a) => a.ID === armazemId)?.Nome;

  return (
    <div className="bs bs-detalhe">
      <div className="bs-detalhe-header">
        <button className="bs-voltar" onClick={onVoltar} aria-label="Voltar às contas">
          <ArrowLeftIcon className="bs-icon-grande" aria-hidden="true" />
        </button>
        <div className="bs-detalhe-titulo">
          <div className="bs-detalhe-nome">{conta.Identificador}</div>
          <div className="bs-detalhe-sub">
            {nomeCliente ?? "Anónimo"}
            {tempo.texto && ` · aberta há ${tempo.texto}`}
          </div>
        </div>
        <div className="bs-menu">
          <button className="bs-menu-botao" onClick={() => setMenuAberto((v) => !v)} aria-label="Mais opções">
            <EllipsisVerticalIcon className="bs-icon-grande" aria-hidden="true" />
          </button>
          {menuAberto && (
            <>
              <div className="bs-menu-fundo" onClick={() => setMenuAberto(false)} />
              <div className="bs-menu-lista">
                {armazens.length > 1 && (
                  <label className="bs-menu-item">
                    <span>Sai do armazém</span>
                    <select value={armazemId} onChange={(e) => setArmazemId(e.target.value ? Number(e.target.value) : "")}>
                      {armazens.map((a) => (
                        <option key={a.ID} value={a.ID}>
                          {a.Nome}
                        </option>
                      ))}
                    </select>
                  </label>
                )}
                <button
                  className="bs-menu-item perigo"
                  onClick={() => {
                    setMenuAberto(false);
                    setACancelar(true);
                  }}
                >
                  <XCircleIcon className="bs-icon" aria-hidden="true" />
                  Cancelar conta
                </button>
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

      <div className="bs-detalhe-body">
        <div className="bs-catalogo">
          {armazemId !== "" && (
            <Catalogo key={armazemId} armazemID={armazemId} IconeBusca={MagnifyingGlassIcon} onEscolher={adicionar} onErro={setErro} />
          )}
          {ultimo && (
            <div className="bs-toast" role="status">
              <CheckCircleIcon className="bs-icon" aria-hidden="true" />+{ultimo.n} {ultimo.nome}
            </div>
          )}
        </div>

        {/* Ecrã estreito: resumo no fundo que abre a comanda. */}
        <div className="bs-resumo-movel">
          <button className="bs-resumo-ver" onClick={() => setComandaAberta(true)} disabled={itens.length === 0}>
            <ChevronUpIcon className="bs-icon" aria-hidden="true" />
            <span>
              <span className="bs-resumo-total">{mt(total)}</span>
              <span className="bs-resumo-qtd">
                {itens.length === 0 ? "Toque nas bebidas para lançar" : `${qtd(nBebidas)} na comanda · ver`}
              </span>
            </span>
          </button>
          <button className="bs-fechar" disabled={itens.length === 0 || armazemId === ""} onClick={() => setMostrarFecho(true)}>
            Fechar conta
          </button>
        </div>

        <aside className={`bs-comanda${comandaAberta ? " aberta" : ""}`}>
          <div className="bs-comanda-header">
            <button className="bs-comanda-fechar-movel" onClick={() => setComandaAberta(false)} aria-label="Voltar às bebidas">
              <ArrowLeftIcon className="bs-icon-grande" aria-hidden="true" />
            </button>
            <span className="bs-comanda-titulo">Comanda · {conta.Identificador}</span>
            <span className="bs-comanda-n">{qtd(nBebidas)}</span>
          </div>
          <div className="bs-comanda-itens">
            {linhas.map((l) => (
              <div key={l.chave} className="bs-linha">
                <div className="bs-linha-topo">
                  <div className="bs-linha-nome">{nomeProduto(l.produtoId)}</div>
                  <div className="bs-linha-total">{mt(l.total)}</div>
                </div>
                <div className="bs-linha-baixo">
                <div className="bs-linha-preco">
                  {mt(l.preco)}
                  {nomesUnidade[l.unidadeId] && nomesUnidade[l.unidadeId] !== "unidade" ? ` · ${nomesUnidade[l.unidadeId]}` : ""}
                </div>
                <div className="bs-stepper">
                  <button onClick={() => menos(l)} aria-label={`Menos uma ${nomeProduto(l.produtoId)}`}>
                    <MinusIcon className="bs-icon" aria-hidden="true" />
                  </button>
                  <span>{qtd(l.quantidade)}</span>
                  <button onClick={() => lancar(l.produtoId, l.unidadeId)} aria-label={`Mais uma ${nomeProduto(l.produtoId)}`}>
                    <PlusIcon className="bs-icon" aria-hidden="true" />
                  </button>
                </div>
                {outrasContas.length > 0 && (
                  <button className="bs-linha-transferir" onClick={() => setATransferir(l)} aria-label="Passar para outra conta">
                    <ArrowsRightLeftIcon className="bs-icon" aria-hidden="true" />
                  </button>
                )}
                </div>
              </div>
            ))}
            {itens.length === 0 && (
              <div className="bs-comanda-vazia">Comanda vazia — toque nas bebidas para lançar a primeira ronda.</div>
            )}
          </div>
          <div className="bs-comanda-footer">
            <div className="bs-comanda-total">
              <span>Total</span>
              <span>{mt(total)}</span>
            </div>
            <button className="bs-fechar" disabled={itens.length === 0 || armazemId === ""} onClick={() => setMostrarFecho(true)}>
              <CheckCircleIcon className="bs-icon" aria-hidden="true" />
              Fechar conta
            </button>
          </div>
        </aside>
      </div>

      {aTransferir && (
        <Folha titulo={`Passar ${qtd(aTransferir.quantidade)}× ${nomeProduto(aTransferir.produtoId)} para…`} onFechar={() => setATransferir(null)}>
          <div className="bs-destinos">
            {outrasContas.map((c) => (
              <button key={c.ID} onClick={() => transferir(aTransferir, c)}>
                {c.Identificador}
              </button>
            ))}
          </div>
        </Folha>
      )}

      {aCancelar && (
        <CancelarContaFolha
          conta={conta}
          temConsumo={itens.length > 0}
          onCancelar={() => setACancelar(false)}
          onCancelada={() => {
            setACancelar(false);
            onVoltar();
          }}
        />
      )}

      {mostrarFecho && armazemId !== "" && (
        <FechoConta
          conta={conta}
          total={total}
          armazemId={armazemId}
          nomeArmazem={nomeArmazem}
          clienteIdentificado={nomeCliente !== undefined}
          nomeCliente={nomeCliente}
          onCancelar={() => setMostrarFecho(false)}
          onFechada={(r) => {
            setMostrarFecho(false);
            onFechada({ ...r, identificador: conta.Identificador });
          }}
        />
      )}
    </div>
  );
}

// --- Cancelar ----------------------------------------------------------------

function CancelarContaFolha({
  conta,
  temConsumo,
  onCancelar,
  onCancelada,
}: {
  conta: domain.Conta;
  temConsumo: boolean;
  onCancelar: () => void;
  onCancelada: () => void;
}) {
  const [motivo, setMotivo] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aCancelar, setACancelar] = useState(false);

  function confirmar() {
    if (!motivo.trim()) return;
    setACancelar(true);
    CancelarConta(conta.ID, motivo.trim())
      .then(onCancelada)
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setACancelar(false));
  }

  return (
    <Folha titulo={`Cancelar ${conta.Identificador}?`} onFechar={onCancelar}>
      <p className="bs-folha-texto">
        {temConsumo
          ? "O consumo lançado não é cobrado nem sai do stock. Fica registado quem cancelou e porquê."
          : "A conta está vazia. Fica registado quem cancelou e porquê."}
      </p>
      <div className="bs-rapidos motivos">
        {MOTIVOS_CANCELAMENTO.map((m) => (
          <button key={m} className={motivo === m ? "ativo" : ""} onClick={() => setMotivo(m)}>
            {m}
          </button>
        ))}
      </div>
      <label className="bs-campo">
        <span>Motivo</span>
        <input value={motivo} onChange={(e) => setMotivo(e.target.value)} placeholder="Escreva ou escolha acima" />
      </label>
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn perigo" disabled={!motivo.trim() || aCancelar} onClick={confirmar}>
          <XCircleIcon className="bs-icon" aria-hidden="true" />
          {aCancelar ? "A cancelar…" : "Cancelar conta"}
        </button>
      </div>
    </Folha>
  );
}

// --- Fechar conta ------------------------------------------------------------

type ModoPagamento = string | "dividir" | "fiado";

function FechoConta({
  conta,
  total: totalBruto,
  armazemId,
  nomeArmazem,
  clienteIdentificado,
  nomeCliente,
  onCancelar,
  onFechada,
}: {
  conta: domain.Conta;
  total: number;
  armazemId: number;
  nomeArmazem?: string;
  clienteIdentificado: boolean;
  nomeCliente?: string;
  onCancelar: () => void;
  onFechada: (r: { id: number; total: number; troco: number; divida: number }) => void;
}) {
  const [modo, setModo] = useState<ModoPagamento>("dinheiro");
  const [recebido, setRecebido] = useState("");
  const [desconto, setDesconto] = useState("");
  const [comDesconto, setComDesconto] = useState(false);
  const [linhas, setLinhas] = useState([{ metodo: "dinheiro", valor: "" }, { metodo: "mpesa", valor: "" }]);
  const [aFechar, setAFechar] = useState(false);
  const [erro, setErro] = useState<string | null>(null);

  const valorDesconto = Math.min(totalBruto, Math.max(0, lerNumero(desconto || "0") || 0));
  const total = arred(totalBruto - valorDesconto);

  // Pagamentos conforme o modo escolhido.
  const pagamentos =
    modo === "fiado"
      ? []
      : modo === "dividir"
        ? linhas
        : modo === "dinheiro"
          ? [{ metodo: "dinheiro", valor: recebido.trim() ? recebido : valorTexto(total) }]
          : [{ metodo: modo, valor: valorTexto(total) }];
  const soma = pagamentos.reduce((s, p) => s + (lerNumero(p.valor || "0") || 0), 0);
  const falta = arred(total - soma);
  const { troco, excessoSemDinheiro, liquidos } = calcularTroco(total, pagamentos);
  const ficaEmDivida = falta > 0.005;
  const bloqueado = aFechar || excessoSemDinheiro > 0 || (ficaEmDivida && !clienteIdentificado);

  function confirmar() {
    if (bloqueado) return;
    const p = new usecase.FecharContaParams();
    p.ArmazemID = armazemId;
    p.DescontoTotal = valorDesconto;
    p.UtilizadorID = 0;
    p.Pagamentos = liquidos.map((pg) => {
      const pag = new usecase.PagamentoVendaParams();
      pag.Metodo = pg.metodo;
      pag.Valor = pg.valor;
      return pag;
    });
    setAFechar(true);
    setErro(null);
    FecharContaAberta(conta.ID, p)
      .then((r) => {
        if (r.Alerta?.Excedido) {
          window.alert(
            `Aviso: o cliente passou o limite de crédito (${mt(r.Alerta.LimiteCredito)}) — deve agora ${mt(r.Alerta.TotalEmAberto)}.`,
          );
        }
        onFechada({ id: r.Venda.Venda.ID, total, troco, divida: ficaEmDivida ? falta : 0 });
      })
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setAFechar(false));
  }

  const opcoes: { chave: ModoPagamento; rotulo: string }[] = [
    ...METODOS.filter((m) => m !== "transferencia").map((m) => ({ chave: m, rotulo: ROTULO_METODO[m] })),
    { chave: "dividir", rotulo: "Dividir" },
    ...(clienteIdentificado ? [{ chave: "fiado", rotulo: "Fiado" }] : []),
  ];

  return (
    <Folha titulo={`Fechar ${conta.Identificador}`} onFechar={onCancelar} larga>
      <div className="bs-fecho-total">
        <span>A pagar</span>
        <strong>{mt(total)}</strong>
        {valorDesconto > 0 && <small>{mt(totalBruto)} − desconto {mt(valorDesconto)}</small>}
      </div>

      <div className="bs-metodos" role="radiogroup" aria-label="Forma de pagamento">
        {opcoes.map((o) => (
          <button key={o.chave} role="radio" aria-checked={modo === o.chave} className={modo === o.chave ? "ativo" : ""} onClick={() => setModo(o.chave)}>
            {o.rotulo}
          </button>
        ))}
      </div>

      {modo === "dinheiro" && (
        <div className="bs-dinheiro">
          <label className="bs-campo">
            <span>O cliente deu</span>
            <input inputMode="decimal" placeholder={valorTexto(total)} value={recebido} onChange={(e) => setRecebido(e.target.value)} />
          </label>
          <div className="bs-notas">
            {valoresSugeridos(total).map((v) => (
              <button key={v} className={arred(lerNumero(recebido || "0")) === v ? "ativo" : ""} onClick={() => setRecebido(valorTexto(v))}>
                {Math.abs(v - total) < 0.005 ? "Exato" : mt(v).replace(",00", "")}
              </button>
            ))}
          </div>
        </div>
      )}

      {modo === "dividir" && (
        <div className="bs-dividir">
          {linhas.map((p, i) => (
            <div key={i} className="bs-dividir-linha">
              <select value={p.metodo} onChange={(e) => setLinhas((prev) => prev.map((x, j) => (j === i ? { ...x, metodo: e.target.value } : x)))}>
                {METODOS.map((m) => (
                  <option key={m} value={m}>
                    {ROTULO_METODO[m]}
                  </option>
                ))}
              </select>
              <input
                inputMode="decimal"
                placeholder="0,00"
                value={p.valor}
                onChange={(e) => setLinhas((prev) => prev.map((x, j) => (j === i ? { ...x, valor: e.target.value } : x)))}
              />
              <button
                className="bs-btn-sec bs-resto"
                onClick={() => {
                  const outros = linhas.reduce((s, x, j) => (j === i ? s : s + (lerNumero(x.valor || "0") || 0)), 0);
                  const resto = Math.max(0, arred(total - outros));
                  setLinhas((prev) => prev.map((x, j) => (j === i ? { ...x, valor: valorTexto(resto) } : x)));
                }}
              >
                Resto
              </button>
              {linhas.length > 1 && (
                <button className="bs-remover" onClick={() => setLinhas((prev) => prev.filter((_, j) => j !== i))} aria-label="Tirar esta forma">
                  <XMarkIcon className="bs-icon" aria-hidden="true" />
                </button>
              )}
            </div>
          ))}
          <button className="bs-link" onClick={() => setLinhas((prev) => [...prev, { metodo: "emola", valor: "" }])}>
            <PlusIcon className="bs-icon" aria-hidden="true" />
            Outra forma de pagamento
          </button>
        </div>
      )}

      {modo === "fiado" && (
        <p className="bs-folha-texto">
          Fica tudo na conta de <strong>{nomeCliente}</strong>, em <em>Clientes e dívidas</em>.
        </p>
      )}

      {comDesconto ? (
        <label className="bs-campo bs-campo-linha">
          <span>Desconto (MT)</span>
          <input inputMode="decimal" placeholder="0" value={desconto} onChange={(e) => setDesconto(e.target.value)} autoFocus />
        </label>
      ) : (
        <button className="bs-link" onClick={() => setComDesconto(true)}>
          <PlusIcon className="bs-icon" aria-hidden="true" />
          Desconto
        </button>
      )}

      <div
        className={`bs-estado${troco > 0 ? " troco" : ""}${(ficaEmDivida && !clienteIdentificado) || excessoSemDinheiro > 0 ? " erro" : ficaEmDivida ? " divida" : ""}`}
      >
        {excessoSemDinheiro > 0 ? (
          `Pago a mais ${mt(excessoSemDinheiro)} — só há troco em dinheiro`
        ) : ficaEmDivida ? (
          clienteIdentificado ? (
            <>
              Fica em dívida <strong>{mt(falta)}</strong>
            </>
          ) : (
            `Falta ${mt(falta)} — cliente anónimo não pode ficar a dever`
          )
        ) : troco > 0 ? (
          <>
            <span>Troco</span>
            <strong>{mt(troco)}</strong>
          </>
        ) : (
          "Pagamento certo"
        )}
      </div>

      {erro && <div className="bs-aviso erro">{erro}</div>}
      {nomeArmazem && <div className="bs-folha-nota">O stock sai de {nomeArmazem}.</div>}

      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn" disabled={bloqueado} onClick={confirmar}>
          <CheckCircleIcon className="bs-icon" aria-hidden="true" />
          {aFechar ? "A fechar…" : "Confirmar"}
        </button>
      </div>
    </Folha>
  );
}

// --- Conta fechada -----------------------------------------------------------

function ContaFechada({
  identificador,
  total,
  troco,
  divida,
  onRecibo,
  onFechar,
}: {
  id: number;
  identificador: string;
  total: number;
  troco: number;
  divida: number;
  onRecibo: () => void;
  onFechar: () => void;
}) {
  return (
    <Folha titulo="" onFechar={onFechar}>
      <div className="bs-fechada">
        <CheckCircleIcon className="bs-fechada-icone" aria-hidden="true" />
        <div className="bs-fechada-titulo">{identificador} fechada</div>
        <div className="bs-fechada-total">{mt(total)}</div>
        {troco > 0 && (
          <div className="bs-fechada-troco">
            <span>Troco a dar</span>
            <strong>{mt(troco)}</strong>
          </div>
        )}
        {divida > 0 && <div className="bs-fechada-divida">Ficou em dívida {mt(divida)}</div>}
      </div>
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onRecibo}>
          <PrinterIcon className="bs-icon" aria-hidden="true" />
          Recibo
        </button>
        <button className="bs-btn" onClick={onFechar}>
          Voltar às contas
        </button>
      </div>
    </Folha>
  );
}

// --- Folha (sobe do fundo no telemóvel, janela no desktop) -------------------

function Folha({
  titulo,
  onFechar,
  larga,
  children,
}: {
  titulo: string;
  onFechar: () => void;
  larga?: boolean;
  children: React.ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onFechar();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onFechar]);
  return (
    <div className="bs-folha-fundo" onClick={onFechar}>
      <div className={`bs-folha${larga ? " larga" : ""}`} role="dialog" aria-label={titulo} onClick={(e) => e.stopPropagation()}>
        <div className="bs-folha-pega" aria-hidden="true" />
        {titulo && (
          <div className="bs-folha-cabeca">
            <span>{titulo}</span>
            <button onClick={onFechar} aria-label="Fechar">
              <XMarkIcon className="bs-icon-grande" aria-hidden="true" />
            </button>
          </div>
        )}
        <div className="bs-folha-corpo">{children}</div>
      </div>
    </div>
  );
}
