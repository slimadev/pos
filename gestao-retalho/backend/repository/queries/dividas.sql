-- name: CriarDivida :one
INSERT INTO dividas (cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status, criado_em;

-- name: ListarDividasPorCliente :many
SELECT id, cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status, criado_em
FROM dividas
WHERE cliente_id = ?
ORDER BY criado_em DESC;
