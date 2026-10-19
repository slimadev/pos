import { useEffect, useState } from "react";
import {
  ListarCategorias,
  ListarProdutos,
  CriarCategoria,
  AtualizarCategoria,
  AtivarCategoria,
  DesativarCategoria,
} from "../wailsjs/go/admin/AdminApp";
import { domain } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";

type ModalEstado = { aberto: false } | { aberto: true; categoria: domain.Categoria | null };

/**
 * Registo de categorias da única loja que este cmd/admin serve — dado de
 * negócio, continua a viver dentro da base da loja (SDD 8.5), este ecrã só
 * escolhe a ligação a abrir (feito no backend, ver admin.CategoriasService).
 * Mostra quantos produtos ativos há em cada categoria, para se ver as que
 * estão vazias antes de as desativar.
 */
export default function CategoriasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [modal, setModal] = useState<ModalEstado>({ aberto: false });

  function carregar() {
    Promise.all([ListarCategorias(), ListarProdutos()])
      .then(([c, p]) => {
        setCategorias(c);
        setProdutos(p);
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(carregar, []);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;
  const nProdutos = (id: number) => produtos.filter((p) => p.Ativo && p.CategoriaID === id).length;
  const ordenadas = [...categorias].sort(
    (a, b) => Number(b.Ativo) - Number(a.Ativo) || a.Nome.localeCompare(b.Nome, "pt"),
  );

  function alternarAtivo(c: domain.Categoria) {
    const acao = c.Ativo ? DesativarCategoria(c.ID) : AtivarCategoria(c.ID);
    acao.then(carregar).catch((e) => onErro(String(e)));
  }

  const fechar = () => setModal({ aberto: false });

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Categorias" }]} />
      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Categorias</h2>
        <button className="admin-btn" onClick={() => setModal({ aberto: true, categoria: null })}>
          + Criar categoria
        </button>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Nome</th>
            <th>Categoria-mãe</th>
            <th style={{ textAlign: "right" }}>Produtos</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {ordenadas.map((c) => (
            <tr key={c.ID} className={c.Ativo ? undefined : "admin-linha-inativa"}>
              <td>
                {c.Nome}
                {!c.Ativo && <span className="admin-etiqueta">inativa</span>}
              </td>
              <td>{c.CategoriaPaiID != null ? nomeCategoria(c.CategoriaPaiID) : "—"}</td>
              <td style={{ textAlign: "right" }}>{nProdutos(c.ID)}</td>
              <td>
                <div className="admin-acoes">
                  <button className="admin-btn-secundario" onClick={() => setModal({ aberto: true, categoria: c })}>
                    Editar
                  </button>
                  <button className="admin-btn-secundario" onClick={() => alternarAtivo(c)}>
                    {c.Ativo ? "Desativar" : "Ativar"}
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {categorias.length === 0 && (
            <tr>
              <td colSpan={4}>{aCarregar ? "A carregar…" : 'Sem categorias — clique em "+ Criar categoria".'}</td>
            </tr>
          )}
        </tbody>
      </table>

      {modal.aberto && (
        <Modal titulo={modal.categoria ? `Editar ${modal.categoria.Nome}` : "Criar categoria"} onFechar={fechar}>
          <CategoriaForm
            categoria={modal.categoria}
            categorias={categorias}
            onGuardado={() => {
              fechar();
              carregar();
            }}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}
    </div>
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
  const [aGuardar, setAGuardar] = useState(false);

  function guardar() {
    if (!nome.trim() || aGuardar) return;
    const c = new domain.Categoria();
    c.ID = categoria?.ID ?? 0;
    c.Nome = nome.trim();
    c.CategoriaPaiID = categoriaPaiId === "" ? undefined : Number(categoriaPaiId);

    setAGuardar(true);
    const pedido = categoria ? AtualizarCategoria(c) : CriarCategoria(c);
    pedido
      .then(onGuardado)
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-produto-form">
      <label className="admin-campo">
        Nome
        <input
          autoFocus
          value={nome}
          onChange={(e) => setNome(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && guardar()}
          placeholder="ex: Ferramentas"
        />
      </label>
      <label className="admin-campo">
        Dentro de (opcional)
        <select value={categoriaPaiId} onChange={(e) => setCategoriaPaiId(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Nenhuma — categoria principal</option>
          {categorias
            .filter((c) => c.ID !== categoria?.ID && c.Ativo)
            .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"))
            .map((c) => (
              <option key={c.ID} value={c.ID}>
                {c.Nome}
              </option>
            ))}
        </select>
      </label>
      <div className="admin-form">
        <button className="admin-btn" disabled={!nome.trim() || aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : categoria ? "Guardar alterações" : "+ Criar categoria"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
