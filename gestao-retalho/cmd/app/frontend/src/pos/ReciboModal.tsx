import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { ObterRecibo, GerarRecibo } from "../../wailsjs/go/api/App";
import { usecase } from "../../wailsjs/go/models";
import { mt, qtd } from "./formato";
import "./ReciboModal.css";

/**
 * Recibo da venda num modal com o aspeto do talão (na largura do papel
 * configurado: rolo de 80/58 mm ou A5) e "Imprimir" pela própria janela —
 * em vez de abrir o PDF noutro programa. Ao imprimir, o CSS de impressão
 * esconde tudo menos o talão. "Guardar PDF" continua disponível.
 */
export default function ReciboModal({
  vendaId,
  troco = 0,
  onFechar,
}: {
  vendaId: number;
  // Troco dado na hora (só se sabe logo a seguir à venda; a venda guarda o
  // valor líquido em dinheiro).
  troco?: number;
  onFechar: () => void;
}) {
  const [recibo, setRecibo] = useState<usecase.ReciboEcra | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    ObterRecibo(vendaId)
      .then(setRecibo)
      .catch((e) => setErro(String(e).replace(/^Error:\s*/, "")));
  }, [vendaId]);

  useEffect(() => {
    function aoTeclar(e: KeyboardEvent) {
      if (e.key === "Escape") onFechar();
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "p") {
        e.preventDefault();
        window.print();
      }
    }
    window.addEventListener("keydown", aoTeclar);
    return () => window.removeEventListener("keydown", aoTeclar);
  }, [onFechar]);

  function guardarPDF() {
    GerarRecibo(vendaId).catch((e) => setErro(String(e).replace(/^Error:\s*/, "")));
  }

  const d = recibo;
  const papel = recibo?.Papel ?? "80";
  // A venda guarda o dinheiro já sem o troco; no talão mostra-se o que o
  // cliente entregou e o troco, como ele contou.
  const iDinheiro = d ? d.Pagamentos.map((p) => p.Metodo).lastIndexOf("Dinheiro") : -1;
  const pagamentos = (d?.Pagamentos ?? []).map((p, i) =>
    troco > 0 && i === iDinheiro ? { Metodo: "Dinheiro entregue", Valor: p.Valor + troco } : p,
  );

  return createPortal(
    <div className="recibo-overlay">
      <div className="recibo-janela" role="dialog" aria-label={`Recibo da venda ${vendaId}`}>
        <div className="recibo-barra">
          <span>{d?.DocumentoCodigo ? `${d.DocumentoNome} ${d.DocumentoCodigo}` : `Recibo #${vendaId}`}</span>
          <button className="recibo-fechar" onClick={onFechar} aria-label="Fechar">
            ×
          </button>
        </div>

        <div className="recibo-fundo">
          {erro && <div className="recibo-erro">{erro}</div>}
          {!d && !erro && <div className="recibo-a-carregar">A preparar o recibo…</div>}
          {d && (
            <div className={`recibo-papel papel-${papel}`}>
              {d.Logo && <img className="recibo-logo" src={d.Logo} alt="" />}
              <div className="recibo-loja">{d.NegocioNome}</div>
              {d.LinhasLoja.map((l) => (
                <div key={l} className="recibo-centro recibo-contacto">
                  {l}
                </div>
              ))}
              {d.DocumentoCodigo ? (
                <div className="recibo-centro recibo-documento">
                  {d.DocumentoNome} {d.DocumentoCodigo}
                </div>
              ) : (
                <div className="recibo-centro">Recibo #{d.VendaID}</div>
              )}
              <div className="recibo-centro">
                {new Date(String(d.CriadoEm)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" })}
              </div>
              <div>Cliente: {d.ClienteNome}</div>
              {d.ClienteNUIT && <div>NUIT: {d.ClienteNUIT}</div>}
              <hr />
              {d.Itens.map((item, i) => (
                <div key={i} className="recibo-item">
                  <div className="recibo-item-nome">{item.NomeProduto}</div>
                  <div className="recibo-linha">
                    <span>
                      {qtd(item.Quantidade)} {item.NomeUnidade} × {mt(item.PrecoUnitario).replace(" MT", "")}
                      {item.Desconto > 0 && ` −${mt(item.Desconto).replace(" MT", "")}`}
                    </span>
                    <span>{mt(item.Subtotal).replace(" MT", "")}</span>
                  </div>
                </div>
              ))}
              <hr />
              <div className="recibo-linha">
                <span>Subtotal</span>
                <span>{mt(d.Subtotal)}</span>
              </div>
              {d.DescontoTotal > 0 && (
                <div className="recibo-linha">
                  <span>Desconto</span>
                  <span>−{mt(d.DescontoTotal)}</span>
                </div>
              )}
              {d.DocumentoCodigo && (
                <>
                  <div className="recibo-linha">
                    <span>Base tributável</span>
                    <span>{mt(d.Total - d.ValorIVA)}</span>
                  </div>
                  <div className="recibo-linha">
                    <span>{d.TaxaIVA > 0 ? `IVA (${d.TaxaIVA.toLocaleString("pt-PT")}%)` : "IVA (isento)"}</span>
                    <span>{mt(d.ValorIVA)}</span>
                  </div>
                </>
              )}
              <div className="recibo-linha recibo-total">
                <span>TOTAL</span>
                <span>{mt(d.Total)}</span>
              </div>
              {pagamentos.map((p, i) => (
                <div key={i} className="recibo-linha">
                  <span>{p.Metodo}</span>
                  <span>{mt(p.Valor)}</span>
                </div>
              ))}
              {troco > 0 && (
                <div className="recibo-linha">
                  <span>Troco</span>
                  <span>{mt(troco)}</span>
                </div>
              )}
              <hr />
              <div className="recibo-centro">Obrigado pela preferência!</div>
              {d.DocumentoCodigo ? (
                <div className="recibo-centro recibo-rodape">Processado por computador — Gestão de Retalho</div>
              ) : (
                <div className="recibo-centro recibo-rodape">Recibo interno — não serve de fatura</div>
              )}
            </div>
          )}
        </div>

        <div className="recibo-acoes">
          <button className="recibo-btn" onClick={onFechar}>
            Fechar
          </button>
          <button className="recibo-btn" onClick={guardarPDF} disabled={!d}>
            Guardar PDF
          </button>
          <button className="recibo-btn recibo-btn-imprimir" onClick={() => window.print()} disabled={!d}>
            Imprimir
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}
