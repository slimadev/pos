import { useEffect, useState } from "react";
import {
  ListarCategorias,
  CriarCategoria,
  AtualizarCategoria,
  AtivarCategoria,
  DesativarCategoria,
} from "../wailsjs/go/admin/AdminApp";
import { domain } from "../wailsjs/go/models";

/**
 * Registo de categorias da única loja que este cmd/admin serve — dado de
 * negócio, continua a viver dentro da base da loja (SDD 8.5), este ecrã só
 * escolhe a ligação a abrir (feito no backend, ver admin.CategoriasService).
 */
export default function CategoriasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [emEdicao, setEmEdicao] = useState<domain.Categoria | null>(null);

  function carregar() {
    ListarCategorias()
      .then(setCategorias)
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(carregar, []);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;

  function alternarAtivo(c: domain.Categoria) {
    const acao = c.Ativo ? DesativarCategoria(c.ID) : AtivarCategoria(c.ID);
    acao.then(carregar).catch((e) => onErro(String(e)));
  }

  return (
    <div className="admin-lojas-screen">
      <h2 style={{ margin: 0 }}>Categorias</h2>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Nome</th>
            <th>Categoria-mãe</th>
            <th>Ativa</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {categorias.map((c) => (
            <tr key={c.ID}>
              <td>{c.Nome}</td>
              <td>{c.CategoriaPaiID != null ? nomeCategoria(c.CategoriaPaiID) : "—"}</td>
              <td>{c.Ativo ? "sim" : "não"}</td>
              <td className="admin-form">
                <button className="admin-btn-secundario" onClick={() => setEmEdicao(c)}>
                  Editar
                </button>
                <button className="admin-btn-secundario" onClick={() => alternarAtivo(c)}>
                  {c.Ativo ? "Desativar" : "Ativar"}
                </button>
              </td>
            </tr>
          ))}
          {categorias.length === 0 && (
            <tr>
              <td colSpan={4}>{aCarregar ? "A carregar…" : "Sem categorias — crie uma abaixo."}</td>
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
          carregar();
        }}
        onCancelar={() => setEmEdicao(null)}
        onErro={onErro}
      />
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
    if (!nome.trim()) return;
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
    <div className="admin-form">
      <input placeholder="Nome da categoria" value={nome} onChange={(e) => setNome(e.target.value)} />
      <select
        value={categoriaPaiId}
        onChange={(e) => setCategoriaPaiId(e.target.value ? Number(e.target.value) : "")}
      >
        <option value="">Sem categoria-mãe</option>
        {categorias
          .filter((c) => c.ID !== categoria?.ID)
          .map((c) => (
            <option key={c.ID} value={c.ID}>
              {c.Nome}
            </option>
          ))}
      </select>
      <button className="admin-btn" disabled={!nome.trim() || aGuardar} onClick={guardar}>
        {aGuardar ? "A guardar…" : categoria ? "Guardar alterações" : "+ Criar categoria"}
      </button>
      {categoria && (
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      )}
    </div>
  );
}
