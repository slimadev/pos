import { useEffect, useState } from "react";
import { Ping, EstaConfigurado, ObterEstadoConfig } from "../wailsjs/go/api/App";
import { usecase } from "../wailsjs/go/models";
import { THEMES, useTheme, type Theme } from "./hooks/useTheme";
import Wizard from "./wizard/Wizard";
import VendaDireta from "./pos/VendaDireta";
import FaseUm from "./FaseUm";
import "./App.css";

type Estado = { tipo: "a-carregar" } | { tipo: "wizard" } | { tipo: "pronto"; config: usecase.EstadoConfig };

function App() {
  const [estado, setEstado] = useState<Estado>({ tipo: "a-carregar" });

  const verificar = () => {
    EstaConfigurado()
      .then((configurado) => {
        if (!configurado) {
          setEstado({ tipo: "wizard" });
          return;
        }
        ObterEstadoConfig().then((config) => setEstado({ tipo: "pronto", config }));
      })
      .catch(() => setEstado({ tipo: "wizard" }));
  };

  useEffect(verificar, []);

  if (estado.tipo === "a-carregar") {
    return null;
  }
  if (estado.tipo === "wizard") {
    return <Wizard onConcluido={verificar} />;
  }
  return <AppPrincipal config={estado.config} />;
}

/**
 * Ecrã pós-wizard. O PDV direto (VendaDireta, Fase 3) já é o ecrã real
 * — o resto (produtos/armazéns/stock via FaseUm) continua scaffolding
 * até as Fases 4+ trazerem os ecrãs de admin definitivos.
 */
function AppPrincipal({ config }: { config: usecase.EstadoConfig }) {
  const { theme, setTheme } = useTheme(config.Tema as Theme);
  const [status, setStatus] = useState("por testar");

  function testPing() {
    setStatus("a testar…");
    Ping()
      .then((result) => setStatus(`ok: ${result}`))
      .catch((err) => setStatus(`erro: ${err}`));
  }

  return (
    <div id="App">
      <div className="app-header">
        <h1>{config.Negocio.Nome}</h1>
        <span className="app-header-tipo">{config.Negocio.TipoNegocio}</span>
      </div>

      <div className="theme-switcher">
        {THEMES.map((t) => (
          <button key={t} className={t === theme ? "active" : ""} onClick={() => setTheme(t)}>
            {t}
          </button>
        ))}
      </div>

      <div className="ping-box">
        <button onClick={testPing}>Testar Ping() → SQLite</button>
        <span>{status}</span>
      </div>

      <VendaDireta />

      <FaseUm />
    </div>
  );
}

export default App;
