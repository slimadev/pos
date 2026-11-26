// O PDV também corre dentro da app Android (terminal Sunmi), onde é o ecrã
// principal: a app abre nele e o admin fica atrás de um login. A ponte JS do
// Android define `__PLATAFORMA` e o objeto `Nativo` antes de a app arrancar;
// no desktop (Wails) nenhum existe.
declare global {
  interface Window {
    __PLATAFORMA?: string;
    Nativo?: { abrirPDV(): void; abrirAdmin(): void };
    go?: { ponte?: { Ponte?: Ponte } };
  }
}

/** Estado da ligação do aparelho ao computador do balcão (modo garçom). */
export type EstadoLigacao = {
  ModoGarcom: boolean;
  Endereco: string;
  Aparelho: string;
  TemImpressora: boolean;
};

type Ponte = {
  EstadoLigacao(): Promise<EstadoLigacao>;
  LigarAoBalcao(endereco: string, chave: string): Promise<EstadoLigacao>;
  DesligarDoBalcao(): Promise<EstadoLigacao>;
  ImprimirNoAparelho(texto: string): Promise<void>;
};

export const noAndroid = window.__PLATAFORMA === "android";

/** Abre a administração, sempre no login (só no Android). */
export function abrirAdministracao() {
  window.Nativo?.abrirAdmin();
}

/** Chamadas à app Android que não vão para a loja (só existe no Android). */
export const ponte: Ponte | null = (noAndroid && window.go?.ponte?.Ponte) || null;
