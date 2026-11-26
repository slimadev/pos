-- name: DefinirContaMesa :exec
INSERT INTO contas_mesa (conta_id, mesa, garcom_id, pessoas) VALUES (?, ?, ?, ?)
ON CONFLICT (conta_id) DO UPDATE SET mesa = excluded.mesa, garcom_id = excluded.garcom_id, pessoas = excluded.pessoas;

-- name: DefinirEtiquetaMesa :exec
UPDATE contas_mesa SET etiqueta = ? WHERE conta_id = ?;

-- name: ObterContaMesa :one
SELECT * FROM contas_mesa WHERE conta_id = ?;

-- name: ListarMesasAbertas :many
SELECT m.conta_id, m.mesa, m.etiqueta, m.garcom_id, m.pessoas, u.nome AS garcom_nome, c.aberta_em,
       CAST(COALESCE((SELECT SUM(i.quantidade * i.preco_unitario_aplicado - i.desconto) FROM itens_conta i WHERE i.conta_id = c.id), 0) AS REAL) AS total,
       CAST(COALESCE((SELECT SUM(MAX(i.quantidade - COALESCE(e.quantidade, 0), 0)) FROM itens_conta i LEFT JOIN itens_conta_enviados e ON e.item_id = i.id WHERE i.conta_id = c.id), 0) AS REAL) AS por_enviar
FROM contas_mesa m
JOIN contas c ON c.id = m.conta_id
JOIN utilizadores u ON u.id = m.garcom_id
WHERE c.negocio_id = ? AND c.status = 'aberta'
ORDER BY u.nome, m.mesa;

-- name: QuantidadeEnviada :one
SELECT CAST(COALESCE((SELECT quantidade FROM itens_conta_enviados WHERE item_id = ?), 0) AS REAL) AS quantidade;

-- name: DefinirQuantidadeEnviada :exec
INSERT INTO itens_conta_enviados (item_id, quantidade) VALUES (?, ?)
ON CONFLICT (item_id) DO UPDATE SET quantidade = excluded.quantidade;

-- name: CriarPedido :one
INSERT INTO pedidos (negocio_id, conta_id, tipo, mesa, garcom_id, motivo, autorizado_por)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: CriarPedidoLinha :exec
INSERT INTO pedido_linhas (pedido_id, produto_id, descricao, unidade, quantidade, impressora_id)
VALUES (?, ?, ?, ?, ?, ?);

-- name: ObterPedido :one
SELECT * FROM pedidos WHERE id = ?;

-- name: ListarPedidoLinhas :many
SELECT * FROM pedido_linhas WHERE pedido_id = ? ORDER BY id;

-- name: ListarPedidosDaConta :many
SELECT * FROM pedidos WHERE conta_id = ? ORDER BY id;

-- name: ListarImpressoras :many
SELECT * FROM impressoras WHERE negocio_id = ? ORDER BY funcao, nome;

-- name: ObterImpressora :one
SELECT * FROM impressoras WHERE id = ?;

-- name: CriarImpressora :one
INSERT INTO impressoras (negocio_id, nome, endereco, porta, colunas, funcao)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: AtualizarImpressora :exec
UPDATE impressoras SET nome = ?, endereco = ?, porta = ?, colunas = ?, funcao = ?, ativa = ? WHERE id = ?;

-- name: ListarCategoriaImpressora :many
SELECT * FROM categoria_impressora;

-- name: DefinirCategoriaImpressora :exec
INSERT INTO categoria_impressora (categoria_id, impressora_id) VALUES (?, ?)
ON CONFLICT (categoria_id) DO UPDATE SET impressora_id = excluded.impressora_id;

-- name: RemoverCategoriaImpressora :exec
DELETE FROM categoria_impressora WHERE categoria_id = ?;

-- name: ObterSessaoCaixaAberta :one
SELECT * FROM sessoes_caixa WHERE negocio_id = ? AND fechada_em IS NULL;

-- name: ObterSessaoCaixa :one
SELECT * FROM sessoes_caixa WHERE id = ?;

-- name: AbrirSessaoCaixa :one
INSERT INTO sessoes_caixa (negocio_id, utilizador_abertura_id, fundo_inicial) VALUES (?, ?, ?)
RETURNING *;

-- name: FecharSessaoCaixa :one
UPDATE sessoes_caixa
SET fechada_em = strftime('%Y-%m-%dT%H:%M:%fZ', 'now'), utilizador_fecho_id = ?, contagem = ?, observacoes = ?
WHERE id = ? AND fechada_em IS NULL
RETURNING *;

-- name: ListarSessoesCaixa :many
SELECT * FROM sessoes_caixa WHERE negocio_id = ? ORDER BY id DESC LIMIT ?;

-- name: AtualizarRoleUtilizador :exec
UPDATE utilizadores SET role = ? WHERE id = ?;

-- name: ListarItensComEnvio :many
SELECT i.id, i.conta_id, i.produto_id, i.unidade_produto_id, i.quantidade, i.preco_unitario_aplicado, i.desconto,
       CAST(COALESCE(e.quantidade, 0) AS REAL) AS enviada,
       p.nome AS produto_nome, u.nome_unidade, p.categoria_id
FROM itens_conta i
JOIN produtos p ON p.id = i.produto_id
JOIN unidades_produto u ON u.id = i.unidade_produto_id
LEFT JOIN itens_conta_enviados e ON e.item_id = i.id
WHERE i.conta_id = ?
ORDER BY i.id;
