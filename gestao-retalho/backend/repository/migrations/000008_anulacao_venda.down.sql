-- Volta ao CHECK antigo: os movimentos 'anulacao_venda' passam a 'entrada'
-- (repõem stock da mesma forma) para caberem na tabela antiga.

DROP VIEW IF EXISTS stock_atual;

CREATE TABLE movimentos_stock_antigo (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produto_id INTEGER NOT NULL REFERENCES produtos (id),
    armazem_origem_id INTEGER REFERENCES armazens (id),
    armazem_destino_id INTEGER REFERENCES armazens (id),
    tipo TEXT NOT NULL CHECK (tipo IN ('entrada', 'transferencia', 'perda', 'consumo_interno', 'ajuste_inventario', 'venda', 'producao')),
    quantidade_unidade_base REAL NOT NULL CHECK (quantidade_unidade_base > 0),
    custo_unitario REAL,
    unidade_produto_id INTEGER REFERENCES unidades_produto (id),
    motivo TEXT,
    referencia_id TEXT,
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CHECK (armazem_origem_id IS NOT NULL OR armazem_destino_id IS NOT NULL)
);

INSERT INTO movimentos_stock_antigo
SELECT id, produto_id, armazem_origem_id, armazem_destino_id,
       CASE tipo WHEN 'anulacao_venda' THEN 'entrada' ELSE tipo END,
       quantidade_unidade_base, custo_unitario, unidade_produto_id, motivo, referencia_id, utilizador_id, criado_em
FROM movimentos_stock;

DROP TABLE movimentos_stock;
ALTER TABLE movimentos_stock_antigo RENAME TO movimentos_stock;

CREATE INDEX idx_movimentos_produto ON movimentos_stock (produto_id);
CREATE INDEX idx_movimentos_criado_em ON movimentos_stock (criado_em);

CREATE VIEW stock_atual AS
SELECT
    m.produto_id AS produto_id,
    m.armazem_id AS armazem_id,
    SUM(m.delta) AS quantidade_unidade_base,
    COALESCE(pcm.custo_medio_ponderado, 0) AS custo_medio_ponderado
FROM (
    SELECT produto_id, armazem_destino_id AS armazem_id, quantidade_unidade_base AS delta
    FROM movimentos_stock WHERE armazem_destino_id IS NOT NULL
    UNION ALL
    SELECT produto_id, armazem_origem_id AS armazem_id, -quantidade_unidade_base AS delta
    FROM movimentos_stock WHERE armazem_origem_id IS NOT NULL
) m
LEFT JOIN produto_custo_medio pcm ON pcm.produto_id = m.produto_id AND pcm.armazem_id = m.armazem_id
GROUP BY m.produto_id, m.armazem_id;
