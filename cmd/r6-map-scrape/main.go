package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/openclaw/r6-map-scrape/internal/scraper"
)

func main() {
	var cfg scraper.Config

	flag.StringVar(&cfg.BaseURL, "base-url", scraper.DefaultBaseURL, "base URL used to resolve relative Ubisoft links")
	flag.StringVar(&cfg.MapsURL, "maps-url", scraper.DefaultMapsURL, "Ubisoft maps listing URL")
	flag.StringVar(&cfg.DownloadDir, "out", scraper.DefaultDownloadDir, "directory for downloaded blueprint ZIP files")
	flag.IntVar(&cfg.MapConcurrency, "map-concurrency", scraper.DefaultMapConcurrency, "concurrent map page requests")
	flag.IntVar(&cfg.DownloadConcurrency, "download-concurrency", scraper.DefaultDownloadConcurrency, "concurrent blueprint downloads")
	flag.BoolVar(&cfg.DryRun, "dry-run", false, "discover blueprint URLs without downloading ZIP files")
	flag.Parse()

	if cfg.MapConcurrency < 1 || cfg.DownloadConcurrency < 1 {
		log.Fatal("concurrency values must be at least 1")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	runner := scraper.New(client, cfg)

	if err := runner.Run(context.Background(), os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "r6-map-scrape: %v\n", err)
		os.Exit(1)
	}
}
