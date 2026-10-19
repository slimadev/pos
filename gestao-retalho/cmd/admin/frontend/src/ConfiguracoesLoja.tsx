import { useRef, useState } from "react";
import type { ChangeEvent } from "react";
import { AtualizarIdentidadeLoja } from "../wailsjs/go/admin/AdminApp";
import { admin } from "../wailsjs/go/models";

const TAMANHO_MAXIMO_LOGO = 1.5 * 1024 * 1024;

/**
 * Configurações da loja depois do assistente inicial: nome e logótipo. O
 * tipo de negócio não muda aqui (define módulos e tema). Sem logótipo, a
 * app mostra o quadrado com a cor do tipo de negócio, como antes.
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

  const mudou = nome.trim() !== loja.Nome || logo !== loja.Logo;

  function guardar() {
    if (!nome.trim() || !mudou || aGuardar) return;
    setAGuardar(true);
    AtualizarIdentidadeLoja(nome.trim(), logo)
      .then(onGuardada)
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-produto-form">
      <label className="admin-campo">
        Nome da loja
        <input autoFocus value={nome} onChange={(e) => setNome(e.target.value)} />
      </label>

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
              {logo ? "Trocar logótipo" : "Escolher logótipo"}
            </button>
            {logo && (
              <button className="admin-btn-secundario" onClick={() => setLogo("")}>
                Remover
              </button>
            )}
          </div>
          <input ref={ficheiroRef} type="file" accept="image/*" hidden onChange={escolherLogo} />
        </div>
        <span className="admin-texto-fraco">
          PNG ou JPG até 1,5 MB, de preferência quadrado. Aparece no menu e no ecrã de entrada.
        </span>
      </div>

      <div className="admin-form">
        <button className="admin-btn" disabled={!nome.trim() || !mudou || aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : "Guardar"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
