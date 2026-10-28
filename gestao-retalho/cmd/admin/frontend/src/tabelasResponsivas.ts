// Em ecrãs estreitos (terminal Android) cada linha de `.admin-tabela` passa
// a cartão (ver mobile.css), e cada célula mostra o título da sua coluna.
// Em vez de repetir esse título em todas as tabelas, copia-se aqui o texto
// do <th> para `data-label` de cada <td>; as células só com botões ficam
// marcadas como `td-acoes` (vão para o fundo do cartão). No desktop os
// atributos existem mas não têm efeito visível.

// Células com campos (input/select) e título de coluna ficam `td-campo`: o
// rótulo mantém-se, senão um formulário em tabela (unidades do produto,
// linhas da receção) aparecia só com campos sem nome. O título pode vir de
// `data-rotulo` no <th> quando o do desktop é curto demais (ex: "Venda").
function rotular(tabela: HTMLTableElement) {
  const titulos = Array.from(tabela.querySelectorAll<HTMLElement>("thead th")).map(
    (th) => th.dataset.rotulo ?? th.textContent?.trim() ?? "",
  );
  if (titulos.length === 0) return;
  tabela.querySelectorAll("tbody tr, tfoot tr").forEach((tr) => {
    const celulas = Array.from((tr as HTMLTableRowElement).cells);
    // Linhas com colSpan (ex: "Sem produtos…") não correspondem às colunas.
    if (celulas.some((c) => c.colSpan > 1)) return;
    celulas.forEach((td, i) => {
      const titulo = titulos[i] ?? "";
      if (titulo && td.getAttribute("data-label") !== titulo) td.setAttribute("data-label", titulo);
      const temCampo = Boolean(td.querySelector("input, select, textarea"));
      const temBotao = Boolean(td.querySelector("button"));
      const texto = td.textContent?.trim() ?? "";
      // Ações: sem título de coluna (botões/seletores soltos), ou só botões.
      const acoes = titulo ? !texto && temBotao && !temCampo : temBotao || temCampo;
      td.classList.toggle("td-acoes", acoes);
      td.classList.toggle("td-campo", temCampo && !acoes);
    });
  });
}

/** Observa o documento e rotula as tabelas sempre que o React as altera. */
export function ativarTabelasResponsivas() {
  let pendente = false;
  const correr = () => {
    pendente = false;
    document.querySelectorAll<HTMLTableElement>("table.admin-tabela").forEach(rotular);
  };
  new MutationObserver(() => {
    if (pendente) return;
    pendente = true;
    requestAnimationFrame(correr);
  }).observe(document.body, { childList: true, subtree: true });
  correr();
}
