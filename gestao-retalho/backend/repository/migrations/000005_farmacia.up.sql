-- Fase 6 do plano de implementação — módulo de farmácia (SDD secção 9.4):
-- lotes/validade (FEFO) e receitas médicas. A mercearia (venda a granel)
-- não precisa de tabela nova — já reutiliza unidades_produto com
-- unidade_base = kg (SDD 9.4).

ALTER TABLE produtos ADD COLUMN lote_controlado INTEGER NOT NULL DEFAULT 0;

CREATE TABLE lotes_produto (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    armazem_id INTEGER NOT NULL REFERENCES armazens (id),
    numero_lote TEXT NOT NULL,
    data_validade TEXT NOT NULL,
    quantidade REAL NOT NULL CHECK (quantidade >= 0),
    custo_unitario REAL NOT NULL,
    UNIQUE (produto_id, armazem_id, numero_lote)
);

-- Índice pela própria ordem que o FEFO usa: mais cedo a vencer primeiro.
CREATE INDEX idx_lotes_produto_fefo ON lotes_produto (produto_id, armazem_id, data_validade);

-- Nome distinto de fichas_tecnicas (SDD 3.3) para nunca confundir: uma é
-- ficha técnica de produto composto, esta é prescrição médica.
CREATE TABLE receitas_medicas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    venda_id INTEGER NOT NULL REFERENCES vendas (id),
    cliente_id INTEGER NOT NULL REFERENCES clientes (id),
    medico TEXT NOT NULL,
    numero_receita TEXT NOT NULL,
    validado_por INTEGER NOT NULL REFERENCES utilizadores (id),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_receitas_medicas_venda ON receitas_medicas (venda_id);
