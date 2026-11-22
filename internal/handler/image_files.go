package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	dbsqlc "github.com/mmuly/my-bookmark-be/db/sqlc"
	"github.com/mmuly/my-bookmark-be/internal/api"
	"github.com/mmuly/my-bookmark-be/internal/middleware"
	"github.com/mmuly/my-bookmark-be/internal/storage"
	_ "golang.org/x/image/webp"
)

const maxFilesPerUpload = 20

var allowedImageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type uploadFailure struct {
	Name    string `json:"name"`
	Message string `json:"message"`
}

type uploadResponse struct {
	Images []api.Image     `json:"images"`
	Failed []uploadFailure `json:"failed"`
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeMessage(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"message": msg})
}

func (h *ImageHandler) UploadImages(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes*maxFilesPerUpload+(1<<20))

	mr, err := r.MultipartReader()
	if err != nil {
		writeMessage(w, http.StatusBadRequest, "expected multipart/form-data")
		return
	}

	resp := uploadResponse{Images: []api.Image{}, Failed: []uploadFailure{}}
	count := 0
	for {
		part, err := mr.NextPart()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			var tooBig *http.MaxBytesError
			if errors.As(err, &tooBig) {
				writeMessage(w, http.StatusRequestEntityTooLarge, "request too large")
				return
			}
			writeMessage(w, http.StatusBadRequest, "malformed multipart body")
			return
		}
		if part.FormName() != "files" || part.FileName() == "" {
			continue
		}
		count++
		name := cleanFileName(part.FileName())
		if count > maxFilesPerUpload {
			resp.Failed = append(resp.Failed, uploadFailure{Name: name, Message: fmt.Sprintf("at most %d files per upload", maxFilesPerUpload)})
			continue
		}
		img, msg := h.saveImage(r, userID, name, part)
		if msg != "" {
			resp.Failed = append(resp.Failed, uploadFailure{Name: name, Message: msg})
			continue
		}
		resp.Images = append(resp.Images, h.toAPIImage(img, nil))
	}

	switch {
	case count == 0:
		writeMessage(w, http.StatusBadRequest, `no "files" parts found`)
	case len(resp.Images) == 0:
		writeJSON(w, http.StatusUnprocessableEntity, resp)
	default:
		writeJSON(w, http.StatusCreated, resp)
	}
}

func (h *ImageHandler) saveImage(r *http.Request, userID uuid.UUID, name string, src io.Reader) (dbsqlc.Image, string) {
	ctx := r.Context()

	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return dbsqlc.Image{}, "could not read file"
	}
	head = head[:n]

	mime := http.DetectContentType(head)
	ext, ok := allowedImageTypes[mime]
	if !ok {
		return dbsqlc.Image{}, "unsupported file type (png, jpeg, gif, webp only)"
	}

	key := fmt.Sprintf("users/%s/%s%s", userID, uuid.NewString(), ext)
	body := io.MultiReader(bytes.NewReader(head), io.LimitReader(src, h.maxBytes+1-int64(len(head))))
	size, err := h.storage.Put(ctx, key, body)
	if err != nil {
		log.Printf("store image: %v", err)
		return dbsqlc.Image{}, "could not store file"
	}
	discard := func() {
		if err := h.storage.Delete(ctx, key); err != nil {
			log.Printf("cleanup %s: %v", key, err)
		}
	}
	if size > h.maxBytes {
		discard()
		return dbsqlc.Image{}, fmt.Sprintf("file larger than %d MB", h.maxBytes>>20)
	}

	w, hgt, err := h.dimensions(r, key)
	if err != nil {
		discard()
		return dbsqlc.Image{}, "file is not a valid image"
	}

	img, err := h.queries.CreateImage(ctx, dbsqlc.CreateImageParams{
		UserID:       userID,
		StorageKey:   key,
		OriginalName: name,
		MimeType:     mime,
		SizeBytes:    size,
		Width:        &w,
		Height:       &hgt,
	})
	if err != nil {
		log.Printf("create image row: %v", err)
		discard()
		return dbsqlc.Image{}, "could not save image"
	}
	return img, ""
}

func (h *ImageHandler) dimensions(r *http.Request, key string) (int32, int32, error) {
	rc, err := h.storage.Open(r.Context(), key)
	if err != nil {
		return 0, 0, err
	}
	defer rc.Close()
	cfg, _, err := image.DecodeConfig(rc)
	if err != nil {
		return 0, 0, err
	}
	return int32(cfg.Width), int32(cfg.Height), nil
}

func (h *ImageHandler) ServeImageFile(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		writeMessage(w, http.StatusNotFound, "Image not found")
		return
	}
	q := r.URL.Query()
	expires, ok := h.signer.Verify(id, q.Get("exp"), q.Get("sig"), time.Now())
	if !ok {
		writeMessage(w, http.StatusForbidden, "Link invalid or expired")
		return
	}
	image, err := h.queries.GetImageForServing(r.Context(), id)
	if err != nil {
		writeMessage(w, http.StatusNotFound, "Image not found")
		return
	}
	rc, err := h.storage.Open(r.Context(), image.StorageKey)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			writeMessage(w, http.StatusNotFound, "Image not found")
			return
		}
		log.Printf("open image %s: %v", image.StorageKey, err)
		writeMessage(w, http.StatusInternalServerError, "could not read image")
		return
	}
	defer rc.Close()

	hd := w.Header()
	hd.Set("Content-Type", image.MimeType)
	hd.Set("Content-Length", strconv.FormatInt(image.SizeBytes, 10))
	hd.Set("X-Content-Type-Options", "nosniff")
	hd.Set("Cache-Control", fmt.Sprintf("private, max-age=%d", int(time.Until(expires).Seconds())))
	_, _ = io.Copy(w, rc)
}

func cleanFileName(raw string) string {
	name := filepath.Base(strings.ReplaceAll(raw, "\\", "/"))
	name = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "." || name == "/" {
		return ""
	}
	for len(name) > 255 {
		_, size := utf8.DecodeLastRuneInString(name)
		name = name[:len(name)-size]
	}
	return name
}
