-- name: CriarReceitaMedica :one
INSERT INTO receitas_medicas (venda_id, cliente_id, medico, numero_receita, validado_por)
VALUES (?, ?, ?, ?, ?)
RETURNING id, venda_id, cliente_id, medico, numero_receita, validado_por, criado_em;

-- name: ListarReceitasPorVenda :many
SELECT id, venda_id, cliente_id, medico, numero_receita, validado_por, criado_em
FROM receitas_medicas
WHERE venda_id = ?
ORDER BY id;
