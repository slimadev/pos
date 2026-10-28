import { useEffect, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import {
  Squares2X2Icon,
  CubeIcon,
  TagIcon,
  ClipboardDocumentListIcon,
  BuildingOffice2Icon,
  ArchiveBoxIcon,
  ClipboardDocumentCheckIcon,
  ShoppingCartIcon,
  UserGroupIcon,
  ChartBarIcon,
  UserCircleIcon,
  ComputerDesktopIcon,
  BeakerIcon,
  ArrowRightStartOnRectangleIcon,
  Cog6ToothIcon,
  DocumentTextIcon,
  ReceiptPercentIcon,
  Bars3Icon,
  ShieldCheckIcon,
  UserPlusIcon,
} from "@heroicons/react/24/outline";
import {
  PrecisaConfigurarAdmin,
  Logout,
  ObterLojaAtual,
  AbrirPDV,
  ObterEstadoArranque,
} from "../wailsjs/go/admin/AdminApp";
import { noAndroid } from "./plataforma";
import { admin } from "../wailsjs/go/models";
import Configurar from "./Configurar";
import Login from "./Login";
import Dashboard from "./Dashboard";
import CategoriasScreen from "./Categorias";
import ProdutosScreen from "./Produtos";
import ArmazensScreen from "./Armazens";
import StockScreen from "./Stock";
import FarmaciaScreen from "./Farmacia";
import InventarioScreen from "./Inventario";
import VendasScreen from "./Vendas";
import RelatoriosScreen from "./Relatorios";
import type { Relatorio } from "./Relatorios";
import ClientesScreen from "./Clientes";
import VendaCreditoScreen from "./VendaCredito";
import CotacoesScreen from "./Cotacoes";
import FaturasScreen from "./Faturas";
import FichasScreen from "./Fichas";
import UtilizadoresScreen from "./Utilizadores";
import CopiasSegurancaScreen from "./CopiasSeguranca";
import ConfiguracoesLoja from "./ConfiguracoesLoja";
import Modal from "./Modal";
import { rotuloTipoNegocio } from "./formato";

type Ecra = "dashboard" | "produtos" | "categorias" | "armazens" | "stock" | "inventario" | "vendas" | "cotacoes" | "faturas" | "relatorios" | "clientes" | "credito" | "fichas" | "utilizadores" | "farmacia" | "copias" | "stub";
type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;

/** Destino de navegação entre ecrãs (ex: "Vendas" → "Fecho de caixa"). */
export type Destino = {
  ecra: Ecra;
  relatorio?: Relatorio;
  cliente?: { ID: number; Nome: string };
  origem?: "vendas" | "clientes";
};
export type Navegar = (d: Destino) => void;

// Ícones outline de uma biblioteca única e neutra (SDD 7.3: "Lucide/Feather"
// — Heroicons segue a mesma linguagem visual e cobre também as variantes
// sólidas que o tema ferragem precisa em cmd/app, por isso ficou só uma
// dependência de ícones no projeto todo em vez de duas).
type ItemNav = { chave: Ecra; nome: string; Icone: IconeSvg; stub?: boolean; soFarmacia?: boolean };

// Menu agrupado pelo trabalho do dia a dia. "Venda a crédito" não tem
// entrada própria — abre-se a partir de Vendas ou da ficha do cliente.
const NAV: { grupo?: string; itens: ItemNav[] }[] = [
  { itens: [{ chave: "dashboard", nome: "Dashboard", Icone: Squares2X2Icon }] },
  {
    grupo: "Vendas",
    itens: [
      { chave: "vendas", nome: "Vendas", Icone: ShoppingCartIcon },
      { chave: "cotacoes", nome: "Cotações", Icone: DocumentTextIcon },
      { chave: "faturas", nome: "Faturas", Icone: ReceiptPercentIcon },
      { chave: "clientes", nome: "Clientes e dívidas", Icone: UserGroupIcon },
    ],
  },
  {
    grupo: "Stock",
    itens: [
      { chave: "stock", nome: "Stock", Icone: ArchiveBoxIcon },
      { chave: "inventario", nome: "Inventário", Icone: ClipboardDocumentCheckIcon },
      { chave: "farmacia", nome: "Farmácia", Icone: BeakerIcon, soFarmacia: true },
      { chave: "armazens", nome: "Armazéns", Icone: BuildingOffice2Icon },
    ],
  },
  {
    grupo: "Catálogo",
    itens: [
      { chave: "produtos", nome: "Produtos", Icone: CubeIcon },
      { chave: "categorias", nome: "Categorias", Icone: TagIcon },
      { chave: "fichas", nome: "Fichas técnicas", Icone: ClipboardDocumentListIcon },
    ],
  },
  {
    grupo: "Gestão",
    itens: [
      { chave: "relatorios", nome: "Relatórios", Icone: ChartBarIcon },
      { chave: "utilizadores", nome: "Utilizadores", Icone: UserCircleIcon },
      { chave: "copias", nome: "Cópias de segurança", Icone: ShieldCheckIcon },
    ],
  },
];

type Estado = { tipo: "a-carregar" } | { tipo: "configurar" } | { tipo: "login" } | { tipo: "pronto" };
type LojaAtual = admin.LojaAtual;

/**
 * Arranque do cmd/admin: admin.db vazio (primeiro arranque de sempre)
 * mostra "Configurar" — cria a única loja que este cmd/admin serve (tipo
 * de negócio, nome, logótipo) e o único administrador de todo o sistema,
 * tudo num só ecrã (Fase 10 addendum, a pedido explícito); já configurado,
 * pede sempre login antes de mostrar o dashboard (o "Admin Geral"). A loja
 * (cmd/app) não tem login nenhum — só este, aqui, é que existe. Um
 * administrador serve sempre exatamente uma loja (que pode ter vários
 * armazéns internamente) — não há registo de várias lojas separadas.
 *
 * A identidade/cor da loja (nome, `data-tipo-negocio` no `<html>`) é
 * carregada aqui, uma vez, e aplica-se tanto ao ecrã de Login como ao
 * resto da app depois de entrar — pedido explícito: "o ecrã de login
 * também deve ter a cor de tipo de loja setado".
 */
export default function App() {
  const [estado, setEstado] = useState<Estado>({ tipo: "a-carregar" });
  const [loja, setLoja] = useState<LojaAtual | null>(null);

  const verificar = () => {
    PrecisaConfigurarAdmin()
      .then((precisa) => setEstado({ tipo: precisa ? "configurar" : "login" }))
      .catch(() => setEstado({ tipo: "login" }));
  };

  const carregarLoja = () => {
    ObterLojaAtual()
      .then((l) => {
        setLoja(l);
        document.documentElement.setAttribute("data-tipo-negocio", l.TipoNegocio);
      })
      .catch(() => {
        // Sem loja configurada ainda (ex: primeiro arranque, ou a criação
        // falhou) — fica com a marca/cor genéricas, não bloqueia a app.
      });
  };

  useEffect(() => {
    verificar();
    carregarLoja();
  }, []);

  if (estado.tipo === "a-carregar") {
    return null;
  }
  if (estado.tipo === "configurar") {
    return (
      <Configurar
        onConcluido={() => {
          carregarLoja();
          setEstado({ tipo: "pronto" });
        }}
      />
    );
  }
  if (estado.tipo === "login") {
    return <Login loja={loja} onEntrar={() => setEstado({ tipo: "pronto" })} />;
  }
  return <AdminPrincipal loja={loja} onLojaAtualizada={setLoja} onSair={() => Logout().finally(verificar)} />;
}

/**
 * Sidebar fiel à tela `1c` dos mocks aprovados (SDD 7.3/8.5, Fase 9), com
 * "Categorias" adicionado fora do mock e "Lojas" removido (Fase 10
 * addendum, a pedido explícito — deixou de fazer sentido um registo de
 * várias lojas quando cada cmd/admin serve sempre uma só). Toda a gestão
 * da loja vive aqui, em grupos; "Farmácia" só aparece em lojas desse tipo e
 * a "Venda a crédito" (só no admin, nunca no PDV) abre-se a partir de
 * Vendas ou da ficha do cliente. O botão "PDV" não
 * troca de ecrã aqui dentro — lança o cmd/app como processo à parte,
 * maximizado, apontado à mesma base desta loja (decisão pós-Fase 10, a
 * pedido explícito: cmd/app deixou de abrir sozinho).
 */
function AdminPrincipal({
  loja,
  onLojaAtualizada,
  onSair,
}: {
  loja: LojaAtual | null;
  onLojaAtualizada: (l: LojaAtual) => void;
  onSair: () => void;
}) {
  const [configAberta, setConfigAberta] = useState(false);
  const [destino, setDestino] = useState<Destino & { n: number }>({ ecra: "dashboard", n: 0 });
  const ecra = destino.ecra;
  // Em ecrãs estreitos (terminal Android) a sidebar é uma gaveta que se abre
  // pelo botão ☰ da barra superior; no desktop isto não tem efeito visível.
  const [menuAberto, setMenuAberto] = useState(false);
  const navegar: Navegar = (d) => {
    setMenuAberto(false);
    setDestino((atual) => ({ ...d, n: atual.n + 1 }));
  };
  const [stubNome, setStubNome] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aAbrirPDV, setAAbrirPDV] = useState(false);
  // "Venda a crédito" não está no menu (abre-se a partir de Vendas/Clientes).
  const tituloEcra =
    ecra === "credito" ? "Venda a crédito" : (NAV.flatMap((g) => g.itens).find((i) => i.chave === ecra)?.nome ?? "");

  // O PDV só vende com operadores: enquanto não houver nenhum, o admin
  // entra direto em Utilizadores e mostra o "último passo" em todos os ecrãs.
  const [operadoresAtivos, setOperadoresAtivos] = useState<number | null>(null);
  useEffect(() => {
    ObterEstadoArranque()
      .then((e) => {
        setOperadoresAtivos(e.OperadoresAtivos);
        if (e.OperadoresAtivos === 0) navegar({ ecra: "utilizadores" });
      })
      .catch(() => setOperadoresAtivos(null));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só ao entrar
  }, []);
  const semOperadores = operadoresAtivos === 0;

  function abrirPDV() {
    if (aAbrirPDV) return;
    if (semOperadores) {
      navegar({ ecra: "utilizadores" });
      return;
    }
    setAAbrirPDV(true);
    AbrirPDV()
      .catch((e) => setErro(String(e)))
      .finally(() => setAAbrirPDV(false));
  }

  // No terminal, sair da administração é voltar ao PDV (com a sessão
  // terminada). Sem operadores não há PDV para onde voltar: fica no login.
  const sair = noAndroid && !semOperadores ? abrirPDV : onSair;

  return (
    <div className={`admin-shell${menuAberto ? " menu-aberto" : ""}`}>
      <header className="admin-appbar">
        <button className="admin-appbar-menu" aria-label="Abrir menu" onClick={() => setMenuAberto(true)}>
          <Bars3Icon className="admin-appbar-menu-icon" aria-hidden="true" />
        </button>
        <div className="admin-appbar-titulo">{tituloEcra || loja?.Nome || "Administração"}</div>
      </header>
      <div className="admin-gaveta-fundo" onClick={() => setMenuAberto(false)} />
      <aside className="admin-sidebar">
        <div className="admin-sidebar-brand">
          {loja?.Logo ? (
            <img src={loja.Logo} alt="" className="admin-sidebar-brand-logo" />
          ) : (
            <div className="admin-sidebar-brand-mark" />
          )}
          <div className="admin-sidebar-brand-texto">
            <div className="admin-sidebar-brand-nome" title={loja?.Nome}>
              {loja?.Nome ?? "Administração"}
            </div>
            {loja && <div className="admin-sidebar-brand-tag">{rotuloTipoNegocio(loja.TipoNegocio)}</div>}
          </div>
          {loja && (
            <button
              className="admin-sidebar-config"
              title="Configurações da loja"
              aria-label="Configurações da loja"
              onClick={() => {
                setMenuAberto(false);
                setConfigAberta(true);
              }}
            >
              <Cog6ToothIcon className="admin-sidebar-config-icon" aria-hidden="true" />
            </button>
          )}
        </div>
        <button className="admin-sidebar-pdv" disabled={aAbrirPDV} onClick={abrirPDV}>
          <ComputerDesktopIcon className="admin-sidebar-icon" aria-hidden="true" />
          {aAbrirPDV ? "A abrir…" : "PDV"}
        </button>
        <nav className="admin-sidebar-nav">
          {NAV.map((g) => {
            const itens = g.itens.filter((item) => !item.soFarmacia || loja?.TipoNegocio === "farmacia");
            return (
              <div key={g.grupo ?? "inicio"} className="admin-sidebar-grupo">
                {g.grupo && <div className="admin-sidebar-grupo-nome">{g.grupo}</div>}
                {itens.map((item) => (
                  <button
                    key={item.nome}
                    className={`admin-sidebar-item${!item.stub && (ecra === item.chave || (ecra === "credito" && item.chave === destino.origem)) ? " active" : ""}${item.stub ? " stub" : ""}`}
                    onClick={() => {
                      navegar({ ecra: item.chave });
                      if (item.stub) setStubNome(item.nome);
                    }}
                  >
                    <item.Icone className="admin-sidebar-icon" aria-hidden="true" />
                    {item.nome}
                  </button>
                ))}
              </div>
            );
          })}
        </nav>
        <button className="admin-sidebar-item admin-sidebar-sair" onClick={sair}>
          <ArrowRightStartOnRectangleIcon className="admin-sidebar-icon" aria-hidden="true" />
          {sair === onSair ? "Sair" : "Sair para o PDV"}
        </button>
      </aside>

      <div className="admin-main">
        {erro && (
          <div className="admin-erro">
            {erro}
            <button onClick={() => setErro(null)}>×</button>
          </div>
        )}
        {semOperadores && (
          <div className="admin-ultimo-passo" role="status">
            <UserPlusIcon className="admin-ultimo-passo-icone" aria-hidden="true" />
            <div>
              <strong>Último passo: crie pelo menos um operador.</strong>
              <span>
                O PDV só vende com operadores — cada um entra com o seu PIN e as vendas ficam no seu nome.
              </span>
            </div>
            {ecra !== "utilizadores" && (
              <button className="admin-btn" onClick={() => navegar({ ecra: "utilizadores" })}>
                Criar operador
              </button>
            )}
          </div>
        )}
        <div key={destino.n} className="admin-ecra">
          {ecra === "dashboard" && <Dashboard onErro={setErro} />}
          {ecra === "produtos" && <ProdutosScreen onErro={setErro} />}
          {ecra === "categorias" && <CategoriasScreen onErro={setErro} />}
          {ecra === "armazens" && <ArmazensScreen onErro={setErro} />}
          {ecra === "stock" && <StockScreen onNavegar={navegar} onErro={setErro} />}
          {ecra === "inventario" && <InventarioScreen onErro={setErro} />}
          {ecra === "vendas" && <VendasScreen onNavegar={navegar} onErro={setErro} />}
          {ecra === "cotacoes" && <CotacoesScreen onNavegar={navegar} onErro={setErro} />}
          {ecra === "faturas" && <FaturasScreen onErro={setErro} />}
          {ecra === "clientes" && <ClientesScreen cliente={destino.cliente} onNavegar={navegar} onErro={setErro} />}
          {ecra === "credito" && (
            <VendaCreditoScreen
              cliente={destino.cliente}
              origem={destino.origem ?? "vendas"}
              onNavegar={navegar}
              onErro={setErro}
            />
          )}
          {ecra === "fichas" && <FichasScreen onErro={setErro} />}
          {ecra === "utilizadores" && <UtilizadoresScreen onErro={setErro} onAtivosMudaram={setOperadoresAtivos} />}
          {ecra === "relatorios" && (
            <RelatoriosScreen tipoNegocio={loja?.TipoNegocio} relatorioInicial={destino.relatorio} onErro={setErro} />
          )}
          {ecra === "farmacia" && <FarmaciaScreen onErro={setErro} />}
          {ecra === "copias" && <CopiasSegurancaScreen onErro={setErro} />}
        </div>
        {ecra === "stub" && (
          <div className="admin-lojas-screen">
            <h2 style={{ margin: 0 }}>{stubNome}</h2>
            <p style={{ color: "var(--text-secondary)", maxWidth: 520 }}>
              Esta secção gere-se a partir do executável da loja — abre a app da loja para gestão detalhada de{" "}
              {stubNome.toLowerCase()}.
            </p>
          </div>
        )}
      </div>

      {configAberta && loja && (
        <Modal titulo="Configurações da loja" onFechar={() => setConfigAberta(false)}>
          <ConfiguracoesLoja
            loja={loja}
            onGuardada={(l) => {
              onLojaAtualizada(l);
              setConfigAberta(false);
            }}
            onCancelar={() => setConfigAberta(false)}
            onErro={setErro}
          />
        </Modal>
      )}
    </div>
  );
}
