-- administradores é o utilizador único do executável de administração
-- universal (decisão pós-Fase 10, a pedido explícito: já não há um
-- utilizador/login por loja — só existe um administrador, para todas as
-- lojas, e é aqui que vive). O tipo de negócio escolhido no wizard de uma
-- loja continua a decidir só tema/cor e ecrã de venda, nunca contas de
-- utilizador.
CREATE TABLE administradores (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    username TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
