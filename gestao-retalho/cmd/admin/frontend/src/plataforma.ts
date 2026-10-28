// O admin também corre dentro da app Android (terminal Sunmi). A ponte JS do
// Android define `__PLATAFORMA` e o objeto `Nativo` antes de a app arrancar;
// no desktop (Wails) nenhum existe.
declare global {
  interface Window {
    __PLATAFORMA?: string;
    Nativo?: {
      abrirPDV(): void;
      abrirAdmin(): void;
      reiniciarApp(): void;
      guardarCopia(nome: string): void;
    };
  }
}

export const noAndroid = window.__PLATAFORMA === "android";

/** "Guardar em…" de uma cópia de segurança (seletor do Android). */
export function guardarCopiaEm(nome: string) {
  window.Nativo?.guardarCopia(nome);
}

/** Reinicia a app Android (ex: para aplicar o restauro de uma cópia). */
export function reiniciarApp() {
  window.Nativo?.reiniciarApp();
}
