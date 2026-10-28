import { useEffect, useState } from "react";
import { MagnifyingGlassIcon, TrashIcon, XMarkIcon } from "@heroicons/react/24/outline";
import { ListarTodasUnidades, RegistarRececaoMercadoria } from "../wailsjs/go/admin/AdminApp";
import { useLeitorCodigo } from "./useLeitorCodigo";
import { domain, usecase } from "../wailsjs/go/models";
import { mt, num } from "./formato";

const lerNumero = (s: string) => Number(s.trim().replace(",", "."));

type Linha = { chave: number; produto: domain.Produto; unidadeID: number; quantidade: string; custo: string };
let proximaChave = 0;

/**
 * Receção de mercadoria: a guia/fatura do fornecedor inteira de uma vez,
 * na unidade em que se compra (ex: 3 caixas de 100). Tudo entra numa só
 * transação no armazém escolhido no ecrã de Stock. Produtos com lotes
 * ficam de fora — entram pela "Entrada em lote", com lote e validade.
 */
export default function Rececao({
  armazem,
  produtos,
  onConcluido,
  onCancelar,
  onErro,
}: {
  armazem: domain.Armazem;
  produtos: domain.Produto[];
  onConcluido: (n: number) => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [unidades, setUnidades] = useState<domain.UnidadeProduto[]>([]);
  const [referencia, setReferencia] = useState("");
  const [pesquisa, setPesquisa] = useState("");
  const [linhas, setLinhas] = useState<Linha[]>([]);
  const [aGuardar, setAGuardar] = useState(false);

  useEffect(() => {
    ListarTodasUnidades()
      .then(setUnidades)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const unidadesDe = (id: number) => unidades.filter((u) => u.ProdutoID === id);
  const elegiveis = produtos.filter((p) => p.Ativo && p.ControlaStock && !p.LoteControlado);
  const comLotes = produtos.some((p) => p.Ativo && p.LoteControlado);

  const termo = pesquisa.trim().toLowerCase();
  const sugestoes = termo
    ? elegiveis
        .filter(
          (p) =>
            p.Nome.toLowerCase().includes(termo) ||
            unidadesDe(p.ID).some((u) => (u.CodigoBarras ?? "").toLowerCase() === termo),
        )
        .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"))
        .slice(0, 8)
    : [];

  function adicionar(p: domain.Produto, porCodigo?: domain.UnidadeProduto) {
    const us = unidadesDe(p.ID);
    const u = porCodigo ?? us.find((x) => x.IsPadraoCompra) ?? us[0];
    if (!u) {
      onErro(`"${p.Nome}" não tem unidades — acrescente uma em Produtos.`);
      return;
    }
    setLinhas((ls) => [
      ...ls,
      { chave: proximaChave++, produto: p, unidadeID: u.ID, quantidade: "1", custo: u.PrecoCusto ? String(u.PrecoCusto) : "" },
    ]);
    setPesquisa("");
  }

  // Leitor do terminal (sem campo focado): o código lido acrescenta a linha.
  useLeitorCodigo((codigo) => {
    const u = unidades.find((x) => (x.CodigoBarras ?? "").toLowerCase() === codigo.toLowerCase());
    const p = u && elegiveis.find((x) => x.ID === u.ProdutoID);
    if (p && u) adicionar(p, u);
    else onErro(`O código ${codigo} não corresponde a nenhum produto com controlo de stock.`);
  });

  // Enter com um código de barras exato (leitor) acrescenta logo a linha.
  function aoTeclar(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key !== "Enter") return;
    const u = unidades.find((x) => (x.CodigoBarras ?? "").toLowerCase() === termo);
    const p = u && elegiveis.find((x) => x.ID === u.ProdutoID);
    if (p && u) adicionar(p, u);
    else if (sugestoes.length === 1) adicionar(sugestoes[0]);
  }

  function mudar(chave: number, campo: "unidadeID" | "quantidade" | "custo", valor: string | number) {
    setLinhas((ls) =>
      ls.map((l) => {
        if (l.chave !== chave) return l;
        if (campo !== "unidadeID") return { ...l, [campo]: valor };
        // Trocar de unidade sugere o custo dessa unidade (se o tiver).
        const u = unidades.find((x) => x.ID === valor);
        return { ...l, unidadeID: Number(valor), custo: u?.PrecoCusto ? String(u.PrecoCusto) : l.custo };
      }),
    );
  }

  const valida = (l: Linha) => {
    const q = lerNumero(l.quantidade);
    const c = l.custo.trim() === "" ? 0 : lerNumero(l.custo);
    return Number.isFinite(q) && q > 0 && Number.isFinite(c) && c >= 0;
  };
  const subtotal = (l: Linha) => (valida(l) ? lerNumero(l.quantidade) * (l.custo.trim() === "" ? 0 : lerNumero(l.custo)) : 0);
  const total = linhas.reduce((s, l) => s + subtotal(l), 0);
  const podeRegistar = linhas.length > 0 && linhas.every(valida) && !aGuardar;

  function registar() {
    if (!podeRegistar) return;
    const p = new usecase.RececaoParams();
    p.ArmazemID = armazem.ID;
    p.Referencia = referencia.trim();
    p.Itens = linhas.map((l) => {
      const i = new usecase.ItemRececao();
      i.ProdutoID = l.produto.ID;
      i.UnidadeProdutoID = l.unidadeID;
      i.Quantidade = lerNumero(l.quantidade);
      i.CustoUnitario = l.custo.trim() === "" ? 0 : lerNumero(l.custo);
      return i;
    });
    setAGuardar(true);
    RegistarRececaoMercadoria(p)
      .then(() => onConcluido(linhas.length))
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-produto-form">
      <div className="admin-produto-campos" style={{ gridTemplateColumns: "2fr 1fr" }}>
        <label className="admin-campo">
          Documento do fornecedor (opcional)
          <input
            value={referencia}
            onChange={(e) => setReferencia(e.target.value)}
            placeholder="ex: Fatura 123 — Distribuidora Matola"
          />
        </label>
        <div className="admin-campo">
          Entra no armazém
          <div className="admin-rececao-armazem">{armazem.Nome}</div>
        </div>
      </div>

      <div className="admin-rececao-pesquisa">
        <label className="admin-pesquisa" style={{ maxWidth: "none" }}>
          <MagnifyingGlassIcon className="admin-pesquisa-icone" aria-hidden="true" />
          <input
            autoFocus
            placeholder="Acrescentar produto: nome ou código de barras"
            value={pesquisa}
            onChange={(e) => setPesquisa(e.target.value)}
            onKeyDown={aoTeclar}
          />
        </label>
        {sugestoes.length > 0 && (
          <div className="admin-rececao-sugestoes">
            {sugestoes.map((p) => (
              <button key={p.ID} onClick={() => adicionar(p)}>
                {p.Nome}
                <span className="admin-texto-fraco"> · {unidadesDe(p.ID).map((u) => u.NomeUnidade).join(", ")}</span>
              </button>
            ))}
          </div>
        )}
        {termo && sugestoes.length === 0 && (
          <div className="admin-texto-fraco" style={{ marginTop: 6 }}>
            Nenhum produto encontrado{comLotes ? " (os produtos com lotes entram em \"Entrada em lote\")" : ""}.
          </div>
        )}
      </div>

      <table className="admin-tabela admin-unidades-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th>Unidade de compra</th>
            <th style={{ width: 110 }}>Quantidade</th>
            <th style={{ width: 140 }}>Custo por unidade</th>
            <th style={{ textAlign: "right" }}>Subtotal</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {linhas.map((l) => {
            const u = unidades.find((x) => x.ID === l.unidadeID);
            const qBase = u && valida(l) ? lerNumero(l.quantidade) * u.QtdEmUnidadeBase : null;
            return (
              <tr key={l.chave}>
                <td>
                  {l.produto.Nome}
                  {qBase != null && u && u.QtdEmUnidadeBase !== 1 && (
                    <div className="admin-texto-fraco">
                      = {num(qBase)} {l.produto.UnidadeBase}
                    </div>
                  )}
                </td>
                <td>
                  <select
                    aria-label="Unidade de compra"
                    value={l.unidadeID}
                    onChange={(e) => mudar(l.chave, "unidadeID", Number(e.target.value))}
                  >
                    {unidadesDe(l.produto.ID).map((x) => (
                      <option key={x.ID} value={x.ID}>
                        {x.NomeUnidade}
                      </option>
                    ))}
                  </select>
                </td>
                <td>
                  <input
                    aria-label="Quantidade"
                    inputMode="decimal"
                    value={l.quantidade}
                    onChange={(e) => mudar(l.chave, "quantidade", e.target.value)}
                  />
                </td>
                <td>
                  <input
                    aria-label="Custo por unidade de compra"
                    inputMode="decimal"
                    placeholder="0,00"
                    value={l.custo}
                    onChange={(e) => mudar(l.chave, "custo", e.target.value)}
                  />
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{valida(l) ? mt(subtotal(l)) : "—"}</td>
                <td>
                  <button
                    className="admin-btn-secundario"
                    aria-label={`Remover ${l.produto.Nome}`}
                    onClick={() => setLinhas((ls) => ls.filter((x) => x.chave !== l.chave))}
                  >
                    <TrashIcon className="admin-btn-icon-inline" aria-hidden="true" />
                  </button>
                </td>
              </tr>
            );
          })}
          {linhas.length === 0 && (
            <tr>
              <td colSpan={6} className="admin-texto-fraco">
                Pesquise acima e acrescente cada produto da guia ou fatura.
              </td>
            </tr>
          )}
        </tbody>
        {linhas.length > 0 && (
          <tfoot>
            <tr>
              <td colSpan={4}>
                <strong>Total da receção</strong>
              </td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                <strong>{mt(total)}</strong>
              </td>
              <td></td>
            </tr>
          </tfoot>
        )}
      </table>

      <div className="admin-form">
        <button className="admin-btn" disabled={!podeRegistar} onClick={registar}>
          {aGuardar ? "A registar…" : `Registar receção${linhas.length ? ` (${linhas.length} produto(s))` : ""}`}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
      </div>
    </div>
  );
}
