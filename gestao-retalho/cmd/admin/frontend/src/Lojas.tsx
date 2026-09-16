import { useEffect, useState } from "react";
import { ListarLojas, CriarLoja, AtualizarLoja, AtivarLoja, DesativarLoja } from "../wailsjs/go/admin/AdminApp";
import { repository } from "../wailsjs/go/models";

/**
 * CRUD de "Lojas" (SDD, tela `1c`, Fase 9) — o registo de quais ficheiros
 * .db o admin conhece. Só guarda nome + caminho; os dados de negócio
 * continuam inteiramente dentro da base de cada loja (SDD 8.5).
 */
export default function LojasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [lojas, setLojas] = useState<repository.Loja[]>([]);
  const [lojaEmEdicao, setLojaEmEdicao] = useState<repository.Loja | null>(null);

  const recarregar = () => {
    ListarLojas().then(setLojas).catch((e) => onErro(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(recarregar, []);

  function alternarAtivo(loja: repository.Loja) {
    const acao = loja.Ativo ? DesativarLoja(loja.ID) : AtivarLoja(loja.ID);
    acao.then(recarregar).catch((e) => onErro(String(e)));
  }

  return (
    <div className="admin-lojas-screen">
      <h2 style={{ margin: 0 }}>Lojas</h2>

      <LojaForm
        key={lojaEmEdicao?.ID ?? "nova"}
        loja={lojaEmEdicao}
        onGuardado={() => {
          setLojaEmEdicao(null);
          recarregar();
        }}
        onErro={onErro}
      />

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Nome</th>
            <th>Ficheiro .db</th>
            <th>Ativa</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {lojas.map((l) => (
            <tr key={l.ID}>
              <td>{l.Nome}</td>
              <td style={{ fontFamily: "monospace", fontSize: 12 }}>{l.CaminhoDB}</td>
              <td>{l.Ativo ? "sim" : "não"}</td>
              <td className="admin-form">
                <button className="admin-btn-secundario" onClick={() => setLojaEmEdicao(l)}>
                  Editar
                </button>
                <button className="admin-btn-secundario" onClick={() => alternarAtivo(l)}>
                  {l.Ativo ? "Desativar" : "Ativar"}
                </button>
              </td>
            </tr>
          ))}
          {lojas.length === 0 && (
            <tr>
              <td colSpan={4}>Nenhuma loja registada — adicione uma acima.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

function LojaForm({
  loja,
  onGuardado,
  onErro,
}: {
  loja: repository.Loja | null;
  onGuardado: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(loja?.Nome ?? "");
  const [caminho, setCaminho] = useState(loja?.CaminhoDB ?? "");
  const [aGuardar, setAGuardar] = useState(false);

  function guardar() {
    if (!nome.trim() || !caminho.trim()) return;
    setAGuardar(true);
    const pedido = loja ? AtualizarLoja(loja.ID, nome.trim(), caminho.trim()) : CriarLoja(nome.trim(), caminho.trim());
    pedido
      .then(() => onGuardado())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-form">
      <input placeholder="Nome (ex: Bottlestore XPTO)" value={nome} onChange={(e) => setNome(e.target.value)} />
      <input
        placeholder="Caminho do ficheiro loja.db"
        value={caminho}
        onChange={(e) => setCaminho(e.target.value)}
        style={{ width: "22em" }}
      />
      <button className="admin-btn" disabled={!nome.trim() || !caminho.trim() || aGuardar} onClick={guardar}>
        {aGuardar ? "A guardar…" : loja ? "Guardar alterações" : "+ Adicionar loja"}
      </button>
    </div>
  );
}
