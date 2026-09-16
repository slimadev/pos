-- name: CriarConta :one
INSERT INTO contas (negocio_id, cliente_id, identificador, status, utilizador_abertura_id)
VALUES (?, ?, ?, 'aberta', ?)
RETURNING id, negocio_id, cliente_id, identificador, status, aberta_em, fechada_em, utilizador_abertura_id, utilizador_fecho_id, motivo_cancelamento;

-- name: FecharConta :one
UPDATE contas
SET status = 'fechada', fechada_em = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), utilizador_fecho_id = ?
WHERE id = ?
RETURNING id, negocio_id, cliente_id, identificador, status, aberta_em, fechada_em, utilizador_abertura_id, utilizador_fecho_id, motivo_cancelamento;

-- name: CancelarConta :one
UPDATE contas
SET status = 'cancelada', fechada_em = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), utilizador_fecho_id = ?, motivo_cancelamento = ?
WHERE id = ?
RETURNING id, negocio_id, cliente_id, identificador, status, aberta_em, fechada_em, utilizador_abertura_id, utilizador_fecho_id, motivo_cancelamento;

-- name: ObterConta :one
SELECT id, negocio_id, cliente_id, identificador, status, aberta_em, fechada_em, utilizador_abertura_id, utilizador_fecho_id, motivo_cancelamento
FROM contas
WHERE id = ?;

-- name: ListarContasAbertas :many
SELECT id, negocio_id, cliente_id, identificador, status, aberta_em, fechada_em, utilizador_abertura_id, utilizador_fecho_id, motivo_cancelamento
FROM contas
WHERE negocio_id = ? AND status = 'aberta'
ORDER BY aberta_em;

-- name: CriarItemConta :one
INSERT INTO itens_conta (conta_id, produto_id, unidade_produto_id, quantidade, preco_unitario_aplicado, desconto, utilizador_id)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id, conta_id, produto_id, unidade_produto_id, quantidade, preco_unitario_aplicado, desconto, criado_em, utilizador_id;

-- name: ListarItensConta :many
SELECT id, conta_id, produto_id, unidade_produto_id, quantidade, preco_unitario_aplicado, desconto, criado_em, utilizador_id
FROM itens_conta
WHERE conta_id = ?
ORDER BY id;

-- name: ObterItemConta :one
SELECT id, conta_id, produto_id, unidade_produto_id, quantidade, preco_unitario_aplicado, desconto, criado_em, utilizador_id
FROM itens_conta
WHERE id = ?;

-- name: TransferirItemConta :exec
UPDATE itens_conta SET conta_id = ? WHERE id = ?;
