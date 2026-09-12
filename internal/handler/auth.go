package handler

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	dbsqlc "github.com/mmuly/my-bookmark-be/db/sqlc"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/auth"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	db      *pgxpool.Pool
	queries *dbsqlc.Queries
	authSvc *auth.Service
}

func NewAuthHandler(db *pgxpool.Pool, authSvc *auth.Service) *AuthHandler {
	return &AuthHandler{db: db, queries: dbsqlc.New(db), authSvc: authSvc}
}

func (h *AuthHandler) issueTokens(userID uuid.UUID) (api.AuthResponse, error) {
	access, err := h.authSvc.GenerateAccessToken(userID)
	if err != nil {
		return api.AuthResponse{}, err
	}
	refresh, err := h.authSvc.GenerateRefreshToken(userID)
	if err != nil {
		return api.AuthResponse{}, err
	}
	return api.AuthResponse{AccessToken: access, RefreshToken: refresh}, nil
}

func (h *AuthHandler) Register(ctx context.Context, request api.RegisterRequestObject) (api.RegisterResponseObject, error) {
	if request.Body == nil {
		return api.Register400JSONResponse{Message: "invalid request body"}, nil
	}
	email := strings.TrimSpace(strings.ToLower(string(request.Body.Email)))
	password := request.Body.Password
	if email == "" || len(password) < 8 {
		return api.Register400JSONResponse{Message: "email and password (min 8 chars) are required"}, nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := h.queries.CreateUser(ctx, dbsqlc.CreateUserParams{
		Email:        email,
		PasswordHash: string(hash),
	})
	if err != nil {
		if strings.Contains(err.Error(), "unique") {
			return api.Register409JSONResponse{Message: "email already taken"}, nil
		}
		return nil, err
	}

	tokens, err := h.issueTokens(user.ID)
	if err != nil {
		return nil, err
	}
	return api.Register201JSONResponse(tokens), nil
}

func (h *AuthHandler) Login(ctx context.Context, request api.LoginRequestObject) (api.LoginResponseObject, error) {
	if request.Body == nil {
		return api.Login401JSONResponse{Message: "invalid credentials"}, nil
	}
	email := strings.TrimSpace(strings.ToLower(string(request.Body.Email)))

	user, err := h.queries.GetUserByEmail(ctx, email)
	if err != nil {
		if err == pgx.ErrNoRows {
			return api.Login401JSONResponse{Message: "invalid credentials"}, nil
		}
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(request.Body.Password)); err != nil {
		return api.Login401JSONResponse{Message: "invalid credentials"}, nil
	}

	tokens, err := h.issueTokens(user.ID)
	if err != nil {
		return nil, err
	}
	return api.Login200JSONResponse(tokens), nil
}

func (h *AuthHandler) RefreshToken(ctx context.Context, request api.RefreshTokenRequestObject) (api.RefreshTokenResponseObject, error) {
	if request.Body == nil {
		return api.RefreshToken401JSONResponse{Message: "invalid or expired refresh token"}, nil
	}
	claims, err := h.authSvc.ValidateToken(request.Body.RefreshToken, auth.RefreshToken)
	if err != nil {
		return api.RefreshToken401JSONResponse{Message: "invalid or expired refresh token"}, nil
	}

	tokens, err := h.issueTokens(claims.UserID)
	if err != nil {
		return nil, err
	}
	return api.RefreshToken200JSONResponse(tokens), nil
}
