import { useRef, useState } from "react";
import type { ChangeEvent } from "react";
import { AtualizarIdentidadeLoja } from "../wailsjs/go/admin/AdminApp";
import { admin, usecase } from "../wailsjs/go/models";
import { TrashIcon, XMarkIcon, FolderOpenIcon, CheckIcon } from "@heroicons/react/24/outline";

const TAMANHO_MAXIMO_LOGO = 1.5 * 1024 * 1024;
const lerNumero = (s: string) => Number(s.trim().replace(",", "."));

/**
 * Configurações da loja depois do assistente inicial: identificação (nome,
 * NUIT, logótipo), contactos (endereço, telefone, email) e faturação (taxa
 * de IVA, papel da impressora de recibos). Os dados da loja aparecem no
 * cabeçalho dos recibos e, mais tarde, das cotações e faturas. O tipo de
 * negócio não muda aqui (define módulos e tema). Sem logótipo, a app
 * mostra o quadrado com a cor do tipo de negócio, como antes.
 */
export default function ConfiguracoesLoja({
  loja,
  onGuardada,
  onCancelar,
  onErro,
}: {
  loja: admin.LojaAtual;
  onGuardada: (l: admin.LojaAtual) => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(loja.Nome);
  const [logo, setLogo] = useState(loja.Logo);
  const [papel, setPapel] = useState(loja.PapelRecibo || "80");
  const [nuit, setNuit] = useState(loja.NUIT ?? "");
  const [endereco, setEndereco] = useState(loja.Endereco ?? "");
  const [telefone, setTelefone] = useState(loja.Telefone ?? "");
  const [email, setEmail] = useState(loja.Email ?? "");
  const [iva, setIva] = useState(String(loja.TaxaIVA ?? 16).replace(".", ","));
  const [aGuardar, setAGuardar] = useState(false);
  const ficheiroRef = useRef<HTMLInputElement>(null);

  function escolherLogo(e: ChangeEvent<HTMLInputElement>) {
    const ficheiro = e.target.files?.[0];
    e.target.value = "";
    if (!ficheiro) return;
    if (ficheiro.size > TAMANHO_MAXIMO_LOGO) {
      onErro("Logótipo demasiado grande (máximo 1,5 MB)");
      return;
    }
    const leitor = new FileReader();
    leitor.onload = () => setLogo(String(leitor.result));
    leitor.readAsDataURL(ficheiro);
  }

  const nuitLimpo = nuit.replace(/\s/g, "");
  const erroNuit = nuitLimpo !== "" && !/^\d{9}$/.test(nuitLimpo) ? "O NUIT tem 9 algarismos." : null;
  const taxa = lerNumero(iva);
  const erroIva = iva.trim() === "" || !Number.isFinite(taxa) || taxa < 0 || taxa > 100 ? "Entre 0 e 100." : null;
  const podeGuardar = !!nome.trim() && !erroNuit && !erroIva && !aGuardar;

  function guardar() {
    if (!podeGuardar) return;
    const d = new usecase.DadosLoja();
    d.Nome = nome.trim();
    d.Logo = logo;
    d.PapelRecibo = papel;
    d.NUIT = nuitLimpo;
    d.Endereco = endereco.trim();
    d.Telefone = telefone.trim();
    d.Email = email.trim();
    d.TaxaIVA = taxa;
    setAGuardar(true);
    AtualizarIdentidadeLoja(d)
      .then(onGuardada)
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-produto-form">
      <div className="admin-config-secao">Identificação</div>
      <div className="admin-config-grelha">
        <label className="admin-campo">
          Nome da loja
          <input autoFocus value={nome} onChange={(e) => setNome(e.target.value)} />
        </label>
        <label className="admin-campo">
          NUIT
          <input inputMode="numeric" placeholder="9 algarismos" value={nuit} onChange={(e) => setNuit(e.target.value)} />
          {erroNuit && <span className="admin-campo-erro">{erroNuit}</span>}
        </label>
      </div>

      <div className="admin-campo">
        Logótipo
        <div className="admin-config-logo">
          {logo ? (
            <img src={logo} alt="Logótipo" className="admin-config-logo-preview" />
          ) : (
            <div className="admin-config-logo-preview vazio" title="Sem logótipo — mostra a cor da loja" />
          )}
          <div className="admin-form">
            <button className="admin-btn-secundario" onClick={() => ficheiroRef.current?.click()}>
              <FolderOpenIcon className="btn-icone" aria-hidden="true" />
              {logo ? "Trocar logótipo" : "Escolher logótipo"}
            </button>
            {logo && (
              <button className="admin-btn-secundario" onClick={() => setLogo("")}>
                <TrashIcon className="btn-icone" aria-hidden="true" />
                Remover
              </button>
            )}
          </div>
          <input ref={ficheiroRef} type="file" accept="image/*" hidden onChange={escolherLogo} />
        </div>
        <span className="admin-texto-fraco">
          PNG ou JPG até 1,5 MB, de preferência quadrado. Aparece no menu, no ecrã de entrada e nos recibos.
        </span>
      </div>

      <div className="admin-config-secao">Contactos</div>
      <label className="admin-campo">
        Endereço
        <input placeholder="ex: Av. Eduardo Mondlane 123, Matola" value={endereco} onChange={(e) => setEndereco(e.target.value)} />
      </label>
      <div className="admin-config-grelha">
        <label className="admin-campo">
          Telefone
          <input inputMode="tel" placeholder="ex: 84 123 4567" value={telefone} onChange={(e) => setTelefone(e.target.value)} />
        </label>
        <label className="admin-campo">
          Email
          <input inputMode="email" placeholder="ex: geral@loja.co.mz" value={email} onChange={(e) => setEmail(e.target.value)} />
        </label>
      </div>

      <div className="admin-config-secao">Faturação e recibos</div>
      <div className="admin-config-grelha">
        <label className="admin-campo">
          Taxa de IVA (%)
          <input inputMode="decimal" value={iva} onChange={(e) => setIva(e.target.value)} />
          {erroIva ? (
            <span className="admin-campo-erro">{erroIva}</span>
          ) : (
            <span className="admin-texto-fraco">{taxa === 0 ? "Isento de IVA." : "A taxa normal é 16%; 0 se a loja for isenta."}</span>
          )}
        </label>
        <label className="admin-campo">
          Papel da impressora de recibos
          <select value={papel} onChange={(e) => setPapel(e.target.value)}>
            <option value="80">Rolo de 80 mm (impressora de talão)</option>
            <option value="58">Rolo de 58 mm (impressora de talão pequena)</option>
            <option value="a5">Folha A5 (impressora normal)</option>
          </select>
        </label>
      </div>

      <div className="admin-form">
        <button className="admin-btn" disabled={!podeGuardar} onClick={guardar}>
          <CheckIcon className="btn-icone" aria-hidden="true" />
          {aGuardar ? "A guardar…" : "Guardar"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
      </div>
    </div>
  );
}
