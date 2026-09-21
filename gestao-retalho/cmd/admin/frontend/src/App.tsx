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
  ArrowRightStartOnRectangleIcon,
} from "@heroicons/react/24/outline";
import { PrecisaConfigurarAdmin, Logout, ObterLojaAtual } from "../wailsjs/go/admin/AdminApp";
import Configurar from "./Configurar";
import Login from "./Login";
import Dashboard from "./Dashboard";
import CategoriasScreen from "./Categorias";
import ProdutosScreen from "./Produtos";

type Ecra = "dashboard" | "produtos" | "categorias" | "stub";
type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;

// Ícones outline de uma biblioteca única e neutra (SDD 7.3: "Lucide/Feather"
// — Heroicons segue a mesma linguagem visual e cobre também as variantes
// sólidas que o tema ferragem precisa em cmd/app, por isso ficou só uma
// dependência de ícones no projeto todo em vez de duas).
const NAV: { chave: Ecra; nome: string; Icone: IconeSvg; stub?: boolean }[] = [
  { chave: "dashboard", nome: "Dashboard", Icone: Squares2X2Icon },
  { chave: "produtos", nome: "Produtos", Icone: CubeIcon },
  { chave: "categorias", nome: "Categorias", Icone: TagIcon },
  { chave: "stub", nome: "Fichas técnicas", Icone: ClipboardDocumentListIcon, stub: true },
  { chave: "stub", nome: "Armazéns", Icone: BuildingOffice2Icon, stub: true },
  { chave: "stub", nome: "Stock", Icone: ArchiveBoxIcon, stub: true },
  { chave: "stub", nome: "Inventário", Icone: ClipboardDocumentCheckIcon, stub: true },
  { chave: "stub", nome: "Vendas", Icone: ShoppingCartIcon, stub: true },
  { chave: "stub", nome: "Clientes e dívidas", Icone: UserGroupIcon, stub: true },
  { chave: "stub", nome: "Relatórios", Icone: ChartBarIcon, stub: true },
  { chave: "stub", nome: "Utilizadores", Icone: UserCircleIcon, stub: true },
];

type Estado = { tipo: "a-carregar" } | { tipo: "configurar" } | { tipo: "login" } | { tipo: "pronto" };
type LojaAtual = { Nome: string; TipoNegocio: string };

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
  return <AdminPrincipal loja={loja} onSair={() => Logout().finally(verificar)} />;
}

/**
 * Sidebar fiel à tela `1c` dos mocks aprovados (SDD 7.3/8.5, Fase 9), com
 * "Categorias" adicionado fora do mock e "Lojas" removido (Fase 10
 * addendum, a pedido explícito — deixou de fazer sentido um registo de
 * várias lojas quando cada cmd/admin serve sempre uma só). "Dashboard",
 * "Produtos" e "Categorias" têm ecrã real; o resto da sidebar existe por
 * fidelidade visual ao mock, mas a gestão detalhada de stock/vendas/etc.
 * continua a acontecer no executável da loja (cmd/app).
 */
function AdminPrincipal({ loja, onSair }: { loja: LojaAtual | null; onSair: () => void }) {
  const [ecra, setEcra] = useState<Ecra>("dashboard");
  const [stubNome, setStubNome] = useState("");
  const [erro, setErro] = useState<string | null>(null);

  return (
    <div className="admin-shell">
      <aside className="admin-sidebar">
        <div className="admin-sidebar-brand">
          <div className="admin-sidebar-brand-mark" />
          <div>
            <div className="admin-sidebar-brand-nome">{loja?.Nome ?? "Administração"}</div>
            {loja && <div className="admin-sidebar-brand-tag">{loja.TipoNegocio}</div>}
          </div>
        </div>
        <nav className="admin-sidebar-nav">
          {NAV.map((item) => (
            <button
              key={item.nome}
              className={`admin-sidebar-item${!item.stub && ecra === item.chave ? " active" : ""}${item.stub ? " stub" : ""}`}
              onClick={() => {
                setEcra(item.chave);
                if (item.stub) setStubNome(item.nome);
              }}
            >
              <item.Icone className="admin-sidebar-icon" aria-hidden="true" />
              {item.nome}
            </button>
          ))}
        </nav>
        <button className="admin-sidebar-item admin-sidebar-sair" onClick={onSair}>
          <ArrowRightStartOnRectangleIcon className="admin-sidebar-icon" aria-hidden="true" />
          Sair
        </button>
      </aside>

      <div className="admin-main">
        {erro && (
          <div className="admin-erro">
            {erro}
            <button onClick={() => setErro(null)}>×</button>
          </div>
        )}
        {ecra === "dashboard" && <Dashboard onErro={setErro} />}
        {ecra === "produtos" && <ProdutosScreen onErro={setErro} />}
        {ecra === "categorias" && <CategoriasScreen onErro={setErro} />}
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
    </div>
  );
}
