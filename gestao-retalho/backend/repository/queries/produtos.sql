-- name: ListarProdutos :many
SELECT id, negocio_id, categoria_id, nome, tipo_produto, unidade_base,
       stock_minimo, controla_stock, ativo, criado_em, atualizado_em, lote_controlado
FROM produtos
WHERE negocio_id = ?
ORDER BY nome;

-- name: ListarProdutosAtivos :many
SELECT id, negocio_id, categoria_id, nome, tipo_produto, unidade_base,
       stock_minimo, controla_stock, ativo, criado_em, atualizado_em, lote_controlado
FROM produtos
WHERE negocio_id = ? AND ativo = 1
ORDER BY nome;

-- name: ObterProduto :one
SELECT id, negocio_id, categoria_id, nome, tipo_produto, unidade_base,
       stock_minimo, controla_stock, ativo, criado_em, atualizado_em, lote_controlado
FROM produtos
WHERE id = ?;

-- name: CriarProduto :one
INSERT INTO produtos (negocio_id, categoria_id, nome, tipo_produto, unidade_base, stock_minimo, controla_stock, lote_controlado, ativo)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, 1)
RETURNING id, negocio_id, categoria_id, nome, tipo_produto, unidade_base,
          stock_minimo, controla_stock, ativo, criado_em, atualizado_em, lote_controlado;

-- name: AtualizarProduto :one
UPDATE produtos
SET categoria_id = ?, nome = ?, stock_minimo = ?, lote_controlado = ?, atualizado_em = strftime('%Y-%m-%dT%H:%M:%fZ', 'now')
WHERE id = ?
RETURNING id, negocio_id, categoria_id, nome, tipo_produto, unidade_base,
          stock_minimo, controla_stock, ativo, criado_em, atualizado_em, lote_controlado;

-- name: DefinirProdutoAtivo :exec
UPDATE produtos SET ativo = ?, atualizado_em = strftime('%Y-%m-%dT%H:%M:%fZ', 'now') WHERE id = ?;
