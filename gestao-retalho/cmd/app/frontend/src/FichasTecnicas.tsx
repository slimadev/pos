import { useState } from "react";
import {
  ListarIngredientes,
  DefinirReceita,
  DuplicarReceita,
  CalcularCustoTeorico,
  SimularDisponibilidade,
  RegistarProducao,
} from "../wailsjs/go/api/App";
import { domain, usecase } from "../wailsjs/go/models";

// Rótulo amigável para o valor interno de TipoProduto — ver a mesma
// constante em Administracao.tsx.
const ROTULO_TIPO_PRODUTO: Record<string, string> = {
  simples: "Simples",
  composto: "Composto",
  semi_elaborado: "Semi Produzido",
};

/**
 * Fichas técnicas / produtos compostos e semi-elaborados (SDD 3.3, Fase 5)
 * — mesma família de scaffolding funcional do resto de FaseUm.tsx, sem
 * tema/polimento: editor de receita, custo teórico + simulador de
 * disponibilidade, e produção em lote para semi-elaborados.
 */
export default function FichasTecnicasSection({
  produtos,
  armazens,
  onErro,
}: {
  produtos: domain.Produto[];
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const compostosEnSemiElaborados = produtos.filter(
    (p) => p.TipoProduto === "composto" || p.TipoProduto === "semi_elaborado",
  );
  const ingredientesPossiveis = produtos.filter(
    (p) => p.TipoProduto === "simples" || p.TipoProduto === "semi_elaborado",
  );

  const [produtoId, setProdutoId] = useState<number | "">("");
  const [armazemIdEscolhido, setArmazemId] = useState<number | "">("");
  const [linhas, setLinhas] = useState<{ ingredienteId: number | ""; quantidade: string; perda: string }[]>([]);
  const [custoTeorico, setCustoTeorico] = useState<number | null>(null);
  const [disponibilidade, setDisponibilidade] = useState<number | null>(null);
  const [produtoDuplicarDestino, setProdutoDuplicarDestino] = useState<number | "">("");

  const [producaoQtd, setProducaoQtd] = useState("1");

  const produtoSelecionado = produtos.find((p) => p.ID === produtoId);
  // Sem useEffect: assumimos o primeiro armazém carregado até o operador
  // escolher outro — evita disparar um setState extra assim que a lista
  // de armazéns (prop) chega do FaseUm.
  const armazemId: number | "" = armazemIdEscolhido !== "" ? armazemIdEscolhido : armazens.length > 0 ? armazens[0].ID : "";

  function carregarReceita(id: number) {
    setProdutoId(id);
    setCustoTeorico(null);
    setDisponibilidade(null);
    ListarIngredientes(id).then((ingredientes) => {
      setLinhas(
        ingredientes.map((i) => ({
          ingredienteId: i.ProdutoIngredienteID,
          quantidade: String(i.Quantidade),
          perda: String(i.PerdaPercentual),
        })),
      );
    });
  }

  function adicionarLinha() {
    setLinhas((prev) => [...prev, { ingredienteId: "", quantidade: "", perda: "0" }]);
  }

  function atualizarLinha(i: number, campo: "ingredienteId" | "quantidade" | "perda", valor: string) {
    setLinhas((prev) =>
      prev.map((l, idx) =>
        idx === i ? { ...l, [campo]: campo === "ingredienteId" ? (valor ? Number(valor) : "") : valor } : l,
      ),
    );
  }

  function removerLinha(i: number) {
    setLinhas((prev) => prev.filter((_, idx) => idx !== i));
  }

  function guardarReceita() {
    if (produtoId === "") return;
    const ingredientes = linhas
      .filter((l) => l.ingredienteId !== "" && Number(l.quantidade) > 0)
      .map((l) => {
        const ing = new usecase.IngredienteInput();
        ing.ProdutoIngredienteID = Number(l.ingredienteId);
        ing.Quantidade = Number(l.quantidade);
        ing.PerdaPercentual = Number(l.perda || 0);
        return ing;
      });
    if (ingredientes.length === 0) {
      onErro("Receita precisa de pelo menos um ingrediente");
      return;
    }
    DefinirReceita(produtoId, ingredientes)
      .then(() => carregarReceita(produtoId))
      .catch((e) => onErro(String(e)));
  }

  function calcularCusto() {
    if (produtoId === "" || armazemId === "") return;
    CalcularCustoTeorico(produtoId, armazemId).then(setCustoTeorico).catch((e) => onErro(String(e)));
    SimularDisponibilidade(produtoId, armazemId).then(setDisponibilidade).catch((e) => onErro(String(e)));
  }

  function duplicar() {
    if (produtoId === "" || produtoDuplicarDestino === "") return;
    DuplicarReceita(produtoId, Number(produtoDuplicarDestino))
      .then(() => onErro(""))
      .catch((e) => onErro(String(e)));
  }

  function produzir() {
    if (produtoId === "" || armazemId === "") return;
    const p = new usecase.ProducaoParams();
    p.ProdutoSemiElaboradoID = produtoId;
    p.ArmazemID = armazemId;
    p.Quantidade = Number(producaoQtd);
    p.UtilizadorID = 0;
    RegistarProducao(p)
      .then((mov) => onErro(`Produção registada: +${mov.QuantidadeUnidadeBase} no armazém`))
      .catch((e) => onErro(String(e)));
  }

  return (
    <section>
      <h2>Fichas técnicas</h2>

      <div className="fase1-form">
        <select value={produtoId} onChange={(e) => (e.target.value ? carregarReceita(Number(e.target.value)) : setProdutoId(""))}>
          <option value="">Produto composto/semi-elaborado…</option>
          {compostosEnSemiElaborados.map((p) => (
            <option key={p.ID} value={p.ID}>
              {p.Nome} ({ROTULO_TIPO_PRODUTO[p.TipoProduto] ?? p.TipoProduto})
            </option>
          ))}
        </select>
        <select value={armazemId} onChange={(e) => setArmazemId(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Armazém…</option>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
      </div>

      {produtoId !== "" && (
        <>
          <table>
            <thead>
              <tr>
                <th>Ingrediente</th>
                <th>Quantidade (unidade base)</th>
                <th>Perda %</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {linhas.map((l, i) => (
                <tr key={i}>
                  <td>
                    <select value={l.ingredienteId} onChange={(e) => atualizarLinha(i, "ingredienteId", e.target.value)}>
                      <option value="">Ingrediente…</option>
                      {ingredientesPossiveis
                        .filter((p) => p.ID !== produtoId)
                        .map((p) => (
                          <option key={p.ID} value={p.ID}>
                            {p.Nome} ({p.UnidadeBase})
                          </option>
                        ))}
                    </select>
                  </td>
                  <td>
                    <input type="number" value={l.quantidade} onChange={(e) => atualizarLinha(i, "quantidade", e.target.value)} />
                  </td>
                  <td>
                    <input type="number" value={l.perda} onChange={(e) => atualizarLinha(i, "perda", e.target.value)} />
                  </td>
                  <td>
                    <button onClick={() => removerLinha(i)}>Remover</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="fase1-form">
            <button onClick={adicionarLinha}>+ ingrediente</button>
            <button onClick={guardarReceita}>Guardar receita</button>
            <button onClick={calcularCusto}>Calcular custo / disponibilidade</button>
          </div>

          {custoTeorico !== null && (
            <p className="fase1-stock-atual">
              Custo teórico: <strong>{custoTeorico.toFixed(2)}</strong>
              {produtoSelecionado && (
                <>
                  {" "}
                  — margem sobre 1ª unidade de venda: ver preço no produto (custo/preço não mostrado automaticamente aqui)
                </>
              )}
              {disponibilidade !== null && (
                <>
                  {" "}
                  · dá para preparar <strong>{disponibilidade}</strong> unidade(s) com o stock atual
                </>
              )}
            </p>
          )}

          <div className="fase1-form">
            <select value={produtoDuplicarDestino} onChange={(e) => setProdutoDuplicarDestino(e.target.value ? Number(e.target.value) : "")}>
              <option value="">Duplicar receita para…</option>
              {compostosEnSemiElaborados
                .filter((p) => p.ID !== produtoId)
                .map((p) => (
                  <option key={p.ID} value={p.ID}>
                    {p.Nome}
                  </option>
                ))}
            </select>
            <button disabled={produtoDuplicarDestino === ""} onClick={duplicar}>
              Duplicar
            </button>
          </div>

          {produtoSelecionado?.TipoProduto === "semi_elaborado" && (
            <div className="fase1-form">
              <input placeholder="Quantidade a produzir" type="number" value={producaoQtd} onChange={(e) => setProducaoQtd(e.target.value)} />
              <button onClick={produzir}>Registar produção em lote</button>
            </div>
          )}
        </>
      )}
    </section>
  );
}
