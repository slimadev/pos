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
  CriarProduto,
  AtivarProduto,
  DesativarProduto,
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
          <ProdutosSection produtos={produtos} categorias={categorias} onChange={recarregar} onErro={setErro} />
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

function ProdutosSection({
  produtos,
  categorias,
  onChange,
  onErro,
}: {
  produtos: domain.Produto[];
  categorias: domain.Categoria[];
  onChange: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState("");
  const [categoriaId, setCategoriaId] = useState<number | "">("");
  const [tipoProduto, setTipoProduto] = useState<"simples" | "composto" | "semi_elaborado">("simples");
  const [unidadeBase, setUnidadeBase] = useState("unidade");
  const [precoCusto, setPrecoCusto] = useState("0");
  const [precoVenda, setPrecoVenda] = useState("0");
  const [loteControlado, setLoteControlado] = useState(false);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;

  function criar() {
    const produto = new domain.Produto();
    produto.Nome = nome;
    produto.CategoriaID = categoriaId === "" ? undefined : Number(categoriaId);
    produto.TipoProduto = tipoProduto;
    produto.UnidadeBase = unidadeBase;
    produto.LoteControlado = loteControlado;

    const unidade = new domain.UnidadeProduto();
    unidade.NomeUnidade = unidadeBase;
    unidade.QtdEmUnidadeBase = 1;
    unidade.PrecoCusto = Number(precoCusto);
    unidade.PrecoVenda = Number(precoVenda);
    unidade.IsPadraoVenda = true;
    unidade.IsPadraoCompra = true;

    const req = new api.CriarProdutoRequest();
    req.Produto = produto;
    req.Unidades = [unidade];

    run(
      () => CriarProduto(req),
      onErro,
      () => {
        setNome("");
        onChange();
      },
    );
  }

  function alternarAtivo(p: domain.Produto) {
    run(() => (p.Ativo ? DesativarProduto(p.ID) : AtivarProduto(p.ID)), onErro, onChange);
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
              <td>{p.TipoProduto}</td>
              <td>{p.UnidadeBase}</td>
              <td>{p.LoteControlado ? "sim" : "não"}</td>
              <td>{p.Ativo ? "sim" : "não"}</td>
              <td>
                <button onClick={() => alternarAtivo(p)}>{p.Ativo ? "Desativar" : "Ativar"}</button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
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
        <select value={tipoProduto} onChange={(e) => setTipoProduto(e.target.value as typeof tipoProduto)}>
          <option value="simples">simples</option>
          <option value="composto">composto</option>
          <option value="semi_elaborado">semi_elaborado</option>
        </select>
        <input
          placeholder="Unidade base (ex: unidade, kg)"
          value={unidadeBase}
          onChange={(e) => setUnidadeBase(e.target.value)}
        />
        <input
          placeholder="Preço custo"
          type="number"
          value={precoCusto}
          onChange={(e) => setPrecoCusto(e.target.value)}
        />
        <input
          placeholder="Preço venda"
          type="number"
          value={precoVenda}
          onChange={(e) => setPrecoVenda(e.target.value)}
        />
        <label>
          <input type="checkbox" checked={loteControlado} onChange={(e) => setLoteControlado(e.target.checked)} /> lote
          controlado (farmácia)
        </label>
        <button disabled={!nome} onClick={criar}>
          + Criar produto
        </button>
      </div>
    </section>
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
