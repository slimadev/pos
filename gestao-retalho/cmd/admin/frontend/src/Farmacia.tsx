import { useEffect, useState } from "react";
import {
  ExclamationTriangleIcon,
  MagnifyingGlassIcon,
  ClipboardDocumentListIcon,
  XMarkIcon,
  LinkIcon,
} from "@heroicons/react/24/outline";
import {
  ListarArmazens,
  ListarProdutos,
  ListarLotesAVencer,
  ListarVendasDoDia,
  ListarReceitasPorVenda,
  RegistarReceitaMedica,
} from "../wailsjs/go/admin/AdminApp";
import { domain, usecase } from "../wailsjs/go/models";
import Modal from "./Modal";
import { mt, num } from "./formato";

const dia = (d: unknown) => String(d).slice(0, 10);

/**
 * Ecrã da farmácia: lotes a vencer (alerta de validade, SDD 9.4) e receitas
 * médicas. A receita é opcional — nunca bloqueia a venda; regista-se aqui
 * depois, associada a uma das vendas do dia, só quando existe.
 */
export default function FarmaciaScreen({ onErro }: { onErro: (e: string) => void }) {
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);

  useEffect(() => {
    ListarProdutos()
      .then(setProdutos)
      .catch((e) => onErro(String(e)));
    ListarArmazens()
      .then(setArmazens)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  const nomeProduto = (id: number) => produtos.find((p) => p.ID === id)?.Nome ?? `#${id}`;
  const nomeArmazem = (id: number) => armazens.find((a) => a.ID === id)?.Nome ?? `#${id}`;

  return (
    <div className="admin-lojas-screen">
      <nav className="admin-breadcrumb unico" aria-label="breadcrumb">
        <span className="admin-breadcrumb-atual">Farmácia</span>
      </nav>
      <h2 style={{ margin: 0 }}>Farmácia</h2>

      <LotesAVencer nomeProduto={nomeProduto} nomeArmazem={nomeArmazem} onErro={onErro} />
      <ReceitasDoDia onErro={onErro} />
    </div>
  );
}

function LotesAVencer({
  nomeProduto,
  nomeArmazem,
  onErro,
}: {
  nomeProduto: (id: number) => string;
  nomeArmazem: (id: number) => string;
  onErro: (e: string) => void;
}) {
  const [dias, setDias] = useState("30");
  const [lotes, setLotes] = useState<domain.Lote[] | null>(null);

  function consultar(nDias: number) {
    ListarLotesAVencer(nDias)
      .then(setLotes)
      .catch((e) => onErro(String(e)));
  }

  useEffect(() => {
    consultar(30);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- só a consulta inicial; as seguintes são pelo botão
  }, []);

  const hoje = new Date().toISOString().slice(0, 10);

  return (
    <section>
      <div className="admin-ecra-header">
        <h3 style={{ margin: 0 }}>Lotes a vencer</h3>
        <div className="admin-form">
          <label>
            Próximos{" "}
            <input type="number" min="0" style={{ width: 80 }} value={dias} onChange={(e) => setDias(e.target.value)} />{" "}
            dias
          </label>
          <button className="admin-btn-secundario" onClick={() => consultar(Number(dias) || 0)}>
            <MagnifyingGlassIcon className="btn-icone" aria-hidden="true" />
            Consultar
          </button>
        </div>
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Produto</th>
            <th>Lote</th>
            <th>Armazém</th>
            <th>Validade</th>
            <th>Quantidade</th>
          </tr>
        </thead>
        <tbody>
          {lotes === null && (
            <tr>
              <td colSpan={5}>A carregar…</td>
            </tr>
          )}
          {lotes?.length === 0 && (
            <tr>
              <td colSpan={5}>Nenhum lote a vencer neste período.</td>
            </tr>
          )}
          {lotes?.map((l) => {
            const vencido = dia(l.DataValidade) < hoje;
            return (
              <tr key={l.ID}>
                <td>{nomeProduto(l.ProdutoID)}</td>
                <td>{l.NumeroLote}</td>
                <td>{nomeArmazem(l.ArmazemID)}</td>
                <td>
                  {dia(l.DataValidade)}
                  {vencido && (
                    <span className="admin-stock-abaixo-minimo">
                      <ExclamationTriangleIcon className="admin-stock-abaixo-minimo-icon" aria-hidden="true" />
                      vencido
                    </span>
                  )}
                </td>
                <td>{num(l.Quantidade)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </section>
  );
}

function ReceitasDoDia({ onErro }: { onErro: (e: string) => void }) {
  const [vendas, setVendas] = useState<domain.Venda[] | null>(null);
  const [vendaAberta, setVendaAberta] = useState<domain.Venda | null>(null);

  useEffect(() => {
    ListarVendasDoDia()
      .then(setVendas)
      .catch((e) => onErro(String(e)));
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  }, []);

  return (
    <section>
      <h3 style={{ margin: "16px 0 0" }}>Receitas médicas (opcional)</h3>
      <p style={{ color: "var(--text-secondary)", fontSize: "0.85rem", margin: "4px 0 8px" }}>
        A receita não é obrigatória para vender. Se o cliente apresentou uma, associe-a aqui à venda do dia.
      </p>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Venda</th>
            <th>Hora</th>
            <th>Total</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {vendas === null && (
            <tr>
              <td colSpan={4}>A carregar…</td>
            </tr>
          )}
          {vendas?.length === 0 && (
            <tr>
              <td colSpan={4}>Ainda não há vendas hoje.</td>
            </tr>
          )}
          {vendas?.map((v) => (
            <tr key={v.ID}>
              <td>#{v.ID}</td>
              <td>{new Date(v.CriadoEm).toLocaleTimeString("pt-PT", { hour: "2-digit", minute: "2-digit" })}</td>
              <td>{mt(v.Total)}</td>
              <td className="admin-form">
                <button className="admin-btn-secundario" onClick={() => setVendaAberta(v)}>
                  <ClipboardDocumentListIcon className="btn-icone" aria-hidden="true" />
                  Receitas
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {vendaAberta && (
        <Modal titulo={`Receitas — venda #${vendaAberta.ID}`} onFechar={() => setVendaAberta(null)}>
          <ReceitasDaVenda venda={vendaAberta} onFechar={() => setVendaAberta(null)} onErro={onErro} />
        </Modal>
      )}
    </section>
  );
}

function ReceitasDaVenda({
  venda,
  onFechar,
  onErro,
}: {
  venda: domain.Venda;
  onFechar: () => void;
  onErro: (e: string) => void;
}) {
  const [receitas, setReceitas] = useState<domain.ReceitaMedica[] | null>(null);
  const [medico, setMedico] = useState("");
  const [numeroReceita, setNumeroReceita] = useState("");
  const [aGuardar, setAGuardar] = useState(false);

  function carregar() {
    ListarReceitasPorVenda(venda.ID)
      .then(setReceitas)
      .catch((e) => onErro(String(e)));
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- venda fixa por instância do modal
  }, []);

  function guardar() {
    if (!medico || !numeroReceita || aGuardar) return;
    const p = new usecase.ReceitaMedicaInput();
    p.VendaID = venda.ID;
    p.ClienteID = venda.ClienteID;
    p.Medico = medico;
    p.NumeroReceita = numeroReceita;
    setAGuardar(true);
    RegistarReceitaMedica(p)
      .then(() => {
        setMedico("");
        setNumeroReceita("");
        carregar();
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Médico</th>
            <th>N.º receita</th>
            <th>Registada</th>
          </tr>
        </thead>
        <tbody>
          {receitas?.length === 0 && (
            <tr>
              <td colSpan={3}>Sem receitas associadas.</td>
            </tr>
          )}
          {receitas?.map((r) => (
            <tr key={r.ID}>
              <td>{r.Medico}</td>
              <td>{r.NumeroReceita}</td>
              <td>{new Date(r.CriadoEm).toLocaleString("pt-PT")}</td>
            </tr>
          ))}
        </tbody>
      </table>

      <div className="admin-form">
        <input placeholder="Médico" value={medico} onChange={(e) => setMedico(e.target.value)} />
        <input placeholder="N.º da receita" value={numeroReceita} onChange={(e) => setNumeroReceita(e.target.value)} />
      </div>
      <div className="admin-form">
        <button className="admin-btn" disabled={!medico || !numeroReceita || aGuardar} onClick={guardar}>
          <LinkIcon className="btn-icone" aria-hidden="true" />
          {aGuardar ? "A guardar…" : "Associar receita"}
        </button>
        <button className="admin-btn-secundario" onClick={onFechar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Fechar
        </button>
      </div>
    </div>
  );
}
