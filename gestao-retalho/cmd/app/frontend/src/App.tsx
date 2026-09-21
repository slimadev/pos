import { useEffect, useState } from "react";
import { EstaConfigurado, ObterEstadoConfig } from "../wailsjs/go/api/App";
import { usecase } from "../wailsjs/go/models";
import { useTheme, type Theme } from "./hooks/useTheme";
import Wizard from "./wizard/Wizard";
import VendaDireta from "./pos/VendaDireta";
import Bottlestore from "./pos/Bottlestore";
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

/**
 * Ecrã principal: cmd/app é agora só o PDV — mostra o ecrã de venda certo
 * consoante os módulos ativos (PDV direto para mercearia/ferragem/
 * farmácia, contas abertas para bottlestore, nunca os dois ao mesmo
 * tempo). A gestão de produtos/categorias/armazéns/stock/etc. deixou de
 * viver aqui — é toda feita a partir do cmd/admin (decisão pós-Fase 10, a
 * pedido explícito: cmd/app deixou de abrir sozinho, só arranca a partir
 * do botão "PDV" do cmd/admin, sempre direto para a venda).
 */
function AppPrincipal({ config }: { config: usecase.EstadoConfig }) {
  useTheme(config.Tema as Theme);
  const temVenda = config.ModulosAtivos.includes("pos_direto") || config.ModulosAtivos.includes("contas_abertas");

  return (
    <div id="App">
      <div className="app-header">
        <h1>{config.Negocio.Nome}</h1>
        <span className="app-header-tipo">{config.Negocio.TipoNegocio}</span>
      </div>

      {temVenda ? (
        <>
          {config.ModulosAtivos.includes("pos_direto") && <VendaDireta tipoNegocio={config.Negocio.TipoNegocio} />}
          {config.ModulosAtivos.includes("contas_abertas") && <Bottlestore />}
        </>
      ) : (
        <p className="app-sem-venda">Nenhum módulo de venda ativo configurado para esta loja.</p>
      )}
    </div>
  );
}

export default App;
