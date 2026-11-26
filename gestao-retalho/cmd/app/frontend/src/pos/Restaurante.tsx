import { useEffect, useState } from "react";
import {
  RectangleGroupIcon,
  BanknotesIcon,
  DevicePhoneMobileIcon,
  BoltIcon,
  LockOpenIcon,
  LockClosedIcon,
  ArrowPathIcon,
} from "@heroicons/react/24/outline";
import {
  EstadoCaixa,
  AbrirCaixa,
  FecharCaixa,
  EstadoServidorAparelhos,
  DefinirServidorAparelhos,
  NovaChaveServidor,
} from "../../wailsjs/go/api/App";
import { api, usecase } from "../../wailsjs/go/models";
import Mesas from "./Mesas";
import { Folha } from "./Bottlestore";
import { textoErro } from "./comum";
import { METODOS, ROTULO_METODO, lerNumero, mt } from "./formato";
import { ponte, type EstadoLigacao } from "../plataforma";
import "./Bottlestore.css";
import "./Mesas.css";

type Separador = "mesas" | "caixa" | "aparelhos" | "venda";

function lerSeparador(): Separador {
  try {
    const s = sessionStorage.getItem("rest-separador");
    if (s === "caixa" || s === "aparelhos" || s === "venda") return s;
  } catch {
    /* sem armazenamento */
  }
  return "mesas";
}

/**
 * Balcão do restaurante: as mesas de todos os garçons (fecha-se aqui), o
 * caixa (abrir com o fundo, fechar com a contagem), os aparelhos dos
 * garçons ligados a este PC e a venda ao balcão de sempre.
 */
export default function Restaurante({ operador, venda }: { operador: usecase.Operador; venda?: React.ReactNode }) {
  const [sep, setSepEstado] = useState<Separador>(lerSeparador);
  const setSep = (s: Separador) => {
    setSepEstado(s);
    try {
      sessionStorage.setItem("rest-separador", s);
    } catch {
      /* sem armazenamento */
    }
  };

  const separadores: { chave: Separador; rotulo: string; Icone: typeof BoltIcon }[] = [
    { chave: "mesas", rotulo: "Mesas", Icone: RectangleGroupIcon },
    { chave: "caixa", rotulo: "Caixa", Icone: BanknotesIcon },
    { chave: "aparelhos", rotulo: "Aparelhos", Icone: DevicePhoneMobileIcon },
    ...(venda ? [{ chave: "venda" as const, rotulo: "Venda", Icone: BoltIcon }] : []),
  ];

  return (
    <div className="rest-raiz">
      <div className="bs-modos" role="tablist">
        {separadores.map(({ chave, rotulo, Icone }) => (
          <button key={chave} role="tab" aria-selected={sep === chave} className={sep === chave ? "ativo" : ""} onClick={() => setSep(chave)}>
            <Icone className="bs-icon" aria-hidden="true" />
            {rotulo}
          </button>
        ))}
      </div>
      {sep === "mesas" && <Mesas balcao operador={operador} onIrParaCaixa={() => setSep("caixa")} />}
      {sep === "caixa" && <Caixa />}
      {sep === "aparelhos" && <Aparelhos />}
      {sep === "venda" && venda}
    </div>
  );
}

// --- Caixa ---------------------------------------------------------------------

const hora = (t: unknown) => new Date(String(t)).toLocaleString("pt-PT", { dateStyle: "short", timeStyle: "short" });

