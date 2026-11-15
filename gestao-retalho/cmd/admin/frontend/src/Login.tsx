import { useState } from "react";
import type { FormEvent } from "react";
import { Login as EntrarBinding } from "../wailsjs/go/admin/AdminApp";
import {
  LockClosedIcon,
  UserIcon,
  ArrowRightEndOnRectangleIcon,
  SparklesIcon,
} from "@heroicons/react/24/outline";

/**
 * Login do cmd/admin com estética glassmorphism moderna e opção
 * de preenchimento rápido para o administrador configurado.
 */
export default function Login({
  loja,
  onEntrar,
}: {
  loja: { Nome: string; TipoNegocio: string; Logo?: string } | null;
  onEntrar: () => void;
}) {
  const [username, setUsername] = useState("beto");
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

  function entrarRapido() {
    setUsername("beto");
    setPassword("123456");
    setErro(null);
    setAEntrar(true);
    EntrarBinding("beto", "123456")
      .then(() => onEntrar())
      .catch((err) => setErro(String(err)))
      .finally(() => setAEntrar(false));
  }

  return (
    <div className="login-shell">
      <form className="login-card" onSubmit={entrar}>
        {loja?.Logo ? (
          <img src={loja.Logo} alt="" className="login-logo login-logo-imagem" />
        ) : (
          <div className="login-logo">
            <SparklesIcon style={{ width: 26, height: 26, color: "#ffffff" }} aria-hidden="true" />
          </div>
        )}
        <h1 className="login-titulo">{loja?.Nome ?? "Administração"}</h1>
        {loja && <div className="login-tipo">{loja.TipoNegocio}</div>}
        <p className="login-sub">Inicie sessão no painel de administração</p>

        {erro && <div className="login-erro">{erro}</div>}

        <label className="login-campo">
          <span style={{ display: "flex", alignItems: "center", gap: 5 }}>
            <UserIcon style={{ width: 14, height: 14 }} aria-hidden="true" />
            Utilizador
          </span>
          <input
            value={username}
            placeholder="Nome de utilizador (ex: beto)"
            onChange={(e) => setUsername(e.target.value)}
            autoFocus
            autoComplete="username"
          />
        </label>
        <label className="login-campo">
          <span style={{ display: "flex", alignItems: "center", gap: 5 }}>
            <LockClosedIcon style={{ width: 14, height: 14 }} aria-hidden="true" />
            Password
          </span>
          <input
            type="password"
            placeholder="Password de acesso"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="current-password"
          />
        </label>

        <button className="login-botao" type="submit" disabled={aEntrar || !username.trim() || !password}>
          <ArrowRightEndOnRectangleIcon style={{ width: 18, height: 18 }} aria-hidden="true" />
          <span>{aEntrar ? "A entrar…" : "Entrar no Painel"}</span>
        </button>

        <button
          type="button"
          className="login-rapido-btn"
          onClick={entrarRapido}
          disabled={aEntrar}
          title="Preenche o utilizador e password do administrador local"
        >
          <SparklesIcon style={{ width: 15, height: 15 }} aria-hidden="true" />
          <span>Entrar como Administrador (beto / 123456)</span>
        </button>
      </form>
    </div>
  );
}
