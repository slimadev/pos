import { useEffect, useState } from "react";
import { ObterPresetsNegocio, ObterTodosModulos, FinalizarWizard } from "../../wailsjs/go/api/App";
import { domain, usecase } from "../../wailsjs/go/models";
import { useTheme, type Theme } from "../hooks/useTheme";
import "./Wizard.css";

const PASSOS = ["Nome do negócio", "Tipo de negócio", "Módulos e armazém", "Utilizador administrador"] as const;

const ROTULO_MODULO: Record<string, string> = {
  pos_direto: "Venda direta (POS)",
  contas_abertas: "Contas abertas (tipo restaurante/bar)",
  fichas_tecnicas: "Fichas técnicas (produtos compostos: pratos/bebidas)",
  lotes_validade: "Lotes e validade",
  receitas_medicas: "Receitas médicas",
  venda_granel: "Venda a granel/peso",
  dividas: "Gestão de dívidas de clientes",
};

/**
 * Wizard de 1º arranque (SDD secção 9.2, tela de design `1b`): 4 passos —
 * nome do negócio, tipo de negócio, módulos+armazém, administrador. Só
 * grava tudo de uma vez no fim (ConfigService.FinalizarWizard); não há
 * estado a meio caminho guardado na base.
 */
export default function Wizard({ onConcluido }: { onConcluido: () => void }) {
  const { setTheme } = useTheme("admin");

  const [passo, setPasso] = useState(1);
  const [presets, setPresets] = useState<domain.PresetNegocio[]>([]);
  const [todosModulos, setTodosModulos] = useState<string[]>([]);

  const [negocioNome, setNegocioNome] = useState("");
  const [tipoNegocio, setTipoNegocio] = useState<string>("");
  const [modulosPersonalizados, setModulosPersonalizados] = useState<Set<string>>(new Set());
  const [armazemNome, setArmazemNome] = useState("Armazém Principal");
  const [adminNome, setAdminNome] = useState("");
  const [adminUsername, setAdminUsername] = useState("");
  const [adminPassword, setAdminPassword] = useState("");
  const [adminPasswordConfirm, setAdminPasswordConfirm] = useState("");

  const [erro, setErro] = useState<string | null>(null);
  const [aGuardar, setAGuardar] = useState(false);

  useEffect(() => {
    ObterPresetsNegocio().then(setPresets).catch((e) => setErro(String(e)));
    ObterTodosModulos().then(setTodosModulos).catch((e) => setErro(String(e)));
  }, []);

  const presetAtual = presets.find((p) => p.Tipo === tipoNegocio);
  const ehPersonalizado = tipoNegocio === "personalizado";

  function escolherTipo(tipo: string) {
    setTipoNegocio(tipo);
    setTheme((tipo === "personalizado" ? "admin" : tipo) as Theme);
  }

  function alternarModulo(m: string) {
    setModulosPersonalizados((prev) => {
      const next = new Set(prev);
      if (next.has(m)) next.delete(m);
      else next.add(m);
      return next;
    });
  }

  function podeAvancar() {
    switch (passo) {
      case 1:
        return negocioNome.trim() !== "";
      case 2:
        return tipoNegocio !== "";
      case 3:
        return armazemNome.trim() !== "" && (!ehPersonalizado || modulosPersonalizados.size > 0);
      default:
        return true;
    }
  }

  function concluir() {
    setErro(null);
    if (adminPassword.length < 6) {
      setErro("Password do administrador deve ter pelo menos 6 caracteres");
      return;
    }
    if (adminPassword !== adminPasswordConfirm) {
      setErro("As passwords não coincidem");
      return;
    }

    const params = new usecase.FinalizarWizardParams();
    params.NegocioNome = negocioNome.trim();
    params.TipoNegocio = tipoNegocio;
    params.ModulosAtivos = ehPersonalizado ? Array.from(modulosPersonalizados) : [];
    params.ArmazemPrincipalNome = armazemNome.trim();
    params.AdminNome = adminNome.trim();
    params.AdminUsername = adminUsername.trim();
    params.AdminPassword = adminPassword;

    setAGuardar(true);
    FinalizarWizard(params)
      .then(() => onConcluido())
      .catch((e) => setErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="wizard">
      <aside className="wizard-sidebar">
        <div className="wizard-brand">
          <div className="wizard-logo" />
          <div className="wizard-title">Configuração inicial</div>
          <div className="wizard-subtitle">Quatro passos, cerca de dois minutos.</div>
        </div>
        <ol className="wizard-steps">
          {PASSOS.map((label, i) => {
            const n = i + 1;
            const estado = n < passo ? "feito" : n === passo ? "atual" : "pendente";
            return (
              <li key={label} className={`wizard-step wizard-step--${estado}`}>
                <span className="wizard-step-bullet">{n < passo ? "✓" : n}</span>
                <span>{label}</span>
              </li>
            );
          })}
        </ol>
        <div className="wizard-footnote">
          um só instalador
          <br />
          um só executável
          <br />
          config decide o resto
        </div>
      </aside>

      <main className="wizard-main">
        <div className="wizard-content">
          {erro && <div className="wizard-erro">{erro}</div>}

          {passo === 1 && (
            <section>
              <div className="wizard-eyebrow">Passo 1 de 4</div>
              <h1>Bem-vindo — configure o seu negócio</h1>
              <label className="wizard-field">
                Nome do negócio
                <input
                  autoFocus
                  placeholder="ex: Mercearia da Dona Ana"
                  value={negocioNome}
                  onChange={(e) => setNegocioNome(e.target.value)}
                />
              </label>
            </section>
          )}

          {passo === 2 && (
            <section>
              <div className="wizard-eyebrow">Passo 2 de 4</div>
              <h1>Que tipo de negócio é esta loja?</h1>
              <div className="wizard-cards">
                {presets.map((preset) => (
                  <div
                    key={preset.Tipo}
                    className={`wizard-card ${tipoNegocio === preset.Tipo ? "wizard-card--ativo" : ""}`}
                    onClick={() => escolherTipo(preset.Tipo)}
                  >
                    <div className="wizard-radio" />
                    <div className="wizard-card-body">
                      <div className="wizard-card-title">{preset.Nome}</div>
                      <div className="wizard-card-desc">{preset.Descricao}</div>
                    </div>
                  </div>
                ))}
                <div
                  className={`wizard-card wizard-card--tracejado ${tipoNegocio === "personalizado" ? "wizard-card--ativo" : ""}`}
                  onClick={() => escolherTipo("personalizado")}
                >
                  <div className="wizard-radio" />
                  <div className="wizard-card-body">
                    <div className="wizard-card-title">Personalizado</div>
                    <div className="wizard-card-desc">Escolher módulos um a um no passo seguinte</div>
                  </div>
                </div>
              </div>
            </section>
          )}

          {passo === 3 && (
            <section>
              <div className="wizard-eyebrow">Passo 3 de 4</div>
              <h1>Módulos e armazéns</h1>

              {ehPersonalizado ? (
                <div className="wizard-checklist">
                  {todosModulos.map((m) => (
                    <label key={m} className="wizard-checkbox">
                      <input
                        type="checkbox"
                        checked={modulosPersonalizados.has(m)}
                        onChange={() => alternarModulo(m)}
                      />
                      {ROTULO_MODULO[m] ?? m}
                    </label>
                  ))}
                </div>
              ) : (
                <div className="wizard-readonly-modulos">
                  <div className="wizard-eyebrow">Módulos ativados para {presetAtual?.Nome}</div>
                  <ul>
                    {presetAtual?.Modulos.map((m) => <li key={m}>{ROTULO_MODULO[m] ?? m}</li>)}
                  </ul>
                </div>
              )}

              <label className="wizard-field">
                Nome do armazém principal
                <input value={armazemNome} onChange={(e) => setArmazemNome(e.target.value)} />
              </label>
            </section>
          )}

          {passo === 4 && (
            <section>
              <div className="wizard-eyebrow">Passo 4 de 4</div>
              <h1>Utilizador administrador</h1>
              <label className="wizard-field">
                Nome
                <input value={adminNome} onChange={(e) => setAdminNome(e.target.value)} />
              </label>
              <label className="wizard-field">
                Username
                <input value={adminUsername} onChange={(e) => setAdminUsername(e.target.value)} />
              </label>
              <label className="wizard-field">
                Password
                <input type="password" value={adminPassword} onChange={(e) => setAdminPassword(e.target.value)} />
              </label>
              <label className="wizard-field">
                Confirmar password
                <input
                  type="password"
                  value={adminPasswordConfirm}
                  onChange={(e) => setAdminPasswordConfirm(e.target.value)}
                />
              </label>
            </section>
          )}
        </div>

        <div className="wizard-nav">
          <button className="wizard-nav-back" disabled={passo === 1} onClick={() => setPasso((p) => p - 1)}>
            ← Voltar
          </button>
          {passo < 4 ? (
            <button className="wizard-nav-next" disabled={!podeAvancar()} onClick={() => setPasso((p) => p + 1)}>
              Continuar
            </button>
          ) : (
            <button
              className="wizard-nav-next"
              disabled={aGuardar || !adminNome || !adminUsername || !adminPassword}
              onClick={concluir}
            >
              {aGuardar ? "A concluir…" : "Concluir"}
            </button>
          )}
        </div>
      </main>
    </div>
  );
}
