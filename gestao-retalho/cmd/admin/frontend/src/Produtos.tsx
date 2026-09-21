import { useEffect, useState } from "react";
import {
  ListarProdutos,
  ListarCategorias,
  ListarUnidadesProduto,
  CriarProduto,
  AtualizarProduto,
  AtivarProduto,
  DesativarProduto,
  DuplicarProduto,
  AdicionarUnidadeProduto,
} from "../wailsjs/go/admin/AdminApp";
import { api, domain } from "../wailsjs/go/models";
import Modal from "./Modal";

type TipoProduto = "simples" | "composto" | "semi_elaborado";

type ModalEstado = { aberto: false } | { aberto: true; produto: domain.Produto | null };

/**
 * CRUD de produtos da única loja que este cmd/admin serve — inclui as
 * "variações ou grupos" do mesmo produto (SDD 4.1: `UnidadeProduto`, ex:
 * unidade/caixa, garrafa/dose): um produto só é vendável com pelo menos
 * uma, e pode ter várias em simultâneo, geridas aqui. Criar/editar abre
 * num modal (pedido explícito) em vez de um formulário sempre visível
 * por baixo da tabela.
 */
export default function ProdutosScreen({ onErro }: { onErro: (e: string) => void }) {
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [modal, setModal] = useState<ModalEstado>({ aberto: false });

  function carregar() {
    Promise.all([ListarProdutos(), ListarCategorias()])
      .then(([p, c]) => {
        setProdutos(p);
        setCategorias(c);
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(carregar, []);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;

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

  return (
    <div className="admin-lojas-screen">
      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Produtos</h2>
        <button className="admin-btn" onClick={() => setModal({ aberto: true, produto: null })}>
          + Criar produto
        </button>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Nome</th>
            <th>Categoria</th>
            <th>Tipo</th>
            <th>Unidade base</th>
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
              <td>{p.Ativo ? "sim" : "não"}</td>
              <td className="admin-form">
                <button className="admin-btn-secundario" onClick={() => setModal({ aberto: true, produto: p })}>
                  Editar
                </button>
                <button className="admin-btn-secundario" onClick={() => duplicar(p)}>
                  Duplicar
                </button>
                <button className="admin-btn-secundario" onClick={() => alternarAtivo(p)}>
                  {p.Ativo ? "Desativar" : "Ativar"}
                </button>
              </td>
            </tr>
          ))}
          {produtos.length === 0 && (
            <tr>
              <td colSpan={6}>{aCarregar ? "A carregar…" : 'Sem produtos — clique em "+ Criar produto".'}</td>
            </tr>
          )}
        </tbody>
      </table>

      {modal.aberto && (
        <Modal titulo={modal.produto ? "Editar produto" : "Criar produto"} onFechar={fecharModal}>
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

    setAGuardar(true);
    if (produto) {
      AtualizarProduto(p)
        .then(onGuardado)
        .catch((e) => onErro(String(e)))
        .finally(() => setAGuardar(false));
      return;
    }

    const validas = unidadesNovas.filter((u) => u.nomeUnidade.trim());
    if (validas.length === 0) {
      onErro("Produto precisa de pelo menos uma unidade (variação/grupo) de venda");
      setAGuardar(false);
      return;
    }
    const req = new api.CriarProdutoRequest();
    req.Produto = p;
    req.Unidades = validas.map((u) => paraUnidadeProduto(u));
    CriarProduto(req)
      .then(onGuardado)
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
    <div className="admin-produto-form">
      <div className="admin-form">
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
          onChange={(e) => setTipoProduto(e.target.value as TipoProduto)}
          disabled={!!produto}
          title={produto ? "Tipo não pode ser alterado depois de criado" : undefined}
        >
          <option value="simples">simples</option>
          <option value="composto">composto</option>
          <option value="semi_elaborado">semi_elaborado</option>
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
          <input type="checkbox" checked={loteControlado} onChange={(e) => setLoteControlado(e.target.checked)} />{" "}
          lote controlado (farmácia)
        </label>
      </div>

      {!produto && (
        <div className="admin-unidades-bloco">
          <div className="admin-unidades-titulo">Variações / grupos (unidades de venda e compra)</div>
          {unidadesNovas.map((u, i) => (
            <div className="admin-form admin-unidade-linha" key={i}>
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
                <button
                  className="admin-btn-secundario"
                  onClick={() => setUnidadesNovas((prev) => prev.filter((_, idx) => idx !== i))}
                >
                  Remover
                </button>
              )}
            </div>
          ))}
          <button
            className="admin-btn-secundario"
            onClick={() => setUnidadesNovas((prev) => [...prev, unidadeVazia(false)])}
          >
            + Adicionar variação/grupo
          </button>
        </div>
      )}

      {produto && (
        <div className="admin-unidades-bloco">
          <div className="admin-unidades-titulo">Variações / grupos (unidades de venda e compra)</div>
          <table className="admin-tabela">
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
          <div className="admin-form admin-unidade-linha">
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
            <button
              className="admin-btn-secundario"
              disabled={!novaUnidade.nomeUnidade.trim() || aAdicionarUnidade}
              onClick={adicionarUnidadeExistente}
            >
              {aAdicionarUnidade ? "A adicionar…" : "+ Adicionar unidade"}
            </button>
          </div>
        </div>
      )}

      <div className="admin-form">
        <button className="admin-btn" disabled={!nome.trim() || aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : produto ? "Guardar alterações" : "+ Criar produto"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
