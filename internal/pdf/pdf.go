// Package pdf detects links that point (directly or indirectly) at a PDF
// and downloads them as-is, bypassing article extraction entirely. This
// matters for things like arXiv papers, where reflowing into an EPUB would
// mangle equations and figures a plain PDF renders fine.
package pdf

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var arxivAbs = regexp.MustCompile(`^https?://arxiv\.org/abs/`)

const userAgent = "Mozilla/5.0 (compatible; raindrop2rm/1.0; +https://github.com/)"

// IsPDF reports whether link points directly at a PDF -- either by file
// extension, a known landing-page pattern (like an arXiv abstract page),
// or the server's Content-Type.
func IsPDF(link string) (bool, error) {
	if arxivAbs.MatchString(link) {
		return true, nil
	}

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
