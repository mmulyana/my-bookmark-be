package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

func TestLocalRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	n, err := s.Put(ctx, "users/u1/a.png", strings.NewReader("hello"))
	if err != nil || n != 5 {
		t.Fatalf("put: n=%d err=%v", n, err)
	}

	rc, err := s.Open(ctx, "users/u1/a.png")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := io.ReadAll(rc)
	rc.Close()
	if string(got) != "hello" {
		t.Fatalf("got %q", got)
	}

	if err := s.Delete(ctx, "users/u1/a.png"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, "users/u1/a.png"); err != nil {
		t.Fatalf("second delete should be a no-op: %v", err)
	}
	if _, err := s.Open(ctx, "users/u1/a.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

func TestLocalRejectsTraversal(t *testing.T) {
	s, _ := NewLocal(t.TempDir())
	for _, key := range []string{"../x", "/etc/passwd", "a/../../x"} {
		if _, err := s.Put(context.Background(), key, strings.NewReader("x")); err == nil {
			t.Errorf("key %q should be rejected", key)
		}
	}
}
