import { useEffect, useState } from "react";
import {
  ListarContasAbertas,
  ListarItensConta,
  AbrirConta,
  AdicionarItemConta,
  TransferirItemConta,
  CancelarConta,
  FecharContaAberta,
  ListarProdutos,
  ListarUnidadesProduto,
  ListarArmazens,
  ListarClientes,
  GerarRecibo,
} from "../../wailsjs/go/api/App";
import { api, domain, usecase } from "../../wailsjs/go/models";
import "./Bottlestore.css";

const METODOS = ["dinheiro", "mpesa", "emola", "cartao", "transferencia"];

interface ContaComTotal {
  conta: domain.Conta;
  itens: domain.ItemConta[];
  total: number;
}

/**
 * Contas abertas (SDD 4.10, telas de design `2a`/`2b`) — lista de comandas
 * tipo cartão, e ao selecionar uma, grelha de produtos + comanda viva +
 * fecho com split de pagamento e dívida opcional (cliente identificado).
 */
export default function Bottlestore() {
  const [contas, setContas] = useState<ContaComTotal[]>([]);
  const [clientes, setClientes] = useState<domain.Cliente[]>([]);
  const [contaSelecionadaId, setContaSelecionadaId] = useState<number | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const [novoIdentificador, setNovoIdentificador] = useState("");
  const [novoClienteId, setNovoClienteId] = useState<number | "">("");

  const recarregarContas = () => {
    ListarContasAbertas().then(async (lista) => {
      const comTotais = await Promise.all(
        lista.map(async (conta) => {
          const itens = await ListarItensConta(conta.ID);
          const total = itens.reduce((acc, i) => acc + i.Quantidade * i.PrecoUnitarioAplicado - i.Desconto, 0);
          return { conta, itens, total };
        }),
      );
      setContas(comTotais);
    });
  };

  useEffect(() => {
    recarregarContas();
    ListarClientes().then(setClientes);
  }, []);

  function abrirConta() {
    if (!novoIdentificador.trim()) return;
    const req = new api.AbrirContaRequest();
    req.Identificador = novoIdentificador.trim();
    req.ClienteID = novoClienteId === "" ? undefined : Number(novoClienteId);
    AbrirConta(req)
      .then((conta) => {
        setNovoIdentificador("");
        setNovoClienteId("");
        recarregarContas();
        setContaSelecionadaId(conta.ID);
      })
      .catch((e) => setErro(String(e)));
  }

  const contaSelecionada = contas.find((c) => c.conta.ID === contaSelecionadaId) ?? null;

  if (contaSelecionada) {
    return (
      <ContaDetalhe
        contaComTotal={contaSelecionada}
        clientes={clientes}
        outrasContasAbertas={contas.filter((c) => c.conta.ID !== contaSelecionadaId).map((c) => c.conta)}
        onVoltar={() => {
          setContaSelecionadaId(null);
          recarregarContas();
        }}
        onMudou={recarregarContas}
        onErro={setErro}
      />
    );
  }

  const totalEmConsumo = contas.reduce((acc, c) => acc + c.total, 0);

  return (
    <div className="bs">
      {erro && (
        <div className="bs-erro">
          {erro} <button onClick={() => setErro(null)}>×</button>
        </div>
      )}

      <div className="bs-kpis">
        <div>
          <span className="bs-kpi-valor">{contas.length}</span>
          <span className="bs-kpi-label">contas abertas</span>
        </div>
        <div>
          <span className="bs-kpi-valor">{totalEmConsumo.toFixed(0)} MT</span>
          <span className="bs-kpi-label">em consumo</span>
        </div>
      </div>

      <div className="bs-nova">
        <input
          placeholder="Identificador (ex: Mesa 3, Grupo do João)"
          value={novoIdentificador}
          onChange={(e) => setNovoIdentificador(e.target.value)}
        />
        <select value={novoClienteId} onChange={(e) => setNovoClienteId(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Cliente: Anónimo</option>
          {clientes.map((c) => (
            <option key={c.ID} value={c.ID}>
              {c.Nome}
            </option>
          ))}
        </select>
        <button disabled={!novoIdentificador.trim()} onClick={abrirConta}>
          + Nova conta
        </button>
      </div>

      <div className="bs-grid">
        {contas.map(({ conta, itens, total }) => (
          <div key={conta.ID} className="bs-card" onClick={() => setContaSelecionadaId(conta.ID)}>
            <div className="bs-card-top">
              <div className="bs-card-id">{conta.Identificador}</div>
              <div className="bs-card-cliente">{clientes.find((c) => c.ID === conta.ClienteID)?.Nome ?? "Anónimo"}</div>
            </div>
            <div className="bs-card-bottom">
              <div className="bs-card-total">{total.toFixed(0)} MT</div>
              <div className="bs-card-itens">{itens.length} itens</div>
            </div>
          </div>
        ))}
        {contas.length === 0 && <div className="bs-vazio">Nenhuma conta aberta — abra uma acima.</div>}
      </div>
    </div>
  );
}

function ContaDetalhe({
  contaComTotal,
  clientes,
  outrasContasAbertas,
  onVoltar,
  onMudou,
  onErro,
}: {
  contaComTotal: ContaComTotal;
  clientes: domain.Cliente[];
  outrasContasAbertas: domain.Conta[];
  onVoltar: () => void;
  onMudou: () => void;
  onErro: (e: string) => void;
}) {
  const { conta, itens } = contaComTotal;
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [unidadesPadrao, setUnidadesPadrao] = useState<Map<number, domain.UnidadeProduto>>(new Map());
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [armazemId, setArmazemId] = useState<number | "">("");
  const [itensAtual, setItensAtual] = useState(itens);
  const [mostrarFecho, setMostrarFecho] = useState(false);

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
  }, []);

  function recarregarItens() {
    ListarItensConta(conta.ID).then(setItensAtual);
  }

  function adicionar(produto: domain.Produto) {
    const unidade = unidadesPadrao.get(produto.ID);
    if (!unidade) return;
    const p = new usecase.AdicionarItemContaParams();
    p.ContaID = conta.ID;
    p.ProdutoID = produto.ID;
    p.UnidadeProdutoID = unidade.ID;
    p.Quantidade = 1;
    p.Desconto = 0;
    AdicionarItemConta(p).then(recarregarItens).catch((e) => onErro(String(e)));
  }

  function transferir(itemId: number, novaContaId: number) {
    TransferirItemConta(itemId, novaContaId)
      .then(() => {
        recarregarItens();
        onMudou();
      })
      .catch((e) => onErro(String(e)));
  }

  function cancelar() {
    const motivo = window.prompt("Motivo do cancelamento:");
    if (!motivo) return;
    CancelarConta(conta.ID, motivo)
      .then(() => {
        onMudou();
        onVoltar();
      })
      .catch((e) => onErro(String(e)));
  }

  const nomeProduto = (id: number) => produtos.find((p) => p.ID === id)?.Nome ?? `#${id}`;
  const total = itensAtual.reduce((acc, i) => acc + i.Quantidade * i.PrecoUnitarioAplicado - i.Desconto, 0);

  return (
    <div className="bs">
      <div className="bs-detalhe-header">
        <button className="bs-voltar" onClick={onVoltar}>
          ← Contas
        </button>
        <div className="bs-detalhe-titulo">
          <div>{conta.Identificador}</div>
          <div className="bs-detalhe-sub">{clientes.find((c) => c.ID === conta.ClienteID)?.Nome ?? "Anónimo"}</div>
        </div>
        <select value={armazemId} onChange={(e) => setArmazemId(e.target.value ? Number(e.target.value) : "")}>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
        <button className="bs-cancelar" onClick={cancelar}>
          Cancelar conta
        </button>
      </div>

      <div className="bs-detalhe-body">
        <div className="bs-produtos">
          {produtos.map((p) => {
            const unidade = unidadesPadrao.get(p.ID);
            return (
              <div key={p.ID} className="bs-produto-card" onClick={() => adicionar(p)}>
                <div className="bs-produto-nome">{p.Nome}</div>
                <div className="bs-produto-preco">{unidade ? `${unidade.PrecoVenda} MT` : "—"}</div>
              </div>
            );
          })}
        </div>

        <div className="bs-comanda">
          <div className="bs-comanda-header">
            <span>Comanda</span>
            <span>{itensAtual.length} itens</span>
          </div>
          <div className="bs-comanda-itens">
            {itensAtual.map((i) => (
              <div key={i.ID} className="bs-comanda-item">
                <div className="bs-comanda-item-nome">
                  {i.Quantidade}x {nomeProduto(i.ProdutoID)}
                </div>
                <div className="bs-comanda-item-total">{(i.Quantidade * i.PrecoUnitarioAplicado - i.Desconto).toFixed(2)}</div>
                {outrasContasAbertas.length > 0 && (
                  <select
                    className="bs-transferir"
                    defaultValue=""
                    onChange={(e) => e.target.value && transferir(i.ID, Number(e.target.value))}
                  >
                    <option value="">Transferir…</option>
                    {outrasContasAbertas.map((c) => (
                      <option key={c.ID} value={c.ID}>
                        {c.Identificador}
                      </option>
                    ))}
                  </select>
                )}
              </div>
            ))}
            {itensAtual.length === 0 && <div className="bs-comanda-vazia">Sem consumo lançado</div>}
          </div>
          <div className="bs-comanda-footer">
            <div className="bs-comanda-total">
              <span>Total</span>
              <span>{total.toFixed(2)} MT</span>
            </div>
            <button
              className="bs-fechar"
              disabled={itensAtual.length === 0 || armazemId === ""}
              onClick={() => setMostrarFecho(true)}
            >
              Fechar conta
            </button>
          </div>
        </div>
      </div>

      {mostrarFecho && armazemId !== "" && (
        <ModalFecho
          conta={conta}
          total={total}
          armazemId={armazemId}
          // ListarClientes nunca devolve o registo fixo "Anónimo" (filtrado
          // no backend) — não vir na lista é como sabemos que é ele.
          clienteAnonimo={!clientes.some((c) => c.ID === conta.ClienteID)}
          onFechar={(vendaId) => {
            setMostrarFecho(false);
            onMudou();
            onVoltar();
            if (vendaId) GerarRecibo(vendaId).catch(() => {});
          }}
          onCancelar={() => setMostrarFecho(false)}
          onErro={onErro}
        />
      )}
    </div>
  );
}

