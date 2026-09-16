-- lojas é o registo (SDD 8.5, Fase 9) de quais ficheiros .db o executável
-- de administração universal conhece — vive na base própria do admin
-- (admin.db), nunca dentro de uma base de loja, porque nenhuma loja sabe
-- sobre as outras.
CREATE TABLE lojas (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    nome TEXT NOT NULL,
    caminho_db TEXT NOT NULL UNIQUE,
    ativo INTEGER NOT NULL DEFAULT 1,
    criado_em TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
