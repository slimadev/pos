import { useEffect, useState } from "react";
import { CheckCircleIcon, ExclamationTriangleIcon, PrinterIcon, TrashIcon } from "@heroicons/react/24/outline";
import {
  ListarArmazens,
  ListarClientes,
  CatalogoVenda,
  RegistarVendaACredito,
  AbrirRecibo,
} from "../wailsjs/go/admin/AdminApp";
import { admin, domain, usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import type { Navegar } from "./App";

const mt = (n: number) => `${n.toLocaleString("pt-PT", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} MT`;
const num = (n: number) => n.toLocaleString("pt-PT", { maximumFractionDigits: 3 });
const lerNumero = (s: string) => Number(s.replace(",", "."));

const ROTULO_METODO: Record<string, string> = {
  dinheiro: "Dinheiro",
  mpesa: "M-Pesa",
  emola: "e-Mola",
  cartao: "Cartão",
  transferencia: "Transferência",
};

type Linha = { chave: number; item: admin.ItemCatalogo; unidadeID: number; quantidade: string };
let proximaChave = 0;

/**
 * Venda a crédito — só existe no cmd/admin, atrás do login do
 * administrador; o PDV continua a exigir o pagamento total. O cliente leva
 * os produtos e paga depois: a entrada é opcional e o resto fica como
 * dívida, que se cobra em "Clientes e dívidas". O limite de crédito só
 * avisa, nunca bloqueia (como no fecho de uma conta).
 */
export default function VendaCreditoScreen({
  cliente,
  origem,
  onNavegar,
  onErro,
}: {
  cliente?: { ID: number; Nome: string };
  origem: "vendas" | "clientes";
  onNavegar: Navegar;
  onErro: (e: string) => void;
}) {
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [clientes, setClientes] = useState<admin.ClienteResumo[]>([]);
  const [resultado, setResultado] = useState<admin.ResultadoVendaCredito | null>(null);
  const [nova, setNova] = useState(0);

  useEffect(() => {
    ListarArmazens()
      .then((lista) => {
        const ativos = lista.filter((a) => a.Ativo);
        setArmazens(ativos);
        if (ativos.length > 0) setArmazemID(ativos[0].ID);
      })
      .catch((e) => onErro(String(e)));
    ListarClientes()
      .then(setClientes)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  function recomecar() {
    setResultado(null);
    setNova((n) => n + 1);
    ListarClientes()
      .then(setClientes)
      .catch((e) => onErro(String(e)));
  }

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb
        itens={
          origem === "clientes"
            ? [
                { rotulo: "Clientes e dívidas", onClick: () => onNavegar({ ecra: "clientes" }) },
                ...(cliente ? [{ rotulo: cliente.Nome, onClick: () => onNavegar({ ecra: "clientes", cliente }) }] : []),
                { rotulo: "Venda a crédito" },
              ]
            : [{ rotulo: "Vendas", onClick: () => onNavegar({ ecra: "vendas" }) }, { rotulo: "Venda a crédito" }]
        }
      />
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>Venda a crédito</h2>
          <div style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
            O cliente leva agora e paga depois. O que não for pago fica em dívida.
          </div>
        </div>
        {!resultado && (
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
        )}
      </div>

      {resultado ? (
        <Concluida
          resultado={resultado}
          clientes={clientes}
          onNova={recomecar}
          onVerCliente={(c) => onNavegar({ ecra: "clientes", cliente: c })}
          onErro={onErro}
        />
      ) : armazemID === "" ? (
        <p style={{ color: "var(--text-secondary)" }}>Nenhum armazém ativo.</p>
      ) : (
        <NovaVenda
          key={`${armazemID}-${nova}`}
          armazemID={armazemID}
          clientes={clientes.filter((c) => c.Cliente.Ativo)}
          clienteInicial={cliente?.ID}
          onConcluida={setResultado}
          onErro={onErro}
        />
      )}
    </div>
  );
}

function NovaVenda({
  armazemID,
  clientes,
  clienteInicial,
  onConcluida,
  onErro,
}: {
  armazemID: number;
  clientes: admin.ClienteResumo[];
  clienteInicial?: number;
  onConcluida: (r: admin.ResultadoVendaCredito) => void;
  onErro: (e: string) => void;
}) {
  const [catalogo, setCatalogo] = useState<admin.ItemCatalogo[] | null>(null);
  const [pesquisa, setPesquisa] = useState("");
  const [clienteID, setClienteID] = useState<number | "">(clienteInicial ?? "");
  const [linhas, setLinhas] = useState<Linha[]>([]);
  const [entrada, setEntrada] = useState("");
  const [metodo, setMetodo] = useState("dinheiro");
  const [aGuardar, setAGuardar] = useState(false);

  useEffect(() => {
    CatalogoVenda(armazemID)
      .then(setCatalogo)
      .catch((e) => {
        setCatalogo([]);
        onErro(String(e));
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- armazém fixo por instância (key)
  }, []);

  const termo = pesquisa.trim().toLowerCase();
  const visiveis = (catalogo ?? [])
    .filter((i) => !termo || i.Produto.Nome.toLowerCase().includes(termo))
    .sort((a, b) => a.Produto.Nome.localeCompare(b.Produto.Nome, "pt"));

  const unidadeDe = (l: Linha) => l.item.Unidades.find((u) => u.ID === l.unidadeID) ?? l.item.Unidades[0];
  const subtotalDe = (l: Linha) => {
    const q = lerNumero(l.quantidade);
    return Number.isNaN(q) ? 0 : q * unidadeDe(l).PrecoVenda;
  };
  // Quanto de stock (em unidade base) as linhas deste produto já pedem.
  const pedidoBase = (produtoID: number) =>
    linhas
      .filter((l) => l.item.Produto.ID === produtoID)
      .reduce((s, l) => s + (lerNumero(l.quantidade) || 0) * unidadeDe(l).QtdEmUnidadeBase, 0);
  const semStock = (l: Linha) =>
    l.item.Produto.ControlaStock && pedidoBase(l.item.Produto.ID) > l.item.Stock + 1e-9;

  const total = linhas.reduce((s, l) => s + subtotalDe(l), 0);
  const entradaNum = entrada.trim() === "" ? 0 : lerNumero(entrada);
  const entradaValida = !Number.isNaN(entradaNum) && entradaNum >= 0 && entradaNum <= total + 0.005;
  const emDivida = Math.max(0, total - (entradaValida ? entradaNum : 0));

  const cliente = clientes.find((c) => c.Cliente.ID === clienteID);
  const limite = cliente?.Cliente.LimiteCredito;
  const ficaraEmAberto = (cliente?.TotalEmAberto ?? 0) + emDivida;
  const ultrapassaLimite = limite != null && ficaraEmAberto > limite + 0.005;

  const quantidadesValidas = linhas.every((l) => lerNumero(l.quantidade) > 0);
  const podeRegistar =
    clienteID !== "" && linhas.length > 0 && quantidadesValidas && entradaValida && !linhas.some(semStock) && !aGuardar;

  function adicionar(item: admin.ItemCatalogo) {
    const padrao = item.Unidades.find((u) => u.IsPadraoVenda) ?? item.Unidades[0];
    const existente = linhas.find((l) => l.item.Produto.ID === item.Produto.ID && l.unidadeID === padrao.ID);
    if (existente) {
      atualizar(existente.chave, { quantidade: String((lerNumero(existente.quantidade) || 0) + 1) });
      return;
    }
    setLinhas((ls) => [...ls, { chave: ++proximaChave, item, unidadeID: padrao.ID, quantidade: "1" }]);
  }

  function atualizar(chave: number, alteracoes: Partial<Linha>) {
    setLinhas((ls) => ls.map((l) => (l.chave === chave ? { ...l, ...alteracoes } : l)));
  }

  function registar() {
    if (!podeRegistar) return;
    const p = new usecase.VendaDiretaParams();
    p.ArmazemID = armazemID;
    p.ClienteID = clienteID;
    p.Itens = linhas.map((l) => {
      const i = new usecase.ItemVendaDiretaParams();
      i.ProdutoID = l.item.Produto.ID;
      i.UnidadeProdutoID = unidadeDe(l).ID;
      i.Quantidade = lerNumero(l.quantidade);
      i.Desconto = 0;
      return i;
    });
    p.Pagamentos = [];
    if (entradaNum > 0) {
      const pag = new usecase.PagamentoVendaParams();
      pag.Metodo = metodo;
      pag.Valor = entradaNum;
      p.Pagamentos.push(pag);
    }
    setAGuardar(true);
    RegistarVendaACredito(p)
      .then(onConcluida)
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-credito">
      <section className="admin-credito-catalogo">
        <input placeholder="Pesquisar produto…" value={pesquisa} onChange={(e) => setPesquisa(e.target.value)} />
        <div className="admin-credito-lista">
          {catalogo === null && <p>A carregar…</p>}
          {catalogo !== null && visiveis.length === 0 && (
            <p style={{ color: "var(--text-secondary)" }}>Nenhum produto encontrado.</p>
          )}
          {visiveis.map((i) => {
            const padrao = i.Unidades.find((u) => u.IsPadraoVenda) ?? i.Unidades[0];
            const esgotado = i.Produto.ControlaStock && i.Stock <= 0;
            return (
              <button
                key={i.Produto.ID}
                className="admin-credito-produto"
                disabled={esgotado}
                onClick={() => adicionar(i)}
              >
                <span className="nome">{i.Produto.Nome}</span>
                <span className="preco">
                  {mt(padrao.PrecoVenda)} / {padrao.NomeUnidade}
                </span>
                <span className={`stock${esgotado ? " esgotado" : ""}`}>
                  {!i.Produto.ControlaStock
                    ? "composto"
                    : esgotado
                      ? "sem stock"
                      : `${num(i.Stock)} ${i.Produto.UnidadeBase} em stock`}
                </span>
              </button>
            );
          })}
        </div>
      </section>

      <section className="admin-credito-venda">
        <label className="admin-campo">
          Cliente *
          <select value={clienteID} onChange={(e) => setClienteID(e.target.value ? Number(e.target.value) : "")}>
            <option value="">Escolha o cliente…</option>
            {[...clientes]
              .sort((a, b) => a.Cliente.Nome.localeCompare(b.Cliente.Nome, "pt"))
              .map((c) => (
                <option key={c.Cliente.ID} value={c.Cliente.ID}>
                  {c.Cliente.Nome}
                  {c.TotalEmAberto > 0 ? ` — deve ${mt(c.TotalEmAberto)}` : ""}
                </option>
              ))}
          </select>
        </label>
        {clientes.length === 0 && (
          <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
            Não há clientes ativos — registe o cliente em "Clientes e dívidas" primeiro.
          </p>
        )}
        {cliente && (
          <div className="admin-credito-cliente">
            <span>
              Deve hoje <strong>{mt(cliente.TotalEmAberto)}</strong>
            </span>
            <span>
              Limite <strong>{limite != null ? mt(limite) : "sem limite"}</strong>
            </span>
          </div>
        )}

        <table className="admin-tabela">
          <thead>
            <tr>
              <th>Produto</th>
              <th>Unidade</th>
              <th style={{ textAlign: "right" }}>Qtd</th>
              <th style={{ textAlign: "right" }}>Subtotal</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {linhas.length === 0 && (
              <tr>
                <td colSpan={5} style={{ color: "var(--text-secondary)" }}>
                  Escolha os produtos à esquerda.
                </td>
              </tr>
            )}
            {linhas.map((l) => (
              <tr key={l.chave}>
                <td>
                  {l.item.Produto.Nome}
                  {semStock(l) && <div className="admin-inventario-dif falta">só há {num(l.item.Stock)} {l.item.Produto.UnidadeBase}</div>}
                </td>
                <td>
                  {l.item.Unidades.length > 1 ? (
                    <select value={l.unidadeID} onChange={(e) => atualizar(l.chave, { unidadeID: Number(e.target.value) })}>
                      {l.item.Unidades.map((u) => (
                        <option key={u.ID} value={u.ID}>
                          {u.NomeUnidade} ({mt(u.PrecoVenda)})
                        </option>
                      ))}
                    </select>
                  ) : (
                    unidadeDe(l).NomeUnidade
                  )}
                </td>
                <td style={{ textAlign: "right" }}>
                  <input
                    className="admin-credito-qtd"
                    inputMode="decimal"
                    value={l.quantidade}
                    onChange={(e) => atualizar(l.chave, { quantidade: e.target.value })}
                  />
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{mt(subtotalDe(l))}</td>
                <td>
                  <button
                    className="admin-btn-secundario admin-credito-remover"
                    aria-label={`Remover ${l.item.Produto.Nome}`}
                    onClick={() => setLinhas((ls) => ls.filter((x) => x.chave !== l.chave))}
                  >
                    <TrashIcon className="admin-btn-icon-inline" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>

        <div className="admin-form">
          <label className="admin-campo">
            Entrada agora (opcional, MT)
            <input inputMode="decimal" placeholder="0,00" value={entrada} onChange={(e) => setEntrada(e.target.value)} />
          </label>
          <label className="admin-campo">
            Método da entrada
            <select value={metodo} disabled={entradaNum <= 0} onChange={(e) => setMetodo(e.target.value)}>
              {Object.entries(ROTULO_METODO).map(([v, r]) => (
                <option key={v} value={v}>
                  {r}
                </option>
              ))}
            </select>
          </label>
        </div>
        {!entradaValida && (
          <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
            A entrada tem de estar entre 0 e o total da venda.
          </p>
        )}

        <div className="admin-venda-totais" style={{ marginLeft: 0 }}>
          <div className="total">
            <span>Total</span>
            <span>{mt(total)}</span>
          </div>
          <div>
            <span>Entrada</span>
            <span>{mt(entradaValida ? entradaNum : 0)}</span>
          </div>
          <div className="falta">
            <span>Fica em dívida</span>
            <span>{mt(emDivida)}</span>
          </div>
        </div>

        {cliente && ultrapassaLimite && (
          <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
            <ExclamationTriangleIcon className="admin-relatorios-alerta" aria-hidden="true" />
            Com esta venda, {cliente.Cliente.Nome} fica a dever {mt(ficaraEmAberto)}, acima do limite de {mt(limite ?? 0)}. A
            venda pode ser registada na mesma.
          </p>
        )}

        <button className="admin-btn admin-credito-registar" disabled={!podeRegistar} onClick={registar}>
          {aGuardar ? "A registar…" : `Registar venda a crédito${emDivida > 0 ? ` (${mt(emDivida)} em dívida)` : ""}`}
        </button>
      </section>
    </div>
  );
}

function Concluida({
  resultado,
  clientes,
  onNova,
  onVerCliente,
  onErro,
}: {
  resultado: admin.ResultadoVendaCredito;
  clientes: admin.ClienteResumo[];
  onNova: () => void;
  onVerCliente: (c: { ID: number; Nome: string }) => void;
  onErro: (e: string) => void;
}) {
  const v = resultado.Venda.Venda;
  const pago = resultado.Venda.Pagamentos.reduce((s, p) => s + p.Valor, 0);
  const nome = clientes.find((c) => c.Cliente.ID === v.ClienteID)?.Cliente.Nome ?? `Cliente #${v.ClienteID}`;
  const [aImprimir, setAImprimir] = useState(false);

  function imprimir() {
    setAImprimir(true);
    AbrirRecibo(v.ID)
      .catch((e) => onErro(String(e)))
      .finally(() => setAImprimir(false));
  }

  return (
    <div className="admin-credito-concluida">
      <CheckCircleIcon className="admin-credito-concluida-icone" aria-hidden="true" />
      <h3 style={{ margin: 0 }}>Venda #{v.ID} registada a crédito</h3>
      <div className="admin-venda-totais" style={{ marginLeft: 0, minWidth: 320 }}>
        <div>
          <span>Cliente</span>
          <span>{nome}</span>
        </div>
        <div className="total">
          <span>Total</span>
          <span>{mt(v.Total)}</span>
        </div>
        <div>
          <span>Entrada</span>
          <span>{mt(pago)}</span>
        </div>
        <div className="falta">
          <span>Em dívida</span>
          <span>{mt(v.Total - pago)}</span>
        </div>
      </div>
      {resultado.Alerta.Excedido && (
        <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
          {nome} deve agora {mt(resultado.Alerta.TotalEmAberto)}, acima do limite de crédito de{" "}
          {mt(resultado.Alerta.LimiteCredito)}.
        </p>
      )}
      <div className="admin-form">
        <button className="admin-btn-secundario" disabled={aImprimir} onClick={imprimir}>
          <PrinterIcon className="admin-btn-icon-inline" aria-hidden="true" />
          {aImprimir ? "A gerar…" : "Recibo (PDF)"}
        </button>
        {v.ClienteID != null && (
          <button className="admin-btn-secundario" onClick={() => onVerCliente({ ID: v.ClienteID!, Nome: nome })}>
            Ver ficha do cliente
          </button>
        )}
        <button className="admin-btn" onClick={onNova}>
          Nova venda a crédito
        </button>
      </div>
    </div>
  );
}
