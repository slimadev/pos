import { useState } from "react";
import { IniciarTurno } from "../../wailsjs/go/api/App";
import { usecase } from "../../wailsjs/go/models";
import {
  BackspaceIcon,
  UserCircleIcon,
  KeyIcon,
  ArrowRightEndOnRectangleIcon,
  SparklesIcon,
} from "@heroicons/react/24/outline";
import "./Turno.css";

const TECLAS = ["1", "2", "3", "4", "5", "6", "7", "8", "9", "", "0", "backspace"];

/**
 * "Quem está a vender?" — o operador escolhe o seu nome e escreve o PIN
 * (4 a 6 algarismos). Interface glassmorphism tátil otimizada para terminais móveis.
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
    if (t === "backspace") setPin((p) => p.slice(0, -1));
    else if (t !== "" && pin.length < 6) setPin((p) => p + t);
  }

  function entrarComPin(p: string) {
    if (!escolhido || p.length < 4 || aEntrar) return;
    setAEntrar(true);
    IniciarTurno(escolhido.ID, p)
      .then(onIniciado)
      .catch((e) => {
        setErro(String(e).replace(/^Error:\s*/, ""));
        setPin("");
      })
      .finally(() => setAEntrar(false));
  }

  function entrar() {
    entrarComPin(pin);
  }

  if (!escolhido) {
    return (
      <div className="turno">
        <div className="turno-header-icon">
          <UserCircleIcon style={{ width: 44, height: 44, color: "var(--color-accent)" }} aria-hidden="true" />
        </div>
        <h2>Quem está a vender?</h2>
        <p className="turno-dica">Selecione o operador para abrir o turno de caixa</p>
        <div className="turno-operadores">
          {operadores.map((op) => (
            <button key={op.ID} className="turno-operador" onClick={() => setEscolhido(op)}>
              <span className="turno-inicial">{op.Nome.trim().charAt(0).toUpperCase()}</span>
              <span className="turno-nome">{op.Nome}</span>
              <span className="turno-badge">Operador</span>
            </button>
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="turno">
      <div className="turno-header-icon">
        <KeyIcon style={{ width: 36, height: 36, color: "var(--color-accent)" }} aria-hidden="true" />
      </div>
      <h2>Olá, {escolhido.Nome}</h2>
      <p className="turno-dica">Introduza o seu PIN de 4 a 6 dígitos</p>
      <input
        className="turno-pin"
        type="password"
        inputMode="numeric"
        autoFocus
        value={pin}
        aria-label="PIN"
        placeholder="••••"
        onChange={(e) => {
          setErro(null);
          setPin(e.target.value.replace(/\D/g, "").slice(0, 6));
        }}
        onKeyDown={(e) => {
          if (e.key === "Enter") entrar();
        }}
      />

      <p className="turno-erro" role="alert">
        {erro ?? "\u00a0"}
      </p>

      <div className="turno-teclado">
        {TECLAS.map((t, i) =>
          t === "" ? (
            <span key={i} />
          ) : t === "backspace" ? (
            <button
              key={i}
              className="turno-tecla turno-tecla-backspace"
              onClick={() => tecla(t)}
              aria-label="Apagar algarismo"
            >
              <BackspaceIcon style={{ width: 24, height: 24 }} aria-hidden="true" />
            </button>
          ) : (
            <button key={i} className="turno-tecla" onClick={() => tecla(t)} aria-label={t}>
              {t}
            </button>
          ),
        )}
      </div>

      <button
        type="button"
        className="turno-pin-rapido"
        onClick={() => {
          setPin("1234");
          entrarComPin("1234");
        }}
      >
        <SparklesIcon style={{ width: 14, height: 14 }} aria-hidden="true" />
        <span>PIN Padrão: 1234 (Entrar Direto)</span>
      </button>

      <div className="turno-acoes">
        <button
          className="turno-voltar"
          onClick={() => {
            setEscolhido(null);
            setPin("");
            setErro(null);
          }}
        >
          Trocar Operador
        </button>
        <button className="turno-entrar" disabled={pin.length < 4 || aEntrar} onClick={entrar}>
          <ArrowRightEndOnRectangleIcon style={{ width: 18, height: 18 }} aria-hidden="true" />
          <span>{aEntrar ? "A entrar…" : "Entrar no PDV"}</span>
        </button>
      </div>
    </div>
  );
}
