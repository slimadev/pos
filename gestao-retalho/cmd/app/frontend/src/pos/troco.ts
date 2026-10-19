import { lerNumero } from "./formato";

export type LinhaPagamentoTexto = { metodo: string; valor: string };

export type ResultadoTroco = {
  // Quanto dar de volta ao cliente (só sai do dinheiro entregue).
  troco: number;
  // Pago a mais que o dinheiro não cobre (ex: M-Pesa acima do total) —
  // não há troco de uma transferência, o operador tem de corrigir.
  excessoSemDinheiro: number;
  // O que se grava na venda: o dinheiro já sem o troco, para a venda e o
  // fecho de caixa baterem com o que fica na gaveta.
  liquidos: { metodo: string; valor: number }[];
};

const arred = (n: number) => Math.round(n * 100) / 100;

/**
 * Troco de uma venda: se o cliente entrega mais dinheiro do que o total,
 * a diferença é troco e o pagamento em dinheiro grava-se só pelo que fica
 * (ex: total 650, entregou 1000 em dinheiro → troco 350, grava 650). O
 * backend recusa pagamentos acima do total, por isso o troco nunca lhe
 * chega.
 */
export function calcularTroco(total: number, pagamentos: LinhaPagamentoTexto[]): ResultadoTroco {
  const valores = pagamentos.map((p) => ({ metodo: p.metodo, valor: lerNumero(p.valor || "0") || 0 }));
  const soma = valores.reduce((s, p) => s + p.valor, 0);
  const excesso = arred(soma - total);
  const dinheiro = valores.filter((p) => p.metodo === "dinheiro").reduce((s, p) => s + p.valor, 0);

  if (excesso <= 0.005) {
    return { troco: 0, excessoSemDinheiro: 0, liquidos: valores.filter((p) => p.valor > 0) };
  }
  if (excesso > dinheiro + 0.005) {
    return { troco: 0, excessoSemDinheiro: arred(excesso - dinheiro), liquidos: valores.filter((p) => p.valor > 0) };
  }

  // Tira o troco das linhas de dinheiro, da última para a primeira.
  let porTirar = excesso;
  const liquidos = [...valores];
  for (let i = liquidos.length - 1; i >= 0 && porTirar > 0.005; i--) {
    if (liquidos[i].metodo !== "dinheiro") continue;
    const tirar = Math.min(liquidos[i].valor, porTirar);
    liquidos[i] = { ...liquidos[i], valor: arred(liquidos[i].valor - tirar) };
    porTirar = arred(porTirar - tirar);
  }
  return { troco: excesso, excessoSemDinheiro: 0, liquidos: liquidos.filter((p) => p.valor > 0) };
}
