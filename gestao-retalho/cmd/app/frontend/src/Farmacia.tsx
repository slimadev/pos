import { useState } from "react";
import {
  RegistarEntradaComLote,
  ListarLotes,
  ListarLotesAVencer,
  RegistarReceitaMedica,
} from "../wailsjs/go/api/App";
import { domain, usecase } from "../wailsjs/go/models";

/**
 * Farmácia — lotes/validade (FEFO) e receitas médicas (SDD 9.4, Fase 6).
 * Mesma família de scaffolding funcional do resto de FaseUm.tsx: sem tema
 * nem polimento, só o suficiente para exercitar os bindings.
 */
export default function FarmaciaSection({
  produtos,
  armazens,
  onErro,
}: {
  produtos: domain.Produto[];
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const produtosLoteControlado = produtos.filter((p) => p.LoteControlado);

  const [produtoId, setProdutoId] = useState<number | "">("");
  const [armazemId, setArmazemId] = useState<number | "">(armazens[0]?.ID ?? "");
  const [numeroLote, setNumeroLote] = useState("");
  const [validade, setValidade] = useState("");
  const [quantidade, setQuantidade] = useState("");
  const [custo, setCusto] = useState("");
  const [lotes, setLotes] = useState<domain.Lote[]>([]);

  const [dias, setDias] = useState("30");
  const [aVencer, setAVencer] = useState<domain.Lote[]>([]);

  const [vendaIdReceita, setVendaIdReceita] = useState("");
  const [clienteIdReceita, setClienteIdReceita] = useState("");
  const [medico, setMedico] = useState("");
  const [numeroReceita, setNumeroReceita] = useState("");

  function carregarLotes(pid: number, aid: number) {
    ListarLotes(pid, aid).then(setLotes).catch((e) => onErro(String(e)));
  }

  function registarEntrada() {
    if (produtoId === "" || armazemId === "" || !numeroLote || !validade) return;
    const p = new usecase.EntradaComLoteParams();
    p.ProdutoID = produtoId;
    p.ArmazemID = armazemId;
    p.NumeroLote = numeroLote;
    // Go time.Time espera RFC3339 — o input type="date" já dá "AAAA-MM-DD".
    p.DataValidade = new Date(validade + "T00:00:00Z").toISOString();
    p.Quantidade = Number(quantidade);
    p.CustoUnitario = Number(custo);
    p.UtilizadorID = 0;
    RegistarEntradaComLote(p)
      .then(() => {
        setNumeroLote("");
        setQuantidade("");
        setCusto("");
        carregarLotes(produtoId, Number(armazemId));
      })
      .catch((e) => onErro(String(e)));
  }

  function consultarAVencer() {
    ListarLotesAVencer(Number(dias)).then(setAVencer).catch((e) => onErro(String(e)));
  }

  function registarReceita() {
    if (!vendaIdReceita || !medico || !numeroReceita) return;
    const p = new usecase.ReceitaMedicaInput();
    p.VendaID = Number(vendaIdReceita);
    // domain.ClienteAnonimoID no backend é sempre 1 (registo fixo semeado
    // pela migration de vendas) — não é uma constante exportada para TS.
    p.ClienteID = clienteIdReceita ? Number(clienteIdReceita) : 1;
    p.Medico = medico;
    p.NumeroReceita = numeroReceita;
    RegistarReceitaMedica(p)
      .then(() => {
        setMedico("");
        setNumeroReceita("");
      })
      .catch((e) => onErro(String(e)));
  }

  return (
    <section>
      <h2>Farmácia — lotes e validade</h2>

      <div className="fase1-form">
        <select
          value={produtoId}
          onChange={(e) => {
            const id = e.target.value ? Number(e.target.value) : "";
            setProdutoId(id);
            if (id !== "" && armazemId !== "") carregarLotes(id, Number(armazemId));
          }}
        >
          <option value="">Produto lote-controlado…</option>
          {produtosLoteControlado.map((p) => (
            <option key={p.ID} value={p.ID}>
              {p.Nome}
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
                <th>Lote</th>
                <th>Validade</th>
                <th>Quantidade</th>
                <th>Custo unitário</th>
              </tr>
            </thead>
            <tbody>
              {lotes.map((l) => (
                <tr key={l.ID}>
                  <td>{l.NumeroLote}</td>
                  <td>{String(l.DataValidade).slice(0, 10)}</td>
                  <td>{l.Quantidade}</td>
                  <td>{l.CustoUnitario}</td>
                </tr>
              ))}
            </tbody>
          </table>

          <div className="fase1-form">
            <input placeholder="Número de lote" value={numeroLote} onChange={(e) => setNumeroLote(e.target.value)} />
            <input type="date" value={validade} onChange={(e) => setValidade(e.target.value)} />
            <input placeholder="Quantidade" type="number" value={quantidade} onChange={(e) => setQuantidade(e.target.value)} />
            <input placeholder="Custo unitário" type="number" value={custo} onChange={(e) => setCusto(e.target.value)} />
            <button onClick={registarEntrada}>Registar entrada com lote</button>
          </div>
        </>
      )}

      <div className="fase1-form">
        <input placeholder="Dias até vencer" type="number" value={dias} onChange={(e) => setDias(e.target.value)} />
        <button onClick={consultarAVencer}>Consultar lotes a vencer</button>
      </div>
      {aVencer.length > 0 && (
        <table>
          <thead>
            <tr>
              <th>Produto</th>
              <th>Lote</th>
              <th>Validade</th>
              <th>Quantidade</th>
            </tr>
          </thead>
          <tbody>
            {aVencer.map((l) => (
              <tr key={l.ID}>
                <td>{produtos.find((p) => p.ID === l.ProdutoID)?.Nome ?? l.ProdutoID}</td>
                <td>{l.NumeroLote}</td>
                <td>{String(l.DataValidade).slice(0, 10)}</td>
                <td>{l.Quantidade}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      <h3>Receita médica</h3>
      <div className="fase1-form">
        <input placeholder="ID da venda" value={vendaIdReceita} onChange={(e) => setVendaIdReceita(e.target.value)} />
        <input placeholder="ID do cliente (vazio = anónimo)" value={clienteIdReceita} onChange={(e) => setClienteIdReceita(e.target.value)} />
        <input placeholder="Médico" value={medico} onChange={(e) => setMedico(e.target.value)} />
        <input placeholder="Número da receita" value={numeroReceita} onChange={(e) => setNumeroReceita(e.target.value)} />
        <button onClick={registarReceita}>Registar receita</button>
      </div>
    </section>
  );
}
