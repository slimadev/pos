import { useEffect, useState } from "react";
import { EstaConfigurado, ObterEstadoConfig } from "../wailsjs/go/api/App";
import { usecase } from "../wailsjs/go/models";
import { useTheme, type Theme } from "./hooks/useTheme";
import Wizard from "./wizard/Wizard";
import VendaDireta from "./pos/VendaDireta";
import Bottlestore from "./pos/Bottlestore";
import Administracao from "./Administracao";
import "./App.css";

type Estado = { tipo: "a-carregar" } | { tipo: "wizard" } | { tipo: "pronto"; config: usecase.EstadoConfig };

/**
 * Arranque da app: sem configuração ainda, abre o wizard (escolher tipo de
 * negócio, que decide só tema/cor e ecrã de venda). Já configurada, entra
 * direto no ecrã principal — cmd/app não pede login (decisão pós-Fase 10:
 * autenticação existe só no cmd/admin, um único administrador para todas
 * as lojas; ver PLANO_IMPLEMENTACAO.md).
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

type Modo = "venda" | "administracao";

/**
 * Ecrã principal: um único executável por loja com dois modos (SDD 9.5) —
 * "Venda" mostra o ecrã de PDV certo consoante os módulos ativos (PDV
 * direto para mercearia/ferragem/farmácia, contas abertas para
 * bottlestore, nunca os dois ao mesmo tempo), "Administração" é o ponto
 * único onde se gere produtos/categorias/armazéns/stock/etc., também
 * filtrado pelos módulos ativos (fichas técnicas e farmácia só aparecem
 * se o wizard os tiver ativado). O tipo de negócio escolhido na instalação
 * decide os dois: que ecrã de venda existe, e o que aparece na
 * administração.
 */
function AppPrincipal({ config }: { config: usecase.EstadoConfig }) {
  useTheme(config.Tema as Theme);
  const temVenda = config.ModulosAtivos.includes("pos_direto") || config.ModulosAtivos.includes("contas_abertas");
  const [modo, setModo] = useState<Modo>(temVenda ? "venda" : "administracao");

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
