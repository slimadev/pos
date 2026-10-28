import { useEffect, useState } from "react";
import {
  DocumentArrowDownIcon,
  TableCellsIcon,
  DocumentCheckIcon,
  PowerIcon,
} from "@heroicons/react/24/outline";
import {
  EstadoFaturacao,
  DefinirFaturacaoAtiva,
  ListarDocumentosMes,
  VendasSemDocumento,
  EmitirDocumentoEmFalta,
  AbrirDocumentoPDF,
  ExportarDocumentosMes,
} from "../wailsjs/go/admin/AdminApp";
import { repository, usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import { mt, num } from "./formato";

const dataHora = (d: unknown) =>
  new Date(String(d)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" });

function mesAtual(): string {
  const d = new Date();
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}`;
}

/**
 * Documentos fiscais: Fatura-Recibo (venda paga), Fatura (venda com parte
 * por pagar), Recibo (pagamento de dívida) e Nota de Crédito (anulação).
 * São emitidos sozinhos com cada operação quando a emissão está ligada —
 * o que só se deve fazer depois de o software estar aprovado pela AT.
 * Depois de emitido, um documento não se altera nem se apaga.
 */
export default function FaturasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [estado, setEstado] = useState<usecase.EstadoFaturacao | null>(null);
  const [mes, setMes] = useState(mesAtual());
  const [docs, setDocs] = useState<usecase.DocumentoCompleto[] | null>(null);
  const [emFalta, setEmFalta] = useState<repository.VendaSemFatura[]>([]);
  const [aMudar, setAMudar] = useState(false);

  const [ano, numMes] = mes.split("-").map(Number);

  function carregar() {
    EstadoFaturacao()
      .then(setEstado)
      .catch((e) => onErro(String(e)));
    VendasSemDocumento()
      .then(setEmFalta)
      .catch((e) => onErro(String(e)));
    if (ano && numMes) {
      ListarDocumentosMes(ano, numMes)
        .then(setDocs)
        .catch((e) => onErro(String(e)));
    }
  }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- recarrega quando o mês muda
  useEffect(carregar, [mes]);

  function mudarEstado(ativa: boolean) {
    const texto = ativa
      ? "Ligar a emissão de documentos fiscais?\n\nSó deve ligar depois de o software estar aprovado pela Autoridade Tributária. A partir daqui, cada venda emite uma Fatura-Recibo ou Fatura numerada, que não se pode apagar."
      : "Desligar a emissão de documentos fiscais? As vendas seguintes ficam sem fatura.";
    if (!window.confirm(texto)) return;
    setAMudar(true);
    DefinirFaturacaoAtiva(ativa)
      .then(() => carregar())
      .catch((e) => onErro(String(e)))
      .finally(() => setAMudar(false));
  }

  function emitir(vendaID: number) {
    EmitirDocumentoEmFalta(vendaID)
      .then(() => carregar())
      .catch((e) => onErro(String(e)));
  }

  const somaTotal = (docs ?? []).reduce((s, c) => {
    if (c.Documento.Tipo === "RC") return s;
    return s + (c.Documento.Tipo === "NC" ? -1 : 1) * c.Documento.Total;
  }, 0);
  const somaIVA = (docs ?? []).reduce((s, c) => {
    if (c.Documento.Tipo === "RC") return s;
    return s + (c.Documento.Tipo === "NC" ? -1 : 1) * c.Documento.ValorIVA;
  }, 0);

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Faturas" }]} />
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>Faturas</h2>
          <div className="admin-texto-fraco">Faturas-recibo, faturas, recibos e notas de crédito emitidos.</div>
        </div>
      </div>

      {estado && (
        <div className={`admin-faturacao-estado ${estado.Ativa ? "ligada" : "desligada"}`}>
          <div>
            <strong>{estado.Ativa ? "Emissão de documentos fiscais ligada" : "Emissão de documentos fiscais desligada"}</strong>
            <div>
              {estado.Ativa
                ? `Desde ${dataHora(estado.Desde)}. Cada venda emite uma Fatura-Recibo (paga) ou Fatura (com parte por pagar); pagamentos de dívida emitem Recibo e anulações Nota de Crédito.`
                : "Ligue só depois de o software estar aprovado pela Autoridade Tributária. Até lá, as vendas continuam com o recibo interno (sem valor fiscal)."}
            </div>
          </div>
          <button
            className={estado.Ativa ? "admin-btn-secundario" : "admin-btn"}
            disabled={aMudar}
            onClick={() => mudarEstado(!estado.Ativa)}
          >
            <PowerIcon className="btn-icone" aria-hidden="true" />
            {estado.Ativa ? "Desligar" : "Ligar emissão"}
          </button>
        </div>
      )}

      {emFalta.length > 0 && (
        <div className="admin-inventario-aviso warn">
          <strong>{emFalta.length} venda(s) sem documento</strong> — a emissão falhou no momento da venda.
          <div className="admin-faturas-falta">
            {emFalta.map((v) => (
              <span key={v.VendaID}>
                Venda #{v.VendaID} · {mt(v.Total)}{" "}
                <button className="admin-btn-secundario" onClick={() => emitir(v.VendaID)}>
                  <DocumentCheckIcon className="btn-icone" aria-hidden="true" />
                  Emitir
                </button>
              </span>
            ))}
          </div>
        </div>
      )}

      <div className="admin-form">
        <label>
          Mês <input type="month" value={mes} onChange={(e) => setMes(e.target.value)} />
        </label>
        <button
          className="admin-btn-secundario"
          disabled={!docs || docs.length === 0}
          title="Ficheiro mensal dos documentos emitidos (estrutura provisória até a AT publicar a do SAF-T)"
          onClick={() => ExportarDocumentosMes(ano, numMes).catch((e) => onErro(String(e)))}
        >
          <TableCellsIcon className="admin-btn-icon-inline" aria-hidden="true" />
          Exportar o mês (Excel)
        </button>
        {docs && docs.length > 0 && (
          <span className="admin-texto-fraco">
            {docs.length} documento(s) · total {mt(somaTotal)} · IVA {mt(somaIVA)}
          </span>
        )}
      </div>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Documento</th>
            <th>Data</th>
            <th>Adquirente</th>
            <th>NUIT</th>
            <th style={{ textAlign: "right" }}>Total</th>
            <th style={{ textAlign: "right" }}>IVA</th>
            <th>Refere-se a</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {(docs ?? []).map((c) => (
            <tr key={c.Documento.ID}>
              <td style={{ whiteSpace: "nowrap" }}>
                <strong>{c.Codigo}</strong>
                <div className="admin-texto-fraco">{c.NomeTipo}</div>
              </td>
              <td>{dataHora(c.Documento.CriadoEm)}</td>
              <td>{c.Documento.ClienteNome}</td>
              <td>{c.Documento.ClienteNUIT ?? "—"}</td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                {c.Documento.Tipo === "NC" ? "−" : ""}
                {mt(c.Documento.Total)}
              </td>
              <td style={{ textAlign: "right", whiteSpace: "nowrap" }}>
                {c.Documento.Tipo === "RC" ? "—" : `${mt(c.Documento.ValorIVA)} (${num(c.Documento.TaxaIVA)}%)`}
              </td>
              <td>{c.OrigemCodigo || "—"}</td>
              <td>
                <div className="admin-acoes">
                  <button className="admin-btn-secundario" onClick={() => AbrirDocumentoPDF(c.Documento.ID).catch((e) => onErro(String(e)))}>
                    <DocumentArrowDownIcon className="admin-btn-icon-inline" aria-hidden="true" />
                    PDF
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {docs !== null && docs.length === 0 && (
            <tr>
              <td colSpan={8}>Nenhum documento emitido neste mês.</td>
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}
