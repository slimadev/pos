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
import "./App.css";

type Estado = { tipo: "a-carregar" } | { tipo: "wizard" } | { tipo: "pronto"; config: usecase.EstadoConfig };

/**
 * Arranque da app: sem configuração ainda, abre o wizard (escolher tipo de
 * negócio, que decide só tema/cor e ecrã de venda). Já configurada, entra
 * no ecrã principal. O login completo existe só no cmd/admin; aqui, se a
 * loja tiver operadores, cada um só se identifica com o seu PIN.
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
type Turno =
  | { tipo: "a-carregar" }
  | { tipo: "sem-operadores" }
  | { tipo: "escolher"; operadores: usecase.Operador[] }
  | { tipo: "ativo"; operador: usecase.Operador };

/**
 * Com operadores registados no cmd/admin, ninguém vende sem se identificar
 * (o backend também o exige). Sem operadores, o PDV funciona como sempre e
 * tudo fica atribuído ao administrador. Ao voltar o foco à janela volta a
 * verificar: um operador criado ou desativado no admin entretanto conta.
 */
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

  return (
    <div id="App">
      <div className="app-header">
        <h1>{config.Negocio.Nome}</h1>
        <span className="app-header-tipo">{config.Negocio.TipoNegocio}</span>
        {turno.tipo === "ativo" && (
          <div className="app-turno">
            <span>
              A vender: <strong>{turno.operador.Nome}</strong>
            </span>
            <button onClick={trocar}>Trocar operador</button>
          </div>
        )}
      </div>

      {turno.tipo === "a-carregar" ? null : turno.tipo === "escolher" ? (
        <EscolherOperador
          operadores={turno.operadores}
          onIniciado={(operador) => setTurno({ tipo: "ativo", operador })}
        />
      ) : temVenda ? (
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
