-- name: AtualizarUtilizadorAdmin :exec
UPDATE utilizadores
SET nome = ?, username = ?, password_hash = ?
WHERE id = ?;

-- name: ObterUtilizadorPorUsername :one
SELECT id, negocio_id, nome, username, password_hash, role, ativo
FROM utilizadores
WHERE username = ?;

-- name: ListarUtilizadores :many
SELECT id, negocio_id, nome, username, password_hash, role, ativo
FROM utilizadores
ORDER BY nome;

-- name: ObterUtilizador :one
SELECT id, negocio_id, nome, username, password_hash, role, ativo
FROM utilizadores
WHERE id = ?;

-- name: CriarUtilizador :one
INSERT INTO utilizadores (negocio_id, nome, username, password_hash, role, ativo)
VALUES (?, ?, ?, ?, ?, 1)
RETURNING id, negocio_id, nome, username, password_hash, role, ativo;

-- name: AtualizarNomeUtilizador :exec
UPDATE utilizadores SET nome = ? WHERE id = ?;

-- name: AtualizarPasswordUtilizador :exec
UPDATE utilizadores SET password_hash = ? WHERE id = ?;

-- name: DefinirUtilizadorAtivo :exec
UPDATE utilizadores SET ativo = ? WHERE id = ?;
