-- Fase 4 do plano de implementação — contas abertas (bottlestore, SDD
-- 4.10) e dívidas (SDD 3.2/4.11). `contas`/`itens_conta`/`vendas` já
-- existem desde a Fase 3 (SDD 3.5) — aqui só o que falta para uma conta
-- poder ficar aberta horas, ser cancelada, ou fechar com saldo em dívida.

ALTER TABLE contas ADD COLUMN motivo_cancelamento TEXT;

CREATE TABLE dividas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    cliente_id INTEGER NOT NULL REFERENCES clientes (id),
    venda_id INTEGER NOT NULL REFERENCES vendas (id),
    valor_original REAL NOT NULL CHECK (valor_original > 0),
    valor_pago REAL NOT NULL DEFAULT 0,
    valor_em_aberto REAL NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('aberta', 'parcial', 'paga')) DEFAULT 'aberta',
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX idx_dividas_cliente ON dividas (cliente_id);
