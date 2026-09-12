package dbsqlc

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const createUser = `
INSERT INTO users (email, password_hash)
VALUES ($1, $2)
RETURNING id, email, password_hash, created_at
`

type CreateUserParams struct {
	Email        string
	PasswordHash string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, createUser, arg.Email, arg.PasswordHash)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

const getUserByEmail = `SELECT id, email, password_hash, created_at FROM users WHERE email = $1 LIMIT 1`

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRow(ctx, getUserByEmail, email)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

const getUserByID = `SELECT id, email, password_hash, created_at FROM users WHERE id = $1 LIMIT 1`

func (q *Queries) GetUserByID(ctx context.Context, id uuid.UUID) (User, error) {
	row := q.db.QueryRow(ctx, getUserByID, id)
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	return u, err
}

const listFoldersByUser = `SELECT id, user_id, name FROM folders WHERE user_id = $1 ORDER BY name ASC`

func (q *Queries) ListFoldersByUser(ctx context.Context, userID uuid.UUID) ([]Folder, error) {
	rows, err := q.db.Query(ctx, listFoldersByUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var folders []Folder
	for rows.Next() {
		var f Folder
		if err := rows.Scan(&f.ID, &f.UserID, &f.Name); err != nil {
			return nil, err
		}
		folders = append(folders, f)
	}
	return folders, rows.Err()
}

const createFolder = `INSERT INTO folders (user_id, name) VALUES ($1, $2) RETURNING id, user_id, name`

type CreateFolderParams struct {
	UserID uuid.UUID
	Name   string
}

func (q *Queries) CreateFolder(ctx context.Context, arg CreateFolderParams) (Folder, error) {
	row := q.db.QueryRow(ctx, createFolder, arg.UserID, arg.Name)
	var f Folder
	err := row.Scan(&f.ID, &f.UserID, &f.Name)
	return f, err
}

const getFolderByID = `SELECT id, user_id, name FROM folders WHERE id = $1 AND user_id = $2 LIMIT 1`

type GetFolderByIDParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) GetFolderByID(ctx context.Context, arg GetFolderByIDParams) (Folder, error) {
	row := q.db.QueryRow(ctx, getFolderByID, arg.ID, arg.UserID)
	var f Folder
	err := row.Scan(&f.ID, &f.UserID, &f.Name)
	return f, err
}

const updateFolder = `UPDATE folders SET name = $2 WHERE id = $1 AND user_id = $3 RETURNING id, user_id, name`

type UpdateFolderParams struct {
	ID     uuid.UUID
	Name   string
	UserID uuid.UUID
}

func (q *Queries) UpdateFolder(ctx context.Context, arg UpdateFolderParams) (Folder, error) {
	row := q.db.QueryRow(ctx, updateFolder, arg.ID, arg.Name, arg.UserID)
	var f Folder
	err := row.Scan(&f.ID, &f.UserID, &f.Name)
	return f, err
}

const deleteFolder = `DELETE FROM folders WHERE id = $1 AND user_id = $2`

type DeleteFolderParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) DeleteFolder(ctx context.Context, arg DeleteFolderParams) error {
	_, err := q.db.Exec(ctx, deleteFolder, arg.ID, arg.UserID)
	return err
}

const listTagsByUser = `SELECT id, user_id, name FROM tags WHERE user_id = $1 ORDER BY name ASC`

func (q *Queries) ListTagsByUser(ctx context.Context, userID uuid.UUID) ([]Tag, error) {
	rows, err := q.db.Query(ctx, listTagsByUser, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []Tag
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.ID, &t.UserID, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

const createTag = `INSERT INTO tags (user_id, name) VALUES ($1, $2) RETURNING id, user_id, name`

type CreateTagParams struct {
	UserID uuid.UUID
	Name   string
}

func (q *Queries) CreateTag(ctx context.Context, arg CreateTagParams) (Tag, error) {
	row := q.db.QueryRow(ctx, createTag, arg.UserID, arg.Name)
	var t Tag
	err := row.Scan(&t.ID, &t.UserID, &t.Name)
	return t, err
}

const getTagByID = `SELECT id, user_id, name FROM tags WHERE id = $1 AND user_id = $2 LIMIT 1`

type GetTagByIDParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) GetTagByID(ctx context.Context, arg GetTagByIDParams) (Tag, error) {
	row := q.db.QueryRow(ctx, getTagByID, arg.ID, arg.UserID)
	var t Tag
	err := row.Scan(&t.ID, &t.UserID, &t.Name)
	return t, err
}

const updateTag = `UPDATE tags SET name = $2 WHERE id = $1 AND user_id = $3 RETURNING id, user_id, name`

type UpdateTagParams struct {
	ID     uuid.UUID
	Name   string
	UserID uuid.UUID
}

func (q *Queries) UpdateTag(ctx context.Context, arg UpdateTagParams) (Tag, error) {
	row := q.db.QueryRow(ctx, updateTag, arg.ID, arg.Name, arg.UserID)
	var t Tag
	err := row.Scan(&t.ID, &t.UserID, &t.Name)
	return t, err
}

const deleteTag = `DELETE FROM tags WHERE id = $1 AND user_id = $2`

type DeleteTagParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) DeleteTag(ctx context.Context, arg DeleteTagParams) error {
	_, err := q.db.Exec(ctx, deleteTag, arg.ID, arg.UserID)
	return err
}

