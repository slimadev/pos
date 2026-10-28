import { useEffect, useRef, useState } from "react";
import type { ChangeEvent, FormEvent } from "react";
import {
  ConfigurarPrimeiroArranque,
  ObterPresetsNegocio,
  EscolherCopiaParaRestaurar,
  PrepararRestauroCopia,
} from "../wailsjs/go/admin/AdminApp";
import { admin, domain } from "../wailsjs/go/models";
import { FolderOpenIcon, CheckIcon, ArrowUturnLeftIcon } from "@heroicons/react/24/outline";
import { noAndroid, reiniciarApp } from "./plataforma";

const TAMANHO_MAXIMO_LOGO = 1.5 * 1024 * 1024;

/**
 * Primeiro arranque do cmd/admin (Fase 10 addendum): um único ecrã que
 * junta o que antes estava dividido em dois arranques separados — tipo de
 * negócio, nome e logótipo da primeira loja (o que o wizard do cmd/app
 * pedia) e o utilizador/password do único administrador de todo o sistema
 * (o que "Configurar" já pedia). Depois disto é sempre o ecrã de Login.
 */
export default function Configurar({ onConcluido }: { onConcluido: () => void }) {
  const [presets, setPresets] = useState<domain.PresetNegocio[]>([]);
  const [negocioNome, setNegocioNome] = useState("");
  const [tipoNegocio, setTipoNegocio] = useState("");
  const [logo, setLogo] = useState("");

  const [nome, setNome] = useState("");
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [passwordConfirm, setPasswordConfirm] = useState("");

  const [erro, setErro] = useState<string | null>(null);
  const [aGuardar, setAGuardar] = useState(false);
  const ficheiroRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    ObterPresetsNegocio().then(setPresets).catch((e) => setErro(String(e)));
  }, []);

  function escolherLogo(e: ChangeEvent<HTMLInputElement>) {
    const ficheiro = e.target.files?.[0];
    if (!ficheiro) return;
    if (ficheiro.size > TAMANHO_MAXIMO_LOGO) {
      setErro("Logótipo demasiado grande (máximo 1.5 MB)");
      return;
    }
    const leitor = new FileReader();
    leitor.onload = () => setLogo(String(leitor.result));
    leitor.readAsDataURL(ficheiro);
  }

  function concluir(e: FormEvent) {
    e.preventDefault();
    if (aGuardar) return;
    setErro(null);

    if (password.length < 6) {
      setErro("A password deve ter pelo menos 6 caracteres");
      return;
    }
    if (password !== passwordConfirm) {
      setErro("As passwords não coincidem");
      return;
    }

    const params = new admin.ConfigurarPrimeiroArranqueParams();
    params.NegocioNome = negocioNome.trim();
    params.TipoNegocio = tipoNegocio;
    params.Logo = logo;
    params.AdminNome = nome.trim();
    params.AdminUsername = username.trim();
    params.AdminPassword = password;

    setAGuardar(true);
    ConfigurarPrimeiroArranque(params)
      .then((resultado) => {
        if (resultado.AvisoLoja) {
          // Administrador ficou criado e a sessão ativa mesmo que a loja
          // tenha falhado — não bloqueia a entrada, só avisa; a loja pode
          // ser adicionada depois em "Lojas".
          console.warn("primeira loja não foi criada:", resultado.AvisoLoja);
        }
        onConcluido();
      })
      .catch((err) => setErro(String(err)))
      .finally(() => setAGuardar(false));
  }

  // Mensagem do restauro em curso (substitui o link enquanto existe).
  const [aRestaurar, setARestaurar] = useState<string | null>(null);

  async function restaurar() {
    setErro(null);
    try {
      const caminho = await EscolherCopiaParaRestaurar();
      if (!caminho) return;
      const m = await PrepararRestauroCopia(caminho);
      const data = new Date(String(m.CriadaEm)).toLocaleString("pt-PT");
      if (noAndroid) {
        setARestaurar(`Cópia de ${data} pronta. A aplicação vai reiniciar…`);
        setTimeout(reiniciarApp, 1500);
      } else {
        setARestaurar(`Cópia de ${data} pronta. Feche e volte a abrir a administração para concluir.`);
      }
    } catch (e) {
      setErro(String(e).replace(/^Error:\s*/, ""));
    }
  }

  const podeSubmeter =
    negocioNome.trim() !== "" &&
    tipoNegocio !== "" &&
    nome.trim() !== "" &&
    username.trim() !== "" &&
    password !== "" &&
    passwordConfirm !== "" &&
    !aGuardar;

  return (
    <div className="login-shell login-shell--alto">
      <form className="login-card config-card" onSubmit={concluir}>
        <div className="login-logo" />
        <h1 className="login-titulo">Administração</h1>
        <p className="login-sub">Primeiro arranque — configure a primeira loja e o administrador</p>

        {erro && <div className="login-erro">{erro}</div>}

        <div className="config-secao">Loja</div>

        <label className="login-campo">
          Nome do negócio
          <input
            value={negocioNome}
            onChange={(e) => setNegocioNome(e.target.value)}
            autoFocus
            autoComplete="off"
            placeholder="ex: Ferragem Cossa"
          />
        </label>

        <div className="config-tipos">
          {presets.map((preset) => (
            <div
              key={preset.Tipo}
              className={`config-tipo${tipoNegocio === preset.Tipo ? " config-tipo--ativo" : ""}`}
              onClick={() => setTipoNegocio(preset.Tipo)}
            >
              <div className="config-tipo-nome">{preset.Nome}</div>
              <div className="config-tipo-desc">{preset.Descricao}</div>
            </div>
          ))}
        </div>

        <div className="config-logo-linha">
          {logo ? (
            <img src={logo} alt="Logótipo" className="config-logo-preview" />
          ) : (
            <div className="config-logo-preview config-logo-preview--vazio" />
          )}
          <button type="button" className="config-logo-botao" onClick={() => ficheiroRef.current?.click()}>
            <FolderOpenIcon className="btn-icone" aria-hidden="true" />
            {logo ? "Trocar logótipo" : "Escolher logótipo (opcional)"}
          </button>
          <input ref={ficheiroRef} type="file" accept="image/*" hidden onChange={escolherLogo} />
        </div>

        <div className="config-secao">Administrador</div>

        <label className="login-campo">
          Nome
          <input value={nome} onChange={(e) => setNome(e.target.value)} autoComplete="off" />
        </label>
        <label className="login-campo">
          Utilizador
          <input value={username} onChange={(e) => setUsername(e.target.value)} autoComplete="off" />
        </label>
        <label className="login-campo">
          Password
          <input
            type="password"
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            autoComplete="new-password"
          />
        </label>
        <label className="login-campo">
          Confirmar password
          <input
            type="password"
            value={passwordConfirm}
            onChange={(e) => setPasswordConfirm(e.target.value)}
            autoComplete="new-password"
          />
        </label>

        <button className="login-botao" type="submit" disabled={!podeSubmeter}>
          <CheckIcon className="btn-icone" aria-hidden="true" />
          {aGuardar ? "A configurar…" : "Concluir"}
        </button>

        {/* Terminal novo ou reposto: a loja volta de uma cópia de segurança,
            sem criar primeiro uma loja vazia. */}
        <div className="config-restaurar">
          {aRestaurar ?? (
            <>
              Já tem uma cópia de segurança?{" "}
              <button type="button" className="config-restaurar-link" onClick={restaurar}>
                <ArrowUturnLeftIcon className="btn-icone" aria-hidden="true" />
                Restaurar
              </button>
            </>
          )}
        </div>
      </form>
    </div>
  );
}
