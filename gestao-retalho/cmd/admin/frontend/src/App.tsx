import { useState } from "react";
import Dashboard from "./Dashboard";
import LojasScreen from "./Lojas";

type Ecra = "dashboard" | "lojas" | "stub";

const NAV: { chave: Ecra; nome: string; stub?: boolean }[] = [
  { chave: "dashboard", nome: "Dashboard" },
  { chave: "stub", nome: "Produtos", stub: true },
  { chave: "stub", nome: "Fichas técnicas", stub: true },
  { chave: "stub", nome: "Armazéns", stub: true },
  { chave: "stub", nome: "Stock", stub: true },
  { chave: "stub", nome: "Inventário", stub: true },
  { chave: "stub", nome: "Vendas", stub: true },
  { chave: "stub", nome: "Clientes e dívidas", stub: true },
  { chave: "stub", nome: "Relatórios", stub: true },
  { chave: "stub", nome: "Utilizadores", stub: true },
  { chave: "lojas", nome: "Lojas" },
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
