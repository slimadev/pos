import { useEffect, useState } from "react";
import { PlusIcon, PencilSquareIcon, PrinterIcon, PowerIcon, KeyIcon, EyeIcon } from "@heroicons/react/24/outline";
import {
  ObterConfigRestaurante,
  DefinirModoRestaurante,
  GuardarImpressora,
  TestarImpressora,
  DefinirRotaCategoria,
  ListarCategorias,
  HistoricoCaixa,
  ObterFechoCaixa,
  NovaChaveServidor,
} from "../wailsjs/go/admin/AdminApp";
import { admin, domain, usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";
import { mt } from "./formato";

const ROTULO_METODO: Record<string, string> = {
  dinheiro: "Dinheiro",
  mpesa: "M-Pesa",
  emola: "e-Mola",
  cartao: "Cartão",
  transferencia: "Transferência",
};

const dataHora = (d: unknown) => (d ? new Date(String(d)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" }) : "—");

/**
 * Restaurante: liga as mesas no PDV (garçons com aparelhos, fecho no
 * caixa), regista as impressoras térmicas de rede (balcão e cozinha/bar) e
 * diz para que impressora vai cada categoria de produtos. Mostra também
 * os fechos de caixa.
 */
export default function RestauranteScreen({ onErro }: { onErro: (e: string) => void }) {
  const [cfg, setCfg] = useState<admin.ConfigRestaurante | null>(null);
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [sessoes, setSessoes] = useState<domain.SessaoCaixa[]>([]);
  const [editar, setEditar] = useState<domain.Impressora | null>(null);
  const [fecho, setFecho] = useState<usecase.FechoSessao | null>(null);
  const [aviso, setAviso] = useState<string | null>(null);

  function carregar() {
    ObterConfigRestaurante()
      .then(setCfg)
      .catch((e) => onErro(String(e)));
    ListarCategorias()
      .then((c) => setCategorias(c.filter((x) => x.Ativo).sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"))))
      .catch((e) => onErro(String(e)));
    HistoricoCaixa()
      .then(setSessoes)
      .catch((e) => onErro(String(e)));
  }
  // eslint-disable-next-line react-hooks/exhaustive-deps -- só ao abrir o ecrã
  useEffect(carregar, []);

  if (!cfg) return null;
  const preparacao = cfg.Impressoras.filter((i) => i.Funcao === "preparacao" && i.Ativa);
  const rotaDe = (categoriaID: number) => cfg.Rotas.find((r) => r.CategoriaID === categoriaID)?.ImpressoraID ?? null;

  function testar(i: domain.Impressora) {
    setAviso(null);
    TestarImpressora(i.ID)
      .then(() => setAviso(`Talão de teste enviado para ${i.Nome}.`))
      .catch((e) => onErro(String(e)));
  }

  function novaImpressora() {
    const i = new domain.Impressora();
    i.ID = 0;
    i.Nome = "";
    i.Endereco = "";
    i.Porta = 9100;
    i.Colunas = 48;
    i.Funcao = cfg!.Impressoras.some((x) => x.Funcao === "balcao") ? "preparacao" : "balcao";
    i.Ativa = true;
    setEditar(i);
  }

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Restaurante" }]} />
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>Restaurante</h2>
          <div className="admin-texto-fraco">Mesas e garçons, impressoras da cozinha e do balcão, e o caixa.</div>
        </div>
      </div>

      <div className={`admin-faturacao-estado ${cfg.Ativo ? "ligada" : "desligada"}`}>
        <div>
          <strong>{cfg.Ativo ? "Mesas ligadas" : "Mesas desligadas"}</strong>
          <div>
            {cfg.Ativo
              ? "O PDV do balcão mostra as mesas de todos os garçons, o caixa e os aparelhos. Os garçons lançam os pedidos nos aparelhos (Sunmi) e as contas fecham-se no balcão."
              : "Ligue para usar mesas com garçons: os pedidos saem impressos no balcão e na cozinha, e as contas fecham-se no caixa."}
          </div>
        </div>
        <button
          className={cfg.Ativo ? "admin-btn-secundario" : "admin-btn"}
          onClick={() =>
            DefinirModoRestaurante(!cfg.Ativo)
              .then(carregar)
              .catch((e) => onErro(String(e)))
          }
        >
          <PowerIcon className="btn-icone" aria-hidden="true" />
          {cfg.Ativo ? "Desligar" : "Ligar mesas"}
        </button>
      </div>

      {aviso && <div className="admin-inventario-aviso ok">{aviso}</div>}

      <h3 style={{ marginBottom: 0 }}>Impressoras de rede</h3>
      <p className="admin-texto-fraco" style={{ margin: 0 }}>
        Impressoras térmicas com cabo de rede (Epson TM-T20, Xprinter…). O <strong>balcão</strong> recebe todos os pedidos e imprime as contas;
        as de <strong>preparação</strong> (cozinha, bar) só recebem os produtos das categorias que lhes atribuir abaixo. Dê a cada impressora um
        IP fixo no router.
      </p>
      <div className="admin-form">
        <button className="admin-btn" onClick={novaImpressora}>
          <PlusIcon className="btn-icone" aria-hidden="true" />
          Nova impressora
        </button>
      </div>
      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Nome</th>
            <th>Função</th>
            <th>Endereço</th>
            <th>Papel</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {cfg.Impressoras.map((i) => (
            <tr key={i.ID}>
              <td>
                <strong>{i.Nome}</strong>
              </td>
              <td>{i.Funcao === "balcao" ? "Balcão (tudo + contas)" : "Preparação"}</td>
              <td>
                {i.Endereco}:{i.Porta}
              </td>
              <td>{i.Colunas >= 42 ? "80 mm" : "58 mm"}</td>
              <td>
                <span className={`admin-inventario-estado ${i.Ativa ? "fechado" : "aberto"}`}>{i.Ativa ? "Ativa" : "Inativa"}</span>
              </td>
              <td>
                <div className="admin-acoes">
                  <button className="admin-btn-secundario" onClick={() => testar(i)}>
                    <PrinterIcon className="admin-btn-icon-inline" aria-hidden="true" />
                    Testar
                  </button>
                  <button className="admin-btn-secundario" onClick={() => setEditar(i)}>
                    <PencilSquareIcon className="admin-btn-icon-inline" aria-hidden="true" />
                    Editar
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {cfg.Impressoras.length === 0 && (
            <tr>
              <td colSpan={6}>Ainda não há impressoras. Sem elas, o garçom imprime os pedidos no próprio aparelho.</td>
            </tr>
          )}
        </tbody>
      </table>

      <h3 style={{ marginBottom: 0 }}>Para onde vai cada categoria</h3>
      <p className="admin-texto-fraco" style={{ margin: 0 }}>
        Ex: "Pratos" e "Petiscos" → Cozinha. As categorias sem impressora só saem no balcão (bebidas, por exemplo).
      </p>
      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Categoria</th>
            <th>Impressora de preparação</th>
          </tr>
        </thead>
        <tbody>
          {categorias.map((c) => (
            <tr key={c.ID}>
              <td>{c.Nome}</td>
              <td>
                <select
                  aria-label={`Impressora da categoria ${c.Nome}`}
                  value={rotaDe(c.ID) ?? ""}
                  disabled={preparacao.length === 0}
                  onChange={(e) =>
                    DefinirRotaCategoria(c.ID, e.target.value ? Number(e.target.value) : null)
                      .then(carregar)
                      .catch((x) => onErro(String(x)))
                  }
                >
                  <option value="">Só o balcão</option>
                  {preparacao.map((i) => (
                    <option key={i.ID} value={i.ID}>
                      {i.Nome}
                    </option>
                  ))}
                </select>
              </td>
            </tr>
          ))}
          {categorias.length === 0 && (
            <tr>
              <td colSpan={2}>Sem categorias — crie-as em Catálogo → Categorias.</td>
            </tr>
          )}
        </tbody>
      </table>

      <h3 style={{ marginBottom: 0 }}>Aparelhos dos garçons</h3>
      <p className="admin-texto-fraco" style={{ margin: 0 }}>
        Os aparelhos ligam-se ao PDV do computador do balcão (tem de estar aberto). No aparelho escreve-se o endereço desse computador (aparece no
        PDV, em Aparelhos) e esta chave:
      </p>
      <div className="admin-form">
        <span style={{ fontFamily: "monospace", fontSize: "1.5rem", fontWeight: 700, letterSpacing: 2 }}>{cfg.Servidor.Chave}</span>
        <span className="admin-texto-fraco">porta {cfg.Servidor.Porta}</span>
        <button
          className="admin-btn-secundario"
          onClick={() => {
            if (window.confirm("Gerar uma chave nova? Todos os aparelhos têm de a escrever outra vez.")) {
              NovaChaveServidor()
                .then(carregar)
                .catch((e) => onErro(String(e)));
            }
          }}
        >
          <KeyIcon className="btn-icone" aria-hidden="true" />
          Chave nova
        </button>
      </div>
      <p className="admin-texto-fraco" style={{ margin: 0 }}>
        Quem autoriza anular um pedido já enviado é um <strong>gerente</strong> — marque-o em Utilizadores.
      </p>

      <h3 style={{ marginBottom: 0 }}>Fechos de caixa</h3>
      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Aberto</th>
            <th>Fechado</th>
            <th style={{ textAlign: "right" }}>Fundo</th>
            <th>Obs.</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {sessoes.map((s) => (
            <tr key={s.ID}>
              <td>{dataHora(s.AbertaEm)}</td>
              <td>{s.FechadaEm ? dataHora(s.FechadaEm) : <strong>aberto</strong>}</td>
              <td style={{ textAlign: "right" }}>{mt(s.FundoInicial)}</td>
              <td>{s.Observacoes ?? ""}</td>
              <td>
                {s.FechadaEm && (
                  <button
                    className="admin-btn-secundario"
                    onClick={() =>
                      ObterFechoCaixa(s.ID)
                        .then(setFecho)
                        .catch((e) => onErro(String(e)))
                    }
                  >
                    <EyeIcon className="admin-btn-icon-inline" aria-hidden="true" />
                    Conferência
                  </button>
                )}
              </td>
            </tr>
          ))}
          {sessoes.length === 0 && (
            <tr>
              <td colSpan={5}>O caixa ainda não foi aberto (abre-se no PDV do balcão).</td>
            </tr>
          )}
        </tbody>
      </table>

      {editar && (
        <Modal titulo={editar.ID ? `Editar ${editar.Nome}` : "Nova impressora"} onFechar={() => setEditar(null)}>
          <FormImpressora
            inicial={editar}
            onCancelar={() => setEditar(null)}
            onGuardar={(i) =>
              GuardarImpressora(i).then(() => {
                setEditar(null);
                carregar();
              })
            }
          />
        </Modal>
      )}

      {fecho && (
        <Modal titulo="Conferência do caixa" onFechar={() => setFecho(null)}>
          <p className="admin-texto-fraco">
            {dataHora(fecho.Sessao.AbertaEm)} → {dataHora(fecho.Sessao.FechadaEm)} · aberto por {fecho.AbertoPor}, fechado por {fecho.FechadoPor}
          </p>
          <table className="admin-tabela">
            <thead>
              <tr>
                <th>Método</th>
                <th style={{ textAlign: "right" }}>Esperado</th>
                <th style={{ textAlign: "right" }}>Contado</th>
                <th style={{ textAlign: "right" }}>Diferença</th>
              </tr>
            </thead>
            <tbody>
              {fecho.Linhas.map((l) => (
                <tr key={l.Metodo}>
                  <td>{ROTULO_METODO[l.Metodo] ?? l.Metodo}</td>
                  <td style={{ textAlign: "right" }}>{mt(l.Esperado)}</td>
                  <td style={{ textAlign: "right" }}>{mt(l.Contado)}</td>
                  <td style={{ textAlign: "right", fontWeight: 700 }}>{mt(l.Diferenca)}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <p>
            <strong>
              {fecho.DiferencaTotal < -0.005 ? "Falta" : fecho.DiferencaTotal > 0.005 ? "Sobra" : "Certo"}: {mt(fecho.DiferencaTotal)}
            </strong>
          </p>
        </Modal>
      )}
    </div>
  );
}

function FormImpressora({
  inicial,
  onGuardar,
  onCancelar,
}: {
  inicial: domain.Impressora;
  onGuardar: (i: domain.Impressora) => Promise<void>;
  onCancelar: () => void;
}) {
  const [i, setI] = useState(inicial);
  const [erro, setErro] = useState<string | null>(null);
  const [aGuardar, setAGuardar] = useState(false);
  const mudar = <K extends keyof domain.Impressora>(k: K, v: domain.Impressora[K]) => setI((x) => ({ ...x, [k]: v }) as domain.Impressora);

  function guardar() {
    setAGuardar(true);
    setErro(null);
    onGuardar(i)
      .catch((e) => setErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-produto-form">
      <div className="admin-produto-campos">
        <label className="admin-campo">
          Nome
          <input autoFocus placeholder="ex: Cozinha" value={i.Nome} onChange={(e) => mudar("Nome", e.target.value)} />
        </label>
        <label className="admin-campo">
          Função
          <select value={i.Funcao} onChange={(e) => mudar("Funcao", e.target.value)}>
            <option value="balcao">Balcão — todos os pedidos e as contas</option>
            <option value="preparacao">Preparação — só as categorias atribuídas</option>
          </select>
        </label>
        <label className="admin-campo">
          Endereço IP
          <input inputMode="decimal" placeholder="192.168.1.100" value={i.Endereco} onChange={(e) => mudar("Endereco", e.target.value)} />
        </label>
        <label className="admin-campo">
          Porta
          <input inputMode="numeric" value={i.Porta} onChange={(e) => mudar("Porta", Number(e.target.value.replace(/\D/g, "")) || 0)} />
        </label>
        <label className="admin-campo">
          Papel
          <select value={i.Colunas >= 42 ? 48 : 32} onChange={(e) => mudar("Colunas", Number(e.target.value))}>
            <option value={48}>80 mm</option>
            <option value={32}>58 mm</option>
          </select>
        </label>
        {i.ID !== 0 && (
          <label className="admin-campo" style={{ flexDirection: "row", alignItems: "center", gap: 8 }}>
            <input type="checkbox" checked={i.Ativa} onChange={(e) => mudar("Ativa", e.target.checked)} />
            Ativa
          </label>
        )}
      </div>
      {erro && <div className="admin-inventario-aviso warn">{erro}</div>}
      <div className="admin-form">
        <button className="admin-btn" disabled={aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : "Guardar"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
