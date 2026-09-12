package handler

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/mmuly/my-bookmark-be/db/sqlc"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/middleware"
)

type BookmarkHandler struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewBookmarkHandler(db *pgxpool.Pool) *BookmarkHandler {
	return &BookmarkHandler{db: db, queries: dbsqlc.New(db)}
}

func toAPIBookmark(b dbsqlc.Bookmark, tagIDs []uuid.UUID) api.Bookmark {
	if tagIDs == nil {
		tagIDs = []uuid.UUID{}
	}
	return api.Bookmark{
		Id:          b.ID,
		Url:         b.Url,
		Title:       b.Title,
		Description: b.Description,
		Favicon:     b.Favicon,
		ImageUrl:    b.ImageUrl,
		FolderId:    b.FolderID,
		TagIds:      tagIDs,
		IsFavorite:  b.IsFavorite,
		CreatedAt:   b.CreatedAt,
	}
}

func (h *BookmarkHandler) ListBookmarks(ctx context.Context, request api.ListBookmarksRequestObject) (api.ListBookmarksResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	page, perPage := 1, 20
	if request.Params.Page != nil {
		page = *request.Params.Page
	}
	if request.Params.PerPage != nil {
		perPage = *request.Params.PerPage
	}
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 20
	}
	if perPage > 100 {
		perPage = 100
	}

	bookmarks, err := h.queries.ListBookmarksByUser(ctx, dbsqlc.ListBookmarksByUserParams{
		UserID:     userID,
		IsFavorite: request.Params.IsFavorite,
		HasFolder:  request.Params.HasFolder,
		HasTags:    request.Params.HasTags,
		PerPage:    perPage,
		Offset:     (page - 1) * perPage,
	})
	if err != nil {
		return nil, err
	}
	if len(bookmarks) == 0 {
		return api.ListBookmarks200JSONResponse{}, nil
	}

	bookmarkIDs := make([]uuid.UUID, len(bookmarks))
	for i, b := range bookmarks {
		bookmarkIDs[i] = b.ID
	}
	tagRows, err := h.queries.GetTagIDsForBookmarks(ctx, bookmarkIDs)
	if err != nil {
		return nil, err
	}
	tagMap := make(map[uuid.UUID][]uuid.UUID)
	for _, row := range tagRows {
		tagMap[row.BookmarkID] = append(tagMap[row.BookmarkID], row.TagID)
	}

	result := make(api.ListBookmarks200JSONResponse, len(bookmarks))
	for i, b := range bookmarks {
		result[i] = toAPIBookmark(b, tagMap[b.ID])
	}
	return result, nil
}

func (h *BookmarkHandler) CreateBookmark(ctx context.Context, request api.CreateBookmarkRequestObject) (api.CreateBookmarkResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if request.Body == nil {
		return api.CreateBookmark400JSONResponse{Message: "invalid request body"}, nil
	}
	input := request.Body
	url := strings.TrimSpace(input.Url)
	title := strings.TrimSpace(input.Title)
	if url == "" || title == "" {
		return api.CreateBookmark400JSONResponse{Message: "URL and title are required"}, nil
	}

	isFavorite := false
	if input.IsFavorite != nil {
		isFavorite = *input.IsFavorite
	}

	bookmark, err := h.queries.CreateBookmark(ctx, dbsqlc.CreateBookmarkParams{
		UserID:      userID,
		Url:         url,
		Title:       title,
		Description: input.Description,
		Favicon:     input.Favicon,
		ImageUrl:    input.ImageUrl,
		FolderID:    input.FolderId,
		IsFavorite:  isFavorite,
	})
	if err != nil {
		return nil, err
	}

	var tagIDStrs []string
	if input.TagIds != nil {
		for _, id := range *input.TagIds {
			tagIDStrs = append(tagIDStrs, id.String())
		}
	}
	tagIDs, err := h.syncTags(ctx, bookmark.ID, tagIDStrs)
	if err != nil {
		return nil, err
	}

	return api.CreateBookmark201JSONResponse(toAPIBookmark(bookmark, tagIDs)), nil
}

