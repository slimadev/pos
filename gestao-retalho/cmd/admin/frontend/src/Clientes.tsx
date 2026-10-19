import { useEffect, useState } from "react";
import { BanknotesIcon, ExclamationTriangleIcon, PencilSquareIcon } from "@heroicons/react/24/outline";
import {
  ListarClientes,
  ObterFichaCliente,
  CriarCliente,
  AtualizarCliente,
  AtivarCliente,
  DesativarCliente,
  ListarPagamentosDivida,
  RegistarPagamentoDivida,
} from "../wailsjs/go/admin/AdminApp";
import { admin, domain, usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";
import type { Navegar } from "./App";

const mt = (n: number) => `${n.toLocaleString("pt-PT", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} MT`;
const data = (d: unknown) => new Date(String(d)).toLocaleDateString("pt-PT", { dateStyle: "short" });
const dataHora = (d: unknown) => new Date(String(d)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" });
const diasDesde = (d: unknown) => Math.floor((Date.now() - new Date(String(d)).getTime()) / 86_400_000);
const haDias = (dias: number) => (dias === 0 ? "hoje" : dias === 1 ? "há 1 dia" : `há ${dias} dias`);

const ROTULO_METODO: Record<string, string> = {
  dinheiro: "Dinheiro",
  mpesa: "M-Pesa",
  emola: "e-Mola",
  cartao: "Cartão",
  transferencia: "Transferência",
};
const ROTULO_DIVIDA: Record<string, string> = { aberta: "Em aberto", parcial: "Parcial", paga: "Paga" };
const ROTULO_VENDA: Record<string, string> = { paga: "Paga", divida: "Dívida", parcial: "Parcial", cancelada: "Cancelada" };
const DIAS_DIVIDA_ANTIGA = 30;

type Vista = { tipo: "lista" } | { tipo: "ficha"; clienteID: number; nome: string };

/**
 * Clientes identificados e o que devem. A lista mostra o saldo em aberto
 * de cada um (o cliente anónimo das vendas ao balcão não aparece); a ficha
 * mostra as dívidas, permite registar pagamentos e lista as compras.
 */
export default function ClientesScreen({
  cliente,
  onNavegar,
  onErro,
}: {
  cliente?: { ID: number; Nome: string };
  onNavegar: Navegar;
  onErro: (e: string) => void;
}) {
  const [vista, setVista] = useState<Vista>(
    cliente ? { tipo: "ficha", clienteID: cliente.ID, nome: cliente.Nome } : { tipo: "lista" },
  );

  if (vista.tipo === "ficha") {
    return (
      <div className="admin-lojas-screen">
        <Breadcrumb
          itens={[{ rotulo: "Clientes e dívidas", onClick: () => setVista({ tipo: "lista" }) }, { rotulo: vista.nome }]}
        />
        <Ficha clienteID={vista.clienteID} onNavegar={onNavegar} onErro={onErro} />
      </div>
    );
  }
  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Clientes e dívidas" }]} />
      <Lista
        onAbrir={(c) => setVista({ tipo: "ficha", clienteID: c.ID, nome: c.Nome })}
        onNavegar={onNavegar}
        onErro={onErro}
      />
    </div>
  );
}

function Lista({
  onAbrir,
  onNavegar,
  onErro,
}: {
  onAbrir: (c: domain.Cliente) => void;
  onNavegar: Navegar;
  onErro: (e: string) => void;
}) {
  const [clientes, setClientes] = useState<admin.ClienteResumo[] | null>(null);
  const [pesquisa, setPesquisa] = useState("");
  const [filtro, setFiltro] = useState<"ativos" | "com_divida" | "inativos">("ativos");
  const [novo, setNovo] = useState(false);

  function carregar() {
    ListarClientes()
      .then(setClientes)
      .catch((e) => {
        setClientes([]);
        onErro(String(e));
      });
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- carga inicial
  }, []);

  const lista = clientes ?? [];
  const ativos = lista.filter((c) => c.Cliente.Ativo);
  const totalEmAberto = ativos.reduce((s, c) => s + c.TotalEmAberto, 0);
  const comDivida = ativos.filter((c) => c.TotalEmAberto > 0).length;
  const antigas = ativos.filter((c) => c.DiasDividaMaisAntiga > DIAS_DIVIDA_ANTIGA).length;
  const acimaLimite = ativos.filter((c) => c.LimiteExcedido).length;

  const termo = pesquisa.trim().toLowerCase();
  const visiveis = lista
    .filter((c) => {
      if (filtro === "inativos") return !c.Cliente.Ativo;
      if (!c.Cliente.Ativo) return false;
      return filtro === "com_divida" ? c.TotalEmAberto > 0 : true;
    })
    .filter(
      (c) =>
        !termo || c.Cliente.Nome.toLowerCase().includes(termo) || (c.Cliente.Telefone ?? "").toLowerCase().includes(termo),
    )
    .sort((a, b) => b.TotalEmAberto - a.TotalEmAberto || a.Cliente.Nome.localeCompare(b.Cliente.Nome, "pt"));

  return (
    <>
      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Clientes e dívidas</h2>
        <div className="admin-form">
          <button
            className="admin-btn-secundario"
            title="Lista de dívidas em aberto, para exportar em Excel ou PDF"
            onClick={() => onNavegar({ ecra: "relatorios", relatorio: "dividas" })}
          >
            Exportar dívidas
          </button>
          <button className="admin-btn" onClick={() => setNovo(true)}>
            + Novo cliente
          </button>
        </div>
      </div>

      <div className="admin-inventario-resumo">
        <span>
          <strong>{mt(totalEmAberto)}</strong> em dívida
        </span>
        <span>
          <strong>{comDivida}</strong> cliente(s) com dívida
        </span>
        {antigas > 0 && (
          <span>
            <strong>{antigas}</strong> com dívida há mais de {DIAS_DIVIDA_ANTIGA} dias
          </span>
        )}
        {acimaLimite > 0 && (
          <span>
            <strong>{acimaLimite}</strong> acima do limite de crédito
          </span>
        )}
      </div>

      <div className="admin-form">
        <input placeholder="Pesquisar nome ou telefone…" value={pesquisa} onChange={(e) => setPesquisa(e.target.value)} />
        <select value={filtro} onChange={(e) => setFiltro(e.target.value as typeof filtro)}>
          <option value="ativos">Todos os ativos</option>
          <option value="com_divida">Só com dívida</option>
          <option value="inativos">Inativos</option>
        </select>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Cliente</th>
            <th>Telefone</th>
            <th style={{ textAlign: "right" }}>Limite de crédito</th>
            <th style={{ textAlign: "right" }}>Em dívida</th>
            <th style={{ textAlign: "right" }}>Dívida mais antiga</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {clientes === null && (
            <tr>
              <td colSpan={6}>A carregar…</td>
            </tr>
          )}
          {clientes !== null && visiveis.length === 0 && (
            <tr>
              <td colSpan={6}>{lista.length === 0 ? "Ainda não há clientes registados." : "Nenhum cliente neste filtro."}</td>
            </tr>
          )}
          {visiveis.map((r) => (
            <tr key={r.Cliente.ID}>
              <td>
                {r.Cliente.Nome}
                {r.LimiteExcedido && (
                  <span className="admin-stock-abaixo-minimo">
                    <ExclamationTriangleIcon className="admin-stock-abaixo-minimo-icon" aria-hidden="true" />
                    acima do limite
                  </span>
                )}
              </td>
              <td>{r.Cliente.Telefone || "—"}</td>
              <td style={{ textAlign: "right" }}>{r.Cliente.LimiteCredito != null ? mt(r.Cliente.LimiteCredito) : "—"}</td>
              <td style={{ textAlign: "right" }}>
                {r.TotalEmAberto > 0 ? <span className="admin-inventario-dif falta">{mt(r.TotalEmAberto)}</span> : "—"}
              </td>
              <td style={{ textAlign: "right" }}>
                {r.TotalEmAberto > 0 ? (
                  <span className={r.DiasDividaMaisAntiga > DIAS_DIVIDA_ANTIGA ? "admin-inventario-dif falta" : undefined}>
                    {haDias(r.DiasDividaMaisAntiga)}
                  </span>
                ) : (
                  "—"
                )}
              </td>
              <td className="admin-form">
                <button className="admin-btn-secundario" onClick={() => onAbrir(r.Cliente)}>
                  Abrir
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {novo && (
        <Modal titulo="Novo cliente" onFechar={() => setNovo(false)}>
          <ClienteForm
            cliente={null}
            onGuardado={() => {
              setNovo(false);
              carregar();
            }}
            onCancelar={() => setNovo(false)}
            onErro={onErro}
          />
        </Modal>
      )}
    </>
  );
}

function ClienteForm({
  cliente,
  onGuardado,
  onCancelar,
  onErro,
}: {
  cliente: domain.Cliente | null;
  onGuardado: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(cliente?.Nome ?? "");
  const [telefone, setTelefone] = useState(cliente?.Telefone ?? "");
  const [nif, setNif] = useState(cliente?.NIF ?? "");
  const [morada, setMorada] = useState(cliente?.Morada ?? "");
  const [limite, setLimite] = useState(cliente?.LimiteCredito != null ? String(cliente.LimiteCredito) : "");
  const [aGuardar, setAGuardar] = useState(false);

  function guardar() {
    if (!nome.trim() || aGuardar) return;
    const limiteNum = limite.trim() === "" ? undefined : Number(limite.replace(",", "."));
    if (limiteNum !== undefined && (Number.isNaN(limiteNum) || limiteNum < 0)) {
      onErro("Limite de crédito inválido");
      return;
    }
    const c = new domain.Cliente({ ...(cliente ?? {}), Ativo: cliente?.Ativo ?? true });
    c.Nome = nome.trim();
    c.Telefone = telefone.trim() || undefined;
    c.NIF = nif.trim() || undefined;
    c.Morada = morada.trim() || undefined;
    c.LimiteCredito = limiteNum;

    setAGuardar(true);
    const pedido = cliente ? AtualizarCliente(c) : CriarCliente(c);
    pedido
      .then(() => onGuardado())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      <label className="admin-campo">
        Nome *
        <input value={nome} onChange={(e) => setNome(e.target.value)} autoFocus />
      </label>
      <div className="admin-form">
        <label className="admin-campo">
          Telefone
          <input value={telefone} onChange={(e) => setTelefone(e.target.value)} />
        </label>
        <label className="admin-campo">
          NUIT
          <input value={nif} onChange={(e) => setNif(e.target.value)} />
        </label>
      </div>
      <label className="admin-campo">
        Morada
        <input value={morada} onChange={(e) => setMorada(e.target.value)} />
      </label>
      <label className="admin-campo">
        Limite de crédito (MT)
        <input
          inputMode="decimal"
          placeholder="sem limite"
          value={limite}
          onChange={(e) => setLimite(e.target.value)}
        />
        <small style={{ color: "var(--text-secondary)" }}>
          Só avisa quando uma venda a crédito ou uma conta em dívida passa do limite — nunca a bloqueia.
        </small>
      </label>
      <div className="admin-form">
        <button className="admin-btn" disabled={!nome.trim() || aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : cliente ? "Guardar alterações" : "Criar cliente"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}

type ModalFicha =
  | { tipo: "nenhum" }
  | { tipo: "editar" }
  | { tipo: "pagar"; divida: domain.Divida }
  | { tipo: "pagamentos"; divida: domain.Divida };

function Ficha({
  clienteID,
  onNavegar,
  onErro,
}: {
  clienteID: number;
  onNavegar: Navegar;
  onErro: (e: string) => void;
}) {
  const [ficha, setFicha] = useState<admin.FichaCliente | null>(null);
  const [modal, setModal] = useState<ModalFicha>({ tipo: "nenhum" });

  function carregar() {
    ObterFichaCliente(clienteID)
      .then(setFicha)
      .catch((e) => onErro(String(e)));
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- cliente fixo por instância
  }, []);

  if (!ficha) return <p>A carregar…</p>;
  const c = ficha.Cliente;
  const emAberto = ficha.Dividas.reduce((s, d) => s + d.ValorEmAberto, 0);
  const limiteExcedido = c.LimiteCredito != null && emAberto > c.LimiteCredito;
  const dividas = [...ficha.Dividas].sort((a, b) => {
    const pagaA = a.Status === "paga" ? 1 : 0;
    const pagaB = b.Status === "paga" ? 1 : 0;
    return pagaA - pagaB || new Date(String(a.CriadoEm)).getTime() - new Date(String(b.CriadoEm)).getTime();
  });
  const fechar = () => setModal({ tipo: "nenhum" });

  function alternarAtivo() {
    (c.Ativo ? DesativarCliente(c.ID) : AtivarCliente(c.ID)).then(carregar).catch((e) => onErro(String(e)));
  }

  return (
    <>
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>
            {c.Nome}
            {!c.Ativo && <span className="admin-venda-estado cancelada" style={{ marginLeft: 8 }}>inativo</span>}
          </h2>
          <div style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
            {[c.Telefone, c.NIF && `NUIT ${c.NIF}`, c.Morada].filter(Boolean).join(" · ") || "Sem contactos registados"}
          </div>
        </div>
        <div className="admin-form">
          {c.Ativo && (
            <button
              className="admin-btn"
              onClick={() => onNavegar({ ecra: "credito", origem: "clientes", cliente: { ID: c.ID, Nome: c.Nome } })}
            >
              <BanknotesIcon className="admin-btn-icon-inline" aria-hidden="true" />
              Vender a crédito
            </button>
          )}
          <button className="admin-btn-secundario" onClick={() => setModal({ tipo: "editar" })}>
            <PencilSquareIcon className="admin-btn-icon-inline" aria-hidden="true" />
            Editar
          </button>
          <button className="admin-btn-secundario" onClick={alternarAtivo}>
            {c.Ativo ? "Desativar" : "Ativar"}
          </button>
        </div>
      </div>

      <div className="admin-inventario-resumo">
        <span>
          <strong className={emAberto > 0 ? "admin-inventario-dif falta" : undefined}>{mt(emAberto)}</strong> em dívida
        </span>
        <span>
          <strong>{c.LimiteCredito != null ? mt(c.LimiteCredito) : "sem limite"}</strong> de limite
        </span>
        {limiteExcedido && (
          <span className="admin-inventario-dif falta">
            <ExclamationTriangleIcon className="admin-relatorios-alerta" aria-hidden="true" />
            acima do limite em {mt(emAberto - (c.LimiteCredito ?? 0))}
          </span>
        )}
      </div>

      <h3 style={{ margin: "8px 0 0" }}>Dívidas</h3>
      <table className="admin-tabela admin-tabela-compacta">
        <thead>
          <tr>
            <th>Venda</th>
            <th>Data</th>
            <th style={{ textAlign: "right" }}>Valor</th>
            <th style={{ textAlign: "right" }}>Pago</th>
            <th style={{ textAlign: "right" }}>Em aberto</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {dividas.length === 0 && (
            <tr>
              <td colSpan={7}>Este cliente não tem dívidas.</td>
            </tr>
          )}
          {dividas.map((d) => {
            const dias = diasDesde(d.CriadoEm);
            return (
              <tr key={d.ID}>
                <td>#{d.VendaID}</td>
                <td>
                  {data(d.CriadoEm)}
                  {d.Status !== "paga" && (
                    <span
                      className={dias > DIAS_DIVIDA_ANTIGA ? "admin-inventario-dif falta" : undefined}
                      style={{ marginLeft: 6, fontSize: "0.85rem", color: dias > DIAS_DIVIDA_ANTIGA ? undefined : "var(--text-secondary)" }}
                    >
                      {haDias(dias)}
                    </span>
                  )}
                </td>
                <td style={{ textAlign: "right" }}>{mt(d.ValorOriginal)}</td>
                <td style={{ textAlign: "right" }}>{mt(d.ValorPago)}</td>
                <td style={{ textAlign: "right" }}>{d.ValorEmAberto > 0 ? mt(d.ValorEmAberto) : "—"}</td>
                <td>
                  <span className={`admin-divida-estado ${d.Status}`}>{ROTULO_DIVIDA[d.Status] ?? d.Status}</span>
                </td>
                <td className="admin-form admin-acoes">
                  {d.Status !== "paga" && (
                    <button className="admin-btn" onClick={() => setModal({ tipo: "pagar", divida: d })}>
                      Pagar
                    </button>
                  )}
                  {d.ValorPago > 0 && (
                    <button className="admin-btn-secundario" onClick={() => setModal({ tipo: "pagamentos", divida: d })}>
                      Pagamentos
                    </button>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      <h3 style={{ margin: "8px 0 0" }}>Compras</h3>
      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Venda</th>
            <th>Data</th>
            <th>Estado</th>
            <th style={{ textAlign: "right" }}>Total</th>
          </tr>
        </thead>
        <tbody>
          {ficha.Vendas.length === 0 && (
            <tr>
              <td colSpan={4}>Ainda sem compras.</td>
            </tr>
          )}
          {ficha.Vendas.map((v) => (
            <tr key={v.ID}>
              <td>#{v.ID}</td>
              <td>{dataHora(v.CriadoEm)}</td>
              <td>
                <span className={`admin-venda-estado ${v.Status}`}>{ROTULO_VENDA[v.Status] ?? v.Status}</span>
              </td>
              <td style={{ textAlign: "right" }}>{mt(v.Total)}</td>
            </tr>
          ))}
        </tbody>
      </table>

      {modal.tipo === "editar" && (
        <Modal titulo={`Editar ${c.Nome}`} onFechar={fechar}>
          <ClienteForm
            cliente={c}
            onGuardado={() => {
              fechar();
              carregar();
            }}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}
      {modal.tipo === "pagar" && (
        <Modal titulo={`Pagar dívida da venda #${modal.divida.VendaID}`} onFechar={fechar}>
          <PagarDivida
            divida={modal.divida}
            onPago={() => {
              fechar();
              carregar();
            }}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}
      {modal.tipo === "pagamentos" && (
        <Modal titulo={`Pagamentos da venda #${modal.divida.VendaID}`} onFechar={fechar}>
          <PagamentosDivida dividaID={modal.divida.ID} onErro={onErro} />
        </Modal>
      )}
    </>
  );
}

function PagarDivida({
  divida,
  onPago,
  onCancelar,
  onErro,
}: {
  divida: domain.Divida;
  onPago: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [valor, setValor] = useState(divida.ValorEmAberto.toFixed(2).replace(".", ","));
  const [metodo, setMetodo] = useState("dinheiro");
  const [aGuardar, setAGuardar] = useState(false);

  const valorNum = Number(valor.replace(",", "."));
  const valido = !Number.isNaN(valorNum) && valorNum > 0 && valorNum <= divida.ValorEmAberto + 0.005;
  const restante = divida.ValorEmAberto - (Number.isNaN(valorNum) ? 0 : valorNum);

  function pagar() {
    if (!valido || aGuardar) return;
    const p = new usecase.RegistarPagamentoDividaParams();
    p.DividaID = divida.ID;
    p.Valor = valorNum;
    p.Metodo = metodo;
    setAGuardar(true);
    RegistarPagamentoDivida(p)
      .then(() => onPago())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      <div className="admin-venda-totais" style={{ marginLeft: 0 }}>
        <div>
          <span>Valor da dívida</span>
          <span>{mt(divida.ValorOriginal)}</span>
        </div>
        <div>
          <span>Já pago</span>
          <span>{mt(divida.ValorPago)}</span>
        </div>
        <div className="total">
          <span>Em aberto</span>
          <span>{mt(divida.ValorEmAberto)}</span>
        </div>
      </div>
      <div className="admin-form">
        <label className="admin-campo">
          Valor a pagar (MT)
          <input inputMode="decimal" value={valor} onChange={(e) => setValor(e.target.value)} autoFocus />
        </label>
        <label className="admin-campo">
          Método
          <select value={metodo} onChange={(e) => setMetodo(e.target.value)}>
            {Object.entries(ROTULO_METODO).map(([v, r]) => (
              <option key={v} value={v}>
                {r}
              </option>
            ))}
          </select>
        </label>
      </div>
      {!valido && valor.trim() !== "" && (
        <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
          O valor tem de ser maior que zero e no máximo {mt(divida.ValorEmAberto)}.
        </p>
      )}
      {valido && (
        <p style={{ margin: 0, color: "var(--text-secondary)" }}>
          {restante <= 0.005 ? "A dívida fica paga." : `Fica em aberto ${mt(restante)}.`}
        </p>
      )}
      <div className="admin-form">
        <button className="admin-btn" disabled={!valido || aGuardar} onClick={pagar}>
          {aGuardar ? "A registar…" : "Registar pagamento"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}

function PagamentosDivida({ dividaID, onErro }: { dividaID: number; onErro: (e: string) => void }) {
  const [pagamentos, setPagamentos] = useState<domain.PagamentoDivida[] | null>(null);

  useEffect(() => {
    ListarPagamentosDivida(dividaID)
      .then(setPagamentos)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- dívida fixa por instância
  }, []);

  return (
    <table className="admin-tabela">
      <thead>
        <tr>
          <th>Data</th>
          <th>Método</th>
          <th style={{ textAlign: "right" }}>Valor</th>
        </tr>
      </thead>
      <tbody>
        {pagamentos === null && (
          <tr>
            <td colSpan={3}>A carregar…</td>
          </tr>
        )}
        {pagamentos?.map((p) => (
          <tr key={p.ID}>
            <td>{dataHora(p.CriadoEm)}</td>
            <td>{ROTULO_METODO[p.Metodo] ?? p.Metodo}</td>
            <td style={{ textAlign: "right" }}>{mt(p.Valor)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
