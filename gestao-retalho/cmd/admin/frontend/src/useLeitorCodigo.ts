import { useEffect, useRef } from "react";

/**
 * Chama `aoLer(codigo)` sempre que o leitor de códigos de barras lê um
 * código. No terminal Android (Sunmi) o lado nativo emite o evento
 * `codigo-lido` — não é preciso nenhum campo com foco. No desktop o leitor
 * USB continua a "escrever" no campo focado, como sempre.
 */
export function useLeitorCodigo(aoLer: (codigo: string) => void) {
  const atual = useRef(aoLer);
  useEffect(() => {
    atual.current = aoLer;
  });
  useEffect(() => {
    function onCodigo(e: Event) {
      const codigo = String((e as CustomEvent<string>).detail ?? "").trim();
      if (codigo) atual.current(codigo);
    }
    window.addEventListener("codigo-lido", onCodigo);
    return () => window.removeEventListener("codigo-lido", onCodigo);
  }, []);
}
