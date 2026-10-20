import { useEffect, useState } from "react";
import { DocumentArrowDownIcon, MagnifyingGlassIcon, TrashIcon } from "@heroicons/react/24/outline";
import {
  ListarCotacoes,
  ObterCotacao,
  CriarCotacao,
  CancelarCotacao,
  ConverterCotacao,
  AbrirCotacaoPDF,
  ListarClientes,
  ListarProdutos,
  ListarTodasUnidades,
  ListarArmazens,
  ObterLojaAtual,
} from "../wailsjs/go/admin/AdminApp";
import { admin, domain, usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";
import type { Navegar } from "./App";
import { mt, num } from "./formato";

const lerNumero = (s: string) => Number(s.trim().replace(",", "."));
const data = (d: unknown) => new Date(String(d)).toLocaleDateString("pt-PT");
const ROTULO_METODO: Record<string, string> = {
  dinheiro: "Dinheiro",
  mpesa: "M-Pesa",
  emola: "e-Mola",
  cartao: "Cartão",
  transferencia: "Transferência",
};

type Vista = { tipo: "lista" } | { tipo: "nova" } | { tipo: "ver"; id: number };

/**
 * Cotações (orçamentos/proformas): o cliente leva o preço por escrito, sem
 * mexer no stock nem contar como venda. Os preços ficam congelados e
 * incluem IVA. Uma cotação aberta converte-se em venda (paga na hora, ou
 * a crédito para um cliente registado), ao preço cotado.
 */
export default function CotacoesScreen({ onNavegar, onErro }: { onNavegar: Navegar; onErro: (e: string) => void }) {
  const [vista, setVista] = useState<Vista>({ tipo: "lista" });
  const voltar = () => setVista({ tipo: "lista" });

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb
        itens={
          vista.tipo === "lista"
            ? [{ rotulo: "Cotações" }]
            : [{ rotulo: "Cotações", onClick: voltar }, { rotulo: vista.tipo === "nova" ? "Nova cotação" : "Cotação" }]
        }
      />
      {vista.tipo === "lista" && (
        <Lista onNova={() => setVista({ tipo: "nova" })} onAbrir={(id) => setVista({ tipo: "ver", id })} onErro={onErro} />
      )}
      {vista.tipo === "nova" && (
        <NovaCotacao onCriada={(id) => setVista({ tipo: "ver", id })} onCancelar={voltar} onErro={onErro} />
      )}
      {vista.tipo === "ver" && <VerCotacao id={vista.id} onNavegar={onNavegar} onErro={onErro} />}
    </div>
  );
}

function Estado({ c }: { c: usecase.CotacaoCompleta }) {
  const e = c.Cotacao.Estado;
  if (e === "convertida") return <span className="admin-venda-estado paga">Venda #{c.Cotacao.VendaID}</span>;
  if (e === "cancelada") return <span className="admin-venda-estado cancelada">Cancelada</span>;
  if (c.Expirada) return <span className="admin-venda-estado divida">Expirada</span>;
  return <span className="admin-venda-estado parcial">Aberta</span>;
}

