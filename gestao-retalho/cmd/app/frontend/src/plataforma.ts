// O PDV também corre dentro da app Android (terminal Sunmi), onde é o ecrã
// principal: a app abre nele e o admin fica atrás de um login. A ponte JS do
// Android define `__PLATAFORMA` e o objeto `Nativo` antes de a app arrancar;
// no desktop (Wails) nenhum existe.
declare global {
  interface Window {
    __PLATAFORMA?: string;
    Nativo?: { abrirPDV(): void; abrirAdmin(): void };
  }
}

export const noAndroid = window.__PLATAFORMA === "android";

/** Abre a administração, sempre no login (só no Android). */
export function abrirAdministracao() {
  window.Nativo?.abrirAdmin();
}
