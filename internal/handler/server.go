package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/auth"
	"github.com/mmuly/my-bookmark-be/internal/storage"
)

type Server struct {
	*AuthHandler
	*BookmarkHandler
	*FolderHandler
	*TagHandler
	*ImageHandler
	*MetadataHandler
}

func NewServer(db *pgxpool.Pool, authSvc *auth.Service, store storage.Storage, maxImageBytes int64, urlSecret string) *Server {
	return &Server{
		AuthHandler:     NewAuthHandler(db, authSvc),
		BookmarkHandler: NewBookmarkHandler(db),
		FolderHandler:   NewFolderHandler(db),
		TagHandler:      NewTagHandler(db),
		ImageHandler:    NewImageHandler(db, store, maxImageBytes, urlSecret),
		MetadataHandler: NewMetadataHandler(),
	}
}

var _ api.StrictServerInterface = (*Server)(nil)
