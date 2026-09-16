import { useState } from "react";
import type { ComponentType, SVGProps } from "react";
import {
  Squares2X2Icon,
  CubeIcon,
  ClipboardDocumentListIcon,
  BuildingOffice2Icon,
  ArchiveBoxIcon,
  ClipboardDocumentCheckIcon,
  ShoppingCartIcon,
  UserGroupIcon,
  ChartBarIcon,
  UserCircleIcon,
  BuildingStorefrontIcon,
} from "@heroicons/react/24/outline";
import Dashboard from "./Dashboard";
import LojasScreen from "./Lojas";

type Ecra = "dashboard" | "lojas" | "stub";
type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;

// Ícones outline de uma biblioteca única e neutra (SDD 7.3: "Lucide/Feather"
// — Heroicons segue a mesma linguagem visual e cobre também as variantes
// sólidas que o tema ferragem precisa em cmd/app, por isso ficou só uma
// dependência de ícones no projeto todo em vez de duas).
const NAV: { chave: Ecra; nome: string; Icone: IconeSvg; stub?: boolean }[] = [
  { chave: "dashboard", nome: "Dashboard", Icone: Squares2X2Icon },
  { chave: "stub", nome: "Produtos", Icone: CubeIcon, stub: true },
  { chave: "stub", nome: "Fichas técnicas", Icone: ClipboardDocumentListIcon, stub: true },
  { chave: "stub", nome: "Armazéns", Icone: BuildingOffice2Icon, stub: true },
  { chave: "stub", nome: "Stock", Icone: ArchiveBoxIcon, stub: true },
  { chave: "stub", nome: "Inventário", Icone: ClipboardDocumentCheckIcon, stub: true },
  { chave: "stub", nome: "Vendas", Icone: ShoppingCartIcon, stub: true },
  { chave: "stub", nome: "Clientes e dívidas", Icone: UserGroupIcon, stub: true },
  { chave: "stub", nome: "Relatórios", Icone: ChartBarIcon, stub: true },
  { chave: "stub", nome: "Utilizadores", Icone: UserCircleIcon, stub: true },
  { chave: "lojas", nome: "Lojas", Icone: BuildingStorefrontIcon },
];

/**
 * Administração universal (SDD 7.3/8.5, Fase 9) — sidebar fiel à tela `1c`
 * dos mocks aprovados. Só "Dashboard" e "Lojas" têm ecrã real nesta fase:
 * o resto da sidebar existe por fidelidade visual ao mock, mas a gestão
 * detalhada de produtos/stock/vendas/etc. continua a acontecer no
 * executável de cada loja (cmd/app) — o admin foca-se na visão agregada
 * (SDD 8.5: "visão agregada primeiro") e no registo de lojas.
 */
export default function App() {
  const [ecra, setEcra] = useState<Ecra>("dashboard");
  const [stubNome, setStubNome] = useState("");
  const [erro, setErro] = useState<string | null>(null);

  return (
    <div className="admin-shell">
      <aside className="admin-sidebar">
        <div className="admin-sidebar-brand">
          <div className="admin-sidebar-brand-mark" />
          <div className="admin-sidebar-brand-nome">Administração</div>
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
        <div className="admin-sidebar-footer">
          <div className="admin-sidebar-footer-nome">Administrador</div>
          <div className="admin-sidebar-footer-sub">Gestão de Retalho</div>
        </div>
      </aside>

      <div className="admin-main">
        {erro && (
          <div className="admin-erro">
            {erro}
            <button onClick={() => setErro(null)}>×</button>
          </div>
        )}
        {ecra === "dashboard" && <Dashboard onErro={setErro} />}
        {ecra === "lojas" && <LojasScreen onErro={setErro} />}
        {ecra === "stub" && (
          <div className="admin-lojas-screen">
            <h2 style={{ margin: 0 }}>{stubNome}</h2>
            <p style={{ color: "var(--text-secondary)", maxWidth: 520 }}>
              Esta secção gere-se a partir do executável de cada loja — abre a app da loja para gestão
              detalhada de {stubNome.toLowerCase()}. O admin universal foca-se na visão agregada
              (Dashboard) e no registo de lojas (Lojas).
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
