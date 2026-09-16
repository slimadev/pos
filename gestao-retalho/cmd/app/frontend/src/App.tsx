import { useEffect, useState } from "react";
import { Ping, EstaConfigurado, ObterEstadoConfig } from "../wailsjs/go/api/App";
import { usecase } from "../wailsjs/go/models";
import { THEMES, useTheme, type Theme } from "./hooks/useTheme";
import Wizard from "./wizard/Wizard";
import VendaDireta from "./pos/VendaDireta";
import Bottlestore from "./pos/Bottlestore";
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
 * Ecrã pós-wizard. Menu dinâmico por módulo (SDD 9.5): pos_direto mostra
 * o PDV direto (Fase 3), contas_abertas mostra a bottlestore (Fase 4) —
 * nunca os dois ao mesmo tempo, tal como os presets do wizard nunca
 * ativam os dois módulos juntos. Produtos/armazéns/stock via FaseUm
 * continua scaffolding até as fases de admin trazerem os ecrãs definitivos.
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

      {config.ModulosAtivos.includes("pos_direto") && <VendaDireta />}
      {config.ModulosAtivos.includes("contas_abertas") && <Bottlestore />}

      <FaseUm />
    </div>
  );
}

export default App;