const listBookmarksByUser = `
SELECT b.id, b.user_id, b.url, b.title, b.description, b.favicon, b.image_url, b.folder_id, b.is_favorite, b.created_at
FROM bookmarks b
WHERE b.user_id = $1
  AND ($2::bool IS NULL OR b.is_favorite = $2)
  AND ($3::bool IS NULL OR (b.folder_id IS NOT NULL) = $3)
  AND (
    $4::bool IS NULL
    OR EXISTS (SELECT 1 FROM bookmark_tags bt WHERE bt.bookmark_id = b.id) = $4
  )
ORDER BY b.created_at DESC
LIMIT $5 OFFSET $6
`

type ListBookmarksByUserParams struct {
	UserID     uuid.UUID
	IsFavorite *bool
	HasFolder  *bool
	HasTags    *bool
	PerPage    int
	Offset     int
}

func (q *Queries) ListBookmarksByUser(ctx context.Context, arg ListBookmarksByUserParams) ([]Bookmark, error) {
	rows, err := q.db.Query(ctx, listBookmarksByUser, arg.UserID, arg.IsFavorite, arg.HasFolder, arg.HasTags, arg.PerPage, arg.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var bookmarks []Bookmark
	for rows.Next() {
		var b Bookmark
		if err := rows.Scan(&b.ID, &b.UserID, &b.Url, &b.Title, &b.Description, &b.Favicon, &b.ImageUrl, &b.FolderID, &b.IsFavorite, &b.CreatedAt); err != nil {
			return nil, err
		}
		bookmarks = append(bookmarks, b)
	}
	return bookmarks, rows.Err()
}

const createBookmark = `
INSERT INTO bookmarks (user_id, url, title, description, favicon, image_url, folder_id, is_favorite)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, url, title, description, favicon, image_url, folder_id, is_favorite, created_at
`

type CreateBookmarkParams struct {
	UserID      uuid.UUID
	Url         string
	Title       string
	Description *string
	Favicon     *string
	ImageUrl    *string
	FolderID    *uuid.UUID
	IsFavorite  bool
}

func (q *Queries) CreateBookmark(ctx context.Context, arg CreateBookmarkParams) (Bookmark, error) {
	row := q.db.QueryRow(ctx, createBookmark,
		arg.UserID, arg.Url, arg.Title, arg.Description,
		arg.Favicon, arg.ImageUrl, arg.FolderID, arg.IsFavorite,
	)
	var b Bookmark
	err := row.Scan(&b.ID, &b.UserID, &b.Url, &b.Title, &b.Description, &b.Favicon, &b.ImageUrl, &b.FolderID, &b.IsFavorite, &b.CreatedAt)
	return b, err
}

const getBookmarkByID = `
SELECT id, user_id, url, title, description, favicon, image_url, folder_id, is_favorite, created_at
FROM bookmarks WHERE id = $1 AND user_id = $2 LIMIT 1
`

type GetBookmarkByIDParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) GetBookmarkByID(ctx context.Context, arg GetBookmarkByIDParams) (Bookmark, error) {
	row := q.db.QueryRow(ctx, getBookmarkByID, arg.ID, arg.UserID)
	var b Bookmark
	err := row.Scan(&b.ID, &b.UserID, &b.Url, &b.Title, &b.Description, &b.Favicon, &b.ImageUrl, &b.FolderID, &b.IsFavorite, &b.CreatedAt)
	return b, err
}

