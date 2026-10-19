import { useEffect, useState } from "react";
import { MagnifyingGlassIcon } from "@heroicons/react/24/outline";
import {
  ListarProdutos,
  ListarCategorias,
  ListarUnidadesProduto,
  ListarTodasUnidades,
  CriarProduto,
  AtualizarProduto,
  AtivarProduto,
  DesativarProduto,
  DuplicarProduto,
  AdicionarUnidadeProduto,
  AtualizarUnidadeProduto,
} from "../wailsjs/go/admin/AdminApp";
import { api, domain } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";

type TipoProduto = "simples" | "composto" | "semi_elaborado";

// Rótulos amigáveis para o valor interno de TipoProduto (nunca muda — é o
// que fica gravado na base e usado nas validações do backend, só a etiqueta
// mostrada ao utilizador é que é mais legível que o enum em bruto).
const ROTULO_TIPO_PRODUTO: Record<TipoProduto, string> = {
  simples: "Simples",
  composto: "Composto",
  semi_elaborado: "Semi Produzido",
};

const AJUDA_TIPO: Record<TipoProduto, string> = {
  simples: "Comprado e vendido tal como está.",
  composto: "Feito na hora a partir de outros produtos (ficha técnica); não guarda stock próprio.",
  semi_elaborado: "Preparado antes a partir de outros produtos e guardado em stock.",
};

const mt = (n: number) => `${n.toLocaleString("pt-PT", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} MT`;
const lerNumero = (s: string) => Number(s.trim().replace(",", "."));
const numeroValido = (s: string) => s.trim() !== "" && Number.isFinite(lerNumero(s));

type ModalEstado = { aberto: false } | { aberto: true; produto: domain.Produto | null };
type FiltroEstado = "ativos" | "inativos" | "todos";

/**
 * CRUD de produtos da única loja que este cmd/admin serve — inclui as
 * "variações ou grupos" do mesmo produto (SDD 4.1: `UnidadeProduto`, ex:
 * unidade/caixa, garrafa/dose): um produto só é vendável com pelo menos
 * uma, e pode ter várias em simultâneo, geridas aqui. A lista pesquisa por
 * nome ou código de barras e mostra o preço da unidade padrão de venda;
 * criar/editar abre num modal.
 */
