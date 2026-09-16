-- name: ObterConfiguracao :one
SELECT chave, valor FROM configuracoes WHERE negocio_id = ? AND chave = ?;

-- name: DefinirConfiguracao :exec
INSERT INTO configuracoes (negocio_id, chave, valor)
VALUES (?, ?, ?)
ON CONFLICT (negocio_id, chave) DO UPDATE SET valor = excluded.valor;

-- name: ListarConfiguracoes :many
SELECT chave, valor FROM configuracoes WHERE negocio_id = ?;