const updateBookmark = `
UPDATE bookmarks
SET url         = COALESCE($3, url),
    title       = COALESCE($4, title),
    description = COALESCE($5, description),
    favicon     = COALESCE($6, favicon),
    image_url   = COALESCE($7, image_url),
    folder_id   = COALESCE($8, folder_id),
    is_favorite = COALESCE($9, is_favorite)
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, url, title, description, favicon, image_url, folder_id, is_favorite, created_at
`

type UpdateBookmarkParams struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Url         *string
	Title       *string
	Description *string
	Favicon     *string
	ImageUrl    *string
	FolderID    *uuid.UUID
	IsFavorite  *bool
}

func (q *Queries) UpdateBookmark(ctx context.Context, arg UpdateBookmarkParams) (Bookmark, error) {
	row := q.db.QueryRow(ctx, updateBookmark,
		arg.ID, arg.UserID, arg.Url, arg.Title,
		arg.Description, arg.Favicon, arg.ImageUrl,
		arg.FolderID, arg.IsFavorite,
	)
	var b Bookmark
	err := row.Scan(&b.ID, &b.UserID, &b.Url, &b.Title, &b.Description, &b.Favicon, &b.ImageUrl, &b.FolderID, &b.IsFavorite, &b.CreatedAt)
	return b, err
}

const removeBookmarkFolder = `
UPDATE bookmarks SET folder_id = NULL
WHERE id = $1 AND user_id = $2
RETURNING id, user_id, url, title, description, favicon, image_url, folder_id, is_favorite, created_at
`

func (q *Queries) RemoveBookmarkFolder(ctx context.Context, arg GetBookmarkByIDParams) (Bookmark, error) {
	row := q.db.QueryRow(ctx, removeBookmarkFolder, arg.ID, arg.UserID)
	var b Bookmark
	err := row.Scan(&b.ID, &b.UserID, &b.Url, &b.Title, &b.Description, &b.Favicon, &b.ImageUrl, &b.FolderID, &b.IsFavorite, &b.CreatedAt)
	return b, err
}

const deleteBookmark = `DELETE FROM bookmarks WHERE id = $1 AND user_id = $2`

type DeleteBookmarkParams struct {
	ID     uuid.UUID
	UserID uuid.UUID
}

func (q *Queries) DeleteBookmark(ctx context.Context, arg DeleteBookmarkParams) error {
	_, err := q.db.Exec(ctx, deleteBookmark, arg.ID, arg.UserID)
	return err
}

const getTagIDsForBookmark = `SELECT tag_id FROM bookmark_tags WHERE bookmark_id = $1`

func (q *Queries) GetTagIDsForBookmark(ctx context.Context, bookmarkID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := q.db.Query(ctx, getTagIDsForBookmark, bookmarkID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

const deleteBookmarkTags = `DELETE FROM bookmark_tags WHERE bookmark_id = $1`

func (q *Queries) DeleteBookmarkTags(ctx context.Context, bookmarkID uuid.UUID) error {
	_, err := q.db.Exec(ctx, deleteBookmarkTags, bookmarkID)
	return err
}

const insertBookmarkTag = `INSERT INTO bookmark_tags (bookmark_id, tag_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`

type InsertBookmarkTagParams struct {
	BookmarkID uuid.UUID
	TagID      uuid.UUID
}

func (q *Queries) InsertBookmarkTag(ctx context.Context, arg InsertBookmarkTagParams) error {
	_, err := q.db.Exec(ctx, insertBookmarkTag, arg.BookmarkID, arg.TagID)
	return err
}

type GetTagIDsForBookmarksRow struct {
	BookmarkID uuid.UUID
	TagID      uuid.UUID
}

const getTagIDsForBookmarks = `SELECT bookmark_id, tag_id FROM bookmark_tags WHERE bookmark_id = ANY($1::uuid[])`

func (q *Queries) GetTagIDsForBookmarks(ctx context.Context, bookmarkIDs []uuid.UUID) ([]GetTagIDsForBookmarksRow, error) {
	rows, err := q.db.Query(ctx, getTagIDsForBookmarks, bookmarkIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []GetTagIDsForBookmarksRow
	for rows.Next() {
		var r GetTagIDsForBookmarksRow
		if err := rows.Scan(&r.BookmarkID, &r.TagID); err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

var _ = pgx.ErrNoRows
