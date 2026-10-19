import { useEffect, useState } from "react";
import type { ComponentType, SVGProps } from "react";
import {
  BuildingStorefrontIcon,
  ShoppingBagIcon,
  ArchiveBoxIcon,
  CheckCircleIcon,
  PauseCircleIcon,
  ShoppingCartIcon,
  NoSymbolIcon,
} from "@heroicons/react/24/outline";
import { ListarArmazens, CriarArmazem, AtualizarArmazem, AtivarArmazem, DesativarArmazem } from "../wailsjs/go/admin/AdminApp";
import { domain } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";

type IconeSvg = ComponentType<SVGProps<SVGSVGElement>>;
type TipoArmazem = "principal" | "ponto_venda" | "deposito";

const TIPO_INFO: Record<TipoArmazem, { rotulo: string; Icone: IconeSvg; cor: string }> = {
  principal: { rotulo: "Principal", Icone: BuildingStorefrontIcon, cor: "accent" },
  ponto_venda: { rotulo: "Ponto de venda", Icone: ShoppingBagIcon, cor: "ok" },
  deposito: { rotulo: "Depósito", Icone: ArchiveBoxIcon, cor: "warn" },
};

function infoDoTipo(tipo: string) {
  return TIPO_INFO[tipo as TipoArmazem] ?? TIPO_INFO.principal;
}

type Vista = { tipo: "lista" } | { tipo: "form"; armazem: domain.Armazem | null };

/**
 * CRUD de armazéns da única loja que este cmd/admin serve. Em vez do
 * padrão modal usado em Produtos.tsx, este ecrã navega entre "lista" e
 * "form" dentro de si mesmo (sem sair da página) — dá uso real ao
 * breadcrumb no topo, que mostra onde estás e deixa voltar com um clique.
 */
