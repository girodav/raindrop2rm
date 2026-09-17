// Package pdf detects links that point (directly or indirectly) at a PDF
// and downloads them as-is, bypassing article extraction entirely. This
// matters for things like arXiv papers, where reflowing into an EPUB would
// mangle equations and figures a plain PDF renders fine.
package pdf

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"raindrop2rm/internal/epub"
)

var arxivAbs = regexp.MustCompile(`^https?://arxiv\.org/abs/(.+)$`)

const userAgent = "Mozilla/5.0 (compatible; raindrop2rm/1.0; +https://github.com/)"

// ResolveURL rewrites known "landing page" URLs (like arXiv abstract pages)
// to their direct PDF link. Any other URL is returned unchanged.
func ResolveURL(link string) string {
	if m := arxivAbs.FindStringSubmatch(link); m != nil {
		return "https://arxiv.org/pdf/" + m[1]
	}
	return link
}

// IsPDF reports whether link points directly at a PDF, either by file
// extension or by the server's Content-Type.
func IsPDF(link string) (bool, error) {
	if before, _, found := strings.Cut(link, "?"); found {
		link = before
	}
	if strings.HasSuffix(strings.ToLower(link), ".pdf") {
		return true, nil
	}

	req, err := http.NewRequest(http.MethodHead, link, nil)
	if err != nil {
		return false, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Errorf("head request: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	return strings.HasPrefix(resp.Header.Get("Content-Type"), "application/pdf"), nil
}

// Download fetches link (a direct PDF URL) into outDir, named after title,
// and returns the local path.
func Download(link, title, outDir string) (string, error) {
	req, err := http.NewRequest(http.MethodGet, link, nil)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("fetch: status %d", resp.StatusCode)
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir outdir: %w", err)
	}

	path := filepath.Join(outDir, epub.Slugify(title)+".pdf")
	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close() //nolint:errcheck

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return path, nil
}
