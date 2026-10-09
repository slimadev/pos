import { useEffect, useState } from "react";
import {
  ListarOperadores,
  CriarOperador,
  RenomearOperador,
  RedefinirPINOperador,
  AtivarOperador,
  DesativarOperador,
} from "../wailsjs/go/admin/AdminApp";
import { usecase } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";

const PIN_VALIDO = /^[0-9]{4,6}$/;

type ModalEstado =
  | { tipo: "nenhum" }
  | { tipo: "novo" }
  | { tipo: "renomear"; operador: usecase.Operador }
  | { tipo: "pin"; operador: usecase.Operador };

/**
 * Operadores do PDV: quem vende e o PIN com que se identifica ao abrir o
 * PDV. Com pelo menos um operador ativo, o PDV pergunta sempre quem está a
 * vender, e cada venda fica com o seu nome. O login completo continua a ser
 * só o do administrador, aqui no cmd/admin.
 */
export default function UtilizadoresScreen({ onErro }: { onErro: (e: string) => void }) {
  const [operadores, setOperadores] = useState<usecase.Operador[] | null>(null);
  const [modal, setModal] = useState<ModalEstado>({ tipo: "nenhum" });

  function carregar() {
    ListarOperadores()
      .then(setOperadores)
      .catch((e) => {
        setOperadores([]);
        onErro(String(e));
      });
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- carga inicial
  }, []);

  const fechar = () => setModal({ tipo: "nenhum" });
  const guardado = () => {
    fechar();
    carregar();
  };

  function alternar(op: usecase.Operador) {
    (op.Ativo ? DesativarOperador(op.ID) : AtivarOperador(op.ID)).then(carregar).catch((e) => onErro(String(e)));
  }

  const lista = [...(operadores ?? [])].sort(
    (a, b) => Number(b.Ativo) - Number(a.Ativo) || a.Nome.localeCompare(b.Nome, "pt"),
  );
  const ativos = lista.filter((o) => o.Ativo).length;

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Utilizadores" }]} />
      <div className="admin-ecra-header">
        <div>
          <h2 style={{ margin: 0 }}>Utilizadores</h2>
          <div style={{ color: "var(--text-secondary)", fontSize: "0.9rem" }}>
            Operadores do PDV — cada um entra com o seu PIN e fica com as suas vendas.
          </div>
        </div>
        <button className="admin-btn" onClick={() => setModal({ tipo: "novo" })}>
          + Novo operador
        </button>
      </div>

      <p className={`admin-inventario-aviso ${ativos > 0 ? "ok" : "warn"}`} style={{ margin: 0 }}>
        {ativos > 0
          ? `O PDV pede o operador e o PIN antes de vender (${ativos} operador${ativos === 1 ? "" : "es"} ativo${ativos === 1 ? "" : "s"}).`
          : "Sem operadores ativos, o PDV vende sem pedir quem é, e tudo fica atribuído ao administrador."}
      </p>

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Operador</th>
            <th>Estado</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {operadores === null && (
            <tr>
              <td colSpan={3}>A carregar…</td>
            </tr>
          )}
          {operadores !== null && lista.length === 0 && (
            <tr>
              <td colSpan={3}>Ainda não há operadores.</td>
            </tr>
          )}
          {lista.map((op) => (
            <tr key={op.ID}>
              <td>{op.Nome}</td>
              <td>
                <span className={`admin-inventario-estado ${op.Ativo ? "fechado" : "aberto"}`}>
                  {op.Ativo ? "Ativo" : "Inativo"}
                </span>
              </td>
              <td className="admin-fichas-acoes">
                <button className="admin-btn-secundario" onClick={() => setModal({ tipo: "renomear", operador: op })}>
                  Mudar nome
                </button>
                <button className="admin-btn-secundario" onClick={() => setModal({ tipo: "pin", operador: op })}>
                  Novo PIN
                </button>
                <button className="admin-btn-secundario" onClick={() => alternar(op)}>
                  {op.Ativo ? "Desativar" : "Ativar"}
                </button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>

      {modal.tipo === "novo" && (
        <Modal titulo="Novo operador" onFechar={fechar}>
          <FormOperador
            pedirNome
            pedirPIN
            textoBotao="Criar operador"
            onGuardar={(nome, pin) => CriarOperador(nome, pin).then(() => undefined)}
            onGuardado={guardado}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}
      {modal.tipo === "renomear" && (
        <Modal titulo={`Mudar o nome de ${modal.operador.Nome}`} onFechar={fechar}>
          <FormOperador
            pedirNome
            nomeInicial={modal.operador.Nome}
            textoBotao="Guardar"
            onGuardar={(nome) => RenomearOperador(modal.operador.ID, nome)}
            onGuardado={guardado}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}
      {modal.tipo === "pin" && (
        <Modal titulo={`Novo PIN para ${modal.operador.Nome}`} onFechar={fechar}>
          <FormOperador
            pedirPIN
            textoBotao="Guardar PIN"
            onGuardar={(_, pin) => RedefinirPINOperador(modal.operador.ID, pin)}
            onGuardado={guardado}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}
    </div>
  );
}

function FormOperador({
  pedirNome = false,
  pedirPIN = false,
  nomeInicial = "",
  textoBotao,
  onGuardar,
  onGuardado,
  onCancelar,
  onErro,
}: {
  pedirNome?: boolean;
  pedirPIN?: boolean;
  nomeInicial?: string;
  textoBotao: string;
  onGuardar: (nome: string, pin: string) => Promise<void>;
  onGuardado: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(nomeInicial);
  const [pin, setPin] = useState("");
  const [confirmacao, setConfirmacao] = useState("");
  const [aGuardar, setAGuardar] = useState(false);

  const nomeOk = !pedirNome || nome.trim() !== "";
  const pinOk = !pedirPIN || (PIN_VALIDO.test(pin) && pin === confirmacao);
  const podeGuardar = nomeOk && pinOk && !aGuardar;

  function guardar() {
    if (!podeGuardar) return;
    setAGuardar(true);
    onGuardar(nome.trim(), pin)
      .then(onGuardado)
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      {pedirNome && (
        <label className="admin-campo">
          Nome
          <input value={nome} onChange={(e) => setNome(e.target.value)} autoFocus />
        </label>
      )}
      {pedirPIN && (
        <>
          <div className="admin-form">
            <label className="admin-campo">
              PIN (4 a 6 algarismos)
              <input
                type="password"
                inputMode="numeric"
                value={pin}
                autoFocus={!pedirNome}
                onChange={(e) => setPin(e.target.value.replace(/\D/g, "").slice(0, 6))}
              />
            </label>
            <label className="admin-campo">
              Repetir o PIN
              <input
                type="password"
                inputMode="numeric"
                value={confirmacao}
                onChange={(e) => setConfirmacao(e.target.value.replace(/\D/g, "").slice(0, 6))}
              />
            </label>
          </div>
          {confirmacao !== "" && pin !== confirmacao && (
            <p className="admin-inventario-aviso warn" style={{ margin: 0 }}>
              Os dois PIN não são iguais.
            </p>
          )}
          <small style={{ color: "var(--text-secondary)" }}>
            O PIN não fica guardado em texto — não é possível vê-lo depois, só definir um novo. Ao fim de 5 PIN errados
            seguidos no PDV, o operador fica 30 segundos sem poder tentar.
          </small>
        </>
      )}
      <div className="admin-form">
        <button className="admin-btn" disabled={!podeGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : textoBotao}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