export default function ArmazensScreen({ onErro }: { onErro: (e: string) => void }) {
  const [armazens, setArmazens] = useState<domain.Armazem[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [vista, setVista] = useState<Vista>({ tipo: "lista" });

  function carregar() {
    ListarArmazens()
      .then(setArmazens)
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(carregar, []);

  function alternarAtivo(a: domain.Armazem) {
    const acao = a.Ativo ? DesativarArmazem(a.ID) : AtivarArmazem(a.ID);
    acao.then(carregar).catch((e) => onErro(String(e)));
  }

  const migalhas =
    vista.tipo === "lista"
      ? [{ rotulo: "Armazéns" }]
      : [
          { rotulo: "Armazéns", onClick: () => setVista({ tipo: "lista" }) },
          { rotulo: vista.armazem ? vista.armazem.Nome : "Novo armazém" },
        ];

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={migalhas} />

      {vista.tipo === "lista" && (
        <>
          <div className="admin-ecra-header">
            <h2 style={{ margin: 0 }}>Armazéns</h2>
            <button className="admin-btn" onClick={() => setVista({ tipo: "form", armazem: null })}>
              + Criar armazém
            </button>
          </div>

          {armazens.length === 0 ? (
            <p style={{ color: "var(--text-secondary)" }}>
              {aCarregar ? "A carregar…" : 'Sem armazéns — clique em "+ Criar armazém".'}
            </p>
          ) : (
            <div className="admin-armazens-grid">
              {armazens.map((a) => {
                const info = infoDoTipo(a.Tipo);
                return (
                  <div key={a.ID} className={`admin-armazem-card${a.Ativo ? "" : " inativo"}`}>
                    <div className="admin-armazem-card-topo">
                      <div className={`admin-armazem-icone cor-${info.cor}`}>
                        <info.Icone className="admin-armazem-icone-svg" aria-hidden="true" />
                      </div>
                      <div className={`admin-armazem-status ${a.Ativo ? "ativo" : "inativo"}`}>
                        {a.Ativo ? (
                          <CheckCircleIcon className="admin-armazem-status-icon" aria-hidden="true" />
                        ) : (
                          <PauseCircleIcon className="admin-armazem-status-icon" aria-hidden="true" />
                        )}
                        {a.Ativo ? "Ativo" : "Inativo"}
                      </div>
                    </div>
                    <div className="admin-armazem-nome">{a.Nome}</div>
                    <div className={`admin-armazem-tag cor-${info.cor}`}>{info.rotulo}</div>
                    <div className={`admin-armazem-venda${a.ForaDeVenda ? " fora" : ""}`}>
                      {a.ForaDeVenda ? (
                        <NoSymbolIcon className="admin-armazem-status-icon" aria-hidden="true" />
                      ) : (
                        <ShoppingCartIcon className="admin-armazem-status-icon" aria-hidden="true" />
                      )}
                      {a.ForaDeVenda ? "Fora de venda — não aparece no PDV" : "Disponível para venda no PDV"}
                    </div>
                    <div className="admin-armazem-card-acoes">
                      <button className="admin-btn-secundario" onClick={() => setVista({ tipo: "form", armazem: a })}>
                        Editar
                      </button>
                      <button className="admin-btn-secundario" onClick={() => alternarAtivo(a)}>
                        {a.Ativo ? "Desativar" : "Ativar"}
                      </button>
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </>
      )}

      {vista.tipo === "form" && (
        <ArmazemForm
          armazem={vista.armazem}
          onGuardado={() => {
            setVista({ tipo: "lista" });
            carregar();
          }}
          onCancelar={() => setVista({ tipo: "lista" })}
          onErro={onErro}
        />
      )}
    </div>
  );
}

function ArmazemForm({
  armazem,
  onGuardado,
  onCancelar,
  onErro,
}: {
  armazem: domain.Armazem | null;
  onGuardado: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(armazem?.Nome ?? "");
  const [tipo, setTipo] = useState<TipoArmazem>((armazem?.Tipo as TipoArmazem) ?? "principal");
  const [disponivel, setDisponivel] = useState(armazem ? !armazem.ForaDeVenda : true);
  // Num armazém novo, a caixa acompanha o tipo (depósito = fora de venda)
  // até o utilizador lhe mexer.
  const [mexeuNaVenda, setMexeuNaVenda] = useState(!!armazem);

  function escolherTipo(t: TipoArmazem) {
    setTipo(t);
    if (!mexeuNaVenda) setDisponivel(t !== "deposito");
  }
  const [aGuardar, setAGuardar] = useState(false);

  function guardar() {
    if (!nome.trim() || aGuardar) return;

    const a = new domain.Armazem();
    a.ID = armazem?.ID ?? 0;
    a.Nome = nome.trim();
    a.Tipo = tipo;
    a.ForaDeVenda = !disponivel;

    setAGuardar(true);
    const pedido = armazem ? AtualizarArmazem(a) : CriarArmazem(a);
    pedido
      .then(() => onGuardado())
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-armazem-form">
      <div className="admin-armazem-form-tipos">
        {(Object.keys(TIPO_INFO) as TipoArmazem[]).map((t) => {
          const info = TIPO_INFO[t];
          return (
            <div
              key={t}
              className={`admin-armazem-tipo-opcao cor-${info.cor}${tipo === t ? " ativo" : ""}`}
              onClick={() => escolherTipo(t)}
            >
              <info.Icone className="admin-armazem-tipo-opcao-icone" aria-hidden="true" />
              {info.rotulo}
            </div>
          );
        })}
      </div>

      <label className="admin-produto-check">
        <input
          type="checkbox"
          checked={disponivel}
          onChange={(e) => {
            setDisponivel(e.target.checked);
            setMexeuNaVenda(true);
          }}
        />
        Disponível para venda — aparece para escolher no PDV
      </label>

      <div className="admin-form">
        <input placeholder="Nome do armazém" value={nome} onChange={(e) => setNome(e.target.value)} autoFocus />
        <button className="admin-btn" disabled={!nome.trim() || aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : armazem ? "Guardar alterações" : "+ Criar armazém"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          Cancelar
        </button>
      </div>
    </div>
  );
}
