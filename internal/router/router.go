package router

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/auth"
	"github.com/mmuly/my-bookmark-be/internal/handler"
	"github.com/mmuly/my-bookmark-be/internal/middleware"
	"github.com/rs/cors"
)

func jsonErrorHandler(status int) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(map[string]string{"message": err.Error()})
	}
}

func New(db *pgxpool.Pool, authSvc *auth.Service) http.Handler {
	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: false,
	})
	r.Use(c.Handler)

	ssi := handler.NewServer(db, authSvc)
	strict := api.NewStrictHandlerWithOptions(ssi, nil, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc:  jsonErrorHandler(http.StatusBadRequest),
		ResponseErrorHandlerFunc: jsonErrorHandler(http.StatusInternalServerError),
	})
	wrapper := api.ServerInterfaceWrapper{Handler: strict}

	r.Post("/auth/register", wrapper.Register)
	r.Post("/auth/login", wrapper.Login)
	r.Post("/auth/refresh", wrapper.RefreshToken)

	r.Group(func(r chi.Router) {
		r.Use(middleware.Authenticate(authSvc))

		r.Get("/bookmarks", wrapper.ListBookmarks)
		r.Post("/bookmarks", wrapper.CreateBookmark)
		r.Patch("/bookmarks/{id}", wrapper.UpdateBookmark)
		r.Delete("/bookmarks/{id}", wrapper.DeleteBookmark)
		r.Delete("/bookmarks/{id}/folder", wrapper.RemoveBookmarkFolder)

		r.Get("/folders", wrapper.ListFolders)
		r.Post("/folders", wrapper.CreateFolder)
		r.Patch("/folders/{id}", wrapper.UpdateFolder)
		r.Delete("/folders/{id}", wrapper.DeleteFolder)

		r.Get("/tags", wrapper.ListTags)
		r.Post("/tags", wrapper.CreateTag)
		r.Patch("/tags/{id}", wrapper.UpdateTag)
		r.Delete("/tags/{id}", wrapper.DeleteTag)

		r.Post("/fetch-metadata", wrapper.FetchMetadata)
	})

	return r
}
