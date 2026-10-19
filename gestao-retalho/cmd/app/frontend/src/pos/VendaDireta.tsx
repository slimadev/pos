import { useEffect, useRef, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import {
  MagnifyingGlassIcon as MagnifyingGlassOutline,
  ShoppingCartIcon as ShoppingCartOutline,
  CheckCircleIcon as CheckCircleOutline,
} from "@heroicons/react/24/outline";
import {
  MagnifyingGlassIcon as MagnifyingGlassSolid,
  ShoppingCartIcon as ShoppingCartSolid,
  CheckCircleIcon as CheckCircleSolid,
} from "@heroicons/react/24/solid";
import {
  ListarArmazens,
  ListarClientes,
  RegistarVendaDireta,
  GerarRecibo,
  ListarVendasDoDia,
} from "../../wailsjs/go/api/App";
import { domain, usecase } from "../../wailsjs/go/models";
import Catalogo, { type CatalogoHandle } from "./Catalogo";
import { METODOS, ROTULO_METODO, lerNumero, mt, qtd } from "./formato";
import "./VendaDireta.css";

interface ItemCarrinho {
  chave: string;
  produtoId: number;
  produtoNome: string;
  unidadeId: number;
  unidadeNome: string;
  precoVenda: number;
  quantidade: number;
  granel: boolean;
}

interface LinhaPagamento {
  metodo: string;
  valor: string;
}

// Venda a granel (SDD 9.4, mercearia): sem tabela nova, só reutiliza
// unidade_base = kg (ou g) para aceitar peso livre em vez de contagem por
// unidade. Heurística simples — não há flag dedicada no produto.
const UNIDADES_GRANEL = ["kg", "g"];
function ehGranel(produto: domain.Produto): boolean {
  return UNIDADES_GRANEL.includes(produto.UnidadeBase.toLowerCase());
}

type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;

// Ícones sólidos para a ferragem (SDD 7.2: "ícones sólidos"), outline para
// mercearia/farmácia — este ecrã é partilhado pelos três tipos de negócio.
const ICONES_OUTLINE = { busca: MagnifyingGlassOutline, carrinho: ShoppingCartOutline, finalizar: CheckCircleOutline };
const ICONES_SOLID = { busca: MagnifyingGlassSolid, carrinho: ShoppingCartSolid, finalizar: CheckCircleSolid };

/**
 * PDV direto (SDD 4.10, tela de design `2c`) — catálogo com pesquisa por
 * nome/código de barras e categorias, carrinho lateral, F12 finaliza. Vende
 * sempre contra o armazém escolhido; internamente é uma `conta` que abre e
 * fecha na mesma transação (SDD 3.5), transparente aqui.
 */
export default function VendaDireta({ tipoNegocio }: { tipoNegocio: string }) {
  const icones: Record<"busca" | "carrinho" | "finalizar", IconeSvg> =
    tipoNegocio === "ferragem" ? ICONES_SOLID : ICONES_OUTLINE;
  const IconeCarrinho = icones.carrinho;
  const IconeFinalizar = icones.finalizar;

  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [clientes, setClientes] = useState<domain.Cliente[]>([]);
  const [armazemId, setArmazemId] = useState<number | "">("");
  const [clienteId, setClienteId] = useState<number | "">("");

  const [carrinho, setCarrinho] = useState<ItemCarrinho[]>([]);
  const [descontoGlobal, setDescontoGlobal] = useState("");
  const [pagamentos, setPagamentos] = useState<LinhaPagamento[]>([{ metodo: "dinheiro", valor: "" }]);

  const [ultimaVendaID, setUltimaVendaID] = useState<number | null>(null);
  const [historicoDia, setHistoricoDia] = useState<domain.Venda[]>([]);
  const [erro, setErro] = useState<string | null>(null);
  const [aFinalizar, setAFinalizar] = useState(false);

  const catalogoRef = useRef<CatalogoHandle>(null);

  useEffect(() => {
    ListarArmazens().then((lista) => {
      const ativos = lista.filter((a) => a.Ativo && !a.ForaDeVenda);
      setArmazens(ativos);
      // Abre no armazém principal (se estiver à venda), não no primeiro por nome.
      const inicial = ativos.find((a) => a.Tipo === "principal") ?? ativos[0];
      if (inicial) setArmazemId(inicial.ID);
    });
    ListarClientes().then(setClientes);
    ListarVendasDoDia()
      .then(setHistoricoDia)
      .catch(() => {});
  }, []);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "F12") {
        e.preventDefault();
        finalizar();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  });

  function adicionarAoCarrinho(item: usecase.ItemCatalogo, unidade: domain.UnidadeProduto) {
    const produto = item.Produto;
    const granel = ehGranel(produto);
    const chave = `${produto.ID}-${unidade.ID}`;
    setUltimaVendaID(null);
    setCarrinho((prev) => {
      if (prev.some((i) => i.chave === chave)) {
        return prev.map((i) => (i.chave === chave ? { ...i, quantidade: i.quantidade + (granel ? 0.1 : 1) } : i));
      }
      return [
        ...prev,
        {
          chave,
          produtoId: produto.ID,
          produtoNome: produto.Nome,
          unidadeId: unidade.ID,
          unidadeNome: unidade.NomeUnidade,
          precoVenda: unidade.PrecoVenda,
          // Peso inicial de 0.1kg por omissão para granel — o operador
          // ajusta ao peso real a seguir.
          quantidade: granel ? 0.1 : 1,
          granel,
        },
      ];
    });
  }

  function alterarQuantidade(chave: string, delta: number) {
    setCarrinho((prev) =>
      prev.map((i) => (i.chave === chave ? { ...i, quantidade: i.quantidade + delta } : i)).filter((i) => i.quantidade > 0),
    );
  }

  // definirQuantidade é o ponto de entrada para o peso de um item granel —
  // hoje só aceita o que o operador escreve. Uma balança eletrónica futura
  // (porta série/USB, SDD 9.4) entraria aqui.
  function definirQuantidade(chave: string, valor: string) {
    const quantidade = lerNumero(valor);
    if (!Number.isFinite(quantidade) || quantidade < 0) return;
    setCarrinho((prev) => prev.map((i) => (i.chave === chave ? { ...i, quantidade } : i)).filter((i) => i.quantidade > 0));
  }

  const subtotal = carrinho.reduce((acc, i) => acc + i.precoVenda * i.quantidade, 0);
  const desconto = lerNumero(descontoGlobal || "0") || 0;
  const total = Math.max(0, subtotal - desconto);
  const somaPagamentos = pagamentos.reduce((acc, p) => acc + (lerNumero(p.valor || "0") || 0), 0);
  const faltaAtribuir = Math.round((total - somaPagamentos) * 100) / 100;

  function atualizarPagamento(i: number, campo: keyof LinhaPagamento, valor: string) {
    setPagamentos((prev) => prev.map((p, idx) => (idx === i ? { ...p, [campo]: valor } : p)));
  }

  // Com uma só forma de pagamento, o valor acompanha o total — o caso mais
  // comum (tudo em dinheiro ou tudo em M-Pesa) fica a um F12 de distância.
  function preencherTotal(i: number) {
    const restante = Math.max(0, Math.round((total - (somaPagamentos - (lerNumero(pagamentos[i].valor || "0") || 0))) * 100) / 100);
    atualizarPagamento(i, "valor", restante.toFixed(2).replace(".", ","));
  }

  function finalizar() {
    if (carrinho.length === 0 || armazemId === "" || aFinalizar || Math.abs(faltaAtribuir) >= 0.01) return;
    setErro(null);

    const params = new usecase.VendaDiretaParams();
    params.ArmazemID = Number(armazemId);
    params.ClienteID = clienteId === "" ? undefined : Number(clienteId);
    params.DescontoTotal = desconto;
    params.Itens = carrinho.map((i) => {
      const item = new usecase.ItemVendaDiretaParams();
      item.ProdutoID = i.produtoId;
      item.UnidadeProdutoID = i.unidadeId;
      item.Quantidade = i.quantidade;
      item.Desconto = 0;
      return item;
    });
    params.Pagamentos = pagamentos
      .filter((p) => lerNumero(p.valor || "0") > 0)
      .map((p) => {
        const pag = new usecase.PagamentoVendaParams();
        pag.Metodo = p.metodo;
        pag.Valor = lerNumero(p.valor);
        return pag;
      });

    setAFinalizar(true);
    RegistarVendaDireta(params)
      .then((resultado) => {
        setUltimaVendaID(resultado.Venda.ID);
        setCarrinho([]);
        setDescontoGlobal("");
        setPagamentos([{ metodo: "dinheiro", valor: "" }]);
        ListarVendasDoDia()
          .then(setHistoricoDia)
          .catch(() => {});
        catalogoRef.current?.recarregar();
        catalogoRef.current?.focarPesquisa();
      })
      .catch((e) => setErro(String(e).replace(/^Error:\s*/, "")))
      .finally(() => setAFinalizar(false));
  }

  function gerarRecibo(vendaId: number) {
    GerarRecibo(vendaId).catch((e) => setErro(String(e)));
  }

  return (
    <div className="pos">
      <div className="pos-vendas">
        {armazemId !== "" && (
          <Catalogo
            ref={catalogoRef}
            armazemID={armazemId}
            IconeBusca={icones.busca}
            onEscolher={adicionarAoCarrinho}
            onErro={(e) => setErro(e)}
            extraToolbar={
              <>
                <select value={armazemId} onChange={(e) => setArmazemId(e.target.value ? Number(e.target.value) : "")}>
                  {armazens.map((a) => (
                    <option key={a.ID} value={a.ID}>
                      {a.Nome}
                    </option>
                  ))}
                </select>
                <select value={clienteId} onChange={(e) => setClienteId(e.target.value ? Number(e.target.value) : "")}>
                  <option value="">Cliente: Anónimo</option>
                  {clientes.map((c) => (
                    <option key={c.ID} value={c.ID}>
                      {c.Nome}
                    </option>
                  ))}
                </select>
              </>
            }
          />
        )}

        {erro && (
          <div className="pos-erro">
            {erro} <button onClick={() => setErro(null)}>×</button>
          </div>
        )}

        {ultimaVendaID && (
          <div className="pos-sucesso">
            Venda #{ultimaVendaID} registada.{" "}
            <button onClick={() => gerarRecibo(ultimaVendaID)}>Gerar recibo</button>
          </div>
        )}

        {historicoDia.length > 0 && (
          <div className="pos-historico">
            <div className="pos-historico-titulo">Vendas de hoje ({historicoDia.length})</div>
            <ul>
              {historicoDia.map((v) => (
                <li key={v.ID}>
                  #{v.ID} — {mt(v.Total)}
                  <button onClick={() => gerarRecibo(v.ID)}>Recibo</button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <aside className="pos-carrinho">
        <div className="pos-carrinho-header">
          <span className="pos-carrinho-titulo">
            <IconeCarrinho className="pos-carrinho-icon" aria-hidden="true" />
            Venda
          </span>
          <span className="pos-carrinho-count">{carrinho.length} itens</span>
        </div>

        <div className="pos-carrinho-itens">
          {carrinho.map((i) => (
            <div key={i.chave} className="pos-carrinho-item">
              {i.granel ? (
                <input
                  className="pos-peso-granel"
                  inputMode="decimal"
                  value={String(i.quantidade).replace(".", ",")}
                  onChange={(e) => definirQuantidade(i.chave, e.target.value)}
                  title="Peso em kg — escrito à mão até haver balança integrada"
                />
              ) : (
                <div className="pos-qtd-stepper">
                  <button onClick={() => alterarQuantidade(i.chave, -1)} aria-label="Menos um">
                    −
                  </button>
                  <span>{qtd(i.quantidade)}</span>
                  <button onClick={() => alterarQuantidade(i.chave, 1)} aria-label="Mais um">
                    +
                  </button>
                </div>
              )}
              <div className="pos-carrinho-item-nome">
                <div>{i.produtoNome}</div>
                <div className="pos-carrinho-item-detalhe">
                  {mt(i.precoVenda)} · {i.unidadeNome}
                </div>
              </div>
              <div className="pos-carrinho-item-total">{mt(i.precoVenda * i.quantidade)}</div>
            </div>
          ))}
          {carrinho.length === 0 && (
            <div className="pos-carrinho-vazio">Carrinho vazio — toque num produto ou leia o código de barras.</div>
          )}
        </div>

        <div className="pos-carrinho-footer">
          <label className="pos-desconto">
            Desconto (MT)
            <input inputMode="decimal" placeholder="0" value={descontoGlobal} onChange={(e) => setDescontoGlobal(e.target.value)} />
          </label>

          <div className="pos-totais">
            <div>
              <span>Subtotal</span>
              <span>{mt(subtotal)}</span>
            </div>
            <div className="pos-total-final">
              <span>Total</span>
              <span>{mt(total)}</span>
            </div>
          </div>

          <div className="pos-pagamentos">
            {pagamentos.map((p, i) => (
              <div key={i} className="pos-pagamento-linha">
                <select value={p.metodo} onChange={(e) => atualizarPagamento(i, "metodo", e.target.value)}>
                  {METODOS.map((m) => (
                    <option key={m} value={m}>
                      {ROTULO_METODO[m]}
                    </option>
                  ))}
                </select>
                <input
                  inputMode="decimal"
                  placeholder="valor"
                  value={p.valor}
                  onChange={(e) => atualizarPagamento(i, "valor", e.target.value)}
                />
                <button className="pos-tudo" onClick={() => preencherTotal(i)} title="Preencher com o que falta">
                  Tudo
                </button>
              </div>
            ))}
            <button
              className="pos-add-pagamento"
              onClick={() => setPagamentos((prev) => [...prev, { metodo: "mpesa", valor: "" }])}
            >
              + forma de pagamento
            </button>
          </div>

          <div className={`pos-falta ${Math.abs(faltaAtribuir) < 0.01 ? "pos-falta--ok" : ""}`}>
            {faltaAtribuir > 0.005
              ? `Falta pagar: ${mt(faltaAtribuir)}`
              : faltaAtribuir < -0.005
                ? `Pago a mais: ${mt(-faltaAtribuir)}`
                : "Pagamento certo"}
          </div>

          <button
            className="pos-finalizar"
            disabled={carrinho.length === 0 || Math.abs(faltaAtribuir) >= 0.01 || aFinalizar}
            onClick={finalizar}
          >
            <IconeFinalizar className="pos-finalizar-icon" aria-hidden="true" />
            {aFinalizar ? "A finalizar…" : "Finalizar venda (F12)"}
          </button>
        </div>
      </aside>
    </div>
  );
}
