import { useEffect, useState } from "react";
import {
  ListarClientes,
  CriarCliente,
  AtualizarCliente,
  AtivarCliente,
  DesativarCliente,
  ListarVendasPorCliente,
  ListarDividasPorCliente,
  ListarDividasAntigas,
  RegistarPagamentoDivida,
} from "../wailsjs/go/api/App";
import { domain, usecase } from "../wailsjs/go/models";

const METODOS = ["dinheiro", "mpesa", "emola", "cartao", "transferencia"];
const DIAS_ANTIGA = 30;

/**
 * Clientes e dívidas (SDD 4.9/4.11, Fase 7) — mesma família de scaffolding
 * funcional do resto de FaseUm.tsx, sem tema/polimento: CRUD de clientes,
 * histórico de compras, dívidas em aberto com alerta de limite de crédito
 * e de dívidas antigas (> 30 dias), e pagamento de dívida como fluxo
 * independente de uma nova venda.
 */
export default function ClientesSection({ onErro }: { onErro: (e: string) => void }) {
  const [clientes, setClientes] = useState<domain.Cliente[]>([]);
  const [dividasAntigas, setDividasAntigas] = useState<domain.Divida[]>([]);
  const [clienteSelecionadoId, setClienteSelecionadoId] = useState<number | null>(null);
  const [clienteEmEdicao, setClienteEmEdicao] = useState<domain.Cliente | null>(null);

  const recarregar = () => {
    ListarClientes().then(setClientes).catch((e) => onErro(String(e)));
    ListarDividasAntigas(DIAS_ANTIGA).then(setDividasAntigas).catch((e) => onErro(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro is a stable setter passed from FaseUm
  useEffect(recarregar, []);

  const clienteSelecionado = clientes.find((c) => c.ID === clienteSelecionadoId) ?? null;
  const nomeCliente = (id: number) => clientes.find((c) => c.ID === id)?.Nome ?? `#${id}`;

  function alternarAtivo(c: domain.Cliente) {
    const acao = c.Ativo ? DesativarCliente(c.ID) : AtivarCliente(c.ID);
    acao.then(recarregar).catch((e) => onErro(String(e)));
  }

  return (
    <section>
      <h2>Clientes e dívidas</h2>

      {dividasAntigas.length > 0 && (
        <p className="fase1-stock-atual">
          ⚠ {dividasAntigas.length} dívida(s) com mais de {DIAS_ANTIGA} dias em aberto:{" "}
          {dividasAntigas.map((d) => nomeCliente(d.ClienteID)).join(", ")}
        </p>
      )}

      <ClienteForm
        key={clienteEmEdicao?.ID ?? "novo"}
        cliente={clienteEmEdicao}
        onGuardado={() => {
          setClienteEmEdicao(null);
          recarregar();
        }}
        onErro={onErro}
      />

      <table>
        <thead>
          <tr>
            <th>Nome</th>
            <th>Telefone</th>
            <th>Limite crédito</th>
            <th>Ativo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {clientes.map((c) => (
            <tr key={c.ID}>
              <td>{c.Nome}</td>
              <td>{c.Telefone ?? "—"}</td>
              <td>{c.LimiteCredito != null ? `${c.LimiteCredito} MT` : "sem limite"}</td>
              <td>{c.Ativo ? "sim" : "não"}</td>
              <td className="fase1-form">
                <button onClick={() => setClienteEmEdicao(c)}>Editar</button>
                <button onClick={() => alternarAtivo(c)}>{c.Ativo ? "Desativar" : "Ativar"}</button>
                <button onClick={() => setClienteSelecionadoId(c.ID === clienteSelecionadoId ? null : c.ID)}>
                  {c.ID === clienteSelecionadoId ? "Fechar detalhe" : "Ver detalhe"}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {clienteSelecionado && (
        <ClienteDetalhe cliente={clienteSelecionado} onErro={onErro} onChange={recarregar} />
      )}
    </section>
  );
}

function ClienteForm({
  cliente,
  onGuardado,
  onErro,
}: {
  cliente: domain.Cliente | null;
  onGuardado: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(cliente?.Nome ?? "");
  const [telefone, setTelefone] = useState(cliente?.Telefone ?? "");
  const [nif, setNif] = useState(cliente?.NIF ?? "");
  const [morada, setMorada] = useState(cliente?.Morada ?? "");
  const [limiteCredito, setLimiteCredito] = useState(cliente?.LimiteCredito != null ? String(cliente.LimiteCredito) : "");

  function guardar() {
    if (!nome.trim()) return;
    const c = new domain.Cliente();
    c.ID = cliente?.ID ?? 0;
    c.Nome = nome.trim();
    c.Telefone = telefone.trim() || undefined;
    c.NIF = nif.trim() || undefined;
    c.Morada = morada.trim() || undefined;
    c.LimiteCredito = limiteCredito.trim() ? Number(limiteCredito) : undefined;
    c.Ativo = cliente?.Ativo ?? true;

    const pedido = cliente ? AtualizarCliente(c).then(() => undefined) : CriarCliente(c).then(() => undefined);
    pedido.then(onGuardado).catch((e) => onErro(String(e)));
  }

  return (
    <div className="fase1-form">
      <input placeholder="Nome" value={nome} onChange={(e) => setNome(e.target.value)} />
      <input placeholder="Telefone" value={telefone} onChange={(e) => setTelefone(e.target.value)} />
      <input placeholder="NIF" value={nif} onChange={(e) => setNif(e.target.value)} />
      <input placeholder="Morada" value={morada} onChange={(e) => setMorada(e.target.value)} />
      <input
        placeholder="Limite de crédito (vazio = sem limite)"
        type="number"
        value={limiteCredito}
        onChange={(e) => setLimiteCredito(e.target.value)}
      />
      <button disabled={!nome.trim()} onClick={guardar}>
        {cliente ? "Guardar alterações" : "+ Criar cliente"}
      </button>
    </div>
  );
}

function ClienteDetalhe({
  cliente,
  onErro,
  onChange,
}: {
  cliente: domain.Cliente;
  onErro: (e: string) => void;
  onChange: () => void;
}) {
  const [vendas, setVendas] = useState<domain.Venda[]>([]);
  const [dividas, setDividas] = useState<domain.Divida[]>([]);

  const recarregar = () => {
    ListarVendasPorCliente(cliente.ID).then(setVendas).catch((e) => onErro(String(e)));
    ListarDividasPorCliente(cliente.ID).then(setDividas).catch((e) => onErro(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro (setErro) is a stable setter
  useEffect(recarregar, [cliente.ID]);

  const totalEmAberto = dividas.reduce((acc, d) => acc + d.ValorEmAberto, 0);

  return (
    <div className="fase1-form" style={{ flexDirection: "column", alignItems: "stretch" }}>
      <h3>
        {cliente.Nome} — total em dívida: {totalEmAberto.toFixed(2)} MT
        {cliente.LimiteCredito != null && totalEmAberto > cliente.LimiteCredito && (
          <> (excede o limite de crédito de {cliente.LimiteCredito.toFixed(2)} MT)</>
        )}
      </h3>

      <h4>Histórico de compras</h4>
      <table>
        <thead>
          <tr>
            <th>ID venda</th>
            <th>Data</th>
            <th>Total</th>
            <th>Status</th>
          </tr>
        </thead>
        <tbody>
          {vendas.map((v) => (
            <tr key={v.ID}>
              <td>{v.ID}</td>
              <td>{new Date(String(v.CriadoEm)).toLocaleString()}</td>
              <td>{v.Total.toFixed(2)} MT</td>
              <td>{v.Status}</td>
            </tr>
          ))}
          {vendas.length === 0 && (
            <tr>
              <td colSpan={4}>Sem compras registadas</td>
            </tr>
          )}
        </tbody>
      </table>

      <h4>Dívidas</h4>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Venda</th>
            <th>Original</th>
            <th>Pago</th>
            <th>Em aberto</th>
            <th>Status</th>
            <th>Criada em</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {dividas.map((d) => (
            <DividaLinha key={d.ID} divida={d} onErro={onErro} onChange={() => { recarregar(); onChange(); }} />
          ))}
          {dividas.length === 0 && (
            <tr>
              <td colSpan={8}>Sem dívidas registadas</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function DividaLinha({
  divida,
  onErro,
  onChange,
}: {
  divida: domain.Divida;
  onErro: (e: string) => void;
  onChange: () => void;
}) {
  const [valor, setValor] = useState("");
  const [metodo, setMetodo] = useState(METODOS[0]);
  const paga = divida.Status === "paga";

  function pagar() {
    if (!valor || Number(valor) <= 0) return;
    const p = new usecase.RegistarPagamentoDividaParams();
    p.DividaID = divida.ID;
    p.Valor = Number(valor);
    p.Metodo = metodo;
    RegistarPagamentoDivida(p)
      .then(() => {
        setValor("");
        onChange();
      })
      .catch((e) => onErro(String(e)));
  }

  return (
    <tr>
      <td>{divida.ID}</td>
      <td>{divida.VendaID}</td>
      <td>{divida.ValorOriginal.toFixed(2)}</td>
      <td>{divida.ValorPago.toFixed(2)}</td>
      <td>{divida.ValorEmAberto.toFixed(2)}</td>
      <td>{divida.Status}</td>
      <td>{new Date(String(divida.CriadoEm)).toLocaleDateString()}</td>
      <td>
        {!paga && (
          <div className="fase1-form">
            <input
              placeholder="Valor a pagar"
              type="number"
              value={valor}
              onChange={(e) => setValor(e.target.value)}
              style={{ width: "8em" }}
            />
            <select value={metodo} onChange={(e) => setMetodo(e.target.value)}>
              {METODOS.map((m) => (
                <option key={m} value={m}>
                  {m}
                </option>
              ))}
            </select>
            <button onClick={pagar}>Registar pagamento</button>
          </div>
        )}
      </td>
    </tr>
  );
}
