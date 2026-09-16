// Package epub turns an extracted article into a single-chapter EPUB file
// on disk, ready to be handed to rmapi.
package epub

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"

	epublib "github.com/go-shiori/go-epub"

	"raindrop2rm/internal/extract"
)

// Build writes a one-chapter EPUB for art into outDir and returns its path.
// link is included as a byline so you can always get back to the source.
func Build(art *extract.Article, link string, outDir string) (string, error) {
	title := strings.TrimSpace(art.Title)
	if title == "" {
		title = "Untitled article"
	}

	book, err := epublib.NewEpub(title)
	if err != nil {
		return "", fmt.Errorf("new epub: %w", err)
	}
	book.SetTitle(title)

	body := fmt.Sprintf(
		`<h1>%s</h1><p><em><a href="%s">%s</a></em></p>%s`,
		html.EscapeString(title),
		html.EscapeString(link),
		html.EscapeString(link),
		art.Content,
	)

	if _, err := book.AddSection(body, title, "", ""); err != nil {
		return "", fmt.Errorf("add section: %w", err)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir outdir: %w", err)
	}

	path := filepath.Join(outDir, Slugify(title)+".epub")
	if err := book.Write(path); err != nil {
		return "", fmt.Errorf("write epub: %w", err)
	}
	return path, nil
}

// Slugify produces a short, filesystem-safe filename from a title.
func Slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))

	var b strings.Builder
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z' || r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		out = "article"
	}
	if len(out) > 80 {
		out = out[:80]
	}
	return out
}
