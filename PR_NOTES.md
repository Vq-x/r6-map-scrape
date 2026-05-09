# PR Notes

## Suggested PR Title

Port Rainbow Six map scraper CLI to Go

## Suggested PR Body

Use [`PR_DESCRIPTION.md`](./PR_DESCRIPTION.md) as the upstream-ready PR body.

## Summary

- Added a Go CLI at `cmd/r6-map-scrape` that mirrors the Python scraper flow: discover map pages, parse blueprint ZIP links, and download ZIPs.
- Added reusable scraper logic under `internal/scraper` with bounded concurrency for map-page requests and blueprint downloads.
- Added configuration flags for source URLs, output directory, concurrency, and dry-run discovery.
- Added 429 retry handling with `Retry-After` support and exponential backoff.
- Preserved the existing Python implementation in `main.py`.
- Documented CLI usage and flags in `README.md`.
- Ignored `blueprints/` so scraped ZIP assets are not committed.

## Tests and Validation

- Added unit tests for map-link parsing, blueprint-link parsing, filename derivation/sanitization, dry-run output, downloads against a fake HTTP client, ordered concurrent blueprint discovery, and retry-delay logic.
- Added Go microbenchmarks for parser/path hot paths.
- Validation commands run:
  - `gofmt -w ./cmd ./internal`
  - `GOCACHE=/tmp/go-build-cache go test ./...`
  - `GOCACHE=/tmp/go-build-cache go run ./cmd/r6-map-scrape -h`

## Performance

Local CPU benchmarks against equivalent Python hot paths show the Go port is faster:

- Parse 80 map cards: ~5.0x faster after optimization.
- Parse blueprint link: ~13.2x faster after optimization.
- Filename from URL: ~9.0x faster.

During benchmarking, `attr()` was optimized to use precompiled `class`/`href` regexes, reducing map-card parsing from ~1.06 ms/op to ~0.38 ms/op and blueprint-link parsing from ~148 µs/op to ~36 µs/op.

Detailed report: `/home/vqx/openclaw-workspace/research/r6-scraper-go-port-performance-report.md`.

## Reviewer Notes

- The Go parser targets the current Ubisoft markup classes used by the Python scraper: `maplist__card` and `map-details__gallery__button`.
- Real scrape wall-clock time is still expected to be network/download bound; the Go port primarily improves local parser overhead and deployment simplicity.
- The existing Python script remains available for compatibility/reference.

## Prior Execution Notes

- Worker sandbox could not create the nested `feature/go-port` branch or commit; the main session created `feature-go-port-map` and handled commit/cleanup.
- Worker sandbox could not write `/home/vqx/openclaw-workspace/research/r6-map-scrape-go-port-progress.md`; the main session wrote it afterward.
