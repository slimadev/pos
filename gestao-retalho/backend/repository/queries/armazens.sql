-- name: ListarArmazens :many
SELECT id, negocio_id, nome, tipo, ativo
FROM armazens
WHERE negocio_id = ?
ORDER BY nome;

-- name: ObterArmazem :one
SELECT id, negocio_id, nome, tipo, ativo
FROM armazens
WHERE id = ?;

-- name: CriarArmazem :one
INSERT INTO armazens (negocio_id, nome, tipo, ativo)
VALUES (?, ?, ?, 1)
RETURNING id, negocio_id, nome, tipo, ativo;

-- name: AtualizarArmazem :exec
UPDATE armazens SET nome = ?, tipo = ? WHERE id = ?;

-- name: DefinirArmazemAtivo :exec
UPDATE armazens SET ativo = ? WHERE id = ?;