func (h *BookmarkHandler) UpdateBookmark(ctx context.Context, request api.UpdateBookmarkRequestObject) (api.UpdateBookmarkResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if request.Body == nil {
		return api.UpdateBookmark404JSONResponse{Message: "invalid request body"}, nil
	}
	input := request.Body

	bookmark, err := h.queries.UpdateBookmark(ctx, dbsqlc.UpdateBookmarkParams{
		ID:          request.Id,
		UserID:      userID,
		Url:         input.Url,
		Title:       input.Title,
		Description: input.Description,
		Favicon:     input.Favicon,
		ImageUrl:    input.ImageUrl,
		FolderID:    input.FolderId,
		IsFavorite:  input.IsFavorite,
	})
	if err != nil {
		return api.UpdateBookmark404JSONResponse{Message: "Bookmark not found"}, nil
	}

	var tagIDs []uuid.UUID
	if input.TagIds != nil {
		tagIDStrs := make([]string, len(*input.TagIds))
		for i, id := range *input.TagIds {
			tagIDStrs[i] = id.String()
		}
		tagIDs, err = h.syncTags(ctx, bookmark.ID, tagIDStrs)
		if err != nil {
			return nil, err
		}
	} else {
		tagIDs, err = h.queries.GetTagIDsForBookmark(ctx, bookmark.ID)
		if err != nil {
			return nil, err
		}
	}

	return api.UpdateBookmark200JSONResponse(toAPIBookmark(bookmark, tagIDs)), nil
}

func (h *BookmarkHandler) DeleteBookmark(ctx context.Context, request api.DeleteBookmarkRequestObject) (api.DeleteBookmarkResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if _, err := h.queries.GetBookmarkByID(ctx, dbsqlc.GetBookmarkByIDParams{ID: request.Id, UserID: userID}); err != nil {
		return api.DeleteBookmark404JSONResponse{Message: "Bookmark not found"}, nil
	}
	if err := h.queries.DeleteBookmark(ctx, dbsqlc.DeleteBookmarkParams{ID: request.Id, UserID: userID}); err != nil {
		return nil, err
	}
	success := true
	return api.DeleteBookmark200JSONResponse{Success: &success}, nil
}

func (h *BookmarkHandler) RemoveBookmarkFolder(ctx context.Context, request api.RemoveBookmarkFolderRequestObject) (api.RemoveBookmarkFolderResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	bookmark, err := h.queries.RemoveBookmarkFolder(ctx, dbsqlc.GetBookmarkByIDParams{ID: request.Id, UserID: userID})
	if err != nil {
		return api.RemoveBookmarkFolder404JSONResponse{Message: "Bookmark not found"}, nil
	}
	tagIDs, err := h.queries.GetTagIDsForBookmark(ctx, bookmark.ID)
	if err != nil {
		return nil, err
	}
	return api.RemoveBookmarkFolder200JSONResponse(toAPIBookmark(bookmark, tagIDs)), nil
}

// syncTags replaces bookmark tag associations. Returns the final tag IDs.
func (h *BookmarkHandler) syncTags(ctx context.Context, bookmarkID uuid.UUID, tagIDStrs []string) ([]uuid.UUID, error) {
	if err := h.queries.DeleteBookmarkTags(ctx, bookmarkID); err != nil {
		return nil, err
	}
	result := []uuid.UUID{}
	for _, tidStr := range tagIDStrs {
		tid, err := uuid.Parse(tidStr)
		if err != nil {
			continue // skip invalid
		}
		if err := h.queries.InsertBookmarkTag(ctx, dbsqlc.InsertBookmarkTagParams{
			BookmarkID: bookmarkID,
			TagID:      tid,
		}); err != nil {
			return nil, err
		}
		result = append(result, tid)
	}
	return result, nil
}
