-- Armazens que o PDV nao mostra nem deixa usar para vender (ex: um
-- deposito). Guardado ao contrario (fora_de_venda) para que um armazem
-- novo, sem nada indicado, continue disponivel para venda como antes.
-- Os depositos que ja existem passam a ficar fora de venda.
ALTER TABLE armazens ADD COLUMN fora_de_venda INTEGER NOT NULL DEFAULT 0;
UPDATE armazens SET fora_de_venda = 1 WHERE tipo = 'deposito';
