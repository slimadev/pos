// Utilitários dos ecrãs de contas e mesas.

/** Há quanto tempo a conta está aberta ("2h 14m"); em aviso a partir das 3h. */
export function tempoAberta(abertaEm: unknown): { texto: string; longa: boolean } {
  const inicio = new Date(String(abertaEm)).getTime();
  if (!inicio) return { texto: "", longa: false };
  const minutos = Math.max(0, Math.round((Date.now() - inicio) / 60000));
  const horas = Math.floor(minutos / 60);
  return {
    texto: horas > 0 ? `${horas}h ${String(minutos % 60).padStart(2, "0")}m` : `${minutos}m`,
    longa: horas >= 3,
  };
}

export const textoErro = (e: unknown) => String(e).replace(/^Error:\s*/, "");