function Lista({
  onNova,
  onAbrir,
  onErro,
}: {
  onNova: () => void;
  onAbrir: (id: number) => void;
  onErro: (e: string) => void;
}) {
  const [cotacoes, setCotacoes] = useState<usecase.CotacaoCompleta[] | null>(null);
  const [filtro, setFiltro] = useState("");

  useEffect(() => {
    ListarCotacoes()
      .then(setCotacoes)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const visiveis = (cotacoes ?? []).filter((c) => {
    if (filtro === "") return true;
    if (filtro === "expirada") return c.Expirada;
    if (filtro === "aberta") return c.Cotacao.Estado === "aberta" && !c.Expirada;
    return c.Cotacao.Estado === filtro;
  });

  return (
    <>
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>Cotações</h2>
          <div className="admin-texto-fraco">Orçamentos para o cliente — não mexem no stock até virarem venda.</div>
        </div>
        <button className="admin-btn" onClick={onNova}>
          + Nova cotação
        </button>
      </div>
      <div className="admin-form">
        <select value={filtro} onChange={(e) => setFiltro(e.target.value)}>
          <option value="">Todas</option>
          <option value="aberta">Abertas</option>
          <option value="expirada">Expiradas</option>
          <option value="convertida">Convertidas em venda</option>
          <option value="cancelada">Canceladas</option>
        </select>
      </div>
      <table className="admin-tabela">
        <thead>
          <tr>
            <th>N.º</th>
            <th>Data</th>
            <th>Cliente</th>
            <th style={{ textAlign: "right" }}>Total</th>
            <th>Válida até</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {visiveis.map((c) => (
            <tr key={c.Cotacao.ID}>
              <td style={{ whiteSpace: "nowrap" }}>{c.Codigo}</td>
              <td>{data(c.Cotacao.CriadoEm)}</td>
              <td>{c.Cotacao.ClienteNome}</td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{mt(c.Cotacao.Total)}</td>
              <td>{data(c.ValidaAte)}</td>
              <td>
                <Estado c={c} />
              </td>
              <td>
                <div className="admin-acoes">
                  <button className="admin-btn-secundario" onClick={() => onAbrir(c.Cotacao.ID)}>
                    Abrir
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {visiveis.length === 0 && (
            <tr>
              <td colSpan={7}>
                {cotacoes === null
                  ? "A carregar…"
                  : cotacoes.length === 0
                    ? 'Ainda não há cotações — clique em "+ Nova cotação".'
                    : "Nenhuma cotação com este filtro."}
              </td>
            </tr>
          )}
        </tbody>
      </table>
    </>
  );
}

type Linha = { chave: number; produto: domain.Produto; unidadeID: number; quantidade: string; desconto: string };
let proximaChave = 0;

function NovaCotacao({
  onCriada,
  onCancelar,
  onErro,
}: {
  onCriada: (id: number) => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [clientes, setClientes] = useState<admin.ClienteResumo[]>([]);
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [unidades, setUnidades] = useState<domain.UnidadeProduto[]>([]);
  const [taxaIVA, setTaxaIVA] = useState(16);
  const [clienteID, setClienteID] = useState<number | "" | "outro">("");
  const [clienteNome, setClienteNome] = useState("");
  const [validade, setValidade] = useState("15");
  const [pesquisa, setPesquisa] = useState("");
  const [linhas, setLinhas] = useState<Linha[]>([]);
  const [descontoGeral, setDescontoGeral] = useState("");
  const [observacoes, setObservacoes] = useState("");
  const [aGuardar, setAGuardar] = useState(false);

  useEffect(() => {
    Promise.all([ListarClientes(), ListarProdutos(), ListarTodasUnidades(), ObterLojaAtual()])
      .then(([c, p, u, l]) => {
        setClientes(c.filter((x) => x.Cliente.Ativo));
        setProdutos(p.filter((x) => x.Ativo));
        setUnidades(u);
        setTaxaIVA(l.TaxaIVA);
      })
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const unidadesDe = (id: number) => unidades.filter((u) => u.ProdutoID === id);
  const termo = pesquisa.trim().toLowerCase();
  const sugestoes = termo
    ? produtos
        .filter(
          (p) =>
            p.Nome.toLowerCase().includes(termo) ||
            unidadesDe(p.ID).some((u) => (u.CodigoBarras ?? "").toLowerCase() === termo),
        )
        .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"))
        .slice(0, 8)
    : [];

  function adicionar(p: domain.Produto) {
    const us = unidadesDe(p.ID);
    const u = us.find((x) => x.IsPadraoVenda) ?? us[0];
    if (!u) {
      onErro(`"${p.Nome}" não tem unidades — acrescente uma em Produtos.`);
      return;
    }
    setLinhas((ls) => [...ls, { chave: proximaChave++, produto: p, unidadeID: u.ID, quantidade: "1", desconto: "" }]);
    setPesquisa("");
  }

  const preco = (l: Linha) => unidades.find((u) => u.ID === l.unidadeID)?.PrecoVenda ?? 0;
  const q = (l: Linha) => lerNumero(l.quantidade);
  const desc = (l: Linha) => (l.desconto.trim() === "" ? 0 : lerNumero(l.desconto));
  const bruto = (l: Linha) => q(l) * preco(l);
  const linhaValida = (l: Linha) =>
    Number.isFinite(q(l)) && q(l) > 0 && Number.isFinite(desc(l)) && desc(l) >= 0 && desc(l) <= bruto(l);
  const subtotal = linhas.reduce((s, l) => s + (linhaValida(l) ? bruto(l) : 0), 0);
  const descontoLinhas = linhas.reduce((s, l) => s + (linhaValida(l) ? desc(l) : 0), 0);
  const dg = descontoGeral.trim() === "" ? 0 : lerNumero(descontoGeral);
  const total = subtotal - descontoLinhas - (Number.isFinite(dg) ? dg : 0);
  const iva = taxaIVA > 0 ? total - total / (1 + taxaIVA / 100) : 0;
  const dias = Number(validade);
  const podeGuardar =
    linhas.length > 0 &&
    linhas.every(linhaValida) &&
    Number.isFinite(dg) &&
    dg >= 0 &&
    total >= 0 &&
    Number.isInteger(dias) &&
    dias >= 1 &&
    dias <= 365 &&
    !aGuardar;

  function guardar() {
    if (!podeGuardar) return;
    const p = new usecase.NovaCotacao();
    if (typeof clienteID === "number") p.ClienteID = clienteID;
    else p.ClienteNome = clienteNome.trim();
    p.ValidadeDias = dias;
    p.DescontoTotal = Number.isFinite(dg) ? dg : 0;
    p.Observacoes = observacoes.trim();
    p.Itens = linhas.map((l) => {
      const i = new usecase.ItemNovaCotacao();
      i.ProdutoID = l.produto.ID;
      i.UnidadeProdutoID = l.unidadeID;
      i.Quantidade = q(l);
      i.Desconto = desc(l);
      return i;
    });
    setAGuardar(true);
    CriarCotacao(p)
      .then((c) => onCriada(c.Cotacao.ID))
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-produto-form">
      <h2 style={{ margin: 0 }}>Nova cotação</h2>
      <div className="admin-config-grelha">
        <label className="admin-campo">
          Cliente
          <select
            value={clienteID}
            onChange={(e) =>
              setClienteID(e.target.value === "" ? "" : e.target.value === "outro" ? "outro" : Number(e.target.value))
            }
          >
            <option value="">Consumidor final</option>
            <option value="outro">Outro (escrever o nome)…</option>
            {clientes.map((c) => (
              <option key={c.Cliente.ID} value={c.Cliente.ID}>
                {c.Cliente.Nome}
                {c.Cliente.NIF ? ` — NUIT ${c.Cliente.NIF}` : ""}
              </option>
            ))}
          </select>
        </label>
        <label className="admin-campo">
          Válida por (dias)
          <input inputMode="numeric" value={validade} onChange={(e) => setValidade(e.target.value)} />
        </label>
      </div>
      {clienteID === "outro" && (
        <label className="admin-campo">
          Nome do cliente (não registado)
          <input
            autoFocus
            placeholder="ex: Obra do Sr. Mabunda"
            value={clienteNome}
            onChange={(e) => setClienteNome(e.target.value)}
          />
          <span className="admin-texto-fraco">
            Para vender a crédito a partir desta cotação, o cliente tem de estar registado em Clientes.
          </span>
        </label>
      )}

      <div className="admin-rececao-pesquisa">
        <label className="admin-pesquisa" style={{ maxWidth: "none" }}>
          <MagnifyingGlassIcon className="admin-pesquisa-icone" aria-hidden="true" />
          <input
            placeholder="Acrescentar produto: nome ou código de barras"
            value={pesquisa}
            onChange={(e) => setPesquisa(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && sugestoes.length === 1 && adicionar(sugestoes[0])}
          />
        </label>
        {sugestoes.length > 0 && (
          <div className="admin-rececao-sugestoes">
            {sugestoes.map((p) => (
              <button key={p.ID} onClick={() => adicionar(p)}>
                {p.Nome}
              </button>
            ))}
          </div>
        )}
      </div>

      <table className="admin-tabela admin-unidades-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th>Unidade</th>
            <th style={{ width: 90 }}>Qtd</th>
            <th style={{ textAlign: "right" }}>Preço</th>
            <th style={{ width: 120 }}>Desconto (MT)</th>
            <th style={{ textAlign: "right" }}>Total</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {linhas.map((l) => (
            <tr key={l.chave}>
              <td>{l.produto.Nome}</td>
              <td>
                <select
                  aria-label="Unidade"
                  value={l.unidadeID}
                  onChange={(e) =>
                    setLinhas((ls) => ls.map((x) => (x.chave === l.chave ? { ...x, unidadeID: Number(e.target.value) } : x)))
                  }
                >
                  {unidadesDe(l.produto.ID).map((u) => (
                    <option key={u.ID} value={u.ID}>
                      {u.NomeUnidade}
                    </option>
                  ))}
                </select>
              </td>
              <td>
                <input
                  aria-label="Quantidade"
                  inputMode="decimal"
                  value={l.quantidade}
                  onChange={(e) =>
                    setLinhas((ls) => ls.map((x) => (x.chave === l.chave ? { ...x, quantidade: e.target.value } : x)))
                  }
                />
              </td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{mt(preco(l))}</td>
              <td>
                <input
                  aria-label="Desconto"
                  inputMode="decimal"
                  placeholder="0"
                  value={l.desconto}
                  onChange={(e) =>
                    setLinhas((ls) => ls.map((x) => (x.chave === l.chave ? { ...x, desconto: e.target.value } : x)))
                  }
                />
              </td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                {linhaValida(l) ? mt(bruto(l) - desc(l)) : "—"}
              </td>
              <td>
                <button
                  className="admin-btn-secundario"
                  aria-label={`Remover ${l.produto.Nome}`}
                  onClick={() => setLinhas((ls) => ls.filter((x) => x.chave !== l.chave))}
                >
                  <TrashIcon className="admin-btn-icon-inline" aria-hidden="true" />
                </button>
              </td>
            </tr>
          ))}
          {linhas.length === 0 && (
            <tr>
              <td colSpan={7} className="admin-texto-fraco">
                Pesquise acima e acrescente os produtos da cotação.
              </td>
            </tr>
          )}
        </tbody>
      </table>

      <div className="admin-cotacao-rodape">
        <label className="admin-campo">
          Observações (aparecem no documento)
          <textarea
            rows={3}
            placeholder="ex: Entrega na obra incluída. Pagamento 50% na encomenda."
            value={observacoes}
            onChange={(e) => setObservacoes(e.target.value)}
          />
        </label>
        <div className="admin-cotacao-totais">
          <div>
            <span>Subtotal</span>
            <span>{mt(subtotal)}</span>
          </div>
          <label>
            <span>Desconto geral (MT)</span>
            <input inputMode="decimal" placeholder="0" value={descontoGeral} onChange={(e) => setDescontoGeral(e.target.value)} />
          </label>
          <div className="admin-cotacao-total">
            <span>Total</span>
            <span>{mt(Math.max(0, total))}</span>
          </div>
          <div className="admin-texto-fraco" style={{ textAlign: "right" }}>
            {taxaIVA > 0 ? `Inclui IVA (${num(taxaIVA)}%): ${mt(Math.max(0, iva))}` : "Isento de IVA"}
          </div>
        </div>
      </div>

      <div className="admin-form">
        <button className="admin-btn" disabled={!podeGuardar} onClick={guardar}>
          {aGuardar ? "A criar…" : "Criar cotação"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}

function VerCotacao({ id, onNavegar, onErro }: { id: number; onNavegar: Navegar; onErro: (e: string) => void }) {
  const [cc, setCC] = useState<usecase.CotacaoCompleta | null>(null);
  const [converter, setConverter] = useState(false);
  const [aviso, setAviso] = useState<string | null>(null);

  function carregar() {
    ObterCotacao(id)
      .then(setCC)
      .catch((e) => onErro(String(e)));
  }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- cotação fixa por instância
  useEffect(carregar, []);

  if (!cc) return <p>A carregar…</p>;
  const c = cc.Cotacao;
  const aberta = c.Estado === "aberta";

  function abrirPDF() {
    AbrirCotacaoPDF(id).catch((e) => onErro(String(e)));
  }

  function cancelar() {
    if (!window.confirm(`Cancelar a cotação ${cc!.Codigo}? Deixa de se poder converter em venda.`)) return;
    CancelarCotacao(id)
      .then(carregar)
      .catch((e) => onErro(String(e)));
  }

  return (
    <>
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>
            {cc.Codigo} <Estado c={cc} />
          </h2>
          <div className="admin-texto-fraco">
            Feita a {data(c.CriadoEm)} · válida até {data(cc.ValidaAte)}
            {cc.Expirada && " (expirou — os preços podem já não valer)"}
          </div>
        </div>
        <div className="admin-form">
          <button className="admin-btn-secundario" onClick={abrirPDF}>
            <DocumentArrowDownIcon className="admin-btn-icon-inline" aria-hidden="true" />
            Abrir PDF
          </button>
          {aberta && (
            <>
              <button className="admin-btn-secundario admin-btn-perigo-contorno" onClick={cancelar}>
                Cancelar cotação
              </button>
              <button className="admin-btn" onClick={() => setConverter(true)}>
                Converter em venda
              </button>
            </>
          )}
        </div>
      </div>

      {aviso && (
        <div className="admin-aviso-ok">
          {aviso}
          <button onClick={() => onNavegar({ ecra: "vendas" })}>Ver em Vendas</button>
        </div>
      )}

      <div className="admin-cotacao-cliente">
        <div className="admin-texto-fraco">Cliente</div>
        <strong>{c.ClienteNome}</strong>
        {c.ClienteNUIT && <div>NUIT {c.ClienteNUIT}</div>}
        {c.ClienteContacto && <div>{c.ClienteContacto}</div>}
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th style={{ textAlign: "right" }}>Qtd</th>
            <th>Unidade</th>
            <th style={{ textAlign: "right" }}>Preço</th>
            <th style={{ textAlign: "right" }}>Desconto</th>
            <th style={{ textAlign: "right" }}>Total</th>
          </tr>
        </thead>
        <tbody>
          {cc.Itens.map((i) => (
            <tr key={i.ID}>
              <td>{i.Descricao}</td>
              <td style={{ textAlign: "right" }}>{num(i.Quantidade)}</td>
              <td>{i.NomeUnidade}</td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{mt(i.PrecoUnitario)}</td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{i.Desconto > 0 ? mt(i.Desconto) : "—"}</td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{mt(i.Subtotal)}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="admin-cotacao-rodape">
        <div>
          {c.Observacoes && (
            <>
              <div className="admin-texto-fraco">Observações</div>
              <div style={{ whiteSpace: "pre-wrap" }}>{c.Observacoes}</div>
            </>
          )}
        </div>
        <div className="admin-cotacao-totais">
          <div>
            <span>Subtotal</span>
            <span>{mt(c.Subtotal)}</span>
          </div>
          {c.DescontoTotal > 0 && (
            <div>
              <span>Desconto</span>
              <span>−{mt(c.DescontoTotal)}</span>
            </div>
          )}
          <div className="admin-cotacao-total">
            <span>Total</span>
            <span>{mt(c.Total)}</span>
          </div>
          <div className="admin-texto-fraco" style={{ textAlign: "right" }}>
            {c.TaxaIVA > 0 ? `Inclui IVA (${num(c.TaxaIVA)}%): ${mt(c.ValorIVA)}` : "Isento de IVA"}
          </div>
        </div>
      </div>

      {converter && (
        <Modal titulo={`Converter ${cc.Codigo} em venda`} onFechar={() => setConverter(false)}>
          <ConverterForm
            cc={cc}
            onConvertida={(vendaID) => {
              setConverter(false);
              setAviso(`Venda #${vendaID} registada a partir da cotação; o stock já saiu.`);
              carregar();
            }}
            onCancelar={() => setConverter(false)}
            onErro={onErro}
          />
        </Modal>
      )}
    </>
  );
}

function ConverterForm({
  cc,
  onConvertida,
  onCancelar,
  onErro,
}: {
  cc: usecase.CotacaoCompleta;
  onConvertida: (vendaID: number) => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const c = cc.Cotacao;
  const registado = c.ClienteID != null;
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [metodo, setMetodo] = useState("dinheiro");
  const [valor, setValor] = useState(String(c.Total).replace(".", ","));
  const [aConverter, setAConverter] = useState(false);

  useEffect(() => {
    ListarArmazens()
      .then((lista) => {
        const ok = lista.filter((a) => a.Ativo && !a.ForaDeVenda);
        setArmazens(ok);
        const inicial = ok.find((a) => a.Tipo === "principal") ?? ok[0];
        if (inicial) setArmazemID(inicial.ID);
      })
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const pago = valor.trim() === "" ? 0 : lerNumero(valor);
  const valido = Number.isFinite(pago) && pago >= 0 && pago <= c.Total + 0.005;
  const falta = Math.max(0, c.Total - (valido ? pago : 0));
  const podeConverter = armazemID !== "" && valido && (registado || falta < 0.01) && !aConverter;

  function converter() {
    if (!podeConverter) return;
    const p = new usecase.ConverterCotacaoParams();
    p.CotacaoID = c.ID;
    p.ArmazemID = Number(armazemID);
    if (pago > 0) {
      const pg = new usecase.PagamentoVendaParams();
      pg.Metodo = metodo;
      pg.Valor = pago;
      p.Pagamentos = [pg];
    } else {
      p.Pagamentos = [];
    }
    setAConverter(true);
    ConverterCotacao(p)
      .then((v) => onConvertida(v.Venda.ID))
      .catch((e) => onErro(String(e)))
      .finally(() => setAConverter(false));
  }

  return (
    <div className="admin-produto-form">
      <p style={{ margin: 0 }}>
        Regista a venda de <strong>{mt(c.Total)}</strong> a <strong>{c.ClienteNome}</strong> ao preço cotado, e o stock
        sai do armazém escolhido.
      </p>
      <label className="admin-campo">
        Sai do armazém
        <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
      </label>
      <div className="admin-config-grelha">
        <label className="admin-campo">
          Pago agora (MT)
          <input inputMode="decimal" value={valor} onChange={(e) => setValor(e.target.value)} />
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
      {!valido ? (
        <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
          O valor pago tem de estar entre 0 e {mt(c.Total)}.
        </p>
      ) : falta >= 0.01 ? (
        <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
          {registado
            ? `Ficam ${mt(falta)} em dívida de ${c.ClienteNome} (venda a crédito).`
            : `Faltam ${mt(falta)} — sem cliente registado a venda tem de ser paga por inteiro.`}
        </p>
      ) : null}
      <div className="admin-form">
        <button className="admin-btn" disabled={!podeConverter} onClick={converter}>
          {aConverter ? "A registar…" : "Registar venda"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
