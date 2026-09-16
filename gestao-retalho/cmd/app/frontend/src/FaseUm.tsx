import { useEffect, useState } from "react";
import {
  ListarArmazens,
  CriarArmazem,
  ListarProdutos,
  CriarProduto,
  ObterStock,
  RegistarEntradaStock,
  TransferirStock,
  RegistarPerdaStock,
  IniciarInventario,
  RegistarContagemInventario,
  FecharInventario,
} from "../wailsjs/go/api/App";
import { api, domain, usecase } from "../wailsjs/go/models";
import FichasTecnicasSection from "./FichasTecnicas";
import FarmaciaSection from "./Farmacia";
import ClientesSection from "./Clientes";
import RelatoriosSection from "./Relatorios";
import "./FaseUm.css";

/**
 * Ecrã de exercício da Fase 1 (core: produtos, armazéns, stock) — sem tema
 * de negócio nem polimento visual (plano de implementação, Fase 1). Serve
 * para validar os bindings ponta a ponta; é substituído pelos ecrãs reais
 * de PDV/admin nas fases seguintes.
 */
export default function FaseUm() {
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [erro, setErro] = useState<string | null>(null);

  const recarregar = () => {
    ListarArmazens().then(setArmazens).catch((e) => setErro(String(e)));
    ListarProdutos().then(setProdutos).catch((e) => setErro(String(e)));
  };

  useEffect(recarregar, []);

  return (
    <div className="fase1">
      {erro && (
        <div className="fase1-erro">
          {erro} <button onClick={() => setErro(null)}>×</button>
        </div>
      )}
      <ArmazensSection armazens={armazens} onChange={recarregar} onErro={setErro} />
      <ProdutosSection produtos={produtos} onChange={recarregar} onErro={setErro} />
      <StockSection produtos={produtos} armazens={armazens} onErro={setErro} />
      <FichasTecnicasSection produtos={produtos} armazens={armazens} onErro={setErro} />
      <FarmaciaSection produtos={produtos} armazens={armazens} onErro={setErro} />
      <InventarioSection produtos={produtos} armazens={armazens} onErro={setErro} />
      <ClientesSection onErro={setErro} />
      <RelatoriosSection armazens={armazens} onErro={setErro} />
    </div>
  );
}

function run(fn: () => Promise<unknown>, onErro: (e: string) => void, then?: () => void) {
  fn()
    .then(() => then?.())
    .catch((e) => onErro(String(e)));
}

// --- Armazéns --------------------------------------------------------------

