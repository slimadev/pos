-- name: ListarUnidadesPorProduto :many
SELECT id, produto_id, nome_unidade, qtd_em_unidade_base, preco_custo, preco_venda,
       codigo_barras, is_padrao_venda, is_padrao_compra
FROM unidades_produto
WHERE produto_id = ?
ORDER BY id;

-- name: ObterUnidadeProduto :one
SELECT id, produto_id, nome_unidade, qtd_em_unidade_base, preco_custo, preco_venda,
       codigo_barras, is_padrao_venda, is_padrao_compra
FROM unidades_produto
WHERE id = ?;

-- name: CriarUnidadeProduto :one
INSERT INTO unidades_produto (produto_id, nome_unidade, qtd_em_unidade_base, preco_custo, preco_venda, codigo_barras, is_padrao_venda, is_padrao_compra)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, produto_id, nome_unidade, qtd_em_unidade_base, preco_custo, preco_venda,
          codigo_barras, is_padrao_venda, is_padrao_compra;

-- name: AtualizarUnidadeProduto :exec
UPDATE unidades_produto
SET nome_unidade = ?, qtd_em_unidade_base = ?, preco_custo = ?, preco_venda = ?,
    codigo_barras = ?, is_padrao_venda = ?, is_padrao_compra = ?
WHERE id = ?;
