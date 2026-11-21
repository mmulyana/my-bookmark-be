package handler

import (
	"context"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/mmuly/my-bookmark-be/db/sqlc"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/middleware"
	"github.com/mmuly/my-bookmark-be/internal/storage"
)

type ImageHandler struct {
	db       *pgxpool.Pool
	queries  *dbsqlc.Queries
	storage  storage.Storage
	maxBytes int64
	signer   urlSigner
}

func NewImageHandler(db *pgxpool.Pool, store storage.Storage, maxBytes int64, secret string) *ImageHandler {
	return &ImageHandler{db: db, queries: dbsqlc.New(db), storage: store, maxBytes: maxBytes, signer: newURLSigner(secret)}
}

func (h *ImageHandler) toAPIImage(i dbsqlc.Image, tagIDs []uuid.UUID) api.Image {
	if tagIDs == nil {
		tagIDs = []uuid.UUID{}
	}
	var width, height *int
	if i.Width != nil {
		v := int(*i.Width)
		width = &v
	}
	if i.Height != nil {
		v := int(*i.Height)
		height = &v
	}
	return api.Image{
		Id:        i.ID,
		Name:      i.OriginalName,
		Url:       h.signer.URL(i.ID, time.Now()),
		MimeType:  i.MimeType,
		SizeBytes: i.SizeBytes,
		Width:     width,
		Height:    height,
		TagIds:    tagIDs,
		CreatedAt: i.CreatedAt,
	}
}

func (h *ImageHandler) ListImages(ctx context.Context, request api.ListImagesRequestObject) (api.ListImagesResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	perPage := 40
	if request.Params.PerPage != nil {
		perPage = *request.Params.PerPage
	}
	if perPage < 1 {
		perPage = 40
	}
	if perPage > 100 {
		perPage = 100
	}

	var filterTags []uuid.UUID
	if request.Params.TagIds != nil {
		filterTags = *request.Params.TagIds
	}

	images, err := h.queries.ListImagesByUser(ctx, dbsqlc.ListImagesByUserParams{
		UserID:  userID,
		HasTags: request.Params.HasTags,
		TagIDs:  filterTags,
		LastID:  request.Params.LastId,
		PerPage: perPage,
	})
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return api.ListImages200JSONResponse{}, nil
	}

	ids := make([]uuid.UUID, len(images))
	for i, img := range images {
		ids[i] = img.ID
	}
	tagRows, err := h.queries.GetTagIDsForImages(ctx, ids)
	if err != nil {
		return nil, err
	}
	tagMap := make(map[uuid.UUID][]uuid.UUID)
	for _, row := range tagRows {
		tagMap[row.ImageID] = append(tagMap[row.ImageID], row.TagID)
	}

	result := make(api.ListImages200JSONResponse, len(images))
	for i, img := range images {
		result[i] = h.toAPIImage(img, tagMap[img.ID])
	}
	return result, nil
}

func (h *ImageHandler) UpdateImage(ctx context.Context, request api.UpdateImageRequestObject) (api.UpdateImageResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	if request.Body == nil {
		return api.UpdateImage400JSONResponse{Message: "invalid request body"}, nil
	}
	input := request.Body

	image, err := h.queries.GetImageByID(ctx, dbsqlc.GetImageByIDParams{ID: request.Id, UserID: userID})
	if err != nil {
		return api.UpdateImage404JSONResponse{Message: "Image not found"}, nil
	}

	if input.Name != nil {
		name := cleanFileName(*input.Name)
		if name == "" {
			return api.UpdateImage400JSONResponse{Message: "Name must not be empty"}, nil
		}
		image, err = h.queries.UpdateImageName(ctx, dbsqlc.UpdateImageNameParams{ID: image.ID, UserID: userID, OriginalName: name})
		if err != nil {
			return nil, err
		}
	}

	if input.TagIds != nil {
		wanted := uniqueUUIDs(*input.TagIds)
		owned, err := h.queries.FilterOwnedTagIDs(ctx, dbsqlc.FilterOwnedTagIDsParams{UserID: userID, TagIDs: wanted})
		if err != nil {
			return nil, err
		}
		if len(owned) != len(wanted) {
			return api.UpdateImage400JSONResponse{Message: "Unknown tag"}, nil
		}
		if err := h.queries.ReplaceImageTags(ctx, dbsqlc.ReplaceImageTagsParams{ImageID: image.ID, TagIDs: wanted}); err != nil {
			return nil, err
		}
	}

	tagIDs, err := h.tagIDsFor(ctx, image.ID)
	if err != nil {
		return nil, err
	}
	return api.UpdateImage200JSONResponse(h.toAPIImage(image, tagIDs)), nil
}

func (h *ImageHandler) DeleteImage(ctx context.Context, request api.DeleteImageRequestObject) (api.DeleteImageResponseObject, error) {
	userID := middleware.GetUserID(ctx)
	image, err := h.queries.GetImageByID(ctx, dbsqlc.GetImageByIDParams{ID: request.Id, UserID: userID})
	if err != nil {
		return api.DeleteImage404JSONResponse{Message: "Image not found"}, nil
	}
	if err := h.queries.DeleteImage(ctx, dbsqlc.DeleteImageParams{ID: image.ID, UserID: userID}); err != nil {
		return nil, err
	}
	if err := h.storage.Delete(ctx, image.StorageKey); err != nil {
		log.Printf("delete image file %s: %v", image.StorageKey, err)
	}
	success := true
	return api.DeleteImage200JSONResponse{Success: &success}, nil
}

func (h *ImageHandler) tagIDsFor(ctx context.Context, imageID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := h.queries.GetTagIDsForImages(ctx, []uuid.UUID{imageID})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(rows))
	for i, r := range rows {
		ids[i] = r.TagID
	}
	return ids, nil
}

func uniqueUUIDs(in []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(in))
	out := make([]uuid.UUID, 0, len(in))
	for _, id := range in {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
