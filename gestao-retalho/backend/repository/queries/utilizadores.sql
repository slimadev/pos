-- name: AtualizarUtilizadorAdmin :exec
UPDATE utilizadores
SET nome = ?, username = ?, password_hash = ?
WHERE id = ?;

-- name: ObterUtilizadorPorUsername :one
SELECT id, negocio_id, nome, username, password_hash, role, ativo
FROM utilizadores
WHERE username = ?;
