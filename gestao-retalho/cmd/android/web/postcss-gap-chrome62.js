// Plugin PostCSS: converte `gap` para algo que a WebView Chrome 62 do Sunmi
// V2 Pro entende. Só é usado no build `vite --mode android`.
//
//  - grid  → `grid-gap` / `grid-row-gap` / `grid-column-gap` (Chrome 57+).
//  - flex  → margem nos filhos, consoante a direção:
//      coluna        `.x > * + * { margin-top: G }`
//      linha         `.x > * { margin-right: G }` (0 no último, salvo ícone)
//      linha c/ wrap `.x > * { margin-right: C; margin-bottom: R }` e
//                    `.x { margin-bottom: -R }`
//
// O `gap` original é removido nos contentores flex: numa WebView mais
// recente (com flex gap) somar-se-ia às margens.
//
// A direção/display de um seletor é lida na própria regra ou, se lá não
// estiver, em qualquer outra regra com o mesmo seletor. Seletores sem
// display conhecido tratam-se como flex em linha e são listados no build.

// `grid-gap` não entra: só se aplica a grelhas e o Chrome 62 já o entende.
// Cada ficheiro CSS é processado à parte, por isso a info de display só
// vem do próprio ficheiro.
const GAPS = new Set(["gap", "row-gap", "column-gap"]);

function partirGap(valor) {
  // "12px" → [12px, 12px]; "2px 10px" → [row, column]. Respeita var(...).
  const partes = [];
  let nivel = 0;
  let atual = "";
  for (const c of valor.trim()) {
    if (c === "(") nivel++;
    if (c === ")") nivel--;
    if (/\s/.test(c) && nivel === 0) {
      if (atual) partes.push(atual);
      atual = "";
    } else atual += c;
  }
  if (atual) partes.push(atual);
  return [partes[0], partes[1] ?? partes[0]];
}

function seletores(rule) {
  return rule.selectors.map((s) => s.trim());
}

export default function gapChrome62(opcoes = {}) {
  const avisar = opcoes.avisar ?? ((m) => console.warn(m));
  return {
    postcssPlugin: "postcss-gap-chrome62",
    Once(root, { Rule, Declaration }) {
      // 1.ª passagem: display / flex-direction / flex-wrap por seletor.
      const info = new Map();
      root.walkRules((rule) => {
        if (rule.parent && rule.parent.type === "atrule" && /keyframes/.test(rule.parent.name)) return;
        rule.each((d) => {
          if (d.type !== "decl") return;
          if (!["display", "flex-direction", "flex-wrap", "flex-flow"].includes(d.prop)) return;
          for (const s of seletores(rule)) {
            const i = info.get(s) ?? {};
            // flex/grid ganham a `none`/`block` de outras regras (ex: @media
            // print), que não dizem nada sobre o gap.
            if (d.prop === "display" && (/flex|grid/.test(d.value) || !i.display)) i.display = d.value;
            if (d.prop === "flex-direction") i.direcao = d.value;
            if (d.prop === "flex-wrap") i.wrap = d.value;
            if (d.prop === "flex-flow") {
              if (/column/.test(d.value)) i.direcao = "column";
              if (/\bwrap\b/.test(d.value)) i.wrap = "wrap";
            }
            info.set(s, i);
          }
        });
      });

      const desconhecidos = [];

      // 2.ª passagem: converter.
      root.walkRules((rule) => {
        const decls = [];
        rule.each((d) => {
          if (d.type === "decl" && GAPS.has(d.prop)) decls.push(d);
        });
        if (decls.length === 0) return;

        // Junta a info de todos os seletores da regra (o primeiro que souber).
        const sels = seletores(rule);
        const i = {};
        for (const s of sels) {
          const x = info.get(s) ?? {};
          i.display ??= x.display;
          i.direcao ??= x.direcao;
          i.wrap ??= x.wrap;
        }

        if (i.display && /grid/.test(i.display)) {
          for (const d of decls) {
            d.cloneBefore({ prop: d.prop === "gap" ? "grid-gap" : "grid-" + d.prop });
          }
          return;
        }

        if (!i.display || !/flex/.test(i.display)) desconhecidos.push(sels.join(", "));

        let linha = null; // row-gap
        let coluna = null; // column-gap
        for (const d of decls) {
          if (d.prop === "gap") [linha, coluna] = partirGap(d.value);
          if (d.prop === "row-gap") linha = d.value.trim();
          if (d.prop === "column-gap") coluna = d.value.trim();
          d.remove();
        }

        const vertical = /column/.test(i.direcao ?? "");
        const wrap = /^wrap/.test(i.wrap ?? "");
        const filhos = (sufixo) => sels.map((s) => `${s}${sufixo}`).join(",\n");

        let nova;
        if (vertical) {
          if (!linha) return;
          nova = new Rule({ selector: filhos(" > * + *") });
          nova.append(new Declaration({ prop: "margin-top", value: linha }));
        } else if (wrap) {
          nova = new Rule({ selector: filhos(" > *") });
          if (coluna) {
            nova.append(new Declaration({ prop: "margin-right", value: coluna }));
            // Compensa a margem do último de cada linha (fica dentro do
            // padding do pai, sem scroll horizontal).
            rule.append(new Declaration({ prop: "margin-right", value: `calc(-1 * ${coluna})` }));
          }
          if (linha) {
            nova.append(new Declaration({ prop: "margin-bottom", value: linha }));
            rule.append(new Declaration({ prop: "margin-bottom", value: `calc(-1 * ${linha})` }));
          }
        } else {
          // Margem à direita (não `* + *` à esquerda): no padrão comum
          // `<svg/> Texto` o texto é um nó de texto que o CSS não seleciona,
          // e o ícone é o único elemento. O último filho perde a margem,
          // exceto se for um ícone (o texto pode vir depois dele).
          if (!coluna) return;
          nova = new Rule({ selector: filhos(" > *") });
          nova.append(new Declaration({ prop: "margin-right", value: coluna }));
          const ultimo = new Rule({ selector: filhos(" > :last-child:not(svg)") });
          ultimo.append(new Declaration({ prop: "margin-right", value: "0" }));
          rule.after(ultimo);
        }
        if (nova.nodes.length) rule.after(nova);
      });

      if (desconhecidos.length && opcoes.relatorio !== false) {
        avisar(
          `[gap-chrome62] ${desconhecidos.length} regra(s) com gap sem display conhecido (tratadas como flex em linha):\n  ` +
            desconhecidos.join("\n  "),
        );
      }
    },
  };
}
gapChrome62.postcss = true;
