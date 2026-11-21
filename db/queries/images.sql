-- name: CreateImage :one
INSERT INTO images (user_id, storage_key, original_name, mime_type, size_bytes, width, height)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: GetImageByID :one
SELECT * FROM images WHERE id = $1 AND user_id = $2 LIMIT 1;

-- name: GetImageForServing :one
SELECT * FROM images WHERE id = $1 LIMIT 1;

-- name: ListImagesByUser :many
SELECT i.* FROM images i
WHERE i.user_id = sqlc.arg(user_id)
  AND (
    sqlc.narg(has_tags)::bool IS NULL
    OR EXISTS (SELECT 1 FROM image_tags it WHERE it.image_id = i.id) = sqlc.narg(has_tags)
  )
  AND (
    cardinality(sqlc.arg(tag_ids)::uuid[]) = 0
    OR (SELECT count(*) FROM image_tags it WHERE it.image_id = i.id AND it.tag_id = ANY(sqlc.arg(tag_ids)::uuid[]))
       = cardinality(sqlc.arg(tag_ids)::uuid[])
  )
  AND (
    sqlc.narg(last_id)::uuid IS NULL
    OR (i.created_at, i.id) < (
      SELECT cursor.created_at, cursor.id
      FROM images cursor
      WHERE cursor.id = sqlc.narg(last_id) AND cursor.user_id = sqlc.arg(user_id)
    )
  )
ORDER BY i.created_at DESC, i.id DESC
LIMIT sqlc.arg(per_page);

-- name: UpdateImageName :one
UPDATE images SET original_name = $3 WHERE id = $1 AND user_id = $2 RETURNING *;

-- name: DeleteImage :exec
DELETE FROM images WHERE id = $1 AND user_id = $2;

-- name: GetTagIDsForImages :many
SELECT image_id, tag_id FROM image_tags WHERE image_id = ANY($1::uuid[]);

-- name: ReplaceImageTags :exec
WITH removed AS (
  DELETE FROM image_tags WHERE image_id = $1 AND NOT (tag_id = ANY($2::uuid[]))
)
INSERT INTO image_tags (image_id, tag_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT DO NOTHING;

-- name: FilterOwnedTagIDs :many
SELECT id FROM tags WHERE user_id = $1 AND id = ANY($2::uuid[]);
