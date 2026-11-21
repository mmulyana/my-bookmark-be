package dbsqlc

import (
	"context"

	"github.com/google/uuid"
)

const imageColumns = `id, user_id, storage_key, original_name, mime_type, size_bytes, width, height, created_at`

func scanImage(row interface{ Scan(dest ...any) error }) (Image, error) {
	var i Image
	err := row.Scan(&i.ID, &i.UserID, &i.StorageKey, &i.OriginalName, &i.MimeType, &i.SizeBytes, &i.Width, &i.Height, &i.CreatedAt)
	return i, err
}

const createImage = `
INSERT INTO images (user_id, storage_key, original_name, mime_type, size_bytes, width, height)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING ` + imageColumns

type CreateImageParams struct {
	UserID       uuid.UUID
	StorageKey   string
	OriginalName string
	MimeType     string
	SizeBytes    int64
	Width        *int32
	Height       *int32
}

func (q *Queries) CreateImage(ctx context.Context, arg CreateImageParams) (Image, error) {
	return scanImage(q.db.QueryRow(ctx, createImage,
		arg.UserID, arg.StorageKey, arg.OriginalName, arg.MimeType, arg.SizeBytes, arg.Width, arg.Height))
}

const getImageByID = `SELECT ` + imageColumns + ` FROM images WHERE id = $1 AND user_id = $2 LIMIT 1`

type GetImageByIDParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) GetImageByID(ctx context.Context, arg GetImageByIDParams) (Image, error) {
	return scanImage(q.db.QueryRow(ctx, getImageByID, arg.ID, arg.UserID))
}

const getImageForServing = `SELECT ` + imageColumns + ` FROM images WHERE id = $1 LIMIT 1`

func (q *Queries) GetImageForServing(ctx context.Context, id uuid.UUID) (Image, error) {
	return scanImage(q.db.QueryRow(ctx, getImageForServing, id))
}

const listImagesByUser = `
SELECT i.id, i.user_id, i.storage_key, i.original_name, i.mime_type, i.size_bytes, i.width, i.height, i.created_at
FROM images i
WHERE i.user_id = $1
  AND (
    $2::bool IS NULL
    OR EXISTS (SELECT 1 FROM image_tags it WHERE it.image_id = i.id) = $2
  )
  AND (
    cardinality($3::uuid[]) = 0
    OR (SELECT count(*) FROM image_tags it WHERE it.image_id = i.id AND it.tag_id = ANY($3::uuid[]))
       = cardinality($3::uuid[])
  )
  AND (
    $4::uuid IS NULL
    OR (i.created_at, i.id) < (
      SELECT cursor.created_at, cursor.id
      FROM images cursor
      WHERE cursor.id = $4 AND cursor.user_id = $1
    )
  )
ORDER BY i.created_at DESC, i.id DESC
LIMIT $5
`

type ListImagesByUserParams struct {
	UserID  uuid.UUID
	HasTags *bool
	TagIDs  []uuid.UUID
	LastID  *uuid.UUID
	PerPage int
}

func (q *Queries) ListImagesByUser(ctx context.Context, arg ListImagesByUserParams) ([]Image, error) {
	tagIDs := arg.TagIDs
	if tagIDs == nil {
		tagIDs = []uuid.UUID{}
	}
	rows, err := q.db.Query(ctx, listImagesByUser, arg.UserID, arg.HasTags, tagIDs, arg.LastID, arg.PerPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	images := []Image{}
	for rows.Next() {
		i, err := scanImage(rows)
		if err != nil {
			return nil, err
		}
		images = append(images, i)
	}
	return images, rows.Err()
}

const updateImageName = `UPDATE images SET original_name = $3 WHERE id = $1 AND user_id = $2 RETURNING ` + imageColumns

type UpdateImageNameParams struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	OriginalName string
}

func (q *Queries) UpdateImageName(ctx context.Context, arg UpdateImageNameParams) (Image, error) {
	return scanImage(q.db.QueryRow(ctx, updateImageName, arg.ID, arg.UserID, arg.OriginalName))
}

const deleteImage = `DELETE FROM images WHERE id = $1 AND user_id = $2`

type DeleteImageParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) DeleteImage(ctx context.Context, arg DeleteImageParams) error {
	_, err := q.db.Exec(ctx, deleteImage, arg.ID, arg.UserID)
	return err
}

type GetTagIDsForImagesRow struct {
	ImageID uuid.UUID
	TagID   uuid.UUID
}

const getTagIDsForImages = `SELECT image_id, tag_id FROM image_tags WHERE image_id = ANY($1::uuid[])`

func (q *Queries) GetTagIDsForImages(ctx context.Context, imageIDs []uuid.UUID) ([]GetTagIDsForImagesRow, error) {
	rows, err := q.db.Query(ctx, getTagIDsForImages, imageIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []GetTagIDsForImagesRow
	for rows.Next() {
		var r GetTagIDsForImagesRow
		if err := rows.Scan(&r.ImageID, &r.TagID); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

const replaceImageTags = `
WITH removed AS (
  DELETE FROM image_tags WHERE image_id = $1 AND NOT (tag_id = ANY($2::uuid[]))
)
INSERT INTO image_tags (image_id, tag_id)
SELECT $1, unnest($2::uuid[])
ON CONFLICT DO NOTHING
`

type ReplaceImageTagsParams struct {
	ImageID uuid.UUID
	TagIDs  []uuid.UUID
}

func (q *Queries) ReplaceImageTags(ctx context.Context, arg ReplaceImageTagsParams) error {
	tagIDs := arg.TagIDs
	if tagIDs == nil {
		tagIDs = []uuid.UUID{}
	}
	_, err := q.db.Exec(ctx, replaceImageTags, arg.ImageID, tagIDs)
	return err
}

const filterOwnedTagIDs = `SELECT id FROM tags WHERE user_id = $1 AND id = ANY($2::uuid[])`

type FilterOwnedTagIDsParams struct {
	UserID uuid.UUID
	TagIDs []uuid.UUID
}

func (q *Queries) FilterOwnedTagIDs(ctx context.Context, arg FilterOwnedTagIDsParams) ([]uuid.UUID, error) {
	rows, err := q.db.Query(ctx, filterOwnedTagIDs, arg.UserID, arg.TagIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []uuid.UUID{}
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
