package handler

import (
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestURLSigner(t *testing.T) {
	s := newURLSigner("secret")
	id := uuid.New()
	now := time.Date(2026, 1, 1, 10, 30, 0, 0, time.UTC)

	raw := s.URL(id, now)
	u, err := url.Parse(raw)
	if err != nil || !strings.HasPrefix(u.Path, "/images/"+id.String()+"/file") {
		t.Fatalf("bad url %q: %v", raw, err)
	}
	exp, sig := u.Query().Get("exp"), u.Query().Get("sig")

	if _, ok := s.Verify(id, exp, sig, now); !ok {
		t.Fatal("fresh url should verify")
	}
	if s.URL(id, now.Add(10*time.Minute)) != raw {
		t.Error("urls in the same window should be identical (cache friendly)")
	}
	if _, ok := s.Verify(id, exp, sig, now.Add(3*time.Hour)); ok {
		t.Error("expired url must not verify")
	}
	if _, ok := s.Verify(uuid.New(), exp, sig, now); ok {
		t.Error("signature must be bound to the image id")
	}
	if _, ok := newURLSigner("other").Verify(id, exp, sig, now); ok {
		t.Error("signature must be bound to the secret")
	}
	if _, ok := s.Verify(id, exp, "zz", now); ok {
		t.Error("garbage signature must not verify")
	}
}
