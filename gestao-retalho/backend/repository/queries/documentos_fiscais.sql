-- name: ProximoNumeroDocumento :one
SELECT CAST(COALESCE(MAX(numero), 0) + 1 AS INTEGER) AS proximo
FROM documentos_fiscais
WHERE negocio_id = ? AND tipo = ? AND ano = ?;

-- name: CriarDocumentoFiscal :one
INSERT INTO documentos_fiscais (
    negocio_id, tipo, ano, numero, venda_id, pagamento_divida_id, documento_origem_id,
    emitente_nome, emitente_nuit, emitente_endereco, emitente_contacto,
    cliente_id, cliente_nome, cliente_nuit, cliente_endereco,
    subtotal, desconto_total, total, taxa_iva, valor_iva, motivo, utilizador_id
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CriarLinhaDocumento :one
INSERT INTO documento_linhas (documento_id, descricao, unidade, quantidade, preco_unitario, desconto, subtotal)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: ObterDocumentoFiscal :one
SELECT * FROM documentos_fiscais WHERE id = ?;

-- name: ListarLinhasDocumento :many
SELECT * FROM documento_linhas WHERE documento_id = ? ORDER BY id;

-- name: ObterFaturaDaVenda :one
SELECT * FROM documentos_fiscais WHERE venda_id = ? AND tipo IN ('FR', 'FT');

-- name: ObterNotaCreditoDaVenda :one
SELECT * FROM documentos_fiscais WHERE venda_id = ? AND tipo = 'NC';

-- name: ObterReciboDoPagamento :one
SELECT * FROM documentos_fiscais WHERE pagamento_divida_id = ?;

-- name: ListarDocumentosEntre :many
SELECT * FROM documentos_fiscais
WHERE negocio_id = sqlc.arg('negocio_id') AND criado_em >= sqlc.arg('desde') AND criado_em < sqlc.arg('ate')
ORDER BY criado_em, id;

-- name: ListarVendasSemFatura :many
SELECT v.id, v.total, v.status, v.criado_em
FROM vendas v
WHERE v.negocio_id = sqlc.arg('negocio_id') AND v.criado_em >= sqlc.arg('desde') AND v.status != 'cancelada'
  AND NOT EXISTS (
    SELECT 1 FROM documentos_fiscais d WHERE d.venda_id = v.id AND d.tipo IN ('FR', 'FT')
  )
ORDER BY v.id;
