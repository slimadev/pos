import { useEffect, useState } from "react";
import { ArrowRightStartOnRectangleIcon } from "@heroicons/react/24/outline";
import { EstaConfigurado, ObterEstadoConfig, Logout } from "../wailsjs/go/api/App";
import { usecase } from "../wailsjs/go/models";
import { useTheme, type Theme } from "./hooks/useTheme";
import Wizard from "./wizard/Wizard";
import Login from "./Login";
import VendaDireta from "./pos/VendaDireta";
import Bottlestore from "./pos/Bottlestore";
import Administracao from "./Administracao";
import "./App.css";

type Estado =
  | { tipo: "a-carregar" }
  | { tipo: "wizard" }
  | { tipo: "login"; config: usecase.EstadoConfig }
  | { tipo: "pronto"; config: usecase.EstadoConfig };

/**
 * Arranque da app (pedido do utilizador): sem configuração ainda, abre o
 * wizard (escolher tipo de negócio, criar o utilizador administrador);
 * assim que a configuração existe — seja porque o wizard acabou de a criar,
 * seja porque já existia de uma sessão anterior — a app pede sempre login
 * antes de mostrar o ecrã principal, nunca entra direto.
 */
function App() {
  const [estado, setEstado] = useState<Estado>({ tipo: "a-carregar" });

  const verificar = () => {
    EstaConfigurado()
      .then((configurado) => {
        if (!configurado) {
          setEstado({ tipo: "wizard" });
          return;
        }
        ObterEstadoConfig().then((config) => setEstado({ tipo: "login", config }));
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
  if (estado.tipo === "login") {
    return <Login config={estado.config} onEntrar={() => setEstado({ tipo: "pronto", config: estado.config })} />;
  }
  return <AppPrincipal config={estado.config} onSair={verificar} />;
}

type Modo = "venda" | "administracao";

/**
 * Ecrã pós-login: um único executável por loja com dois modos (SDD 9.5) —
 * "Venda" mostra o ecrã de PDV certo consoante os módulos ativos (PDV
 * direto para mercearia/ferragem/farmácia, contas abertas para
 * bottlestore, nunca os dois ao mesmo tempo), "Administração" é o ponto
 * único onde se gere produtos/categorias/armazéns/stock/etc., também
 * filtrado pelos módulos ativos (fichas técnicas e farmácia só aparecem
 * se o wizard os tiver ativado). O tipo de negócio escolhido na instalação
 * decide os dois: que ecrã de venda existe, e o que aparece na
 * administração.
 */
function AppPrincipal({ config, onSair }: { config: usecase.EstadoConfig; onSair: () => void }) {
  useTheme(config.Tema as Theme);
  const temVenda = config.ModulosAtivos.includes("pos_direto") || config.ModulosAtivos.includes("contas_abertas");
  const [modo, setModo] = useState<Modo>(temVenda ? "venda" : "administracao");

  function sair() {
    Logout().finally(onSair);
  }

  return (
    <div id="App">
      <div className="app-header">
        <h1>{config.Negocio.Nome}</h1>
        <span className="app-header-tipo">{config.Negocio.TipoNegocio}</span>
        {temVenda && (
          <div className="app-modo-switch">
            <button className={modo === "venda" ? "active" : ""} onClick={() => setModo("venda")}>
              Venda
            </button>
            <button className={modo === "administracao" ? "active" : ""} onClick={() => setModo("administracao")}>
              Administração
            </button>
          </div>
        )}
        <button className="app-sair" onClick={sair} title="Terminar sessão">
          <ArrowRightStartOnRectangleIcon className="app-sair-icon" aria-hidden="true" />
          Sair
        </button>
      </div>

      {modo === "venda" && temVenda ? (
        <>
          {config.ModulosAtivos.includes("pos_direto") && <VendaDireta tipoNegocio={config.Negocio.TipoNegocio} />}
          {config.ModulosAtivos.includes("contas_abertas") && <Bottlestore />}
        </>
      ) : (
        <Administracao modulosAtivos={config.ModulosAtivos} />
      )}
    </div>
  );
}

export default App;
