-- Etiqueta livre de cada conta de mesa, para o garcom distinguir contas da
-- mesma mesa (por exemplo camisa azul).
ALTER TABLE contas_mesa ADD COLUMN etiqueta TEXT NOT NULL DEFAULT '';
