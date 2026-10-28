import { useEffect, useState } from "react";
import {
  ListarCategorias,
  ListarProdutos,
  CriarCategoria,
  AtualizarCategoria,
  AtivarCategoria,
  DesativarCategoria,
  JuntarCategorias,
} from "../wailsjs/go/admin/AdminApp";
import { domain } from "../wailsjs/go/models";
import Breadcrumb from "./Breadcrumb";
import Modal from "./Modal";
import ImportarProdutosAssistente from "./ImportarProdutos";

import {
  PlusIcon,
  PencilSquareIcon,
  ArrowsPointingInIcon,
  ArrowUpTrayIcon,
  XMarkIcon,
  PowerIcon,
} from "@heroicons/react/24/outline";
type ModalEstado =
  | { aberto: false }
  | { aberto: true; categoria: domain.Categoria | null }
  | { aberto: true; juntar: domain.Categoria }
  | { aberto: true; importar: domain.Categoria };

// Mesma regra do backend (mesmoNome): sem maiúsculas, acentos nem espaços a mais.
const chaveNome = (nome: string) =>
  nome
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .trim()
    .replace(/\s+/g, " ")
    .toLowerCase();

/**
 * Registo de categorias da única loja que este cmd/admin serve — dado de
 * negócio, continua a viver dentro da base da loja (SDD 8.5), este ecrã só
 * escolhe a ligação a abrir (feito no backend, ver admin.CategoriasService).
 * Mostra quantos produtos ativos há em cada categoria, para se ver as que
 * estão vazias antes de as desativar.
 */
