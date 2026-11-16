import { useEffect, useState } from "react";
import {
  EstaConfigurado,
  ObterEstadoConfig,
  ListarOperadoresAtivos,
  ObterOperadorAtual,
  TerminarTurno,
} from "../wailsjs/go/api/App";
import { usecase } from "../wailsjs/go/models";
import { useTheme, type Theme } from "./hooks/useTheme";
import Wizard from "./wizard/Wizard";
import VendaDireta from "./pos/VendaDireta";
import Bottlestore from "./pos/Bottlestore";
import EscolherOperador from "./pos/Turno";
import {
  BuildingStorefrontIcon,
  BeakerIcon,
  WrenchScrewdriverIcon,
  SparklesIcon,
  ArrowPathIcon,
} from "@heroicons/react/24/outline";
import "./App.css";

type Estado = { tipo: "a-carregar" } | { tipo: "wizard" } | { tipo: "pronto"; config: usecase.EstadoConfig };

function getBusinessIcon(tipo: string) {
  switch (tipo) {
    case "mercearia":
      return BuildingStorefrontIcon;
    case "farmacia":
      return BeakerIcon;
    case "ferragem":
      return WrenchScrewdriverIcon;
    case "bottlestore":
      return SparklesIcon;
    default:
      return BuildingStorefrontIcon;
  }
}

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

type Turno =
  | { tipo: "a-carregar" }
  | { tipo: "sem-operadores" }
  | { tipo: "escolher"; operadores: usecase.Operador[] }
  | { tipo: "ativo"; operador: usecase.Operador };

function useTurno() {
  const [turno, setTurno] = useState<Turno>({ tipo: "a-carregar" });

  const verificar = () => {
    ObterOperadorAtual()
      .then((atual) => {
        if (atual.ID !== 0) {
          setTurno({ tipo: "ativo", operador: atual });
          return;
        }
        return ListarOperadoresAtivos().then((operadores) =>
          setTurno(operadores.length === 0 ? { tipo: "sem-operadores" } : { tipo: "escolher", operadores }),
        );
      })
      .catch(() => setTurno({ tipo: "sem-operadores" }));
  };

  useEffect(() => {
    verificar();
    window.addEventListener("focus", verificar);
    return () => window.removeEventListener("focus", verificar);
  }, []);

  const trocar = () => {
    TerminarTurno().finally(verificar);
  };

  return { turno, setTurno, trocar };
}

function AppPrincipal({ config }: { config: usecase.EstadoConfig }) {
  useTheme(config.Tema as Theme);
  const temVenda = config.ModulosAtivos.includes("pos_direto") || config.ModulosAtivos.includes("contas_abertas");
  const { turno, setTurno, trocar } = useTurno();
  const BusinessIcon = getBusinessIcon(config.Negocio.TipoNegocio);

  return (
    <div id="App">
      <header className="app-header">
        <div className="app-header-left">
          <div className="app-header-brand-icon">
            {config.Logo ? (
              <img src={config.Logo} alt="" className="app-header-logo" />
            ) : (
              <BusinessIcon className="app-header-icon-svg" aria-hidden="true" />
            )}
          </div>
          <div className="app-header-info">
            <h1 className="app-header-title">{config.Negocio.Nome}</h1>
            <div className="app-header-meta">
              <span className="app-header-tipo">{config.Negocio.TipoNegocio}</span>
              <span className="app-header-status-dot" />
              <span className="app-header-status">Ponto de Venda Ativo</span>
            </div>
          </div>
        </div>

        {turno.tipo === "ativo" && (
          <div className="app-turno-card">
            <div className="app-turno-avatar">
              {turno.operador.Nome.trim().charAt(0).toUpperCase()}
            </div>
            <div className="app-turno-meta">
              <span className="app-turno-label">A vender</span>
              <strong className="app-turno-nome">{turno.operador.Nome}</strong>
            </div>
            <button className="app-turno-btn" onClick={trocar} title="Trocar de operador">
              <ArrowPathIcon className="app-turno-btn-icon" aria-hidden="true" />
              <span>Trocar</span>
            </button>
          </div>
        )}
      </header>

      {turno.tipo === "a-carregar" ? null : turno.tipo === "escolher" ? (
        <EscolherOperador
          operadores={turno.operadores}
          onIniciado={(operador) => setTurno({ tipo: "ativo", operador })}
        />
      ) : temVenda ? (
        <main className="app-main">
          {config.ModulosAtivos.includes("pos_direto") && <VendaDireta tipoNegocio={config.Negocio.TipoNegocio} />}
          {config.ModulosAtivos.includes("contas_abertas") && <Bottlestore />}
        </main>
      ) : (
        <p className="app-sem-venda">Nenhum módulo de venda ativo configurado para esta loja.</p>
      )}
    </div>
  );
}

export default App;
