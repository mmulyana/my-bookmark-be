-- name: ListBookmarksByUser :many
SELECT b.* FROM bookmarks b
WHERE b.user_id = sqlc.arg(user_id)
  AND (sqlc.narg(is_favorite)::bool IS NULL OR b.is_favorite = sqlc.narg(is_favorite))
  AND (sqlc.narg(has_folder)::bool IS NULL OR (b.folder_id IS NOT NULL) = sqlc.narg(has_folder))
  AND (
    sqlc.narg(has_tags)::bool IS NULL
    OR EXISTS (SELECT 1 FROM bookmark_tags bt WHERE bt.bookmark_id = b.id) = sqlc.narg(has_tags)
  )
ORDER BY b.created_at DESC;

-- name: CreateBookmark :one
INSERT INTO bookmarks (user_id, url, title, description, favicon, image_url, folder_id, is_favorite)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetBookmarkByID :one
SELECT * FROM bookmarks WHERE id = $1 AND user_id = $2 LIMIT 1;

-- name: UpdateBookmark :one
UPDATE bookmarks
SET url         = COALESCE(sqlc.narg(url), url),
    title       = COALESCE(sqlc.narg(title), title),
    description = COALESCE(sqlc.narg(description), description),
    favicon     = COALESCE(sqlc.narg(favicon), favicon),
    image_url   = COALESCE(sqlc.narg(image_url), image_url),
    folder_id   = COALESCE(sqlc.narg(folder_id), folder_id),
    is_favorite = COALESCE(sqlc.narg(is_favorite), is_favorite)
WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING *;

-- name: RemoveBookmarkFolder :one
UPDATE bookmarks SET folder_id = NULL WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: DeleteBookmark :exec
DELETE FROM bookmarks WHERE id = $1 AND user_id = $2;

-- name: GetTagIDsForBookmark :many
SELECT tag_id FROM bookmark_tags WHERE bookmark_id = $1;

-- name: DeleteBookmarkTags :exec
DELETE FROM bookmark_tags WHERE bookmark_id = $1;

-- name: InsertBookmarkTag :exec
INSERT INTO bookmark_tags (bookmark_id, tag_id) VALUES ($1, $2);

-- name: GetTagIDsForBookmarks :many
SELECT bookmark_id, tag_id FROM bookmark_tags WHERE bookmark_id = ANY($1::uuid[]);
