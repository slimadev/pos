// Formato dos números em todo o admin: vírgula decimal e espaço nos
// milhares, sempre (o pt-PT do Intl não separa os números de 4 algarismos
// — "4800" ao lado de "12 000" na mesma tabela). O espaço é inquebrável,
// para "1 234,56 MT" nunca partir em duas linhas.
const ESPACO = " ";

function formatar(n: number, minCasas: number, maxCasas: number): string {
  const texto = Math.abs(n).toLocaleString("en-US", {
    minimumFractionDigits: minCasas,
    maximumFractionDigits: maxCasas,
    useGrouping: false,
  });
  const [inteiro, decimal] = texto.split(".");
  const agrupado = inteiro.replace(/\B(?=(\d{3})+(?!\d))/g, ESPACO);
  const sinal = n < 0 && Number(texto) !== 0 ? "−" : "";
  return sinal + agrupado + (decimal ? "," + decimal : "");
}

/** Valor em meticais com dois decimais: "1 234,56 MT". */
export const mt = (n: number) => `${formatar(n, 2, 2)}${ESPACO}MT`;

/** Valor em meticais arredondado ao metical, para cartões: "1 235 MT". */
export const mtInteiro = (n: number) => `${formatar(n, 0, 0)}${ESPACO}MT`;

/** Quantidade sem zeros inúteis: "1 200", "2,5", "0,125". */
export const num = (n: number, casas = 3) => formatar(n, 0, casas);

/** Percentagem com uma casa: "35,7%". */
export const pct = (n: number) => `${formatar(n, 1, 1)}%`;

const ROTULO_TIPO_NEGOCIO: Record<string, string> = {
  ferragem: "Ferragem",
  mercearia: "Mercearia",
  farmacia: "Farmácia",
  bottlestore: "Bottlestore",
  personalizado: "Personalizado",
};

/** Nome do tipo de negócio para mostrar ("farmacia" → "Farmácia"). */
export const rotuloTipoNegocio = (tipo: string) => ROTULO_TIPO_NEGOCIO[tipo] ?? tipo;
