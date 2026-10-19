import { forwardRef, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import { CatalogoPDV, ListarCategorias } from "../../wailsjs/go/api/App";
import { domain, usecase } from "../../wailsjs/go/models";
import { mt, normalizar, qtd } from "./formato";
import "./Catalogo.css";

type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;

export type CatalogoHandle = {
  focarPesquisa: () => void;
  recarregar: () => void;
};

const SEM_CATEGORIA = -1;

/**
 * Catálogo partilhado pelos ecrãs de venda: pesquisa sem acentos por nome
 * ou código de barras (Enter junta o produto — um leitor de código de
 * barras "escreve" o código e carrega Enter), F2 volta à pesquisa, Esc
 * limpa, separadores por categoria e grelha de cartões de tamanho fixo,
 * com os mais vendidos primeiro. Tudo vem de um só pedido (CatalogoPDV).
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
    // eslint-disable-next-line react-hooks/exhaustive-deps -- recarrega quando o armazém muda ou depois de uma venda
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
  // Só os separadores que têm produtos; "Outros" para os sem categoria.
  const separadores = useMemo(() => {
    const usadas = new Set(lista.map((i) => i.Produto.CategoriaID ?? SEM_CATEGORIA));
    const comProdutos = categorias
      .filter((c) => usadas.has(c.ID))
      .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"))
      .map((c) => ({ id: c.ID, nome: c.Nome }));
    if (usadas.has(SEM_CATEGORIA) && comProdutos.length > 0) comProdutos.push({ id: SEM_CATEGORIA, nome: "Outros" });
    return comProdutos;
  }, [lista, categorias]);

  const termo = normalizar(busca);
  const visiveis = lista.filter((i) => {
    if (termo) {
      // A pesquisa procura em todas as categorias — quem escreve sabe o que quer.
      return (
        normalizar(i.Produto.Nome).includes(termo) || i.Unidades.some((u) => u.CodigoBarras && u.CodigoBarras === busca.trim())
      );
    }
    return categoria === null || (i.Produto.CategoriaID ?? SEM_CATEGORIA) === categoria;
  });

  const unidadePadrao = (i: usecase.ItemCatalogo) => i.Unidades.find((u) => u.ID === i.UnidadePadraoID) ?? i.Unidades[0];
  const esgotado = (i: usecase.ItemCatalogo) => i.Disponivel <= 0 && (i.Produto.ControlaStock || i.Produto.TipoProduto === "composto");

  function escolher(i: usecase.ItemCatalogo, u?: domain.UnidadeProduto) {
    if (esgotado(i)) {
      setAviso(`${i.Produto.Nome} não tem stock neste armazém.`);
      return;
    }
    setAviso(null);
    onEscolher(i, u ?? unidadePadrao(i));
  }

  function enter() {
    const texto = busca.trim();
    if (!texto) return;
    // Código de barras exato: junta a unidade desse código (ex: a caixa).
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
    setAviso(visiveis.length === 0 ? `Nada encontrado para "${texto}".` : "Vários produtos — toque no que quer.");
  }

  return (
    <div className="cat">
      <div className="cat-toolbar">
        <div className="cat-busca-wrap">
          <IconeBusca className="cat-busca-icon" aria-hidden="true" />
          <input
            ref={buscaRef}
            className="cat-busca"
            placeholder="Nome ou código de barras — Enter junta (F2)"
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
              ×
            </button>
          )}
        </div>
        {extraToolbar}
      </div>

      {aviso && <div className="cat-aviso">{aviso}</div>}

      {separadores.length > 0 && !termo && (
        <div className="cat-separadores" role="tablist">
          <button
            role="tab"
            aria-selected={categoria === null}
            className={`cat-separador${categoria === null ? " ativo" : ""}`}
            onClick={() => setCategoria(null)}
          >
            Todos
          </button>
          {separadores.map((c) => (
            <button
              key={c.id}
              role="tab"
              aria-selected={categoria === c.id}
              className={`cat-separador${categoria === c.id ? " ativo" : ""}`}
              onClick={() => setCategoria(c.id)}
            >
              {c.nome}
            </button>
          ))}
        </div>
      )}

      <div className="cat-grelha">
        {itens === null && <div className="cat-vazio">A carregar produtos…</div>}
        {itens !== null && visiveis.length === 0 && (
          <div className="cat-vazio">{termo ? `Nenhum produto com "${busca.trim()}".` : "Sem produtos nesta categoria."}</div>
        )}
        {visiveis.map((i) => {
          const u = unidadePadrao(i);
          const sem = esgotado(i);
          return (
            <button key={i.Produto.ID} className={`cat-cartao${sem ? " esgotado" : ""}`} onClick={() => escolher(i)}>
              <span className="cat-cartao-nome">{i.Produto.Nome}</span>
              <span className="cat-cartao-rodape">
                <span className="cat-cartao-preco">{mt(u.PrecoVenda)}</span>
                <span className="cat-cartao-unidade">{u.NomeUnidade}</span>
              </span>
              <span className={`cat-cartao-stock${sem ? " zero" : ""}`}>
                {sem
                  ? "sem stock"
                  : i.Produto.TipoProduto === "composto"
                    ? `dá para ${qtd(i.Disponivel)}`
                    : i.Produto.ControlaStock
                      ? `${qtd(i.Disponivel)} ${i.Produto.UnidadeBase}`
                      : ""}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
});

export default Catalogo;
