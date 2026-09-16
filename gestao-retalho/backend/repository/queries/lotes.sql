-- name: CriarLote :one
INSERT INTO lotes_produto (produto_id, armazem_id, numero_lote, data_validade, quantidade, custo_unitario)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING id, produto_id, armazem_id, numero_lote, data_validade, quantidade, custo_unitario;

-- name: ListarLotesFEFO :many
SELECT id, produto_id, armazem_id, numero_lote, data_validade, quantidade, custo_unitario
FROM lotes_produto
WHERE produto_id = ? AND armazem_id = ? AND quantidade > 0
ORDER BY data_validade ASC, id ASC;

-- name: AtualizarQuantidadeLote :exec
UPDATE lotes_produto SET quantidade = ? WHERE id = ?;

-- name: ListarLotesAVencer :many
SELECT lp.id, lp.produto_id, lp.armazem_id, lp.numero_lote, lp.data_validade, lp.quantidade, lp.custo_unitario
FROM lotes_produto lp
JOIN produtos p ON p.id = lp.produto_id
WHERE p.negocio_id = ? AND lp.quantidade > 0 AND lp.data_validade <= ?
ORDER BY lp.data_validade ASC;
