# PR Notes

## Summary

- Added a Go CLI at `cmd/r6-map-scrape` that mirrors the Python scraper flow:
  discover map pages, parse blueprint ZIP links, and download ZIPs.
- Added reusable scraper logic under `internal/scraper`.
- Preserved the existing Python implementation in `main.py`.
- Documented CLI usage and flags in `README.md`.
- Ignored `blueprints/` so scraped ZIP assets are not committed.

## Tests and Validation

- Added unit tests for map-link parsing, blueprint-link parsing, filename
  derivation, dry-run behavior, downloads against `httptest`, and retry-delay
  logic.
- Validation commands run:
  - `gofmt -w ./cmd ./internal`
  - `GOCACHE=/tmp/go-build-cache go test ./...`
  - `GOCACHE=/tmp/go-build-cache go run ./cmd/r6-map-scrape -h`

## Blockers

- Worker sandbox could not create the nested `feature/go-port` branch or commit; the main session created `feature-go-port-map` and handled commit/cleanup.
- Worker sandbox could not write `/home/vqx/openclaw-workspace/research/r6-map-scrape-go-port-progress.md`; the main session wrote it afterward.

## Suggested PR Title

Port scraper CLI to Go

## Suggested PR Body

Adds a Go implementation of the Rainbow Six Siege map blueprint scraper while
preserving the existing Python version. The Go CLI supports dry runs, bounded
concurrency for map and download requests, configurable source/output paths,
429 retry handling, and focused tests for the pure parsing/path behavior.
