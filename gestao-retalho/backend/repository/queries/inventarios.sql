-- name: CriarInventario :one
INSERT INTO inventarios (negocio_id, armazem_id, status, utilizador_id)
VALUES (?, ?, 'aberto', ?)
RETURNING id, negocio_id, armazem_id, data, status, utilizador_id;

-- name: ObterInventario :one
SELECT id, negocio_id, armazem_id, data, status, utilizador_id
FROM inventarios
WHERE id = ?;

-- name: FecharInventario :exec
UPDATE inventarios SET status = 'fechado' WHERE id = ?;

-- name: CriarInventarioItem :exec
INSERT INTO inventario_itens (inventario_id, produto_id, quantidade_sistema)
VALUES (?, ?, ?);

-- name: RegistarContagemInventarioItem :exec
UPDATE inventario_itens
SET quantidade_contada = ?, diferenca = ?
WHERE id = ?;

-- name: MarcarInventarioItemAjustado :exec
UPDATE inventario_itens SET ajustado = 1 WHERE id = ?;

-- name: ObterInventarioItem :one
SELECT id, inventario_id, produto_id, quantidade_sistema, quantidade_contada, diferenca, ajustado
FROM inventario_itens
WHERE id = ?;

-- name: ListarInventarioItens :many
SELECT id, inventario_id, produto_id, quantidade_sistema, quantidade_contada, diferenca, ajustado
FROM inventario_itens
WHERE inventario_id = ?
ORDER BY id;

-- name: ListarInventarios :many
SELECT id, negocio_id, armazem_id, data, status, utilizador_id
FROM inventarios
WHERE negocio_id = ?
ORDER BY data DESC, id DESC;

-- name: ObterInventarioAbertoPorArmazem :one
SELECT id, negocio_id, armazem_id, data, status, utilizador_id
FROM inventarios
WHERE armazem_id = ? AND status = 'aberto'
ORDER BY id DESC
LIMIT 1;
