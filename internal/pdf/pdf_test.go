package pdf

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveURL(t *testing.T) {
	cases := map[string]string{
		"https://arxiv.org/abs/2309.00001": "https://arxiv.org/pdf/2309.00001",
		"https://arxiv.org/pdf/2309.00001": "https://arxiv.org/pdf/2309.00001",
		"https://example.com/blog/post":    "https://example.com/blog/post",
	}
	for in, want := range cases {
		if got := ResolveURL(in); got != want {
			t.Errorf("ResolveURL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestIsPDF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pdf" {
			w.Header().Set("Content-Type", "application/pdf")
		} else {
			w.Header().Set("Content-Type", "text/html")
		}
	}))
	defer srv.Close()

	cases := map[string]bool{
		"https://example.com/paper.pdf":   true,
		"https://example.com/paper.pdf?x": true,
		srv.URL + "/pdf":                  true,
		srv.URL + "/article":              false,
	}
	for link, want := range cases {
		got, err := IsPDF(link)
		if err != nil {
			t.Fatalf("IsPDF(%q) error: %v", link, err)
		}
		if got != want {
			t.Errorf("IsPDF(%q) = %v, want %v", link, got, want)
		}
	}
}
