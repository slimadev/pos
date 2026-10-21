// Formatação partilhada pelos ecrãs de venda: valores como em pt
// ("1 234,56 MT") e métodos de pagamento com o nome que o cliente conhece.

// Espaço nos milhares mesmo com 4 algarismos (o pt-PT do Intl não o põe:
// "3510" ao lado de "13 260") e inquebrável, para "MT" não mudar de linha.
const ESPACO = "\u00a0";
function formatar(n: number, minCasas: number, maxCasas: number): string {
  const texto = Math.abs(n).toLocaleString("en-US", {
    minimumFractionDigits: minCasas,
    maximumFractionDigits: maxCasas,
    useGrouping: false,
  });
  const [inteiro, decimal] = texto.split(".");
  const sinal = n < 0 && Number(texto) !== 0 ? "−" : "";
  return sinal + inteiro.replace(/\B(?=(\d{3})+(?!\d))/g, ESPACO) + (decimal ? "," + decimal : "");
}

export const mt = (n: number) => `${formatar(n, 2, 2)}${ESPACO}MT`;

export const qtd = (n: number, casas = 3) => formatar(n, 0, casas);

export const METODOS = ["dinheiro", "mpesa", "emola", "cartao", "transferencia"] as const;

export const ROTULO_METODO: Record<string, string> = {
  dinheiro: "Dinheiro",
  mpesa: "M-Pesa",
  emola: "e-Mola",
  cartao: "Cartão",
  transferencia: "Transferência",
};

// Aceita "12,5" e "12.5" — no teclado português a vírgula é o separador.
export const lerNumero = (s: string) => Number(s.trim().replace(",", "."));

// Sem acentos nem maiúsculas: "paracetamol" encontra "Paracetamól".
export const normalizar = (s: string) =>
  s
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .toLowerCase()
    .trim();
