import { useEffect, useState } from "react";
import { PrinterIcon, XCircleIcon } from "@heroicons/react/24/outline";
import { ListarVendas, ObterDetalheVenda, AbrirRecibo, AnularVenda } from "../wailsjs/go/admin/AdminApp";
import { admin } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";

const mt = (n: number) => `${n.toLocaleString("pt-PT", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} MT`;
const qtd = (n: number) => n.toLocaleString("pt-PT", { maximumFractionDigits: 3 });
const dataHora = (d: unknown) => new Date(String(d)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" });

// "AAAA-MM-DD" no fuso local — toISOString() daria o dia em UTC e trocaria a
// data perto da meia-noite.
function diaLocal(d: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
}
// O <input type="date"> segue o idioma do sistema operativo (num sistema
// em inglês mostra mês/dia); o período por extenso no resumo nunca é ambíguo.
function porExtenso(dia: string): string {
  return new Date(`${dia}T00:00:00`).toLocaleDateString("pt-PT", { day: "numeric", month: "long", year: "numeric" });
}

function diasAtras(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return diaLocal(d);
}

const ROTULO_ESTADO: Record<string, string> = {
  paga: "Paga",
  divida: "Dívida",
  parcial: "Parcial",
  cancelada: "Cancelada",
};
const ROTULO_TIPO: Record<string, string> = {
  direta: "Venda direta",
  conta_fechada: "Conta",
};
const ROTULO_METODO: Record<string, string> = {
  dinheiro: "Dinheiro",
  mpesa: "M-Pesa",
  emola: "e-Mola",
  cartao: "Cartão",
  transferencia: "Transferência",
};

const ATALHOS = [
  { rotulo: "Hoje", dias: 0 },
  { rotulo: "7 dias", dias: 6 },
  { rotulo: "30 dias", dias: 29 },
];

/**
 * Histórico de vendas por período (por omissão, hoje), com o detalhe de
 * cada venda — itens, pagamentos e margem — e a reimpressão do recibo.
 */
export default function VendasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [desde, setDesde] = useState(diaLocal(new Date()));
  const [ate, setAte] = useState(diaLocal(new Date()));
  const [estado, setEstado] = useState("");
  const [operador, setOperador] = useState("");
  const [vendas, setVendas] = useState<admin.VendaResumo[] | null>(null);
  const [periodo, setPeriodo] = useState({ desde, ate });
  const [aberta, setAberta] = useState<number | null>(null);

  function consultar(d: string, a: string) {
    setVendas(null);
    setPeriodo({ desde: d, ate: a });
    ListarVendas(d, a)
      .then(setVendas)
      .catch((e) => {
        setVendas([]);
        onErro(String(e));
      });
  }

  useEffect(() => {
    ListarVendas(desde, ate)
      .then(setVendas)
      .catch((e) => {
        setVendas([]);
        onErro(String(e));
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só a consulta inicial; as seguintes são pelos botões
  }, []);

  function atalho(dias: number) {
    const d = diasAtras(dias);
    const a = diaLocal(new Date());
    setDesde(d);
    setAte(a);
    consultar(d, a);
  }

  const operadores = [...new Set((vendas ?? []).map((v) => v.OperadorNome))].sort((a, b) => a.localeCompare(b, "pt"));
  const visiveis = (vendas ?? []).filter(
    (v) => (!estado || v.Venda.Status === estado) && (!operador || v.OperadorNome === operador),
  );
  const validas = visiveis.filter((v) => v.Venda.Status !== "cancelada");
  const total = validas.reduce((s, v) => s + v.Venda.Total, 0);
  const emDivida = validas.filter((v) => v.Venda.Status === "divida" || v.Venda.Status === "parcial").length;

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Vendas" }]} />

      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Vendas</h2>
        <div className="admin-form">
          {ATALHOS.map((a) => (
            <button key={a.rotulo} className="admin-btn-secundario" onClick={() => atalho(a.dias)}>
              {a.rotulo}
            </button>
          ))}
        </div>
      </div>

      <div className="admin-form">
        <label>
          De <input type="date" value={desde} max={ate} onChange={(e) => setDesde(e.target.value)} />
        </label>
        <label>
          até <input type="date" value={ate} min={desde} onChange={(e) => setAte(e.target.value)} />
        </label>
        <select value={estado} onChange={(e) => setEstado(e.target.value)}>
          <option value="">Todos os estados</option>
          {Object.entries(ROTULO_ESTADO).map(([valor, rotulo]) => (
            <option key={valor} value={valor}>
              {rotulo}
            </option>
          ))}
        </select>
        {operadores.length > 1 && (
          <select value={operador} onChange={(e) => setOperador(e.target.value)}>
            <option value="">Todos os operadores</option>
            {operadores.map((o) => (
              <option key={o} value={o}>
                {o}
              </option>
            ))}
          </select>
        )}
        <button className="admin-btn" disabled={!desde || !ate} onClick={() => consultar(desde, ate)}>
          Consultar
        </button>
      </div>

      <div className="admin-inventario-resumo">
        <span>
          {periodo.desde === periodo.ate
            ? porExtenso(periodo.desde)
            : `${porExtenso(periodo.desde)} a ${porExtenso(periodo.ate)}`}
        </span>
        <span>
          <strong>{validas.length}</strong> venda(s)
        </span>
        <span>
          <strong>{mt(total)}</strong> vendido
        </span>
        {validas.length > 0 && (
          <span>
            <strong>{mt(total / validas.length)}</strong> por venda
          </span>
        )}
        {emDivida > 0 && (
          <span>
            <strong>{emDivida}</strong> com dívida
          </span>
        )}
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>N.º</th>
            <th>Data</th>
            <th>Cliente</th>
            <th>Operador</th>
            <th>Tipo</th>
            <th>Estado</th>
            <th style={{ textAlign: "right" }}>Total</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {vendas === null && (
            <tr>
              <td colSpan={8}>A carregar…</td>
            </tr>
          )}
          {vendas !== null && visiveis.length === 0 && (
            <tr>
              <td colSpan={8}>Sem vendas neste período.</td>
            </tr>
          )}
          {visiveis.map(({ Venda: v, ClienteNome, OperadorNome }) => (
            <tr key={v.ID}>
              <td>#{v.ID}</td>
              <td>{dataHora(v.CriadoEm)}</td>
              <td>{ClienteNome}</td>
              <td>{OperadorNome}</td>
              <td>{ROTULO_TIPO[v.TipoVenda] ?? v.TipoVenda}</td>
              <td>
                <span className={`admin-venda-estado ${v.Status}`}>{ROTULO_ESTADO[v.Status] ?? v.Status}</span>
              </td>
              <td style={{ textAlign: "right", fontVariantNumeric: "tabular-nums" }}>{mt(v.Total)}</td>
              <td className="admin-form">
                <button className="admin-btn-secundario" onClick={() => setAberta(v.ID)}>
                  Detalhe
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {aberta !== null && (
        <Modal titulo={`Venda #${aberta}`} onFechar={() => setAberta(null)}>
          <DetalheVenda
            vendaID={aberta}
            onAnulada={() => {
              setAberta(null);
              consultar(periodo.desde, periodo.ate);
            }}
            onErro={onErro}
          />
        </Modal>
      )}
    </div>
  );
}

function DetalheVenda({
  vendaID,
  onAnulada,
  onErro,
}: {
  vendaID: number;
  onAnulada: () => void;
  onErro: (e: string) => void;
}) {
  const [detalhe, setDetalhe] = useState<admin.VendaDetalhe | null>(null);
  const [aImprimir, setAImprimir] = useState(false);
  const [aAnular, setAAnular] = useState(false);
  const [motivo, setMotivo] = useState("");
  const [aConfirmar, setAConfirmar] = useState(false);

  useEffect(() => {
    ObterDetalheVenda(vendaID)
      .then(setDetalhe)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- venda fixa por instância do modal
  }, []);

  if (detalhe === null) return <p>A carregar…</p>;

  const v = detalhe.Venda;
  const custo = detalhe.Itens.reduce((s, i) => s + i.Item.CustoUnitarioNoMomento * i.Item.Quantidade, 0);
  const lucro = v.Total - custo;
  const pago = detalhe.Pagamentos.reduce((s, p) => s + p.Valor, 0);

  // Só se anula no próprio dia (o backend também o exige): o dinheiro
  // devolve-se nesse dia e fechos de dias anteriores nunca mudam.
  const deHoje = new Date(String(v.CriadoEm)).toDateString() === new Date().toDateString();
  const podeAnular = v.Status !== "cancelada" && deHoje;

  function anular() {
    if (!motivo.trim() || aConfirmar) return;
    setAConfirmar(true);
    AnularVenda(vendaID, motivo.trim())
      .then(onAnulada)
      .catch((e) => onErro(String(e)))
      .finally(() => setAConfirmar(false));
  }

  function imprimir() {
    setAImprimir(true);
    AbrirRecibo(vendaID)
      .catch((e) => onErro(String(e)))
      .finally(() => setAImprimir(false));
  }

  return (
    <div className="admin-armazem-form">
      <div className="admin-venda-cabecalho">
        <span>{dataHora(v.CriadoEm)}</span>
        <span>{detalhe.ClienteNome}</span>
        <span>vendido por {detalhe.OperadorNome}</span>
        <span>{ROTULO_TIPO[v.TipoVenda] ?? v.TipoVenda}</span>
        <span className={`admin-venda-estado ${v.Status}`}>{ROTULO_ESTADO[v.Status] ?? v.Status}</span>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th style={{ textAlign: "right" }}>Qtd</th>
            <th style={{ textAlign: "right" }}>Preço</th>
            <th style={{ textAlign: "right" }}>Desconto</th>
            <th style={{ textAlign: "right" }}>Subtotal</th>
          </tr>
        </thead>
        <tbody>
          {detalhe.Itens.map(({ Item: i, ProdutoNome, UnidadeNome }) => (
            <tr key={i.ID}>
              <td>{ProdutoNome}</td>
              <td style={{ textAlign: "right" }}>
                {qtd(i.Quantidade)} {UnidadeNome}
              </td>
              <td style={{ textAlign: "right" }}>{mt(i.PrecoUnitario)}</td>
              <td style={{ textAlign: "right" }}>{i.Desconto ? mt(i.Desconto) : "—"}</td>
              <td style={{ textAlign: "right" }}>{mt(i.Subtotal)}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="admin-venda-totais">
        <div>
          <span>Subtotal</span>
          <span>{mt(v.Subtotal)}</span>
        </div>
        {v.DescontoTotal > 0 && (
          <div>
            <span>Desconto</span>
            <span>− {mt(v.DescontoTotal)}</span>
          </div>
        )}
        <div className="total">
          <span>Total</span>
          <span>{mt(v.Total)}</span>
        </div>
        <div className="secundario">
          <span>Custo / margem</span>
          <span>
            {mt(custo)} / {mt(lucro)}
            {v.Total > 0 && ` (${((lucro / v.Total) * 100).toLocaleString("pt-PT", { maximumFractionDigits: 1 })}%)`}
          </span>
        </div>
      </div>

      <div>
        <strong style={{ fontSize: "0.9rem" }}>Pagamentos</strong>
        {detalhe.Pagamentos.length === 0 ? (
          <p style={{ color: "var(--text-secondary)", margin: "4px 0 0" }}>Nenhum pagamento registado.</p>
        ) : (
          <div className="admin-venda-totais">
            {detalhe.Pagamentos.map((p) => (
              <div key={p.ID}>
                <span>{ROTULO_METODO[p.Metodo] ?? p.Metodo}</span>
                <span>{mt(p.Valor)}</span>
              </div>
            ))}
            {pago < v.Total - 0.01 && (
              <div className="falta">
                <span>Em falta</span>
                <span>{mt(v.Total - pago)}</span>
              </div>
            )}
          </div>
        )}
      </div>

      {aAnular ? (
        <div className="admin-venda-anular">
          <strong>Anular a venda #{v.ID}?</strong>
          <p style={{ margin: 0 }}>
            O stock volta ao armazém (e aos lotes de onde saiu), a venda deixa de contar nos relatórios e no fecho de
            caixa de hoje{detalhe.Pagamentos.length > 0 ? ` — devolva ${mt(pago)} ao cliente` : ""}. Não se pode desfazer.
          </p>
          <label className="admin-campo">
            Motivo
            <input
              autoFocus
              placeholder="ex: registada por engano, cliente desistiu"
              value={motivo}
              onChange={(e) => setMotivo(e.target.value)}
            />
          </label>
          <div className="admin-form">
            <button className="admin-btn admin-btn-perigo" disabled={!motivo.trim() || aConfirmar} onClick={anular}>
              {aConfirmar ? "A anular…" : "Confirmar anulação"}
            </button>
            <button className="admin-btn-secundario" onClick={() => setAAnular(false)}>
              Voltar
            </button>
          </div>
        </div>
      ) : (
        <div className="admin-form">
          <button className="admin-btn" disabled={aImprimir || v.Status === "cancelada"} onClick={imprimir}>
            <PrinterIcon className="admin-btn-icon-inline" aria-hidden="true" />
            {aImprimir ? "A gerar…" : "Recibo (PDF)"}
          </button>
          {podeAnular && (
            <button className="admin-btn-secundario admin-btn-perigo-contorno" onClick={() => setAAnular(true)}>
              <XCircleIcon className="admin-btn-icon-inline" aria-hidden="true" />
              Anular venda
            </button>
          )}
          {v.Status !== "cancelada" && !deHoje && (
            <span style={{ color: "var(--text-secondary)", fontSize: "0.85rem" }}>
              Só se anula uma venda no próprio dia.
            </span>
          )}
        </div>
      )}
    </div>
  );
}
