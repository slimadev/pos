-- name: RelatorioVendasPorDia :many
SELECT
    date(criado_em) AS dia,
    COUNT(*) AS numero_vendas,
    COALESCE(SUM(total), 0) AS total_vendido
FROM vendas
WHERE negocio_id = sqlc.arg('negocio_id')
  AND status != 'cancelada'
  AND criado_em >= sqlc.arg('desde') AND criado_em < sqlc.arg('ate')
  AND (sqlc.narg('armazem_id') IS NULL OR armazem_id = sqlc.narg('armazem_id'))
GROUP BY date(criado_em)
ORDER BY dia;

-- name: RelatorioLucroPorPeriodo :one
SELECT
    COALESCE(SUM(iv.subtotal), 0) AS receita,
    COALESCE(SUM(iv.custo_unitario_no_momento * iv.quantidade), 0) AS custo
FROM itens_venda iv
JOIN vendas v ON v.id = iv.venda_id
WHERE v.negocio_id = sqlc.arg('negocio_id')
  AND v.status != 'cancelada'
  AND v.criado_em >= sqlc.arg('desde') AND v.criado_em < sqlc.arg('ate')
  AND (sqlc.narg('armazem_id') IS NULL OR v.armazem_id = sqlc.narg('armazem_id'));

-- name: RelatorioVendasPorProduto :many
SELECT
    iv.produto_id,
    p.nome AS produto_nome,
    COALESCE(SUM(iv.quantidade), 0) AS quantidade_total,
    COALESCE(SUM(iv.subtotal), 0) AS valor_total
FROM itens_venda iv
JOIN vendas v ON v.id = iv.venda_id
JOIN produtos p ON p.id = iv.produto_id
WHERE v.negocio_id = sqlc.arg('negocio_id')
  AND v.status != 'cancelada'
  AND v.criado_em >= sqlc.arg('desde') AND v.criado_em < sqlc.arg('ate')
  AND (sqlc.narg('armazem_id') IS NULL OR v.armazem_id = sqlc.narg('armazem_id'))
GROUP BY iv.produto_id, p.nome
ORDER BY valor_total DESC;

-- name: RelatorioVendasPorMetodoPagamento :many
SELECT
    p.metodo,
    COALESCE(SUM(p.valor), 0) AS total
FROM pagamentos p
JOIN vendas v ON v.id = p.venda_id
WHERE v.negocio_id = sqlc.arg('negocio_id')
  AND v.status != 'cancelada'
  AND v.criado_em >= sqlc.arg('desde') AND v.criado_em < sqlc.arg('ate')
  AND (sqlc.narg('armazem_id') IS NULL OR v.armazem_id = sqlc.narg('armazem_id'))
GROUP BY p.metodo
ORDER BY total DESC;

-- name: ItensVendaPorPeriodo :many
SELECT
    iv.id, iv.venda_id, iv.produto_id, iv.unidade_produto_id, iv.quantidade,
    iv.preco_unitario, iv.custo_unitario_no_momento, iv.desconto, iv.subtotal
FROM itens_venda iv
JOIN vendas v ON v.id = iv.venda_id
WHERE v.negocio_id = sqlc.arg('negocio_id')
  AND v.status != 'cancelada'
  AND v.criado_em >= sqlc.arg('desde') AND v.criado_em < sqlc.arg('ate')
  AND (sqlc.narg('armazem_id') IS NULL OR v.armazem_id = sqlc.narg('armazem_id'));

-- name: RelatorioMovimentosPorPeriodo :many
SELECT
    m.id, m.produto_id, p.nome AS produto_nome, m.armazem_origem_id, m.armazem_destino_id, m.tipo,
    m.quantidade_unidade_base, m.custo_unitario, m.motivo, m.criado_em
FROM movimentos_stock m
JOIN produtos p ON p.id = m.produto_id
WHERE m.criado_em >= sqlc.arg('desde') AND m.criado_em < sqlc.arg('ate')
  AND (sqlc.narg('armazem_id') IS NULL OR m.armazem_origem_id = sqlc.narg('armazem_id') OR m.armazem_destino_id = sqlc.narg('armazem_id'))
ORDER BY m.criado_em DESC, m.id DESC;

-- name: RelatorioPerdasConsumoInterno :many
SELECT
    m.tipo,
    m.produto_id,
    p.nome AS produto_nome,
    COALESCE(SUM(m.quantidade_unidade_base), 0) AS quantidade_total,
    COALESCE(SUM(m.quantidade_unidade_base * COALESCE(m.custo_unitario, 0)), 0) AS valor_total
FROM movimentos_stock m
JOIN produtos p ON p.id = m.produto_id
WHERE m.tipo IN ('perda', 'consumo_interno')
  AND m.criado_em >= sqlc.arg('desde') AND m.criado_em < sqlc.arg('ate')
  AND (sqlc.narg('armazem_id') IS NULL OR m.armazem_origem_id = sqlc.narg('armazem_id'))
GROUP BY m.tipo, m.produto_id, p.nome
ORDER BY valor_total DESC;

-- name: RelatorioStockAtual :many
SELECT
    sa.produto_id,
    p.nome AS produto_nome,
    sa.armazem_id,
    sa.quantidade_unidade_base,
    sa.custo_medio_ponderado,
    p.stock_minimo
FROM stock_atual sa
JOIN produtos p ON p.id = sa.produto_id
WHERE p.negocio_id = sqlc.arg('negocio_id')
  AND (sqlc.narg('armazem_id') IS NULL OR sa.armazem_id = sqlc.narg('armazem_id'))
ORDER BY p.nome;

-- name: RelatorioContasPorPeriodo :many
SELECT id, negocio_id, cliente_id, identificador, status, aberta_em, fechada_em, utilizador_abertura_id, utilizador_fecho_id, motivo_cancelamento
FROM contas
WHERE negocio_id = sqlc.arg('negocio_id')
  AND aberta_em >= sqlc.arg('desde') AND aberta_em < sqlc.arg('ate')
ORDER BY aberta_em DESC;

-- name: RelatorioValorContas :many
SELECT
    conta_id,
    COALESCE(SUM(quantidade * preco_unitario_aplicado - desconto), 0) AS valor_total
FROM itens_conta
GROUP BY conta_id;
