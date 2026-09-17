import { useEffect, useState } from "react";
import {
  ListarLojas,
  ListarCategoriasDaLoja,
  CriarCategoriaDaLoja,
  AtualizarCategoriaDaLoja,
  AtivarCategoriaDaLoja,
  DesativarCategoriaDaLoja,
} from "../wailsjs/go/admin/AdminApp";
import { domain, repository } from "../wailsjs/go/models";

/**
 * Registo de categorias a partir do cmd/admin — categorias são dado de
 * negócio (vivem dentro da base de cada loja, SDD 8.5), por isso o ecrã
 * começa por escolher qual loja gerir antes de listar/criar nada; sem
 * lojas registadas não há o que escolher (aponta para "Lojas").
 */
export default function CategoriasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [lojas, setLojas] = useState<repository.Loja[]>([]);
  const [lojaID, setLojaID] = useState<number | "">("");

  useEffect(() => {
    ListarLojas()
      .then((ls) => {
        setLojas(ls);
        if (ls.length > 0) setLojaID(ls[0].ID);
      })
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  if (lojas.length === 0) {
    return (
      <div className="admin-lojas-screen">
        <h2 style={{ margin: 0 }}>Categorias</h2>
        <p style={{ color: "var(--text-secondary)", maxWidth: 520 }}>
          Nenhuma loja registada ainda — registe uma primeiro em "Lojas" para lhe gerir as categorias.
        </p>
      </div>
    );
  }

  return (
    <div className="admin-lojas-screen">
      <h2 style={{ margin: 0 }}>Categorias</h2>

      <div className="admin-form">
        <label>
          Loja:{" "}
          <select value={lojaID} onChange={(e) => setLojaID(e.target.value ? Number(e.target.value) : "")}>
            {lojas.map((l) => (
              <option key={l.ID} value={l.ID}>
                {l.Nome}
              </option>
            ))}
          </select>
        </label>
      </div>

      {/* key={lojaID} força um remonte ao trocar de loja — cada instância
          carrega as suas próprias categorias do zero, em vez de reagir a
          uma mudança de prop dentro de um efeito já montado. */}
      {lojaID !== "" && <CategoriasDaLoja key={lojaID} lojaID={lojaID} onErro={onErro} />}
    </div>
  );
}

function CategoriasDaLoja({ lojaID, onErro }: { lojaID: number; onErro: (e: string) => void }) {
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [emEdicao, setEmEdicao] = useState<domain.Categoria | null>(null);

  function carregar() {
    ListarCategoriasDaLoja(lojaID)
      .then(setCategorias)
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App; lojaID está fixo por instância (key troca-a)
  useEffect(carregar, []);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;

  function alternarAtivo(c: domain.Categoria) {
    const acao = c.Ativo ? DesativarCategoriaDaLoja(lojaID, c.ID) : AtivarCategoriaDaLoja(lojaID, c.ID);
    acao.then(carregar).catch((e) => onErro(String(e)));
  }

  return (
    <>
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
        lojaID={lojaID}
        categoria={emEdicao}
        categorias={categorias}
        onGuardado={() => {
          setEmEdicao(null);
          carregar();
        }}
        onCancelar={() => setEmEdicao(null)}
        onErro={onErro}
      />
    </>
  );
}

function CategoriaForm({
  lojaID,
  categoria,
  categorias,
  onGuardado,
  onCancelar,
  onErro,
}: {
  lojaID: number;
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
    const pedido = categoria ? AtualizarCategoriaDaLoja(lojaID, c) : CriarCategoriaDaLoja(lojaID, c);
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
