-- name: ListarCategorias :many
SELECT id, negocio_id, nome, categoria_pai_id, ativo
FROM categorias
WHERE negocio_id = ?
ORDER BY nome;

-- name: CriarCategoria :one
INSERT INTO categorias (negocio_id, nome, categoria_pai_id, ativo)
VALUES (?, ?, ?, 1)
RETURNING id, negocio_id, nome, categoria_pai_id, ativo;

-- name: AtualizarCategoria :exec
UPDATE categorias SET nome = ?, categoria_pai_id = ? WHERE id = ?;

-- name: DefinirCategoriaAtiva :exec
UPDATE categorias SET ativo = ? WHERE id = ?;

-- name: MoverProdutosDeCategoria :execrows
UPDATE produtos SET categoria_id = sqlc.arg('destino') WHERE categoria_id = sqlc.arg('origem');

-- name: MoverSubcategorias :exec
UPDATE categorias SET categoria_pai_id = sqlc.arg('destino')
WHERE categoria_pai_id = sqlc.arg('origem') AND id != sqlc.arg('destino');
