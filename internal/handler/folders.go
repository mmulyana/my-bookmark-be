package handler

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/mmuly/my-bookmark-be/db/sqlc"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/middleware"
)

type FolderHandler struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
}

func NewFolderHandler(db *pgxpool.Pool) *FolderHandler {
	return &FolderHandler{db: db, queries: dbsqlc.New(db)}
}

func toAPIFolder(f dbsqlc.Folder) api.Folder {
	return api.Folder{Id: f.ID, Name: f.Name}
}

func (h *FolderHandler) ListFolders(ctx context.Context, request api.ListFoldersRequestObject) (api.ListFoldersResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	folders, err := h.queries.ListFoldersByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make(api.ListFolders200JSONResponse, len(folders))
	for i, f := range folders {
		result[i] = toAPIFolder(f)
	}
	return result, nil
}

func (h *FolderHandler) CreateFolder(ctx context.Context, request api.CreateFolderRequestObject) (api.CreateFolderResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if request.Body == nil {
		return api.CreateFolder400JSONResponse{Message: "invalid request body"}, nil
	}
	name := strings.TrimSpace(request.Body.Name)
	if name == "" {
		return api.CreateFolder400JSONResponse{Message: "Folder name is required"}, nil
	}
	folder, err := h.queries.CreateFolder(ctx, dbsqlc.CreateFolderParams{
		UserID: userID,
		Name:   name,
	})
	if err != nil {
		return nil, err
	}
	return api.CreateFolder201JSONResponse(toAPIFolder(folder)), nil
}

func (h *FolderHandler) UpdateFolder(ctx context.Context, request api.UpdateFolderRequestObject) (api.UpdateFolderResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if request.Body == nil {
		return api.UpdateFolder404JSONResponse{Message: "invalid request body"}, nil
	}
	name := strings.TrimSpace(request.Body.Name)
	if name == "" {
		return api.UpdateFolder404JSONResponse{Message: "Folder name is required"}, nil
	}
	folder, err := h.queries.UpdateFolder(ctx, dbsqlc.UpdateFolderParams{
		ID:     request.Id,
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		return api.UpdateFolder404JSONResponse{Message: "Folder not found"}, nil
	}
	return api.UpdateFolder200JSONResponse(toAPIFolder(folder)), nil
}

func (h *FolderHandler) DeleteFolder(ctx context.Context, request api.DeleteFolderRequestObject) (api.DeleteFolderResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if _, err := h.queries.GetFolderByID(ctx, dbsqlc.GetFolderByIDParams{ID: request.Id, UserID: userID}); err != nil {
		return api.DeleteFolder404JSONResponse{Message: "Folder not found"}, nil
	}
	if err := h.queries.DeleteFolder(ctx, dbsqlc.DeleteFolderParams{ID: request.Id, UserID: userID}); err != nil {
		return nil, err
	}
	success := true
	return api.DeleteFolder200JSONResponse{Success: &success}, nil
}