function ArmazensSection({
  armazens,
  onChange,
  onErro,
}: {
  armazens: domain.Armazem[];
  onChange: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState("");
  const [tipo, setTipo] = useState<"principal" | "ponto_venda" | "deposito">("principal");

  function criar() {
    const a = new domain.Armazem();
    a.Nome = nome;
    a.Tipo = tipo;
    run(() => CriarArmazem(a), onErro, () => {
      setNome("");
      onChange();
    });
  }

  return (
    <section>
      <h2>Armazéns</h2>
      <table>
        <thead>
          <tr>
            <th>Nome</th>
            <th>Tipo</th>
            <th>Ativo</th>
          </tr>
        </thead>
        <tbody>
          {armazens.map((a) => (
            <tr key={a.ID}>
              <td>{a.Nome}</td>
              <td>{a.Tipo}</td>
              <td>{a.Ativo ? "sim" : "não"}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="fase1-form">
        <input placeholder="Nome do armazém" value={nome} onChange={(e) => setNome(e.target.value)} />
        <select value={tipo} onChange={(e) => setTipo(e.target.value as typeof tipo)}>
          <option value="principal">principal</option>
          <option value="ponto_venda">ponto_venda</option>
          <option value="deposito">deposito</option>
        </select>
        <button disabled={!nome} onClick={criar}>
          + Criar armazém
        </button>
      </div>
    </section>
  );
}

// --- Produtos ----------------------------------------------------------------

function ProdutosSection({
  produtos,
  onChange,
  onErro,
}: {
  produtos: domain.Produto[];
  onChange: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState("");
  const [tipoProduto, setTipoProduto] = useState<"simples" | "composto" | "semi_elaborado">("simples");
  const [unidadeBase, setUnidadeBase] = useState("unidade");
  const [precoCusto, setPrecoCusto] = useState("0");
  const [precoVenda, setPrecoVenda] = useState("0");
  const [loteControlado, setLoteControlado] = useState(false);

  function criar() {
    const produto = new domain.Produto();
    produto.Nome = nome;
    produto.TipoProduto = tipoProduto;
    produto.UnidadeBase = unidadeBase;
    produto.LoteControlado = loteControlado;

    const unidade = new domain.UnidadeProduto();
    unidade.NomeUnidade = unidadeBase;
    unidade.QtdEmUnidadeBase = 1;
    unidade.PrecoCusto = Number(precoCusto);
    unidade.PrecoVenda = Number(precoVenda);
    unidade.IsPadraoVenda = true;
    unidade.IsPadraoCompra = true;

    const req = new api.CriarProdutoRequest();
    req.Produto = produto;
    req.Unidades = [unidade];

    run(
      () => CriarProduto(req),
      onErro,
      () => {
        setNome("");
        onChange();
      },
    );
  }

  return (
    <section>
      <h2>Produtos</h2>
      <table>
        <thead>
          <tr>
            <th>ID</th>
            <th>Nome</th>
            <th>Tipo</th>
            <th>Unidade base</th>
            <th>Controla stock</th>
            <th>Lote controlado</th>
          </tr>
        </thead>
        <tbody>
          {produtos.map((p) => (
            <tr key={p.ID}>
              <td>{p.ID}</td>
              <td>{p.Nome}</td>
              <td>{p.TipoProduto}</td>
              <td>{p.UnidadeBase}</td>
              <td>{p.ControlaStock ? "sim" : "não"}</td>
              <td>{p.LoteControlado ? "sim" : "não"}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div className="fase1-form">
        <input placeholder="Nome do produto" value={nome} onChange={(e) => setNome(e.target.value)} />
        <select value={tipoProduto} onChange={(e) => setTipoProduto(e.target.value as typeof tipoProduto)}>
          <option value="simples">simples</option>
          <option value="composto">composto</option>
          <option value="semi_elaborado">semi_elaborado</option>
        </select>
        <input placeholder="Unidade base (ex: unidade, kg)" value={unidadeBase} onChange={(e) => setUnidadeBase(e.target.value)} />
        <input placeholder="Preço custo" type="number" value={precoCusto} onChange={(e) => setPrecoCusto(e.target.value)} />
        <input placeholder="Preço venda" type="number" value={precoVenda} onChange={(e) => setPrecoVenda(e.target.value)} />
        <label>
          <input type="checkbox" checked={loteControlado} onChange={(e) => setLoteControlado(e.target.checked)} /> lote
          controlado (farmácia)
        </label>
        <button disabled={!nome} onClick={criar}>
          + Criar produto
        </button>
      </div>
    </section>
  );
}

// --- Stock ---------------------------------------------------------------

function StockSection({
  produtos,
  armazens,
  onErro,
}: {
  produtos: domain.Produto[];
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const [produtoID, setProdutoID] = useState<number | "">("");
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [stock, setStock] = useState<domain.StockItem | null>(null);
  const [quantidade, setQuantidade] = useState("0");
  const [custoUnitario, setCustoUnitario] = useState("0");
  const [armazemDestinoID, setArmazemDestinoID] = useState<number | "">("");
  const [motivo, setMotivo] = useState("");

  const consultar = () => {
    if (produtoID === "" || armazemID === "") return;
    ObterStock(Number(produtoID), Number(armazemID)).then(setStock).catch((e) => onErro(String(e)));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro (setErro) is a stable setter
  useEffect(consultar, [produtoID, armazemID]);

  function entrada() {
    if (produtoID === "" || armazemID === "") return;
    const p = new usecase.EntradaParams();
    p.ProdutoID = Number(produtoID);
    p.ArmazemDestinoID = Number(armazemID);
    p.Quantidade = Number(quantidade);
    p.CustoUnitario = Number(custoUnitario);
    run(() => RegistarEntradaStock(p), onErro, consultar);
  }

  function transferir() {
    if (produtoID === "" || armazemID === "" || armazemDestinoID === "") return;
    const p = new usecase.TransferenciaParams();
    p.ProdutoID = Number(produtoID);
    p.ArmazemOrigemID = Number(armazemID);
    p.ArmazemDestinoID = Number(armazemDestinoID);
    p.Quantidade = Number(quantidade);
    run(() => TransferirStock(p), onErro, consultar);
  }

  function perda() {
    if (produtoID === "" || armazemID === "") return;
    const p = new usecase.SaidaParams();
    p.ProdutoID = Number(produtoID);
    p.ArmazemOrigemID = Number(armazemID);
    p.Quantidade = Number(quantidade);
    p.Motivo = motivo;
    run(() => RegistarPerdaStock(p), onErro, consultar);
  }

  return (
    <section>
      <h2>Stock</h2>
      <div className="fase1-form">
        <select value={produtoID} onChange={(e) => setProdutoID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Produto…</option>
          {produtos.map((p) => (
            <option key={p.ID} value={p.ID}>
              {p.Nome}
            </option>
          ))}
        </select>
        <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Armazém…</option>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
      </div>

      {stock && (
        <p className="fase1-stock-atual">
          Stock atual: <strong>{stock.Quantidade}</strong> · custo médio ponderado: <strong>{stock.CustoMedioPonderado}</strong>
        </p>
      )}

      <div className="fase1-form">
        <input placeholder="Quantidade" type="number" value={quantidade} onChange={(e) => setQuantidade(e.target.value)} />
        <input placeholder="Custo unitário (entrada)" type="number" value={custoUnitario} onChange={(e) => setCustoUnitario(e.target.value)} />
        <button onClick={entrada}>Registar entrada</button>
      </div>

      <div className="fase1-form">
        <select value={armazemDestinoID} onChange={(e) => setArmazemDestinoID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Armazém destino…</option>
          {armazens
            .filter((a) => a.ID !== armazemID)
            .map((a) => (
              <option key={a.ID} value={a.ID}>
                {a.Nome}
              </option>
            ))}
        </select>
        <button onClick={transferir}>Transferir</button>
      </div>

      <div className="fase1-form">
        <input placeholder="Motivo da perda" value={motivo} onChange={(e) => setMotivo(e.target.value)} />
        <button onClick={perda}>Registar perda</button>
      </div>
    </section>
  );
}

// --- Inventário ------------------------------------------------------------

function InventarioSection({
  produtos,
  armazens,
  onErro,
}: {
  produtos: domain.Produto[];
  armazens: domain.Armazem[];
  onErro: (e: string) => void;
}) {
  const [armazemID, setArmazemID] = useState<number | "">("");
  const [inventarioID, setInventarioID] = useState<number | null>(null);
  const [itens, setItens] = useState<domain.InventarioItem[]>([]);
  const [ajustes, setAjustes] = useState<domain.MovimentoStock[] | null>(null);

  const nomeProduto = (id: number) => produtos.find((p) => p.ID === id)?.Nome ?? `#${id}`;

  function iniciar() {
    if (armazemID === "") return;
    setAjustes(null);
    IniciarInventario(Number(armazemID))
      .then((res) => {
        setInventarioID(res.Inventario.ID);
        setItens(res.Itens);
      })
      .catch((e) => onErro(String(e)));
  }

  function contar(itemID: number, valor: string) {
    RegistarContagemInventario(itemID, Number(valor))
      .then((atualizado) => setItens((prev) => prev.map((i) => (i.ID === itemID ? atualizado : i))))
      .catch((e) => onErro(String(e)));
  }

  function fechar() {
    if (inventarioID === null) return;
    FecharInventario(inventarioID)
      .then((movs) => {
        setAjustes(movs);
        setInventarioID(null);
        setItens([]);
      })
      .catch((e) => onErro(String(e)));
  }

  return (
    <section>
      <h2>Inventário</h2>
      <div className="fase1-form">
        <select value={armazemID} onChange={(e) => setArmazemID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Armazém…</option>
          {armazens.map((a) => (
            <option key={a.ID} value={a.ID}>
              {a.Nome}
            </option>
          ))}
        </select>
        <button disabled={inventarioID !== null} onClick={iniciar}>
          Iniciar inventário
        </button>
      </div>

      {itens.length > 0 && (
        <>
          <table>
            <thead>
              <tr>
                <th>Produto</th>
                <th>Qtd. sistema</th>
                <th>Qtd. contada</th>
                <th>Diferença</th>
              </tr>
            </thead>
            <tbody>
              {itens.map((item) => (
                <tr key={item.ID}>
                  <td>{nomeProduto(item.ProdutoID)}</td>
                  <td>{item.QuantidadeSistema}</td>
                  <td>
                    <input
                      type="number"
                      defaultValue={item.QuantidadeContada ?? item.QuantidadeSistema}
                      onBlur={(e) => contar(item.ID, e.target.value)}
                    />
                  </td>
                  <td>{item.Diferenca ?? "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <button onClick={fechar}>Fechar inventário e gerar ajustes</button>
        </>
      )}

      {ajustes && (
        <div>
          <h3>Ajustes gerados ({ajustes.length})</h3>
          <ul>
            {ajustes.map((m) => (
              <li key={m.ID}>
                {nomeProduto(m.ProdutoID)}: {m.ArmazemDestinoID ? "+" : "-"}
                {m.QuantidadeUnidadeBase}
              </li>
            ))}
          </ul>
        </div>
      )}
    </section>
  );
}