function ModalFecho({
  conta,
  total,
  armazemId,
  clienteAnonimo,
  onFechar,
  onCancelar,
  onErro,
}: {
  conta: domain.Conta;
  total: number;
  armazemId: number;
  clienteAnonimo: boolean;
  onFechar: (vendaId: number | null) => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [pagamentos, setPagamentos] = useState([{ metodo: "dinheiro", valor: String(total) }]);
  const [aFechar, setAFechar] = useState(false);

  const somaPagamentos = pagamentos.reduce((acc, p) => acc + Number(p.valor || 0), 0);
  const faltaAtribuir = Math.round((total - somaPagamentos) * 100) / 100;
  const iriaGerarDivida = faltaAtribuir > 0.01;

  function atualizar(i: number, campo: "metodo" | "valor", valor: string) {
    setPagamentos((prev) => prev.map((p, idx) => (idx === i ? { ...p, [campo]: valor } : p)));
  }

  function confirmar() {
    if (iriaGerarDivida && clienteAnonimo) {
      onErro("Cliente anónimo não pode ficar com saldo em dívida — identifique o cliente ou complete o pagamento.");
      return;
    }
    const p = new usecase.FecharContaParams();
    p.ArmazemID = armazemId;
    p.DescontoTotal = 0;
    p.UtilizadorID = 0;
    p.Pagamentos = pagamentos
      .filter((pg) => Number(pg.valor) > 0)
      .map((pg) => {
        const pag = new usecase.PagamentoVendaParams();
        pag.Metodo = pg.metodo;
        pag.Valor = Number(pg.valor);
        return pag;
      });

    setAFechar(true);
    FecharContaAberta(conta.ID, p)
      .then((resultado) => {
        if (resultado.Alerta?.Excedido) {
          window.alert(
            `Aviso: cliente ultrapassa o limite de crédito (${resultado.Alerta.LimiteCredito.toFixed(2)} MT) — total em dívida agora ${resultado.Alerta.TotalEmAberto.toFixed(2)} MT.`,
          );
        }
        onFechar(resultado.Venda.Venda.ID);
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setAFechar(false));
  }

  return (
    <div className="bs-modal-overlay">
      <div className="bs-modal">
        <div className="bs-modal-header">
          <div>Fechar conta — {conta.Identificador}</div>
          <div className="bs-modal-total">{total.toFixed(2)} MT</div>
        </div>

        <div className="bs-modal-pagamentos">
          {pagamentos.map((p, i) => (
            <div key={i} className="bs-modal-linha">
              <select value={p.metodo} onChange={(e) => atualizar(i, "metodo", e.target.value)}>
                {METODOS.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
              <input type="number" value={p.valor} onChange={(e) => atualizar(i, "valor", e.target.value)} />
            </div>
          ))}
          <button onClick={() => setPagamentos((prev) => [...prev, { metodo: "dinheiro", valor: "" }])}>
            + forma de pagamento
          </button>
        </div>

        <div className={`bs-modal-falta ${iriaGerarDivida ? "bs-modal-falta--divida" : ""}`}>
          {iriaGerarDivida
            ? clienteAnonimo
              ? `Falta atribuir ${faltaAtribuir.toFixed(2)} MT — cliente anónimo não pode ficar em dívida`
              : `Falta atribuir ${faltaAtribuir.toFixed(2)} MT — vai gerar dívida para o cliente`
            : "Falta atribuir 0.00 MT"}
        </div>

        <div className="bs-modal-actions">
          <button onClick={onCancelar}>Cancelar</button>
          <button
            className="bs-modal-confirmar"
            disabled={aFechar || somaPagamentos > total + 0.01 || (iriaGerarDivida && clienteAnonimo)}
            onClick={confirmar}
          >
            {aFechar ? "A confirmar…" : "Confirmar pagamento"}
          </button>
        </div>
      </div>
    </div>
  );
}
