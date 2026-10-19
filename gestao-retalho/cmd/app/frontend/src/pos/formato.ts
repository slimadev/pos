// Formatação partilhada pelos ecrãs de venda: valores como em pt
// ("1 234,56 MT") e métodos de pagamento com o nome que o cliente conhece.

export const mt = (n: number) =>
  `${n.toLocaleString("pt-PT", { minimumFractionDigits: 2, maximumFractionDigits: 2 })} MT`;

export const qtd = (n: number) => n.toLocaleString("pt-PT", { maximumFractionDigits: 3 });

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
