import { useState } from "react";
import { Ping } from "../wailsjs/go/api/App";
import { THEMES, useTheme } from "./hooks/useTheme";
import "./App.css";

/**
 * Fase 0 smoke test: prova que o binding Go↔React e a ligação SQLite
 * funcionam de ponta a ponta, e que os 5 temas (secção 7/9.6 do SDD)
 * carregam via CSS variables. Substituído pelo wizard real na Fase 2.
 */
function App() {
  const { theme, setTheme } = useTheme();
  const [status, setStatus] = useState("por testar");

  function testPing() {
    setStatus("a testar…");
    Ping()
      .then((result) => setStatus(`ok: ${result}`))
      .catch((err) => setStatus(`erro: ${err}`));
  }

  return (
    <div id="App">
      <h1>Gestão de Retalho — scaffolding Fase 0</h1>

      <div className="theme-switcher">
        {THEMES.map((t) => (
          <button
            key={t}
            className={t === theme ? "active" : ""}
            onClick={() => setTheme(t)}
          >
            {t}
          </button>
        ))}
      </div>

      <div className="ping-box">
        <button onClick={testPing}>Testar Ping() → SQLite</button>
        <span>{status}</span>
      </div>
    </div>
  );
}

export default App;
