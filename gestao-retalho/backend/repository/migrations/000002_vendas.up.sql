-- Fase 3 do plano de implementação — vendas diretas (SDD secção 3.2, 4.10).
--
-- SDD 3.5: a venda direta da ferragem/mercearia/farmácia é internamente uma
-- `conta` que abre e fecha na mesma transação — por isso `contas`/
-- `itens_conta` entram já aqui, não só na Fase 4 (bottlestore), para as
-- duas fases partilharem exatamente o mesmo mecanismo.

CREATE TABLE clientes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    nome TEXT NOT NULL,
    telefone TEXT,
    nif TEXT,
    morada TEXT,
    limite_credito REAL,
    -- SDD 3.2 descreve o cliente anónimo ora como FK nullable, ora como
    -- "registo fixo com flag is_anonimo" — ficamos com o registo fixo (id=1,
    -- semeado abaixo) para nunca ter de tratar cliente_id NULL à parte em
    -- relatórios/joins; is_anonimo é o que distingue esse registo.
    is_anonimo INTEGER NOT NULL DEFAULT 0,
    ativo INTEGER NOT NULL DEFAULT 1
);

INSERT INTO clientes (id, negocio_id, nome, is_anonimo, ativo) VALUES (1, 1, 'Anónimo', 1, 1);

CREATE TABLE contas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    cliente_id INTEGER NOT NULL REFERENCES clientes (id),
    identificador TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('aberta', 'fechada', 'cancelada')) DEFAULT 'aberta',
    aberta_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    fechada_em TEXT,
    utilizador_abertura_id INTEGER NOT NULL REFERENCES utilizadores (id),
    utilizador_fecho_id INTEGER REFERENCES utilizadores (id)
);

CREATE TABLE itens_conta (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    conta_id INTEGER NOT NULL REFERENCES contas (id),
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    unidade_produto_id INTEGER NOT NULL REFERENCES unidades_produto (id),
    quantidade REAL NOT NULL CHECK (quantidade > 0),
    preco_unitario_aplicado REAL NOT NULL,
    desconto REAL NOT NULL DEFAULT 0,
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id)
);

CREATE INDEX idx_itens_conta_conta ON itens_conta (conta_id);

CREATE TABLE vendas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    armazem_id INTEGER NOT NULL REFERENCES armazens (id),
    cliente_id INTEGER NOT NULL REFERENCES clientes (id),
    conta_id INTEGER REFERENCES contas (id),
    tipo_venda TEXT NOT NULL CHECK (tipo_venda IN ('direta', 'conta_fechada')),
    subtotal REAL NOT NULL,
    desconto_total REAL NOT NULL DEFAULT 0,
    total REAL NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('paga', 'divida', 'parcial', 'cancelada')),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id)
);

CREATE INDEX idx_vendas_negocio_criado ON vendas (negocio_id, criado_em);

CREATE TABLE itens_venda (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    venda_id INTEGER NOT NULL REFERENCES vendas (id),
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    unidade_produto_id INTEGER NOT NULL REFERENCES unidades_produto (id),
    quantidade REAL NOT NULL,
    preco_unitario REAL NOT NULL,
    -- Custo por unidade vendida no momento da venda (não a unidade base) —
    -- permite lucro real por venda mesmo que o custo do produto mude depois
    -- (SDD 3.2, nota sobre itens_venda).
    custo_unitario_no_momento REAL NOT NULL,
    desconto REAL NOT NULL DEFAULT 0,
    subtotal REAL NOT NULL
);

CREATE INDEX idx_itens_venda_venda ON itens_venda (venda_id);

CREATE TABLE pagamentos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    venda_id INTEGER NOT NULL REFERENCES vendas (id),
    metodo TEXT NOT NULL CHECK (metodo IN ('dinheiro', 'mpesa', 'emola', 'cartao', 'transferencia')),
    valor REAL NOT NULL CHECK (valor > 0),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_pagamentos_venda ON pagamentos (venda_id);
