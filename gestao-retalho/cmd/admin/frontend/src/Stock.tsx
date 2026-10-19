import { useEffect, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import {
  ArrowDownTrayIcon,
  ArrowsRightLeftIcon,
  XCircleIcon,
  MinusCircleIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
} from "@heroicons/react/24/outline";
import {
  ListarArmazens,
  ListarProdutos,
  ListarStockPorArmazem,
  ListarMovimentosStock,
  RegistarEntradaStock,
  TransferirStock,
  RegistarPerdaStock,
  RegistarConsumoInternoStock,
  RegistarEntradaComLoteStock,
  ListarLotesStock,
  RegistarPerdaDeLoteStock,
  RegistarConsumoInternoDeLoteStock,
  TransferirLoteStock,
} from "../wailsjs/go/admin/AdminApp";
import { domain, usecase } from "../wailsjs/go/models";
import Modal from "./Modal";
import Rececao from "./Rececao";
import type { Navegar } from "./App";
import { mt, num } from "./formato";

type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;
type TipoMovimento = "entrada" | "entrada_com_lote" | "transferencia" | "perda" | "consumo_interno";

const TIPO_MOVIMENTO_INFO: Record<TipoMovimento, { rotulo: string; Icone: IconeSvg; cor: string }> = {
  entrada: { rotulo: "Entrada", Icone: ArrowDownTrayIcon, cor: "ok" },
  entrada_com_lote: { rotulo: "Entrada c/ lote", Icone: ArrowDownTrayIcon, cor: "ok" },
  transferencia: { rotulo: "Transferência", Icone: ArrowsRightLeftIcon, cor: "accent" },
  perda: { rotulo: "Perda", Icone: XCircleIcon, cor: "danger" },
  consumo_interno: { rotulo: "Consumo interno", Icone: MinusCircleIcon, cor: "warn" },
};

// Movimentos que não se registam por este ecrã mas aparecem no histórico.
const ROTULO_OUTROS_MOVIMENTOS: Record<string, string> = {
  venda: "Venda",
  ajuste_inventario: "Ajuste de inventário",
  anulacao_venda: "Anulação de venda",
  producao: "Produção",
};


type ModalEstado =
  | { aberto: false }
  | { aberto: true; produto: domain.Produto }
  | { aberto: true; historico: domain.Produto }
  | { aberto: true; entradaEmLote: true }
  | { aberto: true; rececao: true };

/**
 * Stock da única loja que este cmd/admin serve — escolhe o armazém, vê a
 * quantidade e o custo médio ponderado de cada produto ali, e regista
 * movimentos (entrada, transferência, perda, consumo interno) por um
 * modal com seletor de tipo. "Abaixo do mínimo" reaproveita o mesmo
 * StockMinimo que já alimenta o alerta do Dashboard.
 */
export default function StockScreen({ onNavegar, onErro }: { onNavegar: Navegar; onErro: (e: string) => void }) {
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [modal, setModal] = useState<ModalEstado>({ aberto: false });
  // Incrementado sempre que um movimento é guardado — entra na key de
  // TabelaStock para forçar um remonte (e portanto recarregar) em vez de
  // expor um método imperativo de refresh.
  const [versao, setVersao] = useState(0);
  const [pesquisa, setPesquisa] = useState("");
  const [aviso, setAviso] = useState<string | null>(null);

  useEffect(() => {
    ListarProdutos()
      .then(setProdutos)
      .catch((e) => onErro(String(e)));
    ListarArmazens()
      .then((lista) => {
        setArmazens(lista);
        if (lista.length > 0) setArmazemID(lista[0].ID);
      })
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  function fecharModal() {
    setModal({ aberto: false });
  }

  return (
    <div className="admin-lojas-screen">
      <nav className="admin-breadcrumb" aria-label="breadcrumb">
        <span className="admin-breadcrumb-atual">Stock</span>
      </nav>

      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Stock</h2>
        <label className="admin-stock-armazem-seletor">
          Armazém:{" "}
          <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
            {armazens.map((a) => (
              <option key={a.ID} value={a.ID}>
                {a.Nome}
              </option>
            ))}
          </select>
        </label>
        <div className="admin-form" style={{ marginLeft: "auto" }}>
          <button
            className="admin-btn-secundario"
            title="Valor do stock ao custo, para exportar em Excel ou PDF"
            onClick={() => onNavegar({ ecra: "relatorios", relatorio: "stock" })}
          >
            Valor do stock / exportar
          </button>
          {produtos.some((p) => p.LoteControlado) && armazemID !== "" && (
            <button className="admin-btn-secundario" onClick={() => setModal({ aberto: true, entradaEmLote: true })}>
              + Entrada em lote
            </button>
          )}
          {armazemID !== "" && (
            <button className="admin-btn" onClick={() => setModal({ aberto: true, rececao: true })}>
              + Receção de mercadoria
            </button>
          )}
        </div>
      </div>

      {aviso && (
        <div className="admin-aviso-ok">
          {aviso}
          <button onClick={() => setAviso(null)} aria-label="Fechar">
            ×
          </button>
        </div>
      )}

      {armazens.length > 0 && (
        <div className="admin-form">
          <label className="admin-pesquisa">
            <MagnifyingGlassIcon className="admin-pesquisa-icone" aria-hidden="true" />
            <input placeholder="Pesquisar produto" value={pesquisa} onChange={(e) => setPesquisa(e.target.value)} />
          </label>
        </div>
      )}

      {armazens.length === 0 ? (
        <p style={{ color: "var(--text-secondary)" }}>Nenhum armazém registado — crie um em "Armazéns" primeiro.</p>
      ) : (
        armazemID !== "" && (
          <TabelaStock
            key={`${armazemID}-${versao}`}
            armazemID={armazemID}
            produtos={produtos}
            pesquisa={pesquisa}
            onAbrirMovimento={(produto) => setModal({ aberto: true, produto })}
            onAbrirHistorico={(produto) => setModal({ aberto: true, historico: produto })}
            onErro={onErro}
          />
        )
      )}

      {modal.aberto && "produto" in modal && armazemID !== "" && (
        <Modal titulo={`Movimento — ${modal.produto.Nome}`} onFechar={fecharModal}>
          <MovimentoForm
            produto={modal.produto}
            armazemAtualID={armazemID}
            armazens={armazens}
            onGuardado={() => {
              fecharModal();
              setVersao((v) => v + 1);
            }}
            onCancelar={fecharModal}
            onErro={onErro}
          />
        </Modal>
      )}

      {modal.aberto && "historico" in modal && (
        <Modal titulo={`Histórico — ${modal.historico.Nome}`} onFechar={fecharModal}>
          <HistoricoMovimentos produtoID={modal.historico.ID} armazens={armazens} onErro={onErro} />
        </Modal>
      )}

      {modal.aberto && "rececao" in modal && armazemID !== "" && (
        <Modal titulo="Receção de mercadoria" largo onFechar={fecharModal}>
          <Rececao
            armazem={armazens.find((a) => a.ID === armazemID)!}
            produtos={produtos}
            onConcluido={(n) => {
              fecharModal();
              setVersao((v) => v + 1);
              setAviso(`Receção registada: ${n} produto(s) deram entrada no stock.`);
            }}
            onCancelar={fecharModal}
            onErro={onErro}
          />
        </Modal>
      )}

      {modal.aberto && "entradaEmLote" in modal && armazemID !== "" && (
        <Modal titulo="Entrada em lote — vários produtos" onFechar={fecharModal}>
          <EntradaLoteEmMassa
            armazemID={armazemID}
            produtos={produtos}
            onConcluido={() => {
              fecharModal();
              setVersao((v) => v + 1);
            }}
            onCancelar={fecharModal}
            onErro={onErro}
          />
        </Modal>
      )}
    </div>
  );
}

function TabelaStock({
  armazemID,
  produtos,
  pesquisa,
  onAbrirMovimento,
  onAbrirHistorico,
  onErro,
}: {
  armazemID: number;
  produtos: domain.Produto[];
  pesquisa: string;
  onAbrirMovimento: (p: domain.Produto) => void;
  onAbrirHistorico: (p: domain.Produto) => void;
  onErro: (e: string) => void;
}) {
  // stock_atual é uma view calculada a partir de movimentos_stock (SDD
  // 3.2) — só tem linha para um produto+armazém depois do primeiro
  // movimento. Por isso a tabela lista sempre TODOS os produtos com
  // controlo de stock (não só os que já vieram de ListarStockPorArmazem),
  // com quantidade 0 para os que ainda não tiveram nenhuma entrada; senão
  // um produto recém-criado nunca teria "+ Movimento" em lado nenhum.
  const [stockItens, setStockItens] = useState<domain.StockItem[] | null>(null);

  useEffect(() => {
    ListarStockPorArmazem(armazemID)
      .then(setStockItens)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável; armazemID está fixo por instância (key troca-o)
  }, []);

  const stockDoProduto = (id: number) => stockItens?.find((i) => i.ProdutoID === id);

  // Produtos "composto" nunca guardam stock próprio (é inferido via ficha
  // técnica, ainda não portada) — não fazem sentido aqui.
  const ativos = produtos.filter((p) => p.Ativo && p.ControlaStock);
  const termo = pesquisa.trim().toLowerCase();
  const produtosDoArmazem = ativos
    .filter((p) => !termo || p.Nome.toLowerCase().includes(termo))
    .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"));
  const valorTotal = ativos.reduce((s, p) => {
    const i = stockDoProduto(p.ID);
    return s + (i ? i.Quantidade * i.CustoMedioPonderado : 0);
  }, 0);

  return (
    <table className="admin-tabela">
      <thead>
        <tr>
          <th>Produto</th>
          <th style={{ textAlign: "right" }}>Quantidade</th>
          <th style={{ textAlign: "right" }}>Custo médio</th>
          <th style={{ textAlign: "right" }}>Valor</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        {stockItens === null ? (
          <tr>
            <td colSpan={5}>A carregar…</td>
          </tr>
        ) : produtosDoArmazem.length === 0 ? (
          <tr>
            <td colSpan={5}>
              {ativos.length === 0
                ? 'Sem produtos com controlo de stock — crie um em "Produtos" primeiro.'
                : "Nenhum produto com esse nome."}
            </td>
          </tr>
        ) : (
          produtosDoArmazem.map((produto) => {
            const item = stockDoProduto(produto.ID);
            const quantidade = item?.Quantidade ?? 0;
            const abaixoDoMinimo = produto.StockMinimo != null && quantidade < produto.StockMinimo;
            return (
              <tr key={produto.ID}>
                <td>
                  {produto.Nome}
                  {abaixoDoMinimo && (
                    <span className="admin-stock-abaixo-minimo">
                      <ExclamationTriangleIcon className="admin-stock-abaixo-minimo-icon" aria-hidden="true" />
                      abaixo do mínimo
                    </span>
                  )}
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  {num(quantidade)} {produto.UnidadeBase}
                </td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>{mt(item?.CustoMedioPonderado ?? 0)}</td>
                <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                  {mt(quantidade * (item?.CustoMedioPonderado ?? 0))}
                </td>
                <td className="admin-acoes">
                  <button className="admin-btn-secundario" onClick={() => onAbrirMovimento(produto)}>
                    + Movimento
                  </button>
                  <button className="admin-btn-secundario" onClick={() => onAbrirHistorico(produto)}>
                    <ClockIcon className="admin-btn-icon-inline" aria-hidden="true" />
                    Histórico
                  </button>
                </td>
              </tr>
            );
          })
        )}
      </tbody>
      {stockItens !== null && ativos.length > 0 && (
        <tfoot>
          <tr>
            <td colSpan={3}>
              <strong>Valor em stock neste armazém</strong>
            </td>
            <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
              <strong>{mt(valorTotal)}</strong>
            </td>
            <td></td>
          </tr>
        </tfoot>
      )}
    </table>
  );
}

function MovimentoForm({
  produto,
  armazemAtualID,
  armazens,
  onGuardado,
  onCancelar,
  onErro,
}: {
  produto: domain.Produto;
  armazemAtualID: number;
  armazens: domain.Armazem[];
  onGuardado: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  // Produtos com rastreio por lote (Fase 6, SDD 9.4) não usam a entrada
  // normal — RegistarEntradaStock rejeita-os e pede número de lote e
  // validade (FEFO na venda depende disto), daí "entrada_com_lote" em vez
  // de "entrada" no seletor de tipos.
  const tiposDisponiveis = (Object.keys(TIPO_MOVIMENTO_INFO) as TipoMovimento[]).filter((t) =>
    produto.LoteControlado ? t !== "entrada" : t !== "entrada_com_lote",
  );
  const [tipo, setTipo] = useState<TipoMovimento>(produto.LoteControlado ? "entrada_com_lote" : "entrada");
  const [quantidade, setQuantidade] = useState("1");
  const [custoUnitario, setCustoUnitario] = useState("0");
  const [motivo, setMotivo] = useState("");
  const [armazemDestinoID, setArmazemDestinoID] = useState<number | "">("");
  const [numeroLote, setNumeroLote] = useState("");
  const [validade, setValidade] = useState("");
  const [aGuardar, setAGuardar] = useState(false);
  // Produtos lote-controlados: perda, consumo e transferência saem sempre
  // de um lote escolhido aqui (o backend recusa a saída genérica).
  const [lotes, setLotes] = useState<domain.Lote[] | null>(produto.LoteControlado ? null : []);
  const [loteID, setLoteID] = useState<number | "">("");
  const saiDeLote = produto.LoteControlado && tipo !== "entrada_com_lote";

  useEffect(() => {
    if (!produto.LoteControlado) return;
    ListarLotesStock(produto.ID, armazemAtualID)
      .then(setLotes)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- produto/armazém fixos por instância do modal
  }, []);

  function guardar() {
    const qtd = Number(quantidade);
    if (qtd <= 0 || aGuardar) return;
    if (saiDeLote && loteID === "") {
      onErro("Escolha o lote");
      return;
    }

    let pedido: Promise<unknown>;
    switch (tipo) {
      case "entrada": {
        const p = new usecase.EntradaParams();
        p.ProdutoID = produto.ID;
        p.ArmazemDestinoID = armazemAtualID;
        p.Quantidade = qtd;
        p.CustoUnitario = Number(custoUnitario);
        pedido = RegistarEntradaStock(p);
        break;
      }
      case "entrada_com_lote": {
        if (!numeroLote || !validade) {
          onErro("Indique o número de lote e a validade");
          return;
        }
        const p = new usecase.EntradaComLoteParams();
        p.ProdutoID = produto.ID;
        p.ArmazemID = armazemAtualID;
        p.NumeroLote = numeroLote;
        // Go time.Time espera RFC3339 — o input type="date" já dá "AAAA-MM-DD".
        p.DataValidade = new Date(validade + "T00:00:00Z").toISOString();
        p.Quantidade = qtd;
        p.CustoUnitario = Number(custoUnitario);
        pedido = RegistarEntradaComLoteStock(p);
        break;
      }
      case "transferencia": {
        if (armazemDestinoID === "") {
          onErro("Escolha o armazém de destino");
          return;
        }
        if (saiDeLote) {
          const p = new usecase.TransferenciaLoteParams();
          p.LoteID = loteID as number;
          p.ArmazemDestinoID = armazemDestinoID;
          p.Quantidade = qtd;
          pedido = TransferirLoteStock(p);
          break;
        }
        const p = new usecase.TransferenciaParams();
        p.ProdutoID = produto.ID;
        p.ArmazemOrigemID = armazemAtualID;
        p.ArmazemDestinoID = armazemDestinoID;
        p.Quantidade = qtd;
        pedido = TransferirStock(p);
        break;
      }
      case "perda": {
        if (saiDeLote) {
          const p = new usecase.SaidaLoteParams();
          p.LoteID = loteID as number;
          p.Quantidade = qtd;
          p.Motivo = motivo;
          pedido = RegistarPerdaDeLoteStock(p);
          break;
        }
        const p = new usecase.SaidaParams();
        p.ProdutoID = produto.ID;
        p.ArmazemOrigemID = armazemAtualID;
        p.Quantidade = qtd;
        p.Motivo = motivo;
        pedido = RegistarPerdaStock(p);
        break;
      }
      case "consumo_interno": {
        if (saiDeLote) {
          const p = new usecase.SaidaLoteParams();
          p.LoteID = loteID as number;
          p.Quantidade = qtd;
          p.Motivo = motivo;
          pedido = RegistarConsumoInternoDeLoteStock(p);
          break;
        }
        const p = new usecase.SaidaParams();
        p.ProdutoID = produto.ID;
        p.ArmazemOrigemID = armazemAtualID;
        p.Quantidade = qtd;
        p.Motivo = motivo;
        pedido = RegistarConsumoInternoStock(p);
        break;
      }
    }

    setAGuardar(true);
    pedido
      .then(() => onGuardado())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      <div className="admin-stock-tipos">
        {tiposDisponiveis.map((t) => {
          const info = TIPO_MOVIMENTO_INFO[t];
          return (
            <div
              key={t}
              className={`admin-armazem-tipo-opcao cor-${info.cor}${tipo === t ? " ativo" : ""}`}
              onClick={() => setTipo(t)}
            >
              <info.Icone className="admin-armazem-tipo-opcao-icone" aria-hidden="true" />
              {info.rotulo}
            </div>
          );
        })}
      </div>
      {produto.LoteControlado && lotes !== null && <TabelaLotes lotes={lotes} />}

      <div className="admin-form">
        {saiDeLote && (
          <select value={loteID} onChange={(e) => setLoteID(e.target.value ? Number(e.target.value) : "")}>
            <option value="">Lote…</option>
            {(lotes ?? []).map((l) => (
              <option key={l.ID} value={l.ID}>
                {l.NumeroLote} — validade {String(l.DataValidade).slice(0, 10)} — {num(l.Quantidade)} disponível
              </option>
            ))}
          </select>
        )}
        <input
          placeholder="Quantidade"
          type="number"
          min="0"
          value={quantidade}
          onChange={(e) => setQuantidade(e.target.value)}
        />

        {(tipo === "entrada" || tipo === "entrada_com_lote") && (
          <input
            placeholder="Custo unitário"
            type="number"
            min="0"
            value={custoUnitario}
            onChange={(e) => setCustoUnitario(e.target.value)}
          />
        )}

        {tipo === "entrada_com_lote" && (
          <>
            <input placeholder="Número de lote" value={numeroLote} onChange={(e) => setNumeroLote(e.target.value)} />
            <input
              placeholder="Validade"
              type="date"
              value={validade}
              onChange={(e) => setValidade(e.target.value)}
            />
          </>
        )}

        {tipo === "transferencia" && (
          <select
            value={armazemDestinoID}
            onChange={(e) => setArmazemDestinoID(e.target.value ? Number(e.target.value) : "")}
          >
            <option value="">Armazém destino…</option>
            {armazens
              .filter((a) => a.ID !== armazemAtualID)
              .map((a) => (
                <option key={a.ID} value={a.ID}>
                  {a.Nome}
                </option>
              ))}
          </select>
        )}

        {(tipo === "perda" || tipo === "consumo_interno") && (
          <input placeholder="Motivo" value={motivo} onChange={(e) => setMotivo(e.target.value)} />
        )}
      </div>

      <div className="admin-form">
        <button
          className="admin-btn"
          disabled={Number(quantidade) <= 0 || aGuardar || (saiDeLote && loteID === "")}
          onClick={guardar}
        >
          {aGuardar ? "A guardar…" : "Registar movimento"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}

function TabelaLotes({ lotes }: { lotes: domain.Lote[] }) {
  if (lotes.length === 0) {
    return <p style={{ color: "var(--text-secondary)", fontSize: "0.85rem" }}>Sem lotes ainda neste armazém.</p>;
  }

  return (
    <table className="admin-tabela">
      <thead>
        <tr>
          <th>Lote</th>
          <th>Validade</th>
          <th>Quantidade</th>
        </tr>
      </thead>
      <tbody>
        {lotes.map((l) => (
          <tr key={l.ID}>
            <td>{l.NumeroLote}</td>
            <td>{String(l.DataValidade).slice(0, 10)}</td>
            <td>{num(l.Quantidade)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

let proximoIdLinha = 0;
type LinhaEntradaLote = { id: number; produtoID: number | ""; quantidade: string; custoUnitario: string };
const novaLinhaEntradaLote = (): LinhaEntradaLote => ({
  id: ++proximoIdLinha,
  produtoID: "",
  quantidade: "1",
  custoUnitario: "0",
});

/**
 * Entrada de um lote físico único (o número de lote e a validade do
 * fornecedor) que cobre vários produtos ao mesmo tempo — situação comum em
 * farmácia, onde uma remessa chega com um só número de lote para vários
 * itens. lotes_produto guarda sempre um lote por produto (chave
 * produto+armazém+número), por isso isto não é uma operação atómica nova
 * no backend: é só este formulário a repetir RegistarEntradaComLoteStock
 * uma vez por produto, com o número/validade partilhados.
 */
function EntradaLoteEmMassa({
  armazemID,
  produtos,
  onConcluido,
  onCancelar,
  onErro,
}: {
  armazemID: number;
  produtos: domain.Produto[];
  onConcluido: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const produtosLoteControlado = produtos.filter((p) => p.Ativo && p.LoteControlado);

  const [numeroLote, setNumeroLote] = useState("");
  const [validade, setValidade] = useState("");
  const [linhas, setLinhas] = useState<LinhaEntradaLote[]>([novaLinhaEntradaLote()]);
  const [resultados, setResultados] = useState<Record<number, "ok" | string>>({});
  const [aGuardar, setAGuardar] = useState(false);

  const algumSucesso = Object.values(resultados).some((r) => r === "ok");

  function atualizarLinha(id: number, alteracoes: Partial<LinhaEntradaLote>) {
    setLinhas((ls) => ls.map((l) => (l.id === id ? { ...l, ...alteracoes } : l)));
  }

  function removerLinha(id: number) {
    setLinhas((ls) => ls.filter((l) => l.id !== id));
  }

  async function guardar() {
    if (!numeroLote || !validade) {
      onErro("Indique o número de lote e a validade");
      return;
    }
    const pendentes = linhas.filter((l) => resultados[l.id] !== "ok");
    for (const l of pendentes) {
      if (l.produtoID === "" || Number(l.quantidade) <= 0) {
        onErro("Escolha o produto e uma quantidade em todas as linhas");
        return;
      }
    }

    setAGuardar(true);
    const novosResultados = { ...resultados };
    for (const l of pendentes) {
      const p = new usecase.EntradaComLoteParams();
      p.ProdutoID = l.produtoID as number;
      p.ArmazemID = armazemID;
      p.NumeroLote = numeroLote;
      p.DataValidade = new Date(validade + "T00:00:00Z").toISOString();
      p.Quantidade = Number(l.quantidade);
      p.CustoUnitario = Number(l.custoUnitario || "0");
      try {
        await RegistarEntradaComLoteStock(p);
        novosResultados[l.id] = "ok";
      } catch (e) {
        novosResultados[l.id] = String(e);
      }
    }
    setResultados(novosResultados);
    setAGuardar(false);

    if (linhas.every((l) => novosResultados[l.id] === "ok")) {
      onConcluido();
    }
  }

  return (
    <div className="admin-armazem-form">
      <div className="admin-form">
        <input placeholder="Número de lote" value={numeroLote} onChange={(e) => setNumeroLote(e.target.value)} />
        <input placeholder="Validade" type="date" value={validade} onChange={(e) => setValidade(e.target.value)} />
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th>Quantidade</th>
            <th>Custo unitário</th>
            <th></th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {linhas.map((l) => {
            const feito = resultados[l.id] === "ok";
            const erro = resultados[l.id] && resultados[l.id] !== "ok" ? (resultados[l.id] as string) : null;
            return (
              <tr key={l.id}>
                <td>
                  <select
                    disabled={feito}
                    value={l.produtoID}
                    onChange={(e) =>
                      atualizarLinha(l.id, { produtoID: e.target.value ? Number(e.target.value) : "" })
                    }
                  >
                    <option value="">Produto…</option>
                    {produtosLoteControlado
                      .filter((p) => p.ID === l.produtoID || !linhas.some((o) => o.produtoID === p.ID))
                      .map((p) => (
                        <option key={p.ID} value={p.ID}>
                          {p.Nome}
                        </option>
                      ))}
                  </select>
                </td>
                <td>
                  <input
                    disabled={feito}
                    type="number"
                    min="0"
                    value={l.quantidade}
                    onChange={(e) => atualizarLinha(l.id, { quantidade: e.target.value })}
                  />
                </td>
                <td>
                  <input
                    disabled={feito}
                    type="number"
                    min="0"
                    value={l.custoUnitario}
                    onChange={(e) => atualizarLinha(l.id, { custoUnitario: e.target.value })}
                  />
                </td>
                <td>
                  {feito && "✓ registada"}
                  {erro && <span style={{ color: "var(--danger)" }}>{erro}</span>}
                </td>
                <td>
                  {!feito && linhas.length > 1 && (
                    <button className="admin-btn-secundario" onClick={() => removerLinha(l.id)}>
                      ×
                    </button>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      <div className="admin-form">
        <button className="admin-btn-secundario" onClick={() => setLinhas((ls) => [...ls, novaLinhaEntradaLote()])}>
          + Adicionar produto
        </button>
      </div>

      <div className="admin-form">
        <button className="admin-btn" disabled={aGuardar} onClick={guardar}>
          {aGuardar ? "A registar…" : "Registar entradas"}
        </button>
        <button className="admin-btn-secundario" onClick={algumSucesso ? onConcluido : onCancelar}>
          {algumSucesso ? "Concluir" : "Cancelar"}
        </button>
      </div>
    </div>
  );
}

function HistoricoMovimentos({
  produtoID,
  armazens,
  onErro,
}: {
  produtoID: number;
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const [movimentos, setMovimentos] = useState<domain.MovimentoStock[] | null>(null);

  useEffect(() => {
    ListarMovimentosStock(produtoID)
      .then(setMovimentos)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só corre para o produto desta instância (o modal remonta ao trocar)
  }, []);

  const nomeArmazem = (id?: number) => (id != null ? (armazens.find((a) => a.ID === id)?.Nome ?? `#${id}`) : "—");

  return (
    <table className="admin-tabela">
      <thead>
        <tr>
          <th>Tipo</th>
          <th>Quantidade</th>
          <th>Origem</th>
          <th>Destino</th>
          <th>Data</th>
        </tr>
      </thead>
      <tbody>
        {(movimentos ?? []).map((m) => (
          <tr key={m.ID}>
            <td>{TIPO_MOVIMENTO_INFO[m.Tipo as TipoMovimento]?.rotulo ?? ROTULO_OUTROS_MOVIMENTOS[m.Tipo] ?? m.Tipo}</td>
            <td>{num(m.QuantidadeUnidadeBase)}</td>
            <td>{nomeArmazem(m.ArmazemOrigemID)}</td>
            <td>{nomeArmazem(m.ArmazemDestinoID)}</td>
            <td>{new Date(m.CriadoEm).toLocaleString("pt-PT")}</td>
          </tr>
        ))}
        {movimentos?.length === 0 && (
          <tr>
            <td colSpan={5}>Sem movimentos registados.</td>
          </tr>
        )}
        {movimentos === null && (
          <tr>
            <td colSpan={5}>A carregar…</td>
          </tr>
        )}
      </tbody>
    </table>
  );
}
