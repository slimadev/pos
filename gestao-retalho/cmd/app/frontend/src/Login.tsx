import { useState } from "react";
import type { FormEvent } from "react";
import { LockClosedIcon } from "@heroicons/react/24/outline";
import { Login as EntrarBinding } from "../wailsjs/go/api/App";
import { usecase } from "../wailsjs/go/models";
import { useTheme, type Theme } from "./hooks/useTheme";
import "./Login.css";

/**
 * Ecrã de sessão (pós-wizard, SDD 4.13): a app sempre pede login ao
 * arrancar quando já está configurada — nunca entra direto no ecrã
 * principal, mesmo que só exista um utilizador. Usa o tema já configurado
 * (config.Tema) mesmo antes de autenticar, para o operador reconhecer a
 * loja certa de imediato.
 */
export default function Login({
  config,
  onEntrar,
}: {
  config: usecase.EstadoConfig;
  onEntrar: () => void;
}) {
  useTheme(config.Tema as Theme);

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
        <div className="login-logo">
          <LockClosedIcon className="login-logo-icon" aria-hidden="true" />
        </div>
        <h1 className="login-titulo">{config.Negocio.Nome}</h1>
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
