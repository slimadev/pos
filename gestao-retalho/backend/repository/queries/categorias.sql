-- name: ListarCategorias :many
SELECT id, negocio_id, nome, categoria_pai_id
FROM categorias
WHERE negocio_id = ?
ORDER BY nome;

-- name: CriarCategoria :one
INSERT INTO categorias (negocio_id, nome, categoria_pai_id)
VALUES (?, ?, ?)
RETURNING id, negocio_id, nome, categoria_pai_id;
