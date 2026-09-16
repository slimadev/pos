-- name: ListarIngredientes :many
SELECT id, produto_composto_id, produto_ingrediente_id, quantidade, perda_percentual
FROM fichas_tecnicas
WHERE produto_composto_id = ?
ORDER BY id;

-- name: CriarIngrediente :one
INSERT INTO fichas_tecnicas (produto_composto_id, produto_ingrediente_id, quantidade, perda_percentual)
VALUES (?, ?, ?, ?)
RETURNING id, produto_composto_id, produto_ingrediente_id, quantidade, perda_percentual;

-- name: RemoverIngredientes :exec
DELETE FROM fichas_tecnicas WHERE produto_composto_id = ?;
