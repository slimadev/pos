-- Fase 10 addendum: categorias nunca tiveram como desativar (só listar/
-- criar) — as outras tabelas de cadastro (armazens, produtos, clientes)
-- já tinham `ativo` desde a migration inicial, categorias ficou por fazer.
ALTER TABLE categorias ADD COLUMN ativo INTEGER NOT NULL DEFAULT 1;
