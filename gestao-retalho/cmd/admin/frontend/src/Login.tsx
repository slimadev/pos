import { useState } from "react";
import type { FormEvent } from "react";
import { Login as EntrarBinding } from "../wailsjs/go/admin/AdminApp";

/**
 * Login do cmd/admin — o único de todo o sistema (decisão pós-Fase 10, a
 * pedido explícito). As lojas (cmd/app) não pedem login nenhum.
 */
export default function Login({ onEntrar }: { onEntrar: () => void }) {
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aEntrar, setAEntrar] = useState(false);

  function entrar(e: FormEvent) {
    e.preventDefault();
    if (!username.trim() || !password || aEntrar) return;
    setErro(null);
    setAEntrar(true);
    EntrarBinding(username.trim(), password)
      .then(() => onEntrar())
      .catch((err) => setErro(String(err)))
      .finally(() => setAEntrar(false));
  }

  return (
    <div className="login-shell">
      <form className="login-card" onSubmit={entrar}>
        <div className="login-logo" />
        <h1 className="login-titulo">Administração</h1>
        <p className="login-sub">Inicie sessão para continuar</p>

        {erro && <div className="login-erro">{erro}</div>}

        <label className="login-campo">
          Utilizador
          <input
            value={username}
            onChange={(e) => setUsername(e.target.value)}
            autoFocus
            autoComplete="off"
          />
        </label>
        <label className="login-campo">
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="off"
          />
        </label>

        <button className="login-botao" type="submit" disabled={aEntrar || !username.trim() || !password}>
          {aEntrar ? "A entrar…" : "Entrar"}
        </button>
      </form>
    </div>
  );
}
