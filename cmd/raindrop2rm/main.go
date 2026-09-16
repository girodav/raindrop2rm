// Command raindrop2rm watches a Raindrop.io tag and pushes newly tagged
// articles to a reMarkable tablet as EPUBs, via rmapi.
package main

import (
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"raindrop2rm/internal/epub"
	"raindrop2rm/internal/extract"
	"raindrop2rm/internal/pdf"
	"raindrop2rm/internal/raindrop"
	"raindrop2rm/internal/rmupload"
)

type config struct {
	RaindropToken string
	Tag           string
	ArchiveTag    string
	RemarkableDir string
	RmapiBin      string
	P2RBin        string
	WorkDir       string
	PollInterval  time.Duration
}

func loadConfig() config {
	cfg := config{
		RaindropToken: os.Getenv("RAINDROP_TOKEN"),
		Tag:           getenv("RAINDROP_TAG", "remarkable"),
		ArchiveTag:    getenv("RAINDROP_ARCHIVE_TAG", "remarkable-synced"),
		RemarkableDir: getenv("REMARKABLE_FOLDER", "/Raindrop"),
		RmapiBin:      getenv("RMAPI_BIN", "rmapi"),
		P2RBin:        getenv("P2R_BIN", "p2r"),
		WorkDir:       getenv("WORK_DIR", "/tmp/raindrop2rm"),
	}

	if cfg.RaindropToken == "" {
		log.Fatal("RAINDROP_TOKEN is required (Raindrop.io Settings -> Integrations -> For Developers -> Create test token)")
	}

	pollStr := getenv("POLL_INTERVAL", "0")
	seconds, err := strconv.Atoi(pollStr)
	if err != nil {
		log.Fatalf("invalid POLL_INTERVAL %q: %v", pollStr, err)
	}
	cfg.PollInterval = time.Duration(seconds) * time.Second

	return cfg
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	cfg := loadConfig()

	rd := raindrop.NewClient(cfg.RaindropToken)
	up := rmupload.New(cfg.RmapiBin, cfg.RemarkableDir)

	runOnce := func() {
		if err := syncOnce(rd, up, cfg); err != nil {
			log.Printf("sync pass failed: %v", err)
		}
	}

	if cfg.PollInterval <= 0 {
		runOnce()
		return
	}

	log.Printf("starting sync loop, polling every %s", cfg.PollInterval)
	for {
		runOnce()
		time.Sleep(cfg.PollInterval)
	}
}

func syncOnce(rd *raindrop.Client, up *rmupload.Uploader, cfg config) error {
	items, err := rd.ListByTag(cfg.Tag)
	if err != nil {
		return err
	}
	log.Printf("found %d raindrop(s) tagged #%s", len(items), cfg.Tag)

	for _, item := range items {
		if err := processItem(rd, up, cfg, item); err != nil {
			log.Printf("failed %q (%s): %v", item.Title, item.Link, err)
			continue
		}
	}
	return nil
}

func processItem(rd *raindrop.Client, up *rmupload.Uploader, cfg config, item raindrop.Raindrop) error {
	log.Printf("processing %q (%s)", item.Title, item.Link)

	isDirectPDF, err := pdf.IsPDF(item.Link)
	if err != nil {
		log.Printf("pdf check failed for %q, falling back to article extraction: %v", item.Title, err)
	}

	var path string
	if isDirectPDF {
		path, err = pdf.P2R(cfg.P2RBin, cfg.RmapiBin, item.Link, cfg.WorkDir)
		if err != nil {
			return err
		}
		defer os.RemoveAll(filepath.Dir(path))
	} else {
		art, err := extract.FromURL(item.Link)
		if err != nil {
			return err
		}
		path, err = epub.Build(art, item.Link, cfg.WorkDir)
		if err != nil {
			return err
		}
		defer os.Remove(path)
	}

	if err := up.Upload(path); err != nil {
		return err
	}

	if err := rd.MarkProcessed(item, cfg.Tag, cfg.ArchiveTag); err != nil {
		return err
	}

	log.Printf("synced %q -> reMarkable:%s", item.Title, cfg.RemarkableDir)
	return nil
}
