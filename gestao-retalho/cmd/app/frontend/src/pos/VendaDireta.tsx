import { useEffect, useMemo, useRef, useState } from "react";
import {
  ListarProdutos,
  ListarUnidadesProduto,
  ListarArmazens,
  ListarClientes,
  ObterStock,
  RegistarVendaDireta,
  GerarRecibo,
  ListarVendasDoDia,
} from "../../wailsjs/go/api/App";
import { domain, usecase } from "../../wailsjs/go/models";
import "./VendaDireta.css";

interface ItemCarrinho {
  produtoId: number;
  produtoNome: string;
  unidadeId: number;
  unidadeNome: string;
  precoVenda: number;
  quantidade: number;
}

interface LinhaPagamento {
  metodo: string;
  valor: string;
}

const METODOS = ["dinheiro", "mpesa", "emola", "cartao", "transferencia"];

/**
 * PDV direto (SDD 4.10, tela de design `2c`) — pesquisa/leitor sempre
 * focado, grelha de produtos, carrinho lateral, F12 finaliza. Vende
 * sempre contra o armazém escolhido; internamente é uma `conta` que abre
 * e fecha na mesma transação (SDD 3.5), transparente aqui.
 */
export default function VendaDireta() {
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [unidadesPadrao, setUnidadesPadrao] = useState<Map<number, domain.UnidadeProduto>>(
    new Map(),
  );
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [clientes, setClientes] = useState<domain.Cliente[]>([]);
  const [armazemId, setArmazemId] = useState<number | "">("");
  const [clienteId, setClienteId] = useState<number | "">("");
  const [busca, setBusca] = useState("");
  const [stockPorProduto, setStockPorProduto] = useState<Map<number, number>>(new Map());

  const [carrinho, setCarrinho] = useState<ItemCarrinho[]>([]);
  const [descontoGlobal, setDescontoGlobal] = useState("0");
  const [pagamentos, setPagamentos] = useState<LinhaPagamento[]>([
    { metodo: "dinheiro", valor: "" },
  ]);

  const [ultimaVendaID, setUltimaVendaID] = useState<number | null>(null);
  const [historicoDia, setHistoricoDia] = useState<domain.Venda[]>([]);
  const [erro, setErro] = useState<string | null>(null);
  const [aFinalizar, setAFinalizar] = useState(false);

  const buscaRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    ListarProdutos().then(async (lista) => {
      setProdutos(lista);
      const entries = await Promise.all(
        lista.map(async (p) => {
          const unidades = await ListarUnidadesProduto(p.ID);
          const padrao = unidades.find((u) => u.IsPadraoVenda) ?? unidades[0];
          return [p.ID, padrao] as const;
        }),
      );
      setUnidadesPadrao(new Map(entries.filter(([, u]) => u)));
    });
    ListarArmazens().then((lista) => {
      setArmazens(lista);
      if (lista.length > 0) setArmazemId(lista[0].ID);
    });
    ListarClientes().then(setClientes);
    ListarVendasDoDia()
      .then(setHistoricoDia)
      .catch(() => {});
  }, []);

  useEffect(() => {
    buscaRef.current?.focus();
  }, []);

  useEffect(() => {
    if (armazemId === "") return;
    Promise.all(
      produtos.map(
        async (p) => [p.ID, (await ObterStock(p.ID, Number(armazemId))).Quantidade] as const,
      ),
    ).then((entries) => setStockPorProduto(new Map(entries)));
  }, [armazemId, produtos]);

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

  const produtosFiltrados = useMemo(() => {
    const termo = busca.trim().toLowerCase();
    if (!termo) return produtos;
    return produtos.filter((p) => p.Nome.toLowerCase().includes(termo));
  }, [produtos, busca]);

  function adicionarAoCarrinho(produto: domain.Produto) {
    const unidade = unidadesPadrao.get(produto.ID);
    if (!unidade) return;
    setCarrinho((prev) => {
      const existente = prev.find((i) => i.produtoId === produto.ID);
      if (existente) {
        return prev.map((i) =>
          i.produtoId === produto.ID ? { ...i, quantidade: i.quantidade + 1 } : i,
        );
      }
      return [
        ...prev,
        {
          produtoId: produto.ID,
          produtoNome: produto.Nome,
          unidadeId: unidade.ID,
          unidadeNome: unidade.NomeUnidade,
          precoVenda: unidade.PrecoVenda,
          quantidade: 1,
        },
      ];
    });
  }

  function alterarQuantidade(produtoId: number, delta: number) {
    setCarrinho((prev) =>
      prev
        .map((i) => (i.produtoId === produtoId ? { ...i, quantidade: i.quantidade + delta } : i))
        .filter((i) => i.quantidade > 0),
    );
  }

  const subtotal = carrinho.reduce((acc, i) => acc + i.precoVenda * i.quantidade, 0);
  const total = Math.max(0, subtotal - Number(descontoGlobal || 0));
  const somaPagamentos = pagamentos.reduce((acc, p) => acc + Number(p.valor || 0), 0);
  const faltaAtribuir = Math.round((total - somaPagamentos) * 100) / 100;

  function atualizarPagamento(i: number, campo: keyof LinhaPagamento, valor: string) {
    setPagamentos((prev) => prev.map((p, idx) => (idx === i ? { ...p, [campo]: valor } : p)));
  }

  function finalizar() {
    if (carrinho.length === 0 || armazemId === "" || aFinalizar) return;
    setErro(null);

    const params = new usecase.VendaDiretaParams();
    params.ArmazemID = Number(armazemId);
    params.ClienteID = clienteId === "" ? undefined : Number(clienteId);
    params.DescontoTotal = Number(descontoGlobal || 0);
    params.UtilizadorID = 0; // preenchido no backend (SDD: sessão real só na Fase 7)
    params.Itens = carrinho.map((i) => {
      const item = new usecase.ItemVendaDiretaParams();
      item.ProdutoID = i.produtoId;
      item.UnidadeProdutoID = i.unidadeId;
      item.Quantidade = i.quantidade;
      item.Desconto = 0;
      return item;
    });
    params.Pagamentos = pagamentos
      .filter((p) => Number(p.valor) > 0)
      .map((p) => {
        const pag = new usecase.PagamentoVendaParams();
        pag.Metodo = p.metodo;
        pag.Valor = Number(p.valor);
        return pag;
      });

    setAFinalizar(true);
    RegistarVendaDireta(params)
      .then((resultado) => {
        setUltimaVendaID(resultado.Venda.ID);
        setCarrinho([]);
        setDescontoGlobal("0");
        setPagamentos([{ metodo: "dinheiro", valor: "" }]);
        ListarVendasDoDia()
          .then(setHistoricoDia)
          .catch(() => {});
        buscaRef.current?.focus();
      })
      .catch((e) => setErro(String(e)))
      .finally(() => setAFinalizar(false));
  }

  function gerarRecibo(vendaId: number) {
    GerarRecibo(vendaId).catch((e) => setErro(String(e)));
  }

  return (
    <div className="pos">
      <div className="pos-vendas">
        <div className="pos-toolbar">
          <input
            ref={buscaRef}
            className="pos-busca"
            placeholder="Ler código de barras ou escrever nome do artigo (F2)"
            value={busca}
            onChange={(e) => setBusca(e.target.value)}
          />
          <select
            value={armazemId}
            onChange={(e) => setArmazemId(e.target.value ? Number(e.target.value) : "")}
          >
            {armazens.map((a) => (
              <option key={a.ID} value={a.ID}>
                {a.Nome}
              </option>
            ))}
          </select>
          <select
            value={clienteId}
            onChange={(e) => setClienteId(e.target.value ? Number(e.target.value) : "")}
          >
            <option value="">Cliente: Anónimo</option>
            {clientes.map((c) => (
              <option key={c.ID} value={c.ID}>
                {c.Nome}
              </option>
            ))}
          </select>
        </div>

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

        <div className="pos-grid">
          {produtosFiltrados.map((p) => {
            const stock = stockPorProduto.get(p.ID);
            const unidade = unidadesPadrao.get(p.ID);
            return (
              <div key={p.ID} className="pos-card" onClick={() => adicionarAoCarrinho(p)}>
                <div className="pos-card-top">
                  <div className="pos-card-nome">{p.Nome}</div>
                  {stock !== undefined && (
                    <div className={`pos-card-stock ${stock <= 0 ? "pos-card-stock--zero" : ""}`}>
                      {stock}
                    </div>
                  )}
                </div>
                <div className="pos-card-bottom">
                  <div className="pos-card-preco">{unidade ? `${unidade.PrecoVenda} MT` : "—"}</div>
                  <div className="pos-card-unidade">{unidade?.NomeUnidade}</div>
                </div>
              </div>
            );
          })}
        </div>

        {historicoDia.length > 0 && (
          <div className="pos-historico">
            <div className="pos-historico-titulo">
              Histórico de vendas do dia ({historicoDia.length})
            </div>
            <ul>
              {historicoDia.map((v) => (
                <li key={v.ID}>
                  #{v.ID} — {v.Total.toFixed(2)} MT
                  <button onClick={() => gerarRecibo(v.ID)}>Recibo</button>
                </li>
              ))}
            </ul>
          </div>
        )}
      </div>

      <aside className="pos-carrinho">
        <div className="pos-carrinho-header">
          <span>Venda</span>
          <span className="pos-carrinho-count">{carrinho.length} itens</span>
        </div>

        <div className="pos-carrinho-itens">
          {carrinho.map((i) => (
            <div key={i.produtoId} className="pos-carrinho-item">
              <div className="pos-qtd-stepper">
                <button onClick={() => alterarQuantidade(i.produtoId, -1)}>−</button>
                <span>{i.quantidade}</span>
                <button onClick={() => alterarQuantidade(i.produtoId, 1)}>+</button>
              </div>
              <div className="pos-carrinho-item-nome">
                <div>{i.produtoNome}</div>
                <div className="pos-carrinho-item-detalhe">
                  {i.precoVenda} MT · {i.unidadeNome}
                </div>
              </div>
              <div className="pos-carrinho-item-total">
                {(i.precoVenda * i.quantidade).toFixed(2)}
              </div>
            </div>
          ))}
          {carrinho.length === 0 && <div className="pos-carrinho-vazio">Carrinho vazio</div>}
        </div>

        <div className="pos-carrinho-footer">
          <label className="pos-desconto">
            Desconto (MT)
            <input
              type="number"
              value={descontoGlobal}
              onChange={(e) => setDescontoGlobal(e.target.value)}
            />
          </label>

          <div className="pos-totais">
            <div>
              <span>Subtotal</span>
              <span>{subtotal.toFixed(2)} MT</span>
            </div>
            <div className="pos-total-final">
              <span>Total</span>
              <span>{total.toFixed(2)} MT</span>
            </div>
          </div>

          <div className="pos-pagamentos">
            {pagamentos.map((p, i) => (
              <div key={i} className="pos-pagamento-linha">
                <select
                  value={p.metodo}
                  onChange={(e) => atualizarPagamento(i, "metodo", e.target.value)}
                >
                  {METODOS.map((m) => (
                    <option key={m} value={m}>
                      {m}
                    </option>
                  ))}
                </select>
                <input
                  type="number"
                  placeholder="valor"
                  value={p.valor}
                  onChange={(e) => atualizarPagamento(i, "valor", e.target.value)}
                />
              </div>
            ))}
            <button
              className="pos-add-pagamento"
              onClick={() => setPagamentos((prev) => [...prev, { metodo: "dinheiro", valor: "" }])}
            >
              + forma de pagamento
            </button>
          </div>

          <div className={`pos-falta ${Math.abs(faltaAtribuir) < 0.01 ? "pos-falta--ok" : ""}`}>
            Falta atribuir: {faltaAtribuir.toFixed(2)} MT
          </div>

          <button
            className="pos-finalizar"
            disabled={carrinho.length === 0 || Math.abs(faltaAtribuir) >= 0.01 || aFinalizar}
            onClick={finalizar}
          >
            {aFinalizar ? "A finalizar…" : "Finalizar venda (F12)"}
          </button>
        </div>
      </aside>
    </div>
  );
}
