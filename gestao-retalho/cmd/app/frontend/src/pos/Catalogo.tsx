import { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import {
  XMarkIcon,
  Squares2X2Icon,
  ListBulletIcon,
  PlusIcon,
  SparklesIcon,
} from "@heroicons/react/24/outline";
import { CatalogoPDV, ListarCategorias } from "../../wailsjs/go/api/App";
import { domain, usecase } from "../../wailsjs/go/models";
import { mt, normalizar, qtd } from "./formato";
import { obterVisualProduto, gerarSvgFallback } from "./productVisuals";
import "./Catalogo.css";

type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;

export type CatalogoHandle = {
  focarPesquisa: () => void;
  recarregar: () => void;
};

const SEM_CATEGORIA = -1;

/**
 * Catálogo moderno estilo Zobaze POS:
 * - Cartões visuais com fotografias reais e vibrantes dos produtos
 * - Alternador de vista entre Grelha Visual e Lista Compacta
 * - Badges táteis de stock (Em stock, Stock baixo, Esgotado)
 * - Separadores de categoria com contadores e rolagem horizontal suave
 * - Ação rápida de toque para adicionar ao carrinho com animação de feedback
 */
const Catalogo = forwardRef<
  CatalogoHandle,
  {
    armazemID: number;
    IconeBusca: IconeSvg;
    onEscolher: (item: usecase.ItemCatalogo, unidade: domain.UnidadeProduto) => void;
    onErro: (e: string) => void;
    extraToolbar?: React.ReactNode;
  }
>(function Catalogo({ armazemID, IconeBusca, onEscolher, onErro, extraToolbar }, ref) {
  const [itens, setItens] = useState<usecase.ItemCatalogo[] | null>(null);
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [categoria, setCategoria] = useState<number | null>(null);
  const [busca, setBusca] = useState("");
  const [aviso, setAviso] = useState<string | null>(null);
  const [versao, setVersao] = useState(0);
  const [modoVisualizacao, setModoVisualizacao] = useState<"cards" | "lista">("cards");
  const buscaRef = useRef<HTMLInputElement>(null);

  useImperativeHandle(ref, () => ({
    focarPesquisa: () => buscaRef.current?.focus(),
    recarregar: () => setVersao((v) => v + 1),
  }));

  useEffect(() => {
    CatalogoPDV(armazemID)
      .then(setItens)
      .catch((e) => {
        setItens([]);
        onErro(String(e));
      });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [armazemID, versao]);

  useEffect(() => {
    ListarCategorias()
      .then(setCategorias)
      .catch(() => {});
    buscaRef.current?.focus();
  }, []);

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (e.key === "F2") {
        e.preventDefault();
        buscaRef.current?.focus();
        buscaRef.current?.select();
      }
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, []);

  const lista = useMemo(() => itens ?? [], [itens]);

  // Separadores com contador de itens
  const separadores = useMemo(() => {
    const contagemPorCat = new Map<number, number>();
    for (const i of lista) {
      const catId = i.Produto.CategoriaID ?? SEM_CATEGORIA;
      contagemPorCat.set(catId, (contagemPorCat.get(catId) || 0) + 1);
    }

    const comProdutos = categorias
      .filter((c) => contagemPorCat.has(c.ID))
      .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"))
      .map((c) => ({ id: c.ID, nome: c.Nome, qtd: contagemPorCat.get(c.ID) || 0 }));

    if (contagemPorCat.has(SEM_CATEGORIA) && comProdutos.length > 0) {
      comProdutos.push({ id: SEM_CATEGORIA, nome: "Outros", qtd: contagemPorCat.get(SEM_CATEGORIA) || 0 });
    }
    return comProdutos;
  }, [lista, categorias]);

  const termo = normalizar(busca);
  const visiveis = lista.filter((i) => {
    if (termo) {
      return (
        normalizar(i.Produto.Nome).includes(termo) ||
        i.Unidades.some((u) => u.CodigoBarras && u.CodigoBarras === busca.trim())
      );
    }
    return categoria === null || (i.Produto.CategoriaID ?? SEM_CATEGORIA) === categoria;
  });

  const unidadePadrao = (i: usecase.ItemCatalogo) =>
    i.Unidades.find((u) => u.ID === i.UnidadePadraoID) ?? i.Unidades[0];
  const esgotado = (i: usecase.ItemCatalogo) =>
    i.Disponivel <= 0 && (i.Produto.ControlaStock || i.Produto.TipoProduto === "composto");
  const stockBaixo = (i: usecase.ItemCatalogo) =>
    !esgotado(i) && i.Produto.ControlaStock && i.Disponivel <= (i.Produto.StockMinimo || 5);

  function escolher(i: usecase.ItemCatalogo, u?: domain.UnidadeProduto) {
    if (esgotado(i)) {
      setAviso(`${i.Produto.Nome} está sem stock neste armazém.`);
      return;
    }
    setAviso(null);
    onEscolher(i, u ?? unidadePadrao(i));
  }

  function enter() {
    const texto = busca.trim();
    if (!texto) return;
    for (const i of lista) {
      const u = i.Unidades.find((x) => x.CodigoBarras === texto);
      if (u) {
        escolher(i, u);
        setBusca("");
        return;
      }
    }
    if (visiveis.length === 1) {
      escolher(visiveis[0]);
      setBusca("");
      return;
    }
    setAviso(visiveis.length === 0 ? `Nenhum produto encontrado para "${texto}".` : "Vários produtos — clique no desejado.");
  }

  return (
    <div className="cat">
      <div className="cat-toolbar">
        <div className="cat-busca-wrap">
          <IconeBusca className="cat-busca-icon" aria-hidden="true" />
          <input
            ref={buscaRef}
            className="cat-busca"
            placeholder="Pesquisar por nome ou código de barras... (F2)"
            value={busca}
            onChange={(e) => {
              setBusca(e.target.value);
              setAviso(null);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                enter();
              } else if (e.key === "Escape") {
                setBusca("");
                setAviso(null);
              }
            }}
          />
          {busca && (
            <button className="cat-limpar" onClick={() => setBusca("")} aria-label="Limpar pesquisa">
              <XMarkIcon style={{ width: 16, height: 16 }} aria-hidden="true" />
            </button>
          )}
        </div>

        {extraToolbar}

        {/* Alternador de visualização Zobaze: Cards com foto vs Lista */}
        <div className="cat-view-toggle">
          <button
            type="button"
            className={`cat-view-btn ${modoVisualizacao === "cards" ? "ativo" : ""}`}
            onClick={() => setModoVisualizacao("cards")}
            title="Vista em Grelha com Fotos"
          >
            <Squares2X2Icon style={{ width: 18, height: 18 }} />
          </button>
          <button
            type="button"
            className={`cat-view-btn ${modoVisualizacao === "lista" ? "ativo" : ""}`}
            onClick={() => setModoVisualizacao("lista")}
            title="Vista Compacta em Lista"
          >
            <ListBulletIcon style={{ width: 18, height: 18 }} />
          </button>
        </div>
      </div>

      {aviso && <div className="cat-aviso">{aviso}</div>}

      {/* Barra de categorias com pílulas táteis */}
      {separadores.length > 0 && !termo && (
        <div className="cat-separadores" role="tablist">
          <button
            role="tab"
            aria-selected={categoria === null}
            className={`cat-separador${categoria === null ? " ativo" : ""}`}
            onClick={() => setCategoria(null)}
          >
            <span>Todos</span>
            <span className="cat-separador-qtd">{lista.length}</span>
          </button>
          {separadores.map((c) => (
            <button
              key={c.id}
              role="tab"
              aria-selected={categoria === c.id}
              className={`cat-separador${categoria === c.id ? " ativo" : ""}`}
              onClick={() => setCategoria(c.id)}
            >
              <span>{c.nome}</span>
              <span className="cat-separador-qtd">{c.qtd}</span>
            </button>
          ))}
        </div>
      )}

      {/* Grelha / Lista de produtos estilo Zobaze POS */}
      <div className={`cat-grelha ${modoVisualizacao === "lista" ? "cat-grelha--lista" : "cat-grelha--cards"}`}>
        {itens === null && (
          <div className="cat-vazio">
            <SparklesIcon className="cat-vazio-icon" />
            <span>A carregar catálogo de produtos...</span>
          </div>
        )}
        {itens !== null && visiveis.length === 0 && (
          <div className="cat-vazio">
            <SparklesIcon className="cat-vazio-icon" />
            <span>{termo ? `Nenhum produto encontrado para "${busca.trim()}".` : "Sem produtos disponíveis nesta categoria."}</span>
          </div>
        )}

        {visiveis.map((i) => {
          const u = unidadePadrao(i);
          const sem = esgotado(i);
          const baixo = stockBaixo(i);
          const nomeCat = categorias.find((c) => c.ID === i.Produto.CategoriaID)?.Nome ?? null;
          const visual = obterVisualProduto(i.Produto.Nome, nomeCat);
          const fallback = gerarSvgFallback(i.Produto.Nome, visual.corAcento, u.NomeUnidade);

          if (modoVisualizacao === "lista") {
            // Visualização Compacta em Lista
            return (
              <button
                key={i.Produto.ID}
                className={`cat-linha${sem ? " esgotado" : ""}`}
                onClick={() => escolher(i)}
              >
                <div className="cat-linha-thumb-wrap">
                  <img
                    src={visual.imagemUrl}
                    alt={i.Produto.Nome}
                    className="cat-linha-thumb"
                    loading="lazy"
                    onError={(e) => {
                      e.currentTarget.src = fallback;
                    }}
                  />
                  {sem && <span className="cat-linha-badge-sem">0</span>}
                </div>
                <div className="cat-linha-info">
                  <span className="cat-linha-nome">{i.Produto.Nome}</span>
                  <div className="cat-linha-sub">
                    <span className="cat-linha-unidade">{u.NomeUnidade}</span>
                    <span className={`cat-linha-stock-pill ${sem ? "sem" : baixo ? "baixo" : "ok"}`}>
                      {sem ? "Sem stock" : `${qtd(i.Disponivel)} disp.`}
                    </span>
                  </div>
                </div>
                <div className="cat-linha-preco-wrap">
                  <span className="cat-linha-preco">{mt(u.PrecoVenda)}</span>
                  <span className="cat-linha-add">
                    <PlusIcon style={{ width: 16, height: 16 }} />
                  </span>
                </div>
              </button>
            );
          }

          // Visualização em Cartão com Imagem Grande (Zobaze Tile)
          return (
            <button
              key={i.Produto.ID}
              className={`cat-cartao${sem ? " esgotado" : ""}`}
              onClick={() => escolher(i)}
            >
              <div className="cat-cartao-img-box">
                <img
                  src={visual.imagemUrl}
                  alt={i.Produto.Nome}
                  className="cat-cartao-img"
                  loading="lazy"
                  onError={(e) => {
                    e.currentTarget.src = fallback;
                  }}
                />
                <div className="cat-cartao-overlay-grad" />

                {/* Badges flutuantes sobre a imagem */}
                <div className="cat-cartao-badges">
                  {sem ? (
                    <span className="cat-badge-stock sem">Esgotado</span>
                  ) : baixo ? (
                    <span className="cat-badge-stock baixo">{qtd(i.Disponivel)} restam</span>
                  ) : (
                    <span className="cat-badge-stock ok">{qtd(i.Disponivel)} em stock</span>
                  )}
                  {nomeCat && <span className="cat-badge-cat">{nomeCat}</span>}
                </div>

                {/* Botão flutuante de adição rápida */}
                <div className="cat-cartao-quick-add" title="Adicionar ao carrinho">
                  <PlusIcon style={{ width: 18, height: 18 }} />
                </div>
              </div>

              <div className="cat-cartao-corpo">
                <span className="cat-cartao-nome" title={i.Produto.Nome}>
                  {i.Produto.Nome}
                </span>

                <div className="cat-cartao-rodape">
                  <div className="cat-cartao-precos">
                    <span className="cat-cartao-preco">{mt(u.PrecoVenda)}</span>
                    <span className="cat-cartao-unidade">/ {u.NomeUnidade}</span>
                  </div>
                  {i.Unidades.length > 1 && (
                    <span className="cat-cartao-variacoes">+{i.Unidades.length - 1} un.</span>
                  )}
                </div>
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
});

export default Catalogo;
