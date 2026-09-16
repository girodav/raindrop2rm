// Package extract pulls the readable content out of an article URL using
// go-readability (a Go port of Mozilla's Readability.js).
package extract

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
)

// Article is the cleaned-up content of a page, ready to drop into an EPUB.
type Article struct {
	Title   string
	Content string // sanitized HTML body
}

// FromURL fetches rawURL and extracts its main readable content.
func FromURL(rawURL string) (*Article, error) {
	// Some sites block the default Go User-Agent.
	withUA := func(r *http.Request) {
		r.Header.Set("User-Agent", "Mozilla/5.0 (compatible; raindrop2rm/1.0; +https://github.com/)")
	}

	art, err := readability.FromURL(rawURL, 30*time.Second, withUA)
	if err != nil {
		return nil, fmt.Errorf("readability: %w", err)
	}

	var content strings.Builder
	if err := art.RenderHTML(&content); err != nil {
		return nil, fmt.Errorf("render content: %w", err)
	}

	return &Article{
		Title:   art.Title(),
		Content: content.String(),
	}, nil
}
