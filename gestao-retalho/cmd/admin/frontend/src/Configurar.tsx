import { useState } from "react";
import type { FormEvent } from "react";
import { CriarAdministrador } from "../wailsjs/go/admin/AdminApp";

/**
 * Primeiro arranque do cmd/admin: cria o único administrador de todo o
 * sistema (decisão pós-Fase 10, a pedido explícito — um administrador
 * para todas as lojas, não um por loja). Só aparece quando admin.db ainda
 * não tem nenhum administrador; depois disso é sempre o ecrã de Login.
 */
export default function Configurar({ onConcluido }: { onConcluido: () => void }) {
  const [nome, setNome] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aGuardar, setAGuardar] = useState(false);

  function concluir(e: FormEvent) {
    e.preventDefault();
    if (aGuardar) return;
    setErro(null);

    if (password.length < 6) {
      setErro("A password deve ter pelo menos 6 caracteres");
      return;
    }
    if (password !== passwordConfirm) {
      setErro("As passwords não coincidem");
      return;
    }

    setAGuardar(true);
    CriarAdministrador(nome.trim(), username.trim(), password)
      .then(() => onConcluido())
      .catch((err) => setErro(String(err)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="login-shell">
      <form className="login-card" onSubmit={concluir}>
        <div className="login-logo" />
        <h1 className="login-titulo">Administração</h1>
        <p className="login-sub">Primeiro arranque — crie o administrador</p>

        {erro && <div className="login-erro">{erro}</div>}

        <label className="login-campo">
          Nome
          <input value={nome} onChange={(e) => setNome(e.target.value)} autoFocus autoComplete="off" />
        </label>
        <label className="login-campo">
          Utilizador
          <input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="off" />
        </label>
        <label className="login-campo">
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
          />
        </label>
        <label className="login-campo">
          Confirmar password
          <input
            type="password"
            value={passwordConfirm}
            onChange={(e) => setPasswordConfirm(e.target.value)}
            autoComplete="new-password"
          />
        </label>

        <button
          className="login-botao"
          type="submit"
          disabled={aGuardar || !nome.trim() || !username.trim() || !password || !passwordConfirm}
        >
          {aGuardar ? "A criar…" : "Criar administrador"}
        </button>
      </form>
    </div>
  );
}