function Caixa() {
  const [estado, setEstado] = useState<usecase.EstadoCaixa | null>(null);
  const [fundo, setFundo] = useState("");
  const [aFechar, setAFechar] = useState(false);
  const [fecho, setFecho] = useState<usecase.FechoSessao | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = () =>
    EstadoCaixa()
      .then(setEstado)
      .catch((e) => setErro(textoErro(e)));
  useEffect(() => {
    carregar();
  }, []);

  function abrir() {
    const v = fundo.trim() === "" ? 0 : lerNumero(fundo);
    if (!Number.isFinite(v) || v < 0) {
      setErro("Fundo de caixa inválido");
      return;
    }
    AbrirCaixa(v)
      .then(() => {
        setFundo("");
        setFecho(null);
        carregar();
      })
      .catch((e) => setErro(textoErro(e)));
  }

  if (!estado) return erro ? <div className="bs-erro">{erro}</div> : null;

  return (
    <div className="rest-painel">
      {erro && (
        <div className="bs-erro" role="alert">
          {erro}
          <button onClick={() => setErro(null)} aria-label="Fechar">
            ×
          </button>
        </div>
      )}
      {fecho && <ConferenciaCaixa fecho={fecho} />}
      {!estado.Aberto ? (
        <div className="rest-cartao">
          <h3>
            <LockClosedIcon className="bs-icon" aria-hidden="true" /> Caixa fechado
          </h3>
          <p className="rest-fraco">Conte o dinheiro de troco que fica na gaveta e abra o caixa. As mesas só se fecham com o caixa aberto.</p>
          <div className="rest-linha">
            <label className="bs-campo" style={{ flex: 1 }}>
              <span>Fundo de caixa (MT)</span>
              <input inputMode="decimal" placeholder="0,00" value={fundo} onChange={(e) => setFundo(e.target.value)} onKeyDown={(e) => e.key === "Enter" && abrir()} />
            </label>
          </div>
          <button className="bs-btn" onClick={abrir}>
            <LockOpenIcon className="bs-icon" aria-hidden="true" />
            Abrir caixa
          </button>
        </div>
      ) : (
        <div className="rest-cartao">
          <h3>
            <LockOpenIcon className="bs-icon" aria-hidden="true" /> Caixa aberto
          </h3>
          <p className="rest-fraco">
            Aberto por {estado.AbertoPor} em {hora(estado.Sessao?.AbertaEm)} · fundo {mt(estado.Sessao?.FundoInicial ?? 0)}
          </p>
          <table className="rest-tabela">
            <thead>
              <tr>
                <th>Método</th>
                <th className="num">Vendas</th>
                <th className="num">Dívidas recebidas</th>
                <th className="num">Total</th>
              </tr>
            </thead>
            <tbody>
              {estado.Recebido.PorMetodo.map((l) => (
                <tr key={l.Metodo}>
                  <td>{ROTULO_METODO[l.Metodo] ?? l.Metodo}</td>
                  <td className="num">{mt(l.Vendas)}</td>
                  <td className="num">{mt(l.Dividas)}</td>
                  <td className="num">{mt(l.Total)}</td>
                </tr>
              ))}
              {estado.Recebido.PorMetodo.length === 0 && (
                <tr>
                  <td colSpan={4} className="rest-fraco">
                    Ainda não entrou nada neste turno.
                  </td>
                </tr>
              )}
            </tbody>
          </table>
          <p>
            Dinheiro que deve estar na gaveta: <strong>{mt(estado.DinheiroEsperado)}</strong>
          </p>
          <div className="rest-linha">
            <button className="bs-btn-sec" onClick={carregar}>
              <ArrowPathIcon className="bs-icon" aria-hidden="true" />
              Atualizar
            </button>
            <button className="bs-btn" onClick={() => setAFechar(true)}>
              <LockClosedIcon className="bs-icon" aria-hidden="true" />
              Fechar caixa
            </button>
          </div>
        </div>
      )}
      {aFechar && estado.Aberto && (
        <FecharCaixaFolha
          estado={estado}
          onCancelar={() => setAFechar(false)}
          onFechado={(f) => {
            setAFechar(false);
            setFecho(f);
            carregar();
          }}
        />
      )}
    </div>
  );
}

