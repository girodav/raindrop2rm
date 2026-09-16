// Package rmupload pushes a local file to the reMarkable cloud by shelling
// out to the rmapi CLI (https://github.com/ddvk/rmapi), which already knows
// how to authenticate, pair devices and speak reMarkable's sync protocol.
package rmupload

import (
	"fmt"
	"os/exec"
)

type Uploader struct {
	bin    string
	folder string
}

// New creates an Uploader that runs the rmapi binary at bin (or "rmapi" if
// empty, i.e. looked up on PATH) and uploads into the given remote folder
// (e.g. "/Raindrop"; empty means the reMarkable root).
func New(bin, folder string) *Uploader {
	if bin == "" {
		bin = "rmapi"
	}
	return &Uploader{bin: bin, folder: folder}
}

// Upload puts the local file at localPath into the configured reMarkable
// folder, creating it first if necessary.
func (u *Uploader) Upload(localPath string) error {
	if u.folder != "" && u.folder != "/" {
		mkdir := exec.Command(u.bin, "mkdir", u.folder)
		// Ignore errors here: rmapi has no "mkdir -p" semantics and errors
		// out if the folder already exists, which is the common case.
		_ = mkdir.Run()
	}

	target := u.folder
	if target == "" {
		target = "/"
	}

	cmd := exec.Command(u.bin, "put", localPath, target)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rmapi put failed: %w\n%s", err, out)
	}
	return nil
}
