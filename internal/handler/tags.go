package handler

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/mmuly/my-bookmark-be/db/sqlc"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/middleware"
)

type TagHandler struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewTagHandler(db *pgxpool.Pool) *TagHandler {
	return &TagHandler{db: db, queries: dbsqlc.New(db)}
}

func toAPITag(t dbsqlc.Tag) api.Tag {
	return api.Tag{Id: t.ID, Name: t.Name}
}

func (h *TagHandler) ListTags(ctx context.Context, request api.ListTagsRequestObject) (api.ListTagsResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	tags, err := h.queries.ListTagsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make(api.ListTags200JSONResponse, len(tags))
	for i, t := range tags {
		result[i] = toAPITag(t)
	}
	return result, nil
}

func (h *TagHandler) CreateTag(ctx context.Context, request api.CreateTagRequestObject) (api.CreateTagResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if request.Body == nil {
		return api.CreateTag400JSONResponse{Message: "invalid request body"}, nil
	}
	name := strings.TrimSpace(strings.ToLower(request.Body.Name))
	if name == "" {
		return api.CreateTag400JSONResponse{Message: "Tag name is required"}, nil
	}
	tag, err := h.queries.CreateTag(ctx, dbsqlc.CreateTagParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	return api.CreateTag201JSONResponse(toAPITag(tag)), nil
}

func (h *TagHandler) UpdateTag(ctx context.Context, request api.UpdateTagRequestObject) (api.UpdateTagResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if request.Body == nil {
		return api.UpdateTag400JSONResponse{Message: "invalid request body"}, nil
	}
	name := strings.TrimSpace(strings.ToLower(request.Body.Name))
	if name == "" {
		return api.UpdateTag400JSONResponse{Message: "Tag name is required"}, nil
	}
	tag, err := h.queries.UpdateTag(ctx, dbsqlc.UpdateTagParams{
		ID:     request.Id,
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		return api.UpdateTag404JSONResponse{Message: "Tag not found"}, nil
	}
	return api.UpdateTag200JSONResponse(toAPITag(tag)), nil
}

func (h *TagHandler) DeleteTag(ctx context.Context, request api.DeleteTagRequestObject) (api.DeleteTagResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if _, err := h.queries.GetTagByID(ctx, dbsqlc.GetTagByIDParams{ID: request.Id, UserID: userID}); err != nil {
		return api.DeleteTag404JSONResponse{Message: "Tag not found"}, nil
	}
	if err := h.queries.DeleteTag(ctx, dbsqlc.DeleteTagParams{ID: request.Id, UserID: userID}); err != nil {
		return nil, err
	}
	success := true
	return api.DeleteTag200JSONResponse{Success: &success}, nil
}
