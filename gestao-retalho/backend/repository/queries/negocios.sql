-- name: ObterNegocio :one
SELECT id, nome, tipo_negocio, ativo FROM negocios WHERE id = ?;

-- name: AtualizarNegocio :one
UPDATE negocios SET nome = ?, tipo_negocio = ?
WHERE id = ?
RETURNING id, nome, tipo_negocio, ativo;
