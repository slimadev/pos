-- Cotacoes (proformas): o cliente leva um orcamento, sem mexer no stock
-- nem contar como venda. Os dados do cliente e os precos ficam congelados
-- no momento em que a cotacao e feita. Numeracao sequencial por ano.
CREATE TABLE cotacoes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    ano INTEGER NOT NULL,
    numero INTEGER NOT NULL,
    cliente_id INTEGER REFERENCES clientes (id),
    cliente_nome TEXT NOT NULL,
    cliente_nuit TEXT,
    cliente_contacto TEXT,
    validade_dias INTEGER NOT NULL,
    subtotal REAL NOT NULL,
    desconto_total REAL NOT NULL DEFAULT 0,
    total REAL NOT NULL,
    taxa_iva REAL NOT NULL,
    valor_iva REAL NOT NULL,
    observacoes TEXT,
    estado TEXT NOT NULL DEFAULT 'aberta' CHECK (estado IN ('aberta', 'convertida', 'cancelada')),
    venda_id INTEGER REFERENCES vendas (id),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id),
    UNIQUE (negocio_id, ano, numero)
);

CREATE TABLE cotacao_itens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cotacao_id INTEGER NOT NULL REFERENCES cotacoes (id),
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    unidade_produto_id INTEGER NOT NULL REFERENCES unidades_produto (id),
    descricao TEXT NOT NULL,
    nome_unidade TEXT NOT NULL,
    quantidade REAL NOT NULL,
    preco_unitario REAL NOT NULL,
    desconto REAL NOT NULL DEFAULT 0,
    subtotal REAL NOT NULL
);

CREATE INDEX idx_cotacao_itens_cotacao ON cotacao_itens (cotacao_id);
