import { useEffect, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import {
  CubeIcon,
  TagIcon,
  BuildingOffice2Icon,
  ArchiveBoxIcon,
  ClipboardDocumentCheckIcon,
  ClipboardDocumentListIcon,
  BeakerIcon,
  UserGroupIcon,
  ChartBarIcon,
} from "@heroicons/react/24/outline";
import {
  ListarArmazens,
  CriarArmazem,
  AtivarArmazem,
  DesativarArmazem,
  ListarProdutos,
  ListarTodosProdutos,
  ListarUnidadesProduto,
  CriarProduto,
  AtualizarProduto,
  AtivarProduto,
  DesativarProduto,
  DuplicarProduto,
  AdicionarUnidadeProduto,
  ListarCategorias,
  CriarCategoria,
  AtualizarCategoria,
  AtivarCategoria,
  DesativarCategoria,
  ObterStock,
  RegistarEntradaStock,
  TransferirStock,
  RegistarPerdaStock,
  IniciarInventario,
  RegistarContagemInventario,
  FecharInventario,
} from "../wailsjs/go/api/App";
import { api, domain, usecase } from "../wailsjs/go/models";
import FichasTecnicasSection from "./FichasTecnicas";
import FarmaciaSection from "./Farmacia";
import ClientesSection from "./Clientes";
import RelatoriosSection from "./Relatorios";
import "./Administracao.css";

type Seccao =
  | "produtos"
  | "categorias"
  | "armazens"
  | "stock"
  | "inventario"
  | "fichas"
  | "farmacia"
  | "clientes"
  | "relatorios";

type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;

// Rótulos amigáveis para o valor interno de TipoProduto (nunca muda — é o
// que fica gravado na base e usado nas validações do backend, só a etiqueta
// mostrada ao utilizador é que é mais legível que o enum em bruto).
const ROTULO_TIPO_PRODUTO: Record<"simples" | "composto" | "semi_elaborado", string> = {
  simples: "Simples",
  composto: "Composto",
  semi_elaborado: "Semi Produzido",
};

/**
 * Ponto único de administração da loja (SDD 9.5): produtos, categorias,
 * armazéns, stock e inventário aparecem sempre; fichas técnicas e farmácia
 * só quando o respetivo módulo está ativo (decidido no wizard, SDD 9.1/9.2)
 * — o mesmo `ModulosAtivos` que decide qual ecrã de venda aparece em
 * App.tsx decide também o que aparece aqui. Substitui o scaffolding
 * FaseUm.tsx da Fase 1, que ficava tudo empilhado numa página só, sem
 * navegação nem separação do ecrã de venda.
 */
export default function Administracao({ modulosAtivos }: { modulosAtivos: string[] }) {
  const [seccao, setSeccao] = useState<Seccao>("produtos");
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [erro, setErro] = useState<string | null>(null);

  const recarregar = () => {
    ListarArmazens().then(setArmazens).catch((e) => setErro(String(e)));
    ListarProdutos().then(setProdutos).catch((e) => setErro(String(e)));
    ListarCategorias().then(setCategorias).catch((e) => setErro(String(e)));
  };

  useEffect(recarregar, []);

  const temFichas = modulosAtivos.includes("fichas_tecnicas");
  const temFarmacia = modulosAtivos.includes("lotes_validade") || modulosAtivos.includes("receitas_medicas");

  const itens: { chave: Seccao; nome: string; Icone: IconeSvg }[] = [
    { chave: "produtos", nome: "Produtos", Icone: CubeIcon },
    { chave: "categorias", nome: "Categorias", Icone: TagIcon },
    { chave: "armazens", nome: "Armazéns", Icone: BuildingOffice2Icon },
    { chave: "stock", nome: "Stock", Icone: ArchiveBoxIcon },
    { chave: "inventario", nome: "Inventário", Icone: ClipboardDocumentCheckIcon },
    ...(temFichas ? [{ chave: "fichas" as const, nome: "Fichas técnicas", Icone: ClipboardDocumentListIcon }] : []),
    ...(temFarmacia ? [{ chave: "farmacia" as const, nome: "Farmácia", Icone: BeakerIcon }] : []),
    { chave: "clientes", nome: "Clientes e dívidas", Icone: UserGroupIcon },
    { chave: "relatorios", nome: "Relatórios", Icone: ChartBarIcon },
  ];

  return (
    <div className="admin2">
      <aside className="admin2-sidebar">
        <nav className="admin2-nav">
          {itens.map((item) => (
            <button
              key={item.chave}
              className={`admin2-nav-item${seccao === item.chave ? " active" : ""}`}
              onClick={() => setSeccao(item.chave)}
            >
              <item.Icone className="admin2-nav-icon" aria-hidden="true" />
              {item.nome}
            </button>
          ))}
        </nav>
      </aside>

      <div className="admin2-content fase1">
        {erro && (
          <div className="fase1-erro">
            {erro} <button onClick={() => setErro(null)}>×</button>
          </div>
        )}

        {seccao === "produtos" && (
          <ProdutosSection categorias={categorias} onChangeGlobal={recarregar} onErro={setErro} />
        )}
        {seccao === "categorias" && (
          <CategoriasSection categorias={categorias} onChange={recarregar} onErro={setErro} />
        )}
        {seccao === "armazens" && <ArmazensSection armazens={armazens} onChange={recarregar} onErro={setErro} />}
        {seccao === "stock" && <StockSection produtos={produtos} armazens={armazens} onErro={setErro} />}
        {seccao === "inventario" && (
          <InventarioSection produtos={produtos} armazens={armazens} onErro={setErro} />
        )}
        {seccao === "fichas" && temFichas && (
          <FichasTecnicasSection produtos={produtos} armazens={armazens} onErro={setErro} />
        )}
        {seccao === "farmacia" && temFarmacia && (
          <FarmaciaSection produtos={produtos} armazens={armazens} onErro={setErro} />
        )}
        {seccao === "clientes" && <ClientesSection onErro={setErro} />}
        {seccao === "relatorios" && <RelatoriosSection armazens={armazens} onErro={setErro} />}
      </div>
    </div>
  );
}

function run(fn: () => Promise<unknown>, onErro: (e: string) => void, then?: () => void) {
  fn()
    .then(() => then?.())
    .catch((e) => onErro(String(e)));
}

// --- Armazéns --------------------------------------------------------------

function ArmazensSection({
  armazens,
  onChange,
  onErro,
}: {
  armazens: domain.Armazem[];
  onChange: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState("");
  const [tipo, setTipo] = useState<"principal" | "ponto_venda" | "deposito">("principal");

  function criar() {
    const a = new domain.Armazem();
    a.Nome = nome;
    a.Tipo = tipo;
    run(() => CriarArmazem(a), onErro, () => {
      setNome("");
      onChange();
    });
  }

  function alternarAtivo(a: domain.Armazem) {
    run(() => (a.Ativo ? DesativarArmazem(a.ID) : AtivarArmazem(a.ID)), onErro, onChange);
  }

  return (
    <section>
      <h2>Armazéns</h2>
      <table>
        <thead>
          <tr>
            <th>Nome</th>
            <th>Tipo</th>
            <th>Ativo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {armazens.map((a) => (
            <tr key={a.ID}>
              <td>{a.Nome}</td>
              <td>{a.Tipo}</td>
              <td>{a.Ativo ? "sim" : "não"}</td>
              <td>
                <button onClick={() => alternarAtivo(a)}>{a.Ativo ? "Desativar" : "Ativar"}</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="fase1-form">
        <input placeholder="Nome do armazém" value={nome} onChange={(e) => setNome(e.target.value)} />
        <select value={tipo} onChange={(e) => setTipo(e.target.value as typeof tipo)}>
          <option value="principal">principal</option>
          <option value="ponto_venda">ponto_venda</option>
          <option value="deposito">deposito</option>
        </select>
        <button disabled={!nome} onClick={criar}>
          + Criar armazém
        </button>
      </div>
    </section>
  );
}

// --- Categorias --------------------------------------------------------------

function CategoriasSection({
  categorias,
  onChange,
  onErro,
}: {
  categorias: domain.Categoria[];
  onChange: () => void;
  onErro: (e: string) => void;
}) {
  const [emEdicao, setEmEdicao] = useState<domain.Categoria | null>(null);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;

  function alternarAtivo(c: domain.Categoria) {
    run(() => (c.Ativo ? DesativarCategoria(c.ID) : AtivarCategoria(c.ID)), onErro, onChange);
  }

  return (
    <section>
      <h2>Categorias</h2>
      <table>
        <thead>
          <tr>
            <th>Nome</th>
            <th>Categoria-mãe</th>
            <th>Ativo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {categorias.map((c) => (
            <tr key={c.ID}>
              <td>{c.Nome}</td>
              <td>{c.CategoriaPaiID != null ? nomeCategoria(c.CategoriaPaiID) : "—"}</td>
              <td>{c.Ativo ? "sim" : "não"}</td>
              <td>
                <button onClick={() => setEmEdicao(c)}>Editar</button>{" "}
                <button onClick={() => alternarAtivo(c)}>{c.Ativo ? "Desativar" : "Ativar"}</button>
              </td>
            </tr>
          ))}
          {categorias.length === 0 && (
            <tr>
              <td colSpan={4}>Sem categorias — crie uma abaixo.</td>
            </tr>
          )}
        </tbody>
      </table>
      <CategoriaForm
        key={emEdicao?.ID ?? "nova"}
        categoria={emEdicao}
        categorias={categorias}
        onGuardado={() => {
          setEmEdicao(null);
          onChange();
        }}
        onCancelar={() => setEmEdicao(null)}
        onErro={onErro}
      />
    </section>
  );
}

function CategoriaForm({
  categoria,
  categorias,
  onGuardado,
  onCancelar,
  onErro,
}: {
  categoria: domain.Categoria | null;
  categorias: domain.Categoria[];
  onGuardado: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(categoria?.Nome ?? "");
  const [categoriaPaiId, setCategoriaPaiId] = useState<number | "">(categoria?.CategoriaPaiID ?? "");

  function guardar() {
    const c = new domain.Categoria();
    c.ID = categoria?.ID ?? 0;
    c.Nome = nome;
    c.CategoriaPaiID = categoriaPaiId === "" ? undefined : Number(categoriaPaiId);
    const pedido = categoria ? AtualizarCategoria(c) : CriarCategoria(c);
    run(() => pedido, onErro, onGuardado);
  }

  return (
    <div className="fase1-form">
      <input placeholder="Nome da categoria" value={nome} onChange={(e) => setNome(e.target.value)} />
      <select value={categoriaPaiId} onChange={(e) => setCategoriaPaiId(e.target.value ? Number(e.target.value) : "")}>
        <option value="">Sem categoria-mãe</option>
        {categorias
          .filter((c) => c.ID !== categoria?.ID)
          .map((c) => (
            <option key={c.ID} value={c.ID}>
              {c.Nome}
            </option>
          ))}
      </select>
      <button disabled={!nome} onClick={guardar}>
        {categoria ? "Guardar alterações" : "+ Criar categoria"}
      </button>
      {categoria && <button onClick={onCancelar}>Cancelar</button>}
    </div>
  );
}

// --- Produtos ----------------------------------------------------------------

// ProdutosSection tem a sua própria lista de produtos (ListarTodosProdutos,
// incluindo inativos, para poder reativá-los aqui) em vez de reaproveitar o
// `produtos` do estado partilhado do Administracao.tsx — esse fica só com
// ativos porque alimenta também os seletores de Stock/Inventário/Fichas
// técnicas/Farmácia, onde um produto inativo não devia aparecer. onChangeGlobal
// refresca esse estado partilhado sempre que um produto é criado/editado aqui.
function ProdutosSection({
  categorias,
  onChangeGlobal,
  onErro,
}: {
  categorias: domain.Categoria[];
  onChangeGlobal: () => void;
  onErro: (e: string) => void;
}) {
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [emEdicao, setEmEdicao] = useState<domain.Produto | null>(null);

  function carregar() {
    ListarTodosProdutos()
      .then(setProdutos)
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de Administracao
  useEffect(carregar, []);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;

  function alternarAtivo(p: domain.Produto) {
    run(() => (p.Ativo ? DesativarProduto(p.ID) : AtivarProduto(p.ID)), onErro, () => {
      carregar();
      onChangeGlobal();
    });
  }

  function duplicar(p: domain.Produto) {
    run(() => DuplicarProduto(p.ID), onErro, () => {
      carregar();
      onChangeGlobal();
    });
  }

  return (
    <section>
      <h2>Produtos</h2>
      <table>
        <thead>
          <tr>
            <th>Nome</th>
            <th>Categoria</th>
            <th>Tipo</th>
            <th>Unidade base</th>
            <th>Lote controlado</th>
            <th>Ativo</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {produtos.map((p) => (
            <tr key={p.ID}>
              <td>{p.Nome}</td>
              <td>{p.CategoriaID != null ? nomeCategoria(p.CategoriaID) : "—"}</td>
              <td>{ROTULO_TIPO_PRODUTO[p.TipoProduto as keyof typeof ROTULO_TIPO_PRODUTO] ?? p.TipoProduto}</td>
              <td>{p.UnidadeBase}</td>
              <td>{p.LoteControlado ? "sim" : "não"}</td>
              <td>{p.Ativo ? "sim" : "não"}</td>
              <td>
                <button onClick={() => setEmEdicao(p)}>Editar</button>{" "}
                <button onClick={() => duplicar(p)}>Duplicar</button>{" "}
                <button onClick={() => alternarAtivo(p)}>{p.Ativo ? "Desativar" : "Ativar"}</button>
              </td>
            </tr>
          ))}
          {produtos.length === 0 && (
            <tr>
              <td colSpan={7}>{aCarregar ? "A carregar…" : "Sem produtos — crie um abaixo."}</td>
            </tr>
          )}
        </tbody>
      </table>

      <ProdutoForm
        key={emEdicao?.ID ?? "novo"}
        produto={emEdicao}
        categorias={categorias}
        onGuardado={() => {
          setEmEdicao(null);
          carregar();
          onChangeGlobal();
        }}
        onCancelar={() => setEmEdicao(null)}
        onErro={onErro}
      />
    </section>
  );
}

type UnidadeRascunho = {
  nomeUnidade: string;
  qtdEmUnidadeBase: string;
  precoCusto: string;
  precoVenda: string;
  codigoBarras: string;
  isPadraoVenda: boolean;
  isPadraoCompra: boolean;
};

function unidadeVazia(padrao: boolean): UnidadeRascunho {
  return {
    nomeUnidade: "",
    qtdEmUnidadeBase: "1",
    precoCusto: "0",
    precoVenda: "0",
    codigoBarras: "",
    isPadraoVenda: padrao,
    isPadraoCompra: padrao,
  };
}

function paraUnidadeProduto(u: UnidadeRascunho, produtoID?: number): domain.UnidadeProduto {
  const unidade = new domain.UnidadeProduto();
  if (produtoID != null) unidade.ProdutoID = produtoID;
  unidade.NomeUnidade = u.nomeUnidade.trim();
  unidade.QtdEmUnidadeBase = Number(u.qtdEmUnidadeBase);
  unidade.PrecoCusto = Number(u.precoCusto);
  unidade.PrecoVenda = Number(u.precoVenda);
  unidade.CodigoBarras = u.codigoBarras.trim() || undefined;
  unidade.IsPadraoVenda = u.isPadraoVenda;
  unidade.IsPadraoCompra = u.isPadraoCompra;
  return unidade;
}

// ProdutoForm serve tanto para criar (produto=null, com o bloco completo de
// variações/grupos — SDD 4.1) como para editar (nome/categoria/stock
// mínimo/lote controlado; tipo e unidade base ficam fixos pelo que
// AtualizarProduto aceita, e as unidades já criadas só podem crescer aqui,
// nunca ser removidas/editadas — sem suporte nenhum disso no backend).
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
  const [tipoProduto, setTipoProduto] = useState<"simples" | "composto" | "semi_elaborado">(
    (produto?.TipoProduto as "simples" | "composto" | "semi_elaborado") ?? "simples",
  );
  const [unidadeBase, setUnidadeBase] = useState(produto?.UnidadeBase ?? "unidade");
  const [stockMinimo, setStockMinimo] = useState(produto?.StockMinimo?.toString() ?? "");
  const [loteControlado, setLoteControlado] = useState(produto?.LoteControlado ?? false);
  const [unidadesNovas, setUnidadesNovas] = useState<UnidadeRascunho[]>([unidadeVazia(true)]);
  const [aGuardar, setAGuardar] = useState(false);

  const [unidadesExistentes, setUnidadesExistentes] = useState<domain.UnidadeProduto[]>([]);
  const [novaUnidade, setNovaUnidade] = useState<UnidadeRascunho>(unidadeVazia(false));
  const [aAdicionarUnidade, setAAdicionarUnidade] = useState(false);

  useEffect(() => {
    if (!produto) return;
    ListarUnidadesProduto(produto.ID)
      .then(setUnidadesExistentes)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só corre para o produto desta instância (key troca-o)
  }, []);

  function atualizarUnidadeNova(i: number, campo: keyof UnidadeRascunho, valor: string | boolean) {
    setUnidadesNovas((prev) => prev.map((u, idx) => (idx === i ? { ...u, [campo]: valor } : u)));
  }

  function guardar() {
    if (!nome.trim() || aGuardar) return;

    const p = new domain.Produto();
    p.ID = produto?.ID ?? 0;
    p.Nome = nome.trim();
    p.CategoriaID = categoriaId === "" ? undefined : Number(categoriaId);
    p.TipoProduto = tipoProduto;
    p.UnidadeBase = unidadeBase.trim() || "unidade";
    p.StockMinimo = stockMinimo === "" ? undefined : Number(stockMinimo);
    p.LoteControlado = loteControlado;

    let pedido: Promise<unknown>;
    if (produto) {
      pedido = AtualizarProduto(p);
    } else {
      const validas = unidadesNovas.filter((u) => u.nomeUnidade.trim());
      if (validas.length === 0) {
        onErro("Produto precisa de pelo menos uma unidade (variação/grupo) de venda");
        return;
      }
      const req = new api.CriarProdutoRequest();
      req.Produto = p;
      req.Unidades = validas.map((u) => paraUnidadeProduto(u));
      pedido = CriarProduto(req);
    }

    setAGuardar(true);
    pedido
      .then(() => onGuardado())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  function adicionarUnidadeExistente() {
    if (!produto || !novaUnidade.nomeUnidade.trim() || aAdicionarUnidade) return;
    setAAdicionarUnidade(true);
    AdicionarUnidadeProduto(paraUnidadeProduto(novaUnidade, produto.ID))
      .then((criada) => {
        setUnidadesExistentes((prev) => [...prev, criada]);
        setNovaUnidade(unidadeVazia(false));
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setAAdicionarUnidade(false));
  }

  return (
    <div className="fase1-produto-form">
      <div className="fase1-form">
        <input placeholder="Nome do produto" value={nome} onChange={(e) => setNome(e.target.value)} />
        <select value={categoriaId} onChange={(e) => setCategoriaId(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Sem categoria</option>
          {categorias.map((c) => (
            <option key={c.ID} value={c.ID}>
              {c.Nome}
            </option>
          ))}
        </select>
        <select
          value={tipoProduto}
          onChange={(e) => setTipoProduto(e.target.value as typeof tipoProduto)}
          disabled={!!produto}
          title={produto ? "Tipo não pode ser alterado depois de criado" : undefined}
        >
          <option value="simples">{ROTULO_TIPO_PRODUTO.simples}</option>
          <option value="composto">{ROTULO_TIPO_PRODUTO.composto}</option>
          <option value="semi_elaborado">{ROTULO_TIPO_PRODUTO.semi_elaborado}</option>
        </select>
        <input
          placeholder="Unidade base (ex: unidade, kg)"
          value={unidadeBase}
          onChange={(e) => setUnidadeBase(e.target.value)}
          disabled={!!produto}
          title={produto ? "Unidade base não pode ser alterada depois de criado" : undefined}
        />
        <input
          placeholder="Stock mínimo (opcional)"
          type="number"
          value={stockMinimo}
          onChange={(e) => setStockMinimo(e.target.value)}
        />
        <label>
          <input type="checkbox" checked={loteControlado} onChange={(e) => setLoteControlado(e.target.checked)} /> lote
          controlado (farmácia)
        </label>
      </div>

      {!produto && (
        <div className="fase1-unidades-bloco">
          <div className="fase1-unidades-titulo">Variações / grupos (unidades de venda e compra)</div>
          {unidadesNovas.map((u, i) => (
            <div className="fase1-form" key={i}>
              <input
                placeholder="Nome (ex: unidade, caixa de 24)"
                value={u.nomeUnidade}
                onChange={(e) => atualizarUnidadeNova(i, "nomeUnidade", e.target.value)}
              />
              <input
                placeholder="Qtd em unidade base"
                type="number"
                value={u.qtdEmUnidadeBase}
                onChange={(e) => atualizarUnidadeNova(i, "qtdEmUnidadeBase", e.target.value)}
              />
              <input
                placeholder="Preço custo"
                type="number"
                value={u.precoCusto}
                onChange={(e) => atualizarUnidadeNova(i, "precoCusto", e.target.value)}
              />
              <input
                placeholder="Preço venda"
                type="number"
                value={u.precoVenda}
                onChange={(e) => atualizarUnidadeNova(i, "precoVenda", e.target.value)}
              />
              <input
                placeholder="Código de barras (opcional)"
                value={u.codigoBarras}
                onChange={(e) => atualizarUnidadeNova(i, "codigoBarras", e.target.value)}
              />
              <label>
                <input
                  type="checkbox"
                  checked={u.isPadraoVenda}
                  onChange={(e) => atualizarUnidadeNova(i, "isPadraoVenda", e.target.checked)}
                />{" "}
                padrão venda
              </label>
              <label>
                <input
                  type="checkbox"
                  checked={u.isPadraoCompra}
                  onChange={(e) => atualizarUnidadeNova(i, "isPadraoCompra", e.target.checked)}
                />{" "}
                padrão compra
              </label>
              {unidadesNovas.length > 1 && (
                <button onClick={() => setUnidadesNovas((prev) => prev.filter((_, idx) => idx !== i))}>Remover</button>
              )}
            </div>
          ))}
          <button onClick={() => setUnidadesNovas((prev) => [...prev, unidadeVazia(false)])}>
            + Adicionar variação/grupo
          </button>
        </div>
      )}

      {produto && (
        <div className="fase1-unidades-bloco">
          <div className="fase1-unidades-titulo">Variações / grupos (unidades de venda e compra)</div>
          <table>
            <thead>
              <tr>
                <th>Nome</th>
                <th>Qtd</th>
                <th>Preço custo</th>
                <th>Preço venda</th>
                <th>Código de barras</th>
              </tr>
            </thead>
            <tbody>
              {unidadesExistentes.map((u) => (
                <tr key={u.ID}>
                  <td>{u.NomeUnidade}</td>
                  <td>{u.QtdEmUnidadeBase}</td>
                  <td>{u.PrecoCusto}</td>
                  <td>{u.PrecoVenda}</td>
                  <td>{u.CodigoBarras ?? "—"}</td>
                </tr>
              ))}
              {unidadesExistentes.length === 0 && (
                <tr>
                  <td colSpan={5}>A carregar…</td>
                </tr>
              )}
            </tbody>
          </table>
          <div className="fase1-form">
            <input
              placeholder="Nome (ex: caixa de 24)"
              value={novaUnidade.nomeUnidade}
              onChange={(e) => setNovaUnidade({ ...novaUnidade, nomeUnidade: e.target.value })}
            />
            <input
              placeholder="Qtd em unidade base"
              type="number"
              value={novaUnidade.qtdEmUnidadeBase}
              onChange={(e) => setNovaUnidade({ ...novaUnidade, qtdEmUnidadeBase: e.target.value })}
            />
            <input
              placeholder="Preço custo"
              type="number"
              value={novaUnidade.precoCusto}
              onChange={(e) => setNovaUnidade({ ...novaUnidade, precoCusto: e.target.value })}
            />
            <input
              placeholder="Preço venda"
              type="number"
              value={novaUnidade.precoVenda}
              onChange={(e) => setNovaUnidade({ ...novaUnidade, precoVenda: e.target.value })}
            />
            <input
              placeholder="Código de barras (opcional)"
              value={novaUnidade.codigoBarras}
              onChange={(e) => setNovaUnidade({ ...novaUnidade, codigoBarras: e.target.value })}
            />
            <button disabled={!novaUnidade.nomeUnidade.trim() || aAdicionarUnidade} onClick={adicionarUnidadeExistente}>
              {aAdicionarUnidade ? "A adicionar…" : "+ Adicionar unidade"}
            </button>
          </div>
        </div>
      )}

      <div className="fase1-form">
        <button disabled={!nome.trim() || aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : produto ? "Guardar alterações" : "+ Criar produto"}
        </button>
        {produto && <button onClick={onCancelar}>Cancelar</button>}
      </div>
    </div>
  );
}

// --- Stock ---------------------------------------------------------------

function StockSection({
  produtos,
  armazens,
  onErro,
}: {
  produtos: domain.Produto[];
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const [produtoID, setProdutoID] = useState<number | "">("");
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [stock, setStock] = useState<domain.StockItem | null>(null);
  const [quantidade, setQuantidade] = useState("0");
  const [custoUnitario, setCustoUnitario] = useState("0");
  const [armazemDestinoID, setArmazemDestinoID] = useState<number | "">("");
  const [motivo, setMotivo] = useState("");

  const consultar = () => {
    if (produtoID === "" || armazemID === "") return;
    ObterStock(Number(produtoID), Number(armazemID)).then(setStock).catch((e) => onErro(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro (setErro) is a stable setter
  useEffect(consultar, [produtoID, armazemID]);

  function entrada() {
    if (produtoID === "" || armazemID === "") return;
    const p = new usecase.EntradaParams();
    p.ProdutoID = Number(produtoID);
    p.ArmazemDestinoID = Number(armazemID);
    p.Quantidade = Number(quantidade);
    p.CustoUnitario = Number(custoUnitario);
    run(() => RegistarEntradaStock(p), onErro, consultar);
  }

  function transferir() {
    if (produtoID === "" || armazemID === "" || armazemDestinoID === "") return;
    const p = new usecase.TransferenciaParams();
    p.ProdutoID = Number(produtoID);
    p.ArmazemOrigemID = Number(armazemID);
    p.ArmazemDestinoID = Number(armazemDestinoID);
    p.Quantidade = Number(quantidade);
    run(() => TransferirStock(p), onErro, consultar);
  }

  function perda() {
    if (produtoID === "" || armazemID === "") return;
    const p = new usecase.SaidaParams();
    p.ProdutoID = Number(produtoID);
    p.ArmazemOrigemID = Number(armazemID);
    p.Quantidade = Number(quantidade);
    p.Motivo = motivo;
    run(() => RegistarPerdaStock(p), onErro, consultar);
  }

  return (
    <section>
      <h2>Stock</h2>
      <div className="fase1-form">
        <select value={produtoID} onChange={(e) => setProdutoID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Produto…</option>
          {produtos.map((p) => (
            <option key={p.ID} value={p.ID}>
              {p.Nome}
            </option>
          ))}
        </select>
        <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Armazém…</option>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
      </div>

      {stock && (
        <p className="fase1-stock-atual">
          Stock atual: <strong>{stock.Quantidade}</strong> · custo médio ponderado: <strong>{stock.CustoMedioPonderado}</strong>
        </p>
      )}

      <div className="fase1-form">
        <input placeholder="Quantidade" type="number" value={quantidade} onChange={(e) => setQuantidade(e.target.value)} />
        <input
          placeholder="Custo unitário (entrada)"
          type="number"
          value={custoUnitario}
          onChange={(e) => setCustoUnitario(e.target.value)}
        />
        <button onClick={entrada}>Registar entrada</button>
      </div>

      <div className="fase1-form">
        <select value={armazemDestinoID} onChange={(e) => setArmazemDestinoID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Armazém destino…</option>
          {armazens
            .filter((a) => a.ID !== armazemID)
            .map((a) => (
              <option key={a.ID} value={a.ID}>
                {a.Nome}
              </option>
            ))}
        </select>
        <button onClick={transferir}>Transferir</button>
      </div>

      <div className="fase1-form">
        <input placeholder="Motivo da perda" value={motivo} onChange={(e) => setMotivo(e.target.value)} />
        <button onClick={perda}>Registar perda</button>
      </div>
    </section>
  );
}

// --- Inventário ------------------------------------------------------------

function InventarioSection({
  produtos,
  armazens,
  onErro,
}: {
  produtos: domain.Produto[];
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [inventarioID, setInventarioID] = useState<number | null>(null);
  const [itens, setItens] = useState<domain.InventarioItem[]>([]);
  const [ajustes, setAjustes] = useState<domain.MovimentoStock[] | null>(null);

  const nomeProduto = (id: number) => produtos.find((p) => p.ID === id)?.Nome ?? `#${id}`;

  function iniciar() {
    if (armazemID === "") return;
    setAjustes(null);
    IniciarInventario(Number(armazemID))
      .then((res) => {
        setInventarioID(res.Inventario.ID);
        setItens(res.Itens);
      })
      .catch((e) => onErro(String(e)));
  }

  function contar(itemID: number, valor: string) {
    RegistarContagemInventario(itemID, Number(valor))
      .then((atualizado) => setItens((prev) => prev.map((i) => (i.ID === itemID ? atualizado : i))))
      .catch((e) => onErro(String(e)));
  }

  function fechar() {
    if (inventarioID === null) return;
    FecharInventario(inventarioID)
      .then((movs) => {
        setAjustes(movs);
        setInventarioID(null);
        setItens([]);
      })
      .catch((e) => onErro(String(e)));
  }

  return (
    <section>
      <h2>Inventário</h2>
      <div className="fase1-form">
        <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Armazém…</option>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
        <button disabled={inventarioID !== null} onClick={iniciar}>
          Iniciar inventário
        </button>
      </div>

      {itens.length > 0 && (
        <>
          <table>
            <thead>
              <tr>
                <th>Produto</th>
                <th>Qtd. sistema</th>
                <th>Qtd. contada</th>
                <th>Diferença</th>
              </tr>
            </thead>
            <tbody>
              {itens.map((item) => (
                <tr key={item.ID}>
                  <td>{nomeProduto(item.ProdutoID)}</td>
                  <td>{item.QuantidadeSistema}</td>
                  <td>
                    <input
                      type="number"
                      defaultValue={item.QuantidadeContada ?? item.QuantidadeSistema}
                      onBlur={(e) => contar(item.ID, e.target.value)}
                    />
                  </td>
                  <td>{item.Diferenca ?? "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <button onClick={fechar}>Fechar inventário e gerar ajustes</button>
        </>
      )}

      {ajustes && (
        <div>
          <h3>Ajustes gerados ({ajustes.length})</h3>
          <ul>
            {ajustes.map((m) => (
              <li key={m.ID}>
                {nomeProduto(m.ProdutoID)}: {m.ArmazemDestinoID ? "+" : "-"}
                {m.QuantidadeUnidadeBase}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}
