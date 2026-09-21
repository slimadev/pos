import { useEffect } from "react";
import type { ReactNode } from "react";
import { XMarkIcon } from "@heroicons/react/24/outline";

/**
 * Modal genérico (overlay + cartão centrado) — primeiro uso em
 * Produtos.tsx (criar/editar), reutilizável por outros ecrãs. Fecha com
 * Escape ou o botão ×; não fecha ao clicar no fundo, para não perder um
 * formulário grande por engano com um clique a mais.
 */
export default function Modal({ titulo, onFechar, children }: { titulo: string; onFechar: () => void; children: ReactNode }) {
  useEffect(() => {
    function aoTeclar(e: KeyboardEvent) {
      if (e.key === "Escape") onFechar();
    }
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [onFechar]);

  return (
    <div className="admin-modal-overlay">
      <div className="admin-modal-card">
        <div className="admin-modal-header">
          <h2 className="admin-modal-titulo">{titulo}</h2>
          <button className="admin-modal-fechar" onClick={onFechar} aria-label="Fechar">
            <XMarkIcon className="admin-modal-fechar-icon" aria-hidden="true" />
          </button>
        </div>
        <div className="admin-modal-body">{children}</div>
      </div>
    </div>
  );
}
