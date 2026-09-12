-- name: ListFoldersByUser :many
SELECT * FROM folders WHERE user_id = $1 ORDER BY name ASC;

-- name: CreateFolder :one
INSERT INTO folders (user_id, name)
VALUES ($1, $2)
RETURNING *;

-- name: GetFolderByID :one
SELECT * FROM folders WHERE id = $1 AND user_id = $2 LIMIT 1;

-- name: UpdateFolder :one
UPDATE folders SET name = $2
WHERE id = $1 AND user_id = $3
RETURNING *;

-- name: DeleteFolder :exec
DELETE FROM folders WHERE id = $1 AND user_id = $2;
