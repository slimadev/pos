-- name: ProximoNumeroCotacao :one
SELECT CAST(COALESCE(MAX(numero), 0) + 1 AS INTEGER) AS proximo
FROM cotacoes
WHERE negocio_id = ? AND ano = ?;

-- name: CriarCotacao :one
INSERT INTO cotacoes (
    negocio_id, ano, numero, cliente_id, cliente_nome, cliente_nuit, cliente_contacto,
    validade_dias, subtotal, desconto_total, total, taxa_iva, valor_iva, observacoes, utilizador_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CriarItemCotacao :one
INSERT INTO cotacao_itens (
    cotacao_id, produto_id, unidade_produto_id, descricao, nome_unidade, quantidade, preco_unitario, desconto, subtotal
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ListarCotacoes :many
SELECT * FROM cotacoes
WHERE negocio_id = ?
ORDER BY id DESC;

-- name: ObterCotacao :one
SELECT * FROM cotacoes WHERE id = ?;

-- name: ListarItensCotacao :many
SELECT * FROM cotacao_itens WHERE cotacao_id = ? ORDER BY id;

-- name: CancelarCotacao :execrows
UPDATE cotacoes SET estado = 'cancelada' WHERE id = ? AND estado = 'aberta';

-- name: MarcarCotacaoConvertida :execrows
UPDATE cotacoes SET estado = 'convertida', venda_id = sqlc.arg('venda_id')
WHERE id = sqlc.arg('id') AND estado = 'aberta';
