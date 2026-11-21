package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

const urlWindow = time.Hour

type urlSigner struct {
	key []byte
}

func newURLSigner(secret string) urlSigner {
	sum := sha256.Sum256([]byte("image-url:" + secret))
	return urlSigner{key: sum[:]}
}

func (s urlSigner) mac(id uuid.UUID, exp int64) []byte {
	m := hmac.New(sha256.New, s.key)
	fmt.Fprintf(m, "%s.%d", id, exp)
	return m.Sum(nil)
}

func (s urlSigner) URL(id uuid.UUID, now time.Time) string {
	w := int64(urlWindow / time.Second)
	exp := (now.Unix()/w + 2) * w
	return fmt.Sprintf("/images/%s/file?exp=%d&sig=%s", id, exp, hex.EncodeToString(s.mac(id, exp)))
}

func (s urlSigner) Verify(id uuid.UUID, expStr, sig string, now time.Time) (time.Time, bool) {
	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || now.Unix() >= exp {
		return time.Time{}, false
	}
	got, err := hex.DecodeString(sig)
	if err != nil || !hmac.Equal(got, s.mac(id, exp)) {
		return time.Time{}, false
	}
	return time.Unix(exp, 0), true
}
