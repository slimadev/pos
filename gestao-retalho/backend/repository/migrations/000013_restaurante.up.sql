-- Restaurante: mesa, garcom e pessoas de cada conta; quanto de cada item ja
-- foi enviado para preparacao; pedidos impressos (envios e anulacoes);
-- impressoras de rede e para onde vai cada categoria; sessoes de caixa.
-- Tabelas a parte das contas para nao mexer no que o bar e os relatorios ja
-- leem.
CREATE TABLE contas_mesa (
    conta_id INTEGER PRIMARY KEY REFERENCES contas (id) ON DELETE CASCADE,
    mesa TEXT NOT NULL,
    garcom_id INTEGER NOT NULL REFERENCES utilizadores (id),
    pessoas INTEGER NOT NULL DEFAULT 1 CHECK (pessoas >= 1)
);

CREATE TABLE itens_conta_enviados (
    item_id INTEGER PRIMARY KEY REFERENCES itens_conta (id) ON DELETE CASCADE,
    quantidade REAL NOT NULL CHECK (quantidade >= 0)
);

CREATE TABLE impressoras (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    nome TEXT NOT NULL,
    endereco TEXT NOT NULL,
    porta INTEGER NOT NULL DEFAULT 9100,
    colunas INTEGER NOT NULL DEFAULT 48,
    funcao TEXT NOT NULL CHECK (funcao IN ('balcao', 'preparacao')),
    ativa INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE categoria_impressora (
    categoria_id INTEGER PRIMARY KEY REFERENCES categorias (id) ON DELETE CASCADE,
    impressora_id INTEGER NOT NULL REFERENCES impressoras (id) ON DELETE CASCADE
);

CREATE TABLE pedidos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    conta_id INTEGER NOT NULL REFERENCES contas (id),
    tipo TEXT NOT NULL CHECK (tipo IN ('pedido', 'anulacao')),
    mesa TEXT NOT NULL,
    garcom_id INTEGER NOT NULL REFERENCES utilizadores (id),
    motivo TEXT,
    autorizado_por INTEGER REFERENCES utilizadores (id),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_pedidos_conta ON pedidos (conta_id);

CREATE TABLE pedido_linhas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pedido_id INTEGER NOT NULL REFERENCES pedidos (id) ON DELETE CASCADE,
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    descricao TEXT NOT NULL,
    unidade TEXT NOT NULL,
    quantidade REAL NOT NULL,
    impressora_id INTEGER REFERENCES impressoras (id) ON DELETE SET NULL
);

CREATE INDEX idx_pedido_linhas_pedido ON pedido_linhas (pedido_id);

CREATE TABLE sessoes_caixa (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    aberta_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    utilizador_abertura_id INTEGER NOT NULL REFERENCES utilizadores (id),
    fundo_inicial REAL NOT NULL CHECK (fundo_inicial >= 0),
    fechada_em TEXT,
    utilizador_fecho_id INTEGER REFERENCES utilizadores (id),
    contagem TEXT,
    observacoes TEXT
);

-- So pode haver uma sessao aberta de cada vez.
CREATE UNIQUE INDEX ux_sessao_caixa_aberta ON sessoes_caixa (negocio_id) WHERE fechada_em IS NULL;