export default function CategoriasScreen({ onErro }: { onErro: (e: string) => void }) {
  const [categorias, setCategorias] = useState<domain.Categoria[]>([]);
  const [produtos, setProdutos] = useState<domain.Produto[]>([]);
  const [aCarregar, setACarregar] = useState(true);
  const [modal, setModal] = useState<ModalEstado>({ aberto: false });
  const [aviso, setAviso] = useState<string | null>(null);

  function carregar() {
    Promise.all([ListarCategorias(), ListarProdutos()])
      .then(([c, p]) => {
        setCategorias(c);
        setProdutos(p);
      })
      .catch((e) => onErro(String(e)))
      .finally(() => setACarregar(false));
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps -- onErro é um setter estável vindo de App
  useEffect(carregar, []);

  const nomeCategoria = (id: number) => categorias.find((c) => c.ID === id)?.Nome ?? `#${id}`;
  const nProdutos = (id: number) => produtos.filter((p) => p.Ativo && p.CategoriaID === id).length;
  const ordenadas = [...categorias].sort(
    (a, b) => Number(b.Ativo) - Number(a.Ativo) || a.Nome.localeCompare(b.Nome, "pt"),
  );

  function alternarAtivo(c: domain.Categoria) {
    const acao = c.Ativo ? DesativarCategoria(c.ID) : AtivarCategoria(c.ID);
    acao.then(carregar).catch((e) => onErro(String(e)));
  }

  const fechar = () => setModal({ aberto: false });
  const ativas = categorias.filter((c) => c.Ativo);
  const repetida = (c: domain.Categoria) =>
    c.Ativo && ativas.some((o) => o.ID !== c.ID && chaveNome(o.Nome) === chaveNome(c.Nome));

  return (
    <div className="admin-lojas-screen">
      <Breadcrumb itens={[{ rotulo: "Categorias" }]} />
      <div className="admin-ecra-header">
        <h2 style={{ margin: 0 }}>Categorias</h2>
        <button className="admin-btn" onClick={() => setModal({ aberto: true, categoria: null })}>
          <PlusIcon className="btn-icone" aria-hidden="true" />
          Criar categoria
        </button>
      </div>

      {aviso && (
        <div className="admin-aviso-ok">
          {aviso}
          <button onClick={() => setAviso(null)} aria-label="Fechar">
            ×
          </button>
        </div>
      )}

      <table className="admin-tabela">
        <thead>
          <tr>
            <th>Nome</th>
            <th>Categoria-mãe</th>
            <th style={{ textAlign: "right" }}>Produtos</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {ordenadas.map((c) => (
            <tr key={c.ID} className={c.Ativo ? undefined : "admin-linha-inativa"}>
              <td>
                {c.Nome}
                {!c.Ativo && <span className="admin-etiqueta">inativa</span>}
                {repetida(c) && <span className="admin-etiqueta aviso">repetida</span>}
              </td>
              <td>{c.CategoriaPaiID != null ? nomeCategoria(c.CategoriaPaiID) : "—"}</td>
              <td style={{ textAlign: "right" }}>{nProdutos(c.ID)}</td>
              <td>
                <div className="admin-acoes">
                  <button className="admin-btn-secundario" onClick={() => setModal({ aberto: true, categoria: c })}>
                    <PencilSquareIcon className="btn-icone" aria-hidden="true" />
                    Editar
                  </button>
                  {c.Ativo && ativas.length > 1 && (
                    <button className="admin-btn-secundario" onClick={() => setModal({ aberto: true, juntar: c })}>
                      <ArrowsPointingInIcon className="btn-icone" aria-hidden="true" />
                      Juntar com…
                    </button>
                  )}
                  {c.Ativo && (
                    <button className="admin-btn-secundario" onClick={() => setModal({ aberto: true, importar: c })}>
                      <ArrowUpTrayIcon className="btn-icone" aria-hidden="true" />
                      Importar Excel
                    </button>
                  )}
                  <button className="admin-btn-secundario" onClick={() => alternarAtivo(c)}>
                    <PowerIcon className="btn-icone" aria-hidden="true" />
                    {c.Ativo ? "Desativar" : "Ativar"}
                  </button>
                </div>
              </td>
            </tr>
          ))}
          {categorias.length === 0 && (
            <tr>
              <td colSpan={4}>{aCarregar ? "A carregar…" : 'Sem categorias — clique em "+ Criar categoria".'}</td>
            </tr>
          )}
        </tbody>
      </table>

      {modal.aberto && "importar" in modal && (
        <Modal titulo={`Importar produtos para ${modal.importar.Nome}`} largo onFechar={fechar}>
          <ImportarProdutosAssistente
            categorias={categorias}
            categoriaInicial={modal.importar.ID}
            onConcluido={(r) => {
              fechar();
              carregar();
              setAviso(`${r.Produtos} produto(s) importado(s)${r.Categorias > 0 ? `, ${r.Categorias} categoria(s) nova(s)` : ""}; o stock já entrou.`);
            }}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}

      {modal.aberto && "juntar" in modal && (
        <Modal titulo={`Juntar ${modal.juntar.Nome}`} onFechar={fechar}>
          <JuntarForm
            origem={modal.juntar}
            nProdutos={nProdutos(modal.juntar.ID)}
            destinos={ativas.filter((c) => c.ID !== modal.juntar.ID)}
            nomeDe={nomeCategoria}
            nProdutosDe={nProdutos}
            onJuntada={(movidos, destino) => {
              fechar();
              carregar();
              setAviso(
                `${movidos} produto(s) passaram para "${destino}" e a categoria repetida foi desativada.`,
              );
            }}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}

      {modal.aberto && "categoria" in modal && (
        <Modal titulo={modal.categoria ? `Editar ${modal.categoria.Nome}` : "Criar categoria"} onFechar={fechar}>
          <CategoriaForm
            categoria={modal.categoria}
            categorias={categorias}
            onGuardado={() => {
              fechar();
              carregar();
            }}
            onCancelar={fechar}
            onErro={onErro}
          />
        </Modal>
      )}
    </div>
  );
}

function CategoriaForm({
  categoria,
  categorias,
  onGuardado,
  onCancelar,
  onErro,
}: {
  categoria: domain.Categoria | null;
  categorias: domain.Categoria[];
  onGuardado: () => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  const [nome, setNome] = useState(categoria?.Nome ?? "");
  const [categoriaPaiId, setCategoriaPaiId] = useState<number | "">(categoria?.CategoriaPaiID ?? "");
  const [aGuardar, setAGuardar] = useState(false);

  function guardar() {
    if (!nome.trim() || aGuardar) return;
    const c = new domain.Categoria();
    c.ID = categoria?.ID ?? 0;
    c.Nome = nome.trim();
    c.CategoriaPaiID = categoriaPaiId === "" ? undefined : Number(categoriaPaiId);

    setAGuardar(true);
    const pedido = categoria ? AtualizarCategoria(c) : CriarCategoria(c);
    pedido
      .then(onGuardado)
      .catch((e) => onErro(String(e)))
      .finally(() => setAGuardar(false));
  }

  return (
    <div className="admin-produto-form">
      <label className="admin-campo">
        Nome
        <input
          autoFocus
          value={nome}
          onChange={(e) => setNome(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && guardar()}
          placeholder="ex: Ferramentas"
        />
      </label>
      <label className="admin-campo">
        Dentro de (opcional)
        <select value={categoriaPaiId} onChange={(e) => setCategoriaPaiId(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Nenhuma — categoria principal</option>
          {categorias
            .filter((c) => c.ID !== categoria?.ID && c.Ativo)
            .sort((a, b) => a.Nome.localeCompare(b.Nome, "pt"))
            .map((c) => (
              <option key={c.ID} value={c.ID}>
                {c.Nome}
              </option>
            ))}
        </select>
      </label>
      <div className="admin-form">
        <button className="admin-btn" disabled={!nome.trim() || aGuardar} onClick={guardar}>
          {aGuardar ? "A guardar…" : categoria ? "Guardar alterações" : "+ Criar categoria"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
      </div>
    </div>
  );
}

function JuntarForm({
  origem,
  nProdutos,
  destinos,
  nomeDe,
  nProdutosDe,
  onJuntada,
  onCancelar,
  onErro,
}: {
  origem: domain.Categoria;
  nProdutos: number;
  destinos: domain.Categoria[];
  nomeDe: (id: number) => string;
  nProdutosDe: (id: number) => number;
  onJuntada: (movidos: number, destino: string) => void;
  onCancelar: () => void;
  onErro: (e: string) => void;
}) {
  // As de nome igual primeiro (é para elas que isto existe), já escolhida.
  const ordenados = [...destinos].sort(
    (a, b) =>
      Number(chaveNome(b.Nome) === chaveNome(origem.Nome)) - Number(chaveNome(a.Nome) === chaveNome(origem.Nome)) ||
      a.Nome.localeCompare(b.Nome, "pt"),
  );
  const [destinoID, setDestinoID] = useState<number | "">(
    ordenados.length > 0 && chaveNome(ordenados[0].Nome) === chaveNome(origem.Nome) ? ordenados[0].ID : "",
  );
  const [aJuntar, setAJuntar] = useState(false);
  const destino = destinos.find((c) => c.ID === destinoID);

  function juntar() {
    if (destinoID === "" || aJuntar) return;
    setAJuntar(true);
    JuntarCategorias(origem.ID, destinoID)
      .then((n) => onJuntada(n, destino?.Nome ?? ""))
      .catch((e) => onErro(String(e)))
      .finally(() => setAJuntar(false));
  }

  const rotulo = (c: domain.Categoria) =>
    `${c.Nome}${c.CategoriaPaiID != null ? ` (em ${nomeDe(c.CategoriaPaiID)})` : ""} — ${nProdutosDe(c.ID)} produto(s)`;

  return (
    <div className="admin-produto-form">
      <p style={{ margin: 0 }}>
        Os {nProdutos} produto(s) de <strong>{origem.Nome}</strong> e as suas subcategorias passam para a categoria
        escolhida, e <strong>{origem.Nome}</strong> fica desativada.
      </p>
      <label className="admin-campo">
        Juntar com
        <select value={destinoID} onChange={(e) => setDestinoID(e.target.value ? Number(e.target.value) : "")}>
          <option value="">Escolha a categoria…</option>
          {ordenados.map((c) => (
            <option key={c.ID} value={c.ID}>
              {rotulo(c)}
            </option>
          ))}
        </select>
      </label>
      <div className="admin-form">
        <button className="admin-btn" disabled={destinoID === "" || aJuntar} onClick={juntar}>
          <ArrowsPointingInIcon className="btn-icone" aria-hidden="true" />
          {aJuntar ? "A juntar…" : "Juntar"}
        </button>
        <button className="admin-btn-secundario" onClick={onCancelar}>
          <XMarkIcon className="btn-icone" aria-hidden="true" />
          Cancelar
        </button>
      </div>
    </div>
  );
}
