-- name: ListarClientes :many
SELECT id, negocio_id, nome, telefone, nif, morada, limite_credito, is_anonimo, ativo
FROM clientes
WHERE negocio_id = ? AND is_anonimo = 0
ORDER BY nome;

-- name: ObterCliente :one
SELECT id, negocio_id, nome, telefone, nif, morada, limite_credito, is_anonimo, ativo
FROM clientes
WHERE id = ?;

-- name: CriarCliente :one
INSERT INTO clientes (negocio_id, nome, telefone, nif, morada, limite_credito, ativo)
VALUES (?, ?, ?, ?, ?, ?, 1)
RETURNING id, negocio_id, nome, telefone, nif, morada, limite_credito, is_anonimo, ativo;

-- name: AtualizarCliente :exec
UPDATE clientes
SET nome = ?, telefone = ?, nif = ?, morada = ?, limite_credito = ?
WHERE id = ?;

-- name: DefinirClienteAtivo :exec
UPDATE clientes SET ativo = ? WHERE id = ?;
