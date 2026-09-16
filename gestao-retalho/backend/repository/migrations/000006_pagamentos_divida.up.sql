-- Fase 7 do plano de implementação — pagamento de dívida como fluxo
-- independente de nova venda (SDD 3.2/4.11). `dividas` já existe desde a
-- Fase 4 (criada ao fechar uma conta/venda sem cobrir o total).

CREATE TABLE pagamentos_divida (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    divida_id INTEGER NOT NULL REFERENCES dividas (id),
    valor REAL NOT NULL CHECK (valor > 0),
    metodo TEXT NOT NULL CHECK (metodo IN ('dinheiro', 'mpesa', 'emola', 'cartao', 'transferencia')),
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    utilizador_id INTEGER NOT NULL REFERENCES utilizadores (id)
);

CREATE INDEX idx_pagamentos_divida_divida ON pagamentos_divida (divida_id);
