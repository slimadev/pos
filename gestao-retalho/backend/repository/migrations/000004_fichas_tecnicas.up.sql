-- Fase 5 do plano de implementação — fichas técnicas / produtos
-- compostos e semi-elaborados (SDD secção 3.3).

CREATE TABLE fichas_tecnicas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    produto_composto_id INTEGER NOT NULL REFERENCES produtos (id),
    produto_ingrediente_id INTEGER NOT NULL REFERENCES produtos (id),
    -- Na unidade base do ingrediente, já "líquida" (depois de qualquer
    -- perda de preparo) — perda_percentual é o que converte isto no
    -- consumo real de stock (SDD 3.3: "aumenta o consumo real face ao
    -- líquido da receita").
    quantidade REAL NOT NULL CHECK (quantidade > 0),
    perda_percentual REAL NOT NULL DEFAULT 0 CHECK (perda_percentual >= 0 AND perda_percentual < 100),
    UNIQUE (produto_composto_id, produto_ingrediente_id)
);

CREATE INDEX idx_fichas_tecnicas_composto ON fichas_tecnicas (produto_composto_id);
