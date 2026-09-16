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
	"os/exec"
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
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}
	return path, nil
}

// P2R runs paper2remarkable (https://github.com/GjjvdBurg/paper2remarkable)
// against originalLink -- the un-resolved raindrop URL, since p2r has its
// own, better source-specific handling for arXiv/PubMed/ACM/etc. -- and
// returns the path to the resulting formatted PDF (cropped margins,
// stripped arXiv timestamp, descriptive filename). It runs p2r with
// --no-upload: we hand the file to our own uploader afterwards, so there's
// one consistent upload/mark-processed path regardless of file type.
//
// The caller is responsible for removing filepath.Dir(path) once done.
func P2R(bin, rmapiBin, originalLink, workDir string) (string, error) {
	if bin == "" {
		bin = "p2r"
	}

	if err := os.MkdirAll(workDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir workdir: %w", err)
	}
	outDir, err := os.MkdirTemp(workDir, "p2r-")
	if err != nil {
		return "", fmt.Errorf("mkdir temp: %w", err)
	}

	args := []string{"--no-upload"}
	if rmapiBin != "" {
		args = append(args, "--rmapi", rmapiBin)
	}
	args = append(args, originalLink)

	cmd := exec.Command(bin, args...)
	cmd.Dir = outDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(outDir)
		return "", fmt.Errorf("p2r failed: %w\n%s", err, out)
	}

	matches, err := filepath.Glob(filepath.Join(outDir, "*.pdf"))
	if err != nil {
		os.RemoveAll(outDir)
		return "", fmt.Errorf("glob output: %w", err)
	}
	if len(matches) == 0 {
		os.RemoveAll(outDir)
		return "", fmt.Errorf("p2r produced no pdf (output: %s)", out)
	}
	return matches[0], nil
}