export default function ProdutosScreen({ onErro }: { onErro: (e: string) => void }) {
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [unidades, setUnidades] = useState<domain.UnidadeProduto[]>([]);
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [modal, setModal] = useState<ModalEstado>({ aberto: false });
  const [pesquisa, setPesquisa] = useState("");
  const [categoriaFiltro, setCategoriaFiltro] = useState<number | "" | "sem">("");
  const [estado, setEstado] = useState<FiltroEstado>("ativos");

  function carregar() {
    Promise.all([ListarProdutos(), ListarCategorias(), ListarTodasUnidades()])
      .then(([p, c, u]) => {
        setProdutos(p);
        setCategorias(c);
        setUnidades(u);
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(carregar, []);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;
  const unidadesDe = (id: number) => unidades.filter((u) => u.ProdutoID === id);

  function alternarAtivo(p: domain.Produto) {
    const acao = p.Ativo ? DesativarProduto(p.ID) : AtivarProduto(p.ID);
    acao.then(carregar).catch((e) => onErro(String(e)));
  }

  function duplicar(p: domain.Produto) {
    DuplicarProduto(p.ID).then(carregar).catch((e) => onErro(String(e)));
  }

  function fecharModal() {
    setModal({ aberto: false });
  }

  const termo = pesquisa.trim().toLowerCase();
  const visiveis = produtos
    .filter((p) => estado === "todos" || (estado === "ativos" ? p.Ativo : !p.Ativo))
    .filter((p) =>
      categoriaFiltro === "" ? true : categoriaFiltro === "sem" ? p.CategoriaID == null : p.CategoriaID === categoriaFiltro,
    )
    .filter(
      (p) =>
        !termo ||
        p.Nome.toLowerCase().includes(termo) ||
        unidadesDe(p.ID).some((u) => (u.CodigoBarras ?? "").toLowerCase() === termo),
    )
    .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"));

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Produtos" }]} />
      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Produtos</h2>
        <button className="admin-btn" onClick={() => setModal({ aberto: true, produto: null })}>
          + Criar produto
        </button>
      </div>

      <div className="admin-form">
        <label className="admin-pesquisa">
          <MagnifyingGlassIcon className="admin-pesquisa-icone" aria-hidden="true" />
          <input
            placeholder="Pesquisar nome ou código de barras"
            value={pesquisa}
            onChange={(e) => setPesquisa(e.target.value)}
          />
        </label>
        <select
          value={categoriaFiltro}
          onChange={(e) =>
            setCategoriaFiltro(e.target.value === "" ? "" : e.target.value === "sem" ? "sem" : Number(e.target.value))
          }
        >
          <option value="">Todas as categorias</option>
          <option value="sem">Sem categoria</option>
          {categorias.map((c) => (
            <option key={c.ID} value={c.ID}>
              {c.Nome}
            </option>
          ))}
        </select>
        <select value={estado} onChange={(e) => setEstado(e.target.value as FiltroEstado)}>
          <option value="ativos">Ativos</option>
          <option value="inativos">Inativos</option>
          <option value="todos">Todos</option>
        </select>
        <span style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>{visiveis.length} produto(s)</span>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th style={{ width: "28%" }}>Nome</th>
            <th>Categoria</th>
            <th>Tipo</th>
            <th style={{ textAlign: "right" }}>Preço de venda</th>
            <th style={{ textAlign: "right" }}>Stock mínimo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {visiveis.map((p) => {
            const us = unidadesDe(p.ID);
            const padrao = us.find((u) => u.IsPadraoVenda) ?? us[0];
            return (
              <tr key={p.ID} className={p.Ativo ? undefined : "admin-linha-inativa"}>
                <td>
                  {p.Nome}
                  {p.LoteControlado && <span className="admin-etiqueta">lotes</span>}
                  {!p.Ativo && <span className="admin-etiqueta">inativo</span>}
                </td>
                <td>{p.CategoriaID != null ? nomeCategoria(p.CategoriaID) : "—"}</td>
                <td>{ROTULO_TIPO_PRODUTO[p.TipoProduto as TipoProduto] ?? p.TipoProduto}</td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  {padrao ? (
                    <>
                      {mt(padrao.PrecoVenda)}
                      <span className="admin-texto-fraco"> / {padrao.NomeUnidade}</span>
                      {us.length > 1 && <div className="admin-texto-fraco">+{us.length - 1} outra(s) unidade(s)</div>}
                    </>
                  ) : (
                    "—"
                  )}
                </td>
                <td style={{ textAlign: "right" }}>
                  {p.StockMinimo != null ? `${p.StockMinimo.toLocaleString("pt-PT")} ${p.UnidadeBase}` : "—"}
                </td>
                <td>
                  <div className="admin-acoes">
                    <button className="admin-btn-secundario" onClick={() => setModal({ aberto: true, produto: p })}>
                      Editar
                    </button>
                    <button className="admin-btn-secundario" onClick={() => duplicar(p)}>
                      Duplicar
                    </button>
                    <button className="admin-btn-secundario" onClick={() => alternarAtivo(p)}>
                      {p.Ativo ? "Desativar" : "Ativar"}
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
          {visiveis.length === 0 && (
            <tr>
              <td colSpan={6}>
                {aCarregar
                  ? "A carregar…"
                  : produtos.length === 0
                    ? 'Sem produtos — clique em "+ Criar produto".'
                    : "Nenhum produto com estes filtros."}
              </td>
            </tr>
          )}
        </tbody>
      </table>

      {modal.aberto && (
        <Modal titulo={modal.produto ? `Editar ${modal.produto.Nome}` : "Criar produto"} largo onFechar={fecharModal}>
          <ProdutoForm
            produto={modal.produto}
            categorias={categorias}
            onGuardado={() => {
              fecharModal();
              carregar();
            }}
            onCancelar={fecharModal}
            onErro={onErro}
          />
        </Modal>
      )}
    </div>
  );
}

type UnidadeRascunho = {
  chave: number;
  id?: number;
  original?: domain.UnidadeProduto;
  nomeUnidade: string;
  qtdEmUnidadeBase: string;
  precoCusto: string;
  precoVenda: string;
  codigoBarras: string;
  isPadraoVenda: boolean;
  isPadraoCompra: boolean;
};

let proximaChave = 0;

function unidadeVazia(padrao: boolean): UnidadeRascunho {
  return {
    chave: proximaChave++,
    nomeUnidade: "",
    qtdEmUnidadeBase: "1",
    precoCusto: "",
    precoVenda: "",
    codigoBarras: "",
    isPadraoVenda: padrao,
    isPadraoCompra: padrao,
  };
}

function deUnidade(u: domain.UnidadeProduto): UnidadeRascunho {
  return {
    chave: proximaChave++,
    id: u.ID,
    original: u,
    nomeUnidade: u.NomeUnidade,
    qtdEmUnidadeBase: String(u.QtdEmUnidadeBase),
    precoCusto: String(u.PrecoCusto),
    precoVenda: String(u.PrecoVenda),
    codigoBarras: u.CodigoBarras ?? "",
    isPadraoVenda: u.IsPadraoVenda,
    isPadraoCompra: u.IsPadraoCompra,
  };
}

function paraUnidadeProduto(u: UnidadeRascunho, produtoID?: number): domain.UnidadeProduto {
  const unidade = new domain.UnidadeProduto();
  if (u.id != null) unidade.ID = u.id;
  if (produtoID != null) unidade.ProdutoID = produtoID;
  unidade.NomeUnidade = u.nomeUnidade.trim();
  unidade.QtdEmUnidadeBase = lerNumero(u.qtdEmUnidadeBase);
  unidade.PrecoCusto = u.precoCusto.trim() === "" ? 0 : lerNumero(u.precoCusto);
  unidade.PrecoVenda = u.precoVenda.trim() === "" ? 0 : lerNumero(u.precoVenda);
  unidade.CodigoBarras = u.codigoBarras.trim() || undefined;
  unidade.IsPadraoVenda = u.isPadraoVenda;
  unidade.IsPadraoCompra = u.isPadraoCompra;
  return unidade;
}

function alterada(u: UnidadeRascunho): boolean {
  if (!u.original) return false;
  const nova = paraUnidadeProduto(u);
  const o = u.original;
  return (
    nova.NomeUnidade !== o.NomeUnidade ||
    nova.QtdEmUnidadeBase !== o.QtdEmUnidadeBase ||
    nova.PrecoCusto !== o.PrecoCusto ||
    nova.PrecoVenda !== o.PrecoVenda ||
    (nova.CodigoBarras ?? "") !== (o.CodigoBarras ?? "") ||
    nova.IsPadraoVenda !== o.IsPadraoVenda ||
    nova.IsPadraoCompra !== o.IsPadraoCompra
  );
}

function ProdutoForm({
  produto,
  categorias,
  onGuardado,
  onCancelar,
  onErro,
}: {
  produto: domain.Produto | null;
  categorias: domain.Categoria[];
  onGuardado: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(produto?.Nome ?? "");
  const [categoriaId, setCategoriaId] = useState<number | "">(produto?.CategoriaID ?? "");
  const [tipoProduto, setTipoProduto] = useState<TipoProduto>((produto?.TipoProduto as TipoProduto) ?? "simples");
  const [unidadeBase, setUnidadeBase] = useState(produto?.UnidadeBase ?? "unidade");
  const [stockMinimo, setStockMinimo] = useState(produto?.StockMinimo?.toString() ?? "");
  const [loteControlado, setLoteControlado] = useState(produto?.LoteControlado ?? false);
  const [linhas, setLinhas] = useState<UnidadeRascunho[]>(produto ? [] : [unidadeVazia(true)]);
  const [aCarregarUnidades, setACarregarUnidades] = useState(!!produto);
  const [aGuardar, setAGuardar] = useState(false);

  useEffect(() => {
    if (!produto) return;
    ListarUnidadesProduto(produto.ID)
      .then((us) => setLinhas(us.map(deUnidade)))
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregarUnidades(false));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só corre para o produto desta instância
  }, []);

  function mudar(chave: number, campo: keyof UnidadeRascunho, valor: string | boolean) {
    setLinhas((prev) => prev.map((u) => (u.chave === chave ? { ...u, [campo]: valor } : u)));
  }

  // Só uma unidade é a padrão de venda (e só uma a de compra).
  function escolherPadrao(chave: number, campo: "isPadraoVenda" | "isPadraoCompra") {
    setLinhas((prev) => prev.map((u) => ({ ...u, [campo]: u.chave === chave })));
  }

  const preenchidas = linhas.filter((u) => u.id != null || u.nomeUnidade.trim());
  const erroUnidade = preenchidas
    .map((u) => {
      const nomeU = u.nomeUnidade.trim() || "sem nome";
      if (!u.nomeUnidade.trim()) return "Todas as unidades precisam de nome.";
      if (!numeroValido(u.qtdEmUnidadeBase) || lerNumero(u.qtdEmUnidadeBase) <= 0)
        return `"${nomeU}": a quantidade tem de ser maior que zero.`;
      for (const v of [u.precoCusto, u.precoVenda])
        if (v.trim() !== "" && (!numeroValido(v) || lerNumero(v) < 0)) return `"${nomeU}": preço inválido.`;
      return null;
    })
    .find((e) => e);
  const stockMinimoValido = stockMinimo.trim() === "" || (numeroValido(stockMinimo) && lerNumero(stockMinimo) >= 0);
  const podeGuardar = !!nome.trim() && preenchidas.length > 0 && !erroUnidade && stockMinimoValido && !aGuardar;

  async function guardar() {
    if (!podeGuardar) return;

    const p = new domain.Produto();
    p.ID = produto?.ID ?? 0;
    p.Nome = nome.trim();
    p.CategoriaID = categoriaId === "" ? undefined : Number(categoriaId);
    p.TipoProduto = tipoProduto;
    p.UnidadeBase = unidadeBase.trim() || "unidade";
    p.StockMinimo = stockMinimo.trim() === "" ? undefined : lerNumero(stockMinimo);
    p.LoteControlado = tipoProduto !== "composto" && loteControlado;

    setAGuardar(true);
    try {
      if (!produto) {
        const req = new api.CriarProdutoRequest();
        req.Produto = p;
        req.Unidades = preenchidas.map((u) => paraUnidadeProduto(u));
        await CriarProduto(req);
      } else {
        await AtualizarProduto(p);
        for (const u of preenchidas) {
          if (u.id == null) await AdicionarUnidadeProduto(paraUnidadeProduto(u, produto.ID));
          else if (alterada(u)) await AtualizarUnidadeProduto(paraUnidadeProduto(u, produto.ID));
        }
      }
      onGuardado();
    } catch (e) {
      onErro(String(e));
    } finally {
      setAGuardar(false);
    }
  }

  const base = unidadeBase.trim() || "unidade";

  return (
    <div className="admin-produto-form">
      <div className="admin-produto-campos">
        <label className="admin-campo admin-produto-nome">
          Nome do produto
          <input autoFocus value={nome} onChange={(e) => setNome(e.target.value)} placeholder="ex: Martelo de unha 500g" />
        </label>
        <label className="admin-campo">
          Categoria
          <select value={categoriaId} onChange={(e) => setCategoriaId(e.target.value ? Number(e.target.value) : "")}>
            <option value="">Sem categoria</option>
            {categorias.map((c) => (
              <option key={c.ID} value={c.ID}>
                {c.Nome}
              </option>
            ))}
          </select>
        </label>
        <label className="admin-campo">
          Tipo
          <select
            value={tipoProduto}
            onChange={(e) => setTipoProduto(e.target.value as TipoProduto)}
            disabled={!!produto}
            title={produto ? "O tipo não se altera depois de criado" : undefined}
          >
            <option value="simples">{ROTULO_TIPO_PRODUTO.simples}</option>
            <option value="composto">{ROTULO_TIPO_PRODUTO.composto}</option>
            <option value="semi_elaborado">{ROTULO_TIPO_PRODUTO.semi_elaborado}</option>
          </select>
        </label>
        <label className="admin-campo">
          Unidade base (como se conta o stock)
          <input
            value={unidadeBase}
            onChange={(e) => setUnidadeBase(e.target.value)}
            disabled={!!produto}
            title={produto ? "A unidade base não se altera depois de criado" : undefined}
            placeholder="ex: unidade, kg, metro"
          />
        </label>
        <label className="admin-campo">
          Stock mínimo (opcional)
          <input
            inputMode="decimal"
            value={stockMinimo}
            onChange={(e) => setStockMinimo(e.target.value)}
            placeholder={`em ${base}`}
          />
        </label>
        {tipoProduto !== "composto" && (
          <label className="admin-produto-check">
            <input type="checkbox" checked={loteControlado} onChange={(e) => setLoteControlado(e.target.checked)} />
            Controla lotes e validade
          </label>
        )}
      </div>
      <div className="admin-texto-fraco">{AJUDA_TIPO[tipoProduto]}</div>

      <div className="admin-unidades-bloco">
        <div>
          <div className="admin-unidades-titulo">Unidades de venda e compra</div>
          <div className="admin-texto-fraco">
            Ex: vender à unidade e também em caixa de 24. A padrão de venda é a que o PDV usa por defeito.
          </div>
        </div>
        <table className="admin-tabela admin-unidades-tabela">
          <thead>
            <tr>
              <th>Nome</th>
              <th>Contém ({base})</th>
              <th>Preço de custo</th>
              <th>Preço de venda</th>
              <th>Código de barras</th>
              <th title="Unidade usada por defeito na venda">Venda</th>
              <th title="Unidade usada por defeito na compra">Compra</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {linhas.map((u) => (
              <tr key={u.chave}>
                <td>
                  <input
                    aria-label="Nome da unidade"
                    placeholder="ex: caixa de 24"
                    value={u.nomeUnidade}
                    onChange={(e) => mudar(u.chave, "nomeUnidade", e.target.value)}
                  />
                </td>
                <td>
                  <input
                    aria-label={`Quantidade em ${base}`}
                    inputMode="decimal"
                    value={u.qtdEmUnidadeBase}
                    onChange={(e) => mudar(u.chave, "qtdEmUnidadeBase", e.target.value)}
                  />
                </td>
                <td>
                  <input
                    aria-label="Preço de custo"
                    inputMode="decimal"
                    placeholder="0,00"
                    value={u.precoCusto}
                    onChange={(e) => mudar(u.chave, "precoCusto", e.target.value)}
                  />
                </td>
                <td>
                  <input
                    aria-label="Preço de venda"
                    inputMode="decimal"
                    placeholder="0,00"
                    value={u.precoVenda}
                    onChange={(e) => mudar(u.chave, "precoVenda", e.target.value)}
                  />
                </td>
                <td>
                  <input
                    aria-label="Código de barras"
                    placeholder="opcional"
                    value={u.codigoBarras}
                    onChange={(e) => mudar(u.chave, "codigoBarras", e.target.value)}
                  />
                </td>
                <td className="admin-unidades-radio">
                  <input
                    type="radio"
                    aria-label="Padrão de venda"
                    checked={u.isPadraoVenda}
                    onChange={() => escolherPadrao(u.chave, "isPadraoVenda")}
                  />
                </td>
                <td className="admin-unidades-radio">
                  <input
                    type="radio"
                    aria-label="Padrão de compra"
                    checked={u.isPadraoCompra}
                    onChange={() => escolherPadrao(u.chave, "isPadraoCompra")}
                  />
                </td>
                <td>
                  {u.id == null && linhas.length > 1 && (
                    <button
                      className="admin-btn-secundario"
                      onClick={() => setLinhas((prev) => prev.filter((x) => x.chave !== u.chave))}
                    >
                      Remover
                    </button>
                  )}
                </td>
              </tr>
            ))}
            {aCarregarUnidades && (
              <tr>
                <td colSpan={8}>A carregar…</td>
              </tr>
            )}
          </tbody>
        </table>
        <div>
          <button className="admin-btn-secundario" onClick={() => setLinhas((prev) => [...prev, unidadeVazia(false)])}>
            + Adicionar unidade
          </button>
        </div>
      </div>

      {(erroUnidade || !stockMinimoValido) && (
        <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
          {erroUnidade ?? "O stock mínimo tem de ser um número igual ou maior que zero."}
        </p>
      )}

      <div className="admin-form">
        <button className="admin-btn" disabled={!podeGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : produto ? "Guardar alterações" : "+ Criar produto"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
