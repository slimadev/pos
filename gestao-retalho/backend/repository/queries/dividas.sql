-- name: CriarDivida :one
INSERT INTO dividas (cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status, criado_em;

-- name: ObterDivida :one
SELECT id, cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status, criado_em
FROM dividas
WHERE id = ?;

-- name: ListarDividasPorCliente :many
SELECT id, cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status, criado_em
FROM dividas
WHERE cliente_id = ?
ORDER BY criado_em DESC;

-- name: ListarDividasAbertas :many
SELECT d.id, d.cliente_id, d.venda_id, d.valor_original, d.valor_pago, d.valor_em_aberto, d.status, d.criado_em
FROM dividas d
JOIN clientes c ON c.id = d.cliente_id
WHERE c.negocio_id = ? AND d.status IN ('aberta', 'parcial')
ORDER BY d.criado_em ASC;

-- name: AtualizarDivida :one
UPDATE dividas
SET valor_pago = ?, valor_em_aberto = ?, status = ?
WHERE id = ?
RETURNING id, cliente_id, venda_id, valor_original, valor_pago, valor_em_aberto, status, criado_em;

-- name: CriarPagamentoDivida :one
INSERT INTO pagamentos_divida (divida_id, valor, metodo, utilizador_id)
VALUES (?, ?, ?, ?)
RETURNING id, divida_id, valor, metodo, criado_em, utilizador_id;

-- name: ListarPagamentosDivida :many
SELECT id, divida_id, valor, metodo, criado_em, utilizador_id
FROM pagamentos_divida
WHERE divida_id = ?
ORDER BY id;
