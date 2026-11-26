-- Caixa: no restaurante, quem pode fechar contas, vender direto, abrir e
-- fechar o caixa e mexer nas mesas dos outros (o gerente tambem pode).
ALTER TABLE utilizadores ADD COLUMN caixa INTEGER NOT NULL DEFAULT 0;
