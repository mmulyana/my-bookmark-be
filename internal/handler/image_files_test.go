package handler

import (
	"strings"
	"testing"
)

func TestCleanFileName(t *testing.T) {
	cases := map[string]string{
		"photo.png":           "photo.png",
		"../../etc/passwd":    "passwd",
		`C:\Users\me\cat.jpg`: "cat.jpg",
		"  spaced name.gif  ": "spaced name.gif",
		"bad\x00name\n.png":   "badname.png",
		"..":                  "..",
		".":                   "",
		"":                    "",
	}
	for in, want := range cases {
		if got := cleanFileName(in); got != want {
			t.Errorf("cleanFileName(%q) = %q, want %q", in, got, want)
		}
	}

	long := cleanFileName(strings.Repeat("é", 200) + ".png")
	if len(long) > 255 || !strings.HasPrefix(long, "é") {
		t.Errorf("long name not truncated on a rune boundary: len=%d", len(long))
	}
}
