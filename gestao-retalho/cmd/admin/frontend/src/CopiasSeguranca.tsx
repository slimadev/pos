import { useEffect, useState } from "react";
import {
  ShieldCheckIcon,
  ExclamationTriangleIcon,
  ArrowUpTrayIcon,
  ShareIcon,
  ArrowPathIcon,
  ArrowUturnLeftIcon,
  ArrowDownTrayIcon,
} from "@heroicons/react/24/outline";
import {
  EstadoCopiasSeguranca,
  CriarCopiaSeguranca,
  PartilharCopiaSeguranca,
  EscolherCopiaParaRestaurar,
  PrepararRestauroCopia,
} from "../wailsjs/go/admin/AdminApp";
import { admin } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import { noAndroid, reiniciarApp, guardarCopiaEm } from "./plataforma";

const dataHora = (d: unknown) =>
  new Date(String(d)).toLocaleString("pt-PT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });

function tamanho(bytes: number) {
  if (bytes < 1024 * 1024) return `${Math.max(1, Math.round(bytes / 1024))} KB`;
  return `${(bytes / 1024 / 1024).toFixed(1).replace(".", ",")} MB`;
}

const limpar = (e: unknown) => String(e).replace(/^Error:\s*/, "");

/**
 * Cópias de segurança de todos os dados (loja, administrador, configuração).
 * No terminal Android os dados existem só lá: a cópia tem de sair dele —
 * criar abre logo a partilha (Drive, WhatsApp, email…). O restauro troca os
 * dados no próximo arranque da aplicação.
 */
export default function CopiasSegurancaScreen({ onErro }: { onErro: (e: string) => void }) {
  const [estado, setEstado] = useState<admin.EstadoCopias | null>(null);
  const [aCriar, setACriar] = useState(false);
  const [aviso, setAviso] = useState<string | null>(null);

  function carregar() {
    EstadoCopiasSeguranca()
      .then(setEstado)
      .catch((e) => onErro(limpar(e)));
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- carga inicial
  }, []);

  function criar() {
    setACriar(true);
    setAviso(null);
    CriarCopiaSeguranca()
      .then((c) => {
        carregar();
        setAviso(
          noAndroid
            ? "Cópia criada. Guarde-a fora do terminal: escolha Drive, WhatsApp ou email na janela que abriu."
            : `Cópia criada: ${c.Caminho}`,
        );
        // No terminal a cópia só protege se sair dele.
        if (noAndroid) return PartilharCopiaSeguranca(c.Nome);
      })
      .catch((e) => onErro(limpar(e)))
      .finally(() => setACriar(false));
  }

  function partilhar(nome: string) {
    PartilharCopiaSeguranca(nome).catch((e) => onErro(limpar(e)));
  }

  async function restaurar() {
    try {
      const caminho = await EscolherCopiaParaRestaurar();
      if (!caminho) return;
      if (
        !window.confirm(
          "Restaurar esta cópia substitui TODOS os dados atuais (produtos, stock, vendas, clientes) pelos da cópia. Continuar?",
        )
      )
        return;
      if (
        !window.confirm(
          "Tem a certeza? Os dados atuais ficam guardados à parte, mas deixam de estar em uso.",
        )
      )
        return;
      const m = await PrepararRestauroCopia(caminho);
      carregar();
      setAviso(
        `Cópia de ${dataHora(m.CriadaEm)} pronta a restaurar. ` +
          (noAndroid
            ? "A aplicação vai reiniciar."
            : "Feche e volte a abrir a administração para concluir."),
      );
      if (noAndroid) setTimeout(reiniciarApp, 1500);
    } catch (e) {
      onErro(limpar(e));
    }
  }

  const dias = estado?.DiasDesdeUltima ?? -1;
  const atrasada = dias < 0 || dias >= 7;

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Cópias de segurança" }]} />
      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Cópias de segurança</h2>
      </div>

      {aviso && <div className="admin-aviso-ok">{aviso}</div>}

      <div className={`admin-card admin-copias-estado${atrasada ? " atrasada" : ""}`}>
        {atrasada ? (
          <ExclamationTriangleIcon className="admin-copias-estado-icone" aria-hidden="true" />
        ) : (
          <ShieldCheckIcon className="admin-copias-estado-icone" aria-hidden="true" />
        )}
        <div className="admin-copias-estado-texto">
          <strong>
            {estado === null
              ? "A verificar…"
              : dias < 0
                ? "Ainda não fez nenhuma cópia de segurança"
                : dias === 0
                  ? "Última cópia: hoje"
                  : `Última cópia: há ${dias} dia${dias === 1 ? "" : "s"}`}
          </strong>
          <span>
            {noAndroid
              ? "Os dados estão só neste terminal. Faça uma cópia pelo menos uma vez por semana e guarde-a fora dele."
              : "Faça uma cópia pelo menos uma vez por semana e guarde-a fora deste computador (pen, Drive)."}
          </span>
        </div>
        <button className="admin-btn" disabled={aCriar} onClick={criar}>
          {aCriar ? (
            <ArrowPathIcon className="btn-icone a-girar" aria-hidden="true" />
          ) : (
            <ArrowUpTrayIcon className="btn-icone" aria-hidden="true" />
          )}
          {aCriar ? "A criar…" : "Criar cópia agora"}
        </button>
      </div>

      {estado?.RestauroPendente && (
        <div className="admin-inventario-aviso warn">
          Há uma cópia pronta a restaurar — reinicie a aplicação para a aplicar.
        </div>
      )}

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Cópia</th>
            <th>Tamanho</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {(estado?.Copias ?? []).map((c) => (
            <tr key={c.Nome}>
              <td>{dataHora(c.CriadaEm)}</td>
              <td>{tamanho(c.Tamanho)}</td>
              <td style={{ textAlign: "right" }}>
                {noAndroid && (
                  <button className="admin-btn-secundario" onClick={() => guardarCopiaEm(c.Nome)}>
                    <ArrowDownTrayIcon className="btn-icone" aria-hidden="true" />
                    Guardar em…
                  </button>
                )}{" "}
                <button className="admin-btn-secundario" onClick={() => partilhar(c.Nome)}>
                  <ShareIcon className="btn-icone" aria-hidden="true" />
                  {noAndroid ? "Partilhar" : "Abrir"}
                </button>
              </td>
            </tr>
          ))}
          {estado !== null && estado.Copias.length === 0 && (
            <tr>
              <td colSpan={3} style={{ color: "var(--text-secondary)" }}>
                Nenhuma cópia guardada nesta máquina.
              </td>
            </tr>
          )}
        </tbody>
      </table>

      <div className="admin-card admin-copias-restaurar">
        <div>
          <strong>Restaurar uma cópia</strong>
          <span>
            Substitui todos os dados atuais pelos de uma cópia (ex: num terminal novo ou depois de
            uma avaria). Os dados atuais ficam guardados à parte.
          </span>
        </div>
        <button className="admin-btn-secundario" onClick={restaurar}>
          <ArrowUturnLeftIcon className="btn-icone" aria-hidden="true" />
          Restaurar de um ficheiro
        </button>
      </div>
    </div>
  );
}
