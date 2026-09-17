package epub

import (
	"strings"
	"testing"
)

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Hello, World!":        "hello-world",
		"  leading/trailing  ": "leading-trailing",
		"":                     "article",
		"already-slug":         "already-slug",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}

	if got := Slugify(strings.Repeat("a", 200)); len(got) > 80 {
		t.Errorf("Slugify did not truncate: len=%d", len(got))
	}
}