function FecharCaixaFolha({
  estado,
  onCancelar,
  onFechado,
}: {
  estado: usecase.EstadoCaixa;
  onCancelar: () => void;
  onFechado: (f: usecase.FechoSessao) => void;
}) {
  const metodos = METODOS.filter((m) => m === "dinheiro" || estado.Recebido.PorMetodo.some((l) => l.Metodo === m && l.Total > 0));
  const esperado = (m: string) =>
    (m === "dinheiro" ? (estado.Sessao?.FundoInicial ?? 0) : 0) + (estado.Recebido.PorMetodo.find((l) => l.Metodo === m)?.Total ?? 0);
  const [contado, setContado] = useState<Record<string, string>>({});
  const [obs, setObs] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aFechar, setAFechar] = useState(false);
  const valor = (m: string) => (contado[m]?.trim() ? lerNumero(contado[m]) : NaN);
  const completo = metodos.every((m) => Number.isFinite(valor(m)) && valor(m) >= 0);

  function fechar() {
    if (!completo || aFechar) return;
    const c: Record<string, number> = {};
    for (const m of metodos) c[m] = valor(m);
    setAFechar(true);
    FecharCaixa(c, obs)
      .then(onFechado)
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setAFechar(false));
  }

  return (
    <Folha titulo="Fechar caixa" onFechar={onCancelar} larga>
      <p className="bs-folha-texto">Conte o que tem em cada forma de pagamento (o dinheiro da gaveta com o fundo, o total do M-Pesa, do cartão…).</p>
      <table className="rest-tabela">
        <thead>
          <tr>
            <th>Método</th>
            <th className="num">Contado</th>
          </tr>
        </thead>
        <tbody>
          {metodos.map((m) => (
            <tr key={m}>
              <td>{ROTULO_METODO[m]}</td>
              <td className="num">
                <input
                  inputMode="decimal"
                  aria-label={`Contado em ${ROTULO_METODO[m]}`}
                  placeholder={mt(esperado(m)).replace(/\s*MT$/, "")}
                  value={contado[m] ?? ""}
                  onChange={(e) => setContado((c) => ({ ...c, [m]: e.target.value }))}
                />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <label className="bs-campo">
        <span>Observações (opcional)</span>
        <input value={obs} onChange={(e) => setObs(e.target.value)} />
      </label>
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        <button className="bs-btn-sec" onClick={onCancelar}>
          Voltar
        </button>
        <button className="bs-btn" disabled={!completo || aFechar} onClick={fechar}>
          {aFechar ? "A fechar…" : "Fechar caixa"}
        </button>
      </div>
    </Folha>
  );
}

function ConferenciaCaixa({ fecho }: { fecho: usecase.FechoSessao }) {
  return (
    <div className="rest-cartao">
      <h3>Caixa fechado — conferência</h3>
      <p className="rest-fraco">
        {hora(fecho.Sessao.AbertaEm)} → {hora(fecho.Sessao.FechadaEm)} · aberto por {fecho.AbertoPor}, fechado por {fecho.FechadoPor}
      </p>
      <table className="rest-tabela">
        <thead>
          <tr>
            <th>Método</th>
            <th className="num">Esperado</th>
            <th className="num">Contado</th>
            <th className="num">Diferença</th>
          </tr>
        </thead>
        <tbody>
          {fecho.Linhas.map((l) => (
            <tr key={l.Metodo}>
              <td>{ROTULO_METODO[l.Metodo] ?? l.Metodo}</td>
              <td className="num">{mt(l.Esperado)}</td>
              <td className="num">{mt(l.Contado)}</td>
              <td className={`num${l.Diferenca < -0.005 ? " rest-dif-falta" : l.Diferenca > 0.005 ? " rest-dif-sobra" : ""}`}>{mt(l.Diferenca)}</td>
            </tr>
          ))}
        </tbody>
        <tfoot>
          <tr>
            <td colSpan={3}>
              <strong>{fecho.DiferencaTotal < -0.005 ? "Falta" : fecho.DiferencaTotal > 0.005 ? "Sobra" : "Certo"}</strong>
            </td>
            <td className="num">
              <strong>{mt(fecho.DiferencaTotal)}</strong>
            </td>
          </tr>
        </tfoot>
      </table>
      {fecho.Sessao.Observacoes && <p className="rest-fraco">Obs.: {fecho.Sessao.Observacoes}</p>}
    </div>
  );
}

// --- Aparelhos dos garçons ------------------------------------------------------------

function Aparelhos() {
  const [e, setE] = useState<api.EstadoServidor | null>(null);
  const [erro, setErro] = useState<string | null>(null);

  const carregar = () =>
    EstadoServidorAparelhos()
      .then(setE)
      .catch((x) => setErro(textoErro(x)));
  useEffect(() => {
    carregar();
    const t = window.setInterval(carregar, 10000);
    return () => window.clearInterval(t);
  }, []);

  if (!e) return erro ? <div className="bs-erro">{erro}</div> : null;
  if (!e.Disponivel) {
    return <p className="rest-fraco">Os aparelhos ligam-se ao computador do balcão — configure-os lá.</p>;
  }
  const mudar = (ativo: boolean) =>
    DefinirServidorAparelhos(ativo, e.Config.Porta)
      .then(setE)
      .catch((x) => {
        setErro(textoErro(x));
        carregar();
      });

  return (
    <div className="rest-painel">
      {erro && (
        <div className="bs-erro" role="alert">
          {erro}
          <button onClick={() => setErro(null)} aria-label="Fechar">
            ×
          </button>
        </div>
      )}
      <div className="rest-cartao">
        <h3>Aparelhos dos garçons</h3>
        <p className="rest-fraco">
          Os aparelhos (Sunmi) ligam-se a este computador pela rede Wi-Fi da loja. Este PDV tem de estar aberto durante o serviço.
        </p>
        <div className="rest-linha">
          <strong>{e.Ligado ? "Ligado" : "Desligado"}</strong>
          <button className={e.Ligado ? "bs-btn-sec" : "bs-btn"} onClick={() => mudar(!e.Ligado)}>
            {e.Ligado ? "Desligar" : "Ligar"}
          </button>
        </div>
        {e.Erro && <div className="bs-aviso erro">{e.Erro}</div>}
      </div>

      <div className="rest-cartao">
        <h3>Para ligar um aparelho</h3>
        <p className="rest-fraco">No aparelho: Administração → "Aparelho de garçom", ou o botão "Balcão" no ecrã do operador. Escreva:</p>
        <p>
          Endereço:{" "}
          {e.Enderecos.length > 0 ? (
            e.Enderecos.map((ip) => (
              <span key={ip} className="rest-chave" style={{ fontSize: 18, marginRight: 8 }}>
                {ip}
                {e.Config.Porta !== 8765 ? `:${e.Config.Porta}` : ""}
              </span>
            ))
          ) : (
            <em>este computador não está ligado a nenhuma rede</em>
          )}
        </p>
        <p>
          Chave da loja: <span className="rest-chave">{e.Config.Chave}</span>
        </p>
        <button
          className="bs-btn-sec"
          onClick={() => {
            if (window.confirm("Gerar uma chave nova? Todos os aparelhos têm de a escrever outra vez.")) {
              NovaChaveServidor()
                .then(setE)
                .catch((x) => setErro(textoErro(x)));
            }
          }}
        >
          Gerar chave nova
        </button>
      </div>

      <div className="rest-cartao">
        <h3>Ligados agora</h3>
        <table className="rest-tabela">
          <thead>
            <tr>
              <th>Aparelho</th>
              <th>Garçom</th>
              <th>Última vez</th>
            </tr>
          </thead>
          <tbody>
            {e.Aparelhos.map((a) => (
              <tr key={a.ID}>
                <td>{a.ID}</td>
                <td>{a.Operador || <span className="rest-fraco">ninguém</span>}</td>
                <td>{new Date(String(a.Visto)).toLocaleTimeString("pt-PT", { hour: "2-digit", minute: "2-digit" })}</td>
              </tr>
            ))}
            {e.Aparelhos.length === 0 && (
              <tr>
                <td colSpan={3} className="rest-fraco">
                  Nenhum aparelho ligou desde que o PDV abriu.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

// --- No aparelho: ligação ao computador do balcão ----------------------------------------

/** Liga o aparelho (modo garçom) ao PDV do computador do balcão. Só Android. */
export function LigacaoBalcao({ onFechar, onMudou }: { onFechar: () => void; onMudou: (e: EstadoLigacao) => void }) {
  const [estado, setEstado] = useState<EstadoLigacao | null>(null);
  const [endereco, setEndereco] = useState("");
  const [chave, setChave] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aLigar, setALigar] = useState(false);

  useEffect(() => {
    ponte?.EstadoLigacao().then((e) => {
      setEstado(e);
      setEndereco(e.Endereco.replace(/:8765$/, ""));
    });
  }, []);

  if (!ponte) return null;

  function ligar() {
    setALigar(true);
    setErro(null);
    ponte!
      .LigarAoBalcao(endereco, chave)
      .then(onMudou)
      .catch((e) => setErro(textoErro(e)))
      .finally(() => setALigar(false));
  }

  return (
    <Folha titulo="Computador do balcão" onFechar={onFechar}>
      <p className="bs-folha-texto">
        Num restaurante, os aparelhos dos garçons usam as mesas, os pedidos e as impressoras do computador do balcão. O endereço e a chave
        aparecem lá, no PDV, em <strong>Aparelhos</strong>.
      </p>
      {estado?.ModoGarcom && (
        <div className="bs-aviso">
          Ligado a <strong>{estado.Endereco}</strong>.
        </div>
      )}
      <label className="bs-campo">
        <span>Endereço do computador</span>
        <input inputMode="decimal" placeholder="192.168.1.10" value={endereco} onChange={(e) => setEndereco(e.target.value)} />
      </label>
      <label className="bs-campo">
        <span>Chave da loja</span>
        <input autoCapitalize="characters" placeholder="ABCD-EFGH" value={chave} onChange={(e) => setChave(e.target.value)} />
      </label>
      {erro && <div className="bs-aviso erro">{erro}</div>}
      <div className="bs-folha-acoes">
        {estado?.ModoGarcom ? (
          <button
            className="bs-btn-sec"
            onClick={() =>
              ponte!
                .DesligarDoBalcao()
                .then(onMudou)
                .catch((e) => setErro(textoErro(e)))
            }
          >
            Usar só este aparelho
          </button>
        ) : (
          <button className="bs-btn-sec" onClick={onFechar}>
            Voltar
          </button>
        )}
        <button className="bs-btn" disabled={!endereco.trim() || !chave.trim() || aLigar} onClick={ligar}>
          {aLigar ? "A ligar…" : "Ligar"}
        </button>
      </div>
    </Folha>
  );
}
