-- Documentos fiscais: FR (fatura-recibo), FT (fatura), RC (recibo de
-- pagamento de divida) e NC (nota de credito). Numeracao por tipo e ano.
-- Emitente e adquirente ficam congelados no documento. Depois de emitido,
-- um documento nunca se altera nem se apaga (gatilhos abaixo): corrige-se
-- com uma nota de credito.
CREATE TABLE documentos_fiscais (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    negocio_id INTEGER NOT NULL REFERENCES negocios (id),
    tipo TEXT NOT NULL CHECK (tipo IN ('FR', 'FT', 'RC', 'NC')),
    ano INTEGER NOT NULL,
    numero INTEGER NOT NULL,
    venda_id INTEGER REFERENCES vendas (id),
    pagamento_divida_id INTEGER REFERENCES pagamentos_divida (id),
    documento_origem_id INTEGER REFERENCES documentos_fiscais (id),
    emitente_nome TEXT NOT NULL,
    emitente_nuit TEXT,
    emitente_endereco TEXT,
    emitente_contacto TEXT,
    cliente_id INTEGER REFERENCES clientes (id),
    cliente_nome TEXT NOT NULL,
    cliente_nuit TEXT,
    cliente_endereco TEXT,
    subtotal REAL NOT NULL,
    desconto_total REAL NOT NULL DEFAULT 0,
    total REAL NOT NULL,
    taxa_iva REAL NOT NULL,
    valor_iva REAL NOT NULL,
    motivo TEXT,
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id),
    UNIQUE (negocio_id, tipo, ano, numero)
);

-- Uma venda tem no maximo uma FR/FT e uma NC; um pagamento, um RC.
CREATE UNIQUE INDEX ux_documentos_venda ON documentos_fiscais (venda_id) WHERE tipo IN ('FR', 'FT');
CREATE UNIQUE INDEX ux_documentos_nc_venda ON documentos_fiscais (venda_id) WHERE tipo = 'NC';
CREATE UNIQUE INDEX ux_documentos_pagamento ON documentos_fiscais (pagamento_divida_id) WHERE pagamento_divida_id IS NOT NULL;
CREATE INDEX idx_documentos_criado ON documentos_fiscais (negocio_id, criado_em);

CREATE TABLE documento_linhas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    documento_id INTEGER NOT NULL REFERENCES documentos_fiscais (id),
    descricao TEXT NOT NULL,
    unidade TEXT NOT NULL,
    quantidade REAL NOT NULL,
    preco_unitario REAL NOT NULL,
    desconto REAL NOT NULL DEFAULT 0,
    subtotal REAL NOT NULL
);

CREATE INDEX idx_documento_linhas_documento ON documento_linhas (documento_id);

CREATE TRIGGER documentos_fiscais_sem_alteracao BEFORE UPDATE ON documentos_fiscais
BEGIN
    SELECT RAISE(ABORT, 'documento fiscal emitido nao pode ser alterado');
END;

CREATE TRIGGER documentos_fiscais_sem_apagar BEFORE DELETE ON documentos_fiscais
BEGIN
    SELECT RAISE(ABORT, 'documento fiscal emitido nao pode ser apagado');
END;

CREATE TRIGGER documento_linhas_sem_alteracao BEFORE UPDATE ON documento_linhas
BEGIN
    SELECT RAISE(ABORT, 'documento fiscal emitido nao pode ser alterado');
END;

CREATE TRIGGER documento_linhas_sem_apagar BEFORE DELETE ON documento_linhas
BEGIN
    SELECT RAISE(ABORT, 'documento fiscal emitido nao pode ser apagado');
END;
