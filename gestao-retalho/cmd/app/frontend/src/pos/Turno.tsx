import { useState } from "react";
import { IniciarTurno } from "../../wailsjs/go/api/App";
import { usecase } from "../../wailsjs/go/models";
import "./Turno.css";

const TECLAS = ["1", "2", "3", "4", "5", "6", "7", "8", "9", "", "0", "⌫"];

/**
 * "Quem está a vender?" — o operador escolhe o seu nome e escreve o PIN
 * (4 a 6 algarismos). Não é um login completo: só identifica quem fica com
 * as vendas deste turno. Os operadores e os PIN gerem-se no cmd/admin.
 */
export default function EscolherOperador({
  operadores,
  onIniciado,
}: {
  operadores: usecase.Operador[];
  onIniciado: (op: usecase.Operador) => void;
}) {
  const [escolhido, setEscolhido] = useState<usecase.Operador | null>(null);
  const [pin, setPin] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [aEntrar, setAEntrar] = useState(false);

  function tecla(t: string) {
    setErro(null);
    if (t === "⌫") setPin((p) => p.slice(0, -1));
    else if (t !== "" && pin.length < 6) setPin((p) => p + t);
  }

  function entrar() {
    if (!escolhido || pin.length < 4 || aEntrar) return;
    setAEntrar(true);
    IniciarTurno(escolhido.ID, pin)
      .then(onIniciado)
      .catch((e) => {
        setErro(String(e).replace(/^Error:\s*/, ""));
        setPin("");
      })
      .finally(() => setAEntrar(false));
  }

  if (!escolhido) {
    return (
      <div className="turno">
        <h2>Quem está a vender?</h2>
        <div className="turno-operadores">
          {operadores.map((op) => (
            <button key={op.ID} className="turno-operador" onClick={() => setEscolhido(op)}>
              <span className="turno-inicial">{op.Nome.trim().charAt(0).toUpperCase()}</span>
              {op.Nome}
            </button>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="turno">
      <h2>Olá, {escolhido.Nome}</h2>
      <p className="turno-dica">Escreva o seu PIN</p>
      <input
        className="turno-pin"
        type="password"
        inputMode="numeric"
        autoFocus
        value={pin}
        aria-label="PIN"
        onChange={(e) => {
          setErro(null);
          setPin(e.target.value.replace(/\D/g, "").slice(0, 6));
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter") entrar();
        }}
      />
      {/* Espaço sempre reservado: se a mensagem aparecesse/desaparecesse, o
          teclado mudava de sítio entre toques. */}
      <p className="turno-erro" role="alert">
        {erro ?? "\u00a0"}
      </p>
      <div className="turno-teclado">
        {TECLAS.map((t, i) =>
          t === "" ? (
            <span key={i} />
          ) : (
            <button key={i} className="turno-tecla" onClick={() => tecla(t)} aria-label={t === "⌫" ? "Apagar" : t}>
              {t}
            </button>
          ),
        )}
      </div>
      <div className="turno-acoes">
        <button
          className="turno-voltar"
          onClick={() => {
            setEscolhido(null);
            setPin("");
            setErro(null);
          }}
        >
          Não sou eu
        </button>
        <button className="turno-entrar" disabled={pin.length < 4 || aEntrar} onClick={entrar}>
          {aEntrar ? "A entrar…" : "Entrar"}
        </button>
      </div>
    </div>
  );
}
