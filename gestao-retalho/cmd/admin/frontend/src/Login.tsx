import { useState } from "react";
import type { FormEvent } from "react";
import { Login as EntrarBinding } from "../wailsjs/go/admin/AdminApp";

/**
 * Login do cmd/admin — o único de todo o sistema (decisão pós-Fase 10, a
 * pedido explícito). As lojas (cmd/app) não pedem login nenhum. `loja`
 * (nome + tipo de negócio) já vem carregada por App.tsx antes deste ecrã
 * aparecer — mostra a identidade real da loja em vez de "Administração"
 * genérico; a cor (App.tsx aplica `data-tipo-negocio` no `<html>`) já
 * também se aplica aqui, antes de autenticar.
 */
export default function Login({
  loja,
  onEntrar,
}: {
  loja: { Nome: string; TipoNegocio: string } | null;
  onEntrar: () => void;
}) {
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
        <h1 className="login-titulo">{loja?.Nome ?? "Administração"}</h1>
        {loja && <div className="login-tipo">{loja.TipoNegocio}</div>}
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
