-- name: ObterStockProdutoArmazem :one
SELECT produto_id, armazem_id, quantidade_unidade_base, custo_medio_ponderado
FROM stock_atual
WHERE produto_id = ? AND armazem_id = ?;

-- name: ListarStockPorArmazem :many
SELECT produto_id, armazem_id, quantidade_unidade_base, custo_medio_ponderado
FROM stock_atual
WHERE armazem_id = ?
ORDER BY produto_id;

-- name: ObterCustoMedioProdutoArmazem :one
SELECT produto_id, armazem_id, custo_medio_ponderado
FROM produto_custo_medio
WHERE produto_id = ? AND armazem_id = ?;

-- name: DefinirCustoMedioProdutoArmazem :exec
INSERT INTO produto_custo_medio (produto_id, armazem_id, custo_medio_ponderado)
VALUES (?, ?, ?)
ON CONFLICT (produto_id, armazem_id) DO UPDATE SET custo_medio_ponderado = excluded.custo_medio_ponderado;

-- name: ListarStockParaInventario :many
SELECT p.id AS produto_id, COALESCE(sa.quantidade_unidade_base, 0) AS quantidade_sistema
FROM produtos p
LEFT JOIN stock_atual sa ON sa.produto_id = p.id AND sa.armazem_id = ?
WHERE p.negocio_id = ? AND p.ativo = 1 AND p.controla_stock = 1
ORDER BY p.nome;
