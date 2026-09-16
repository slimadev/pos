-- name: CriarMovimentoStock :one
INSERT INTO movimentos_stock (
    produto_id, armazem_origem_id, armazem_destino_id, tipo,
    quantidade_unidade_base, custo_unitario, unidade_produto_id,
    motivo, referencia_id, utilizador_id
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id, produto_id, armazem_origem_id, armazem_destino_id, tipo,
          quantidade_unidade_base, custo_unitario, unidade_produto_id,
          motivo, referencia_id, utilizador_id, criado_em;

-- name: ListarMovimentosPorProduto :many
SELECT id, produto_id, armazem_origem_id, armazem_destino_id, tipo,
       quantidade_unidade_base, custo_unitario, unidade_produto_id,
       motivo, referencia_id, utilizador_id, criado_em
FROM movimentos_stock
WHERE produto_id = ?
ORDER BY criado_em DESC, id DESC;
