package handler

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/auth"
)

type Server struct {
	*AuthHandler
	*BookmarkHandler
	*FolderHandler
	*TagHandler
	*MetadataHandler
}

func NewServer(db *pgxpool.Pool, authSvc *auth.Service) *Server {
	return &Server{
		AuthHandler:     NewAuthHandler(db, authSvc),
		BookmarkHandler: NewBookmarkHandler(db),
		FolderHandler:   NewFolderHandler(db),
		TagHandler:      NewTagHandler(db),
		MetadataHandler: NewMetadataHandler(),
	}
}

var _ api.StrictServerInterface = (*Server)(nil)
