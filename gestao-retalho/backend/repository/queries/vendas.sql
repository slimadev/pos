-- name: CriarVenda :one
INSERT INTO vendas (negocio_id, armazem_id, cliente_id, conta_id, tipo_venda, subtotal, desconto_total, total, status, utilizador_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, negocio_id, armazem_id, cliente_id, conta_id, tipo_venda, subtotal, desconto_total, total, status, criado_em, utilizador_id;

-- name: ObterVenda :one
SELECT id, negocio_id, armazem_id, cliente_id, conta_id, tipo_venda, subtotal, desconto_total, total, status, criado_em, utilizador_id
FROM vendas
WHERE id = ?;

-- name: ListarVendasPorPeriodo :many
SELECT id, negocio_id, armazem_id, cliente_id, conta_id, tipo_venda, subtotal, desconto_total, total, status, criado_em, utilizador_id
FROM vendas
WHERE negocio_id = ? AND criado_em >= ? AND criado_em < ?
ORDER BY criado_em DESC;

-- name: CriarItemVenda :one
INSERT INTO itens_venda (venda_id, produto_id, unidade_produto_id, quantidade, preco_unitario, custo_unitario_no_momento, desconto, subtotal)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, venda_id, produto_id, unidade_produto_id, quantidade, preco_unitario, custo_unitario_no_momento, desconto, subtotal;

-- name: ListarItensVenda :many
SELECT id, venda_id, produto_id, unidade_produto_id, quantidade, preco_unitario, custo_unitario_no_momento, desconto, subtotal
FROM itens_venda
WHERE venda_id = ?
ORDER BY id;

-- name: CriarPagamento :one
INSERT INTO pagamentos (venda_id, metodo, valor)
VALUES (?, ?, ?)
RETURNING id, venda_id, metodo, valor, criado_em;

-- name: ListarPagamentos :many
SELECT id, venda_id, metodo, valor, criado_em
FROM pagamentos
WHERE venda_id = ?
ORDER BY id;
