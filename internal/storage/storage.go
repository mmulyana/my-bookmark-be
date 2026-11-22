package storage

import (
	"context"
	"errors"
	"io"
)

var ErrNotFound = errors.New("storage: object not found")

type Storage interface {
	Put(ctx context.Context, key string, r io.Reader) (int64, error)
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}
