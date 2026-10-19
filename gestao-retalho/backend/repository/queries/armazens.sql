-- name: ListarArmazens :many
SELECT id, negocio_id, nome, tipo, ativo, fora_de_venda
FROM armazens
WHERE negocio_id = ?
ORDER BY nome;

-- name: ObterArmazem :one
SELECT id, negocio_id, nome, tipo, ativo, fora_de_venda
FROM armazens
WHERE id = ?;

-- name: CriarArmazem :one
INSERT INTO armazens (negocio_id, nome, tipo, ativo, fora_de_venda)
VALUES (?, ?, ?, 1, ?)
RETURNING id, negocio_id, nome, tipo, ativo, fora_de_venda;

-- name: AtualizarArmazem :exec
UPDATE armazens SET nome = ?, tipo = ?, fora_de_venda = ? WHERE id = ?;

-- name: DefinirArmazemAtivo :exec
UPDATE armazens SET ativo = ? WHERE id = ?;
