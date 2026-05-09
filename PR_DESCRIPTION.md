# Port Rainbow Six map scraper CLI to Go

## Summary

This PR adds a Go implementation of the Rainbow Six Siege map blueprint scraper while preserving the existing Python script. The Go CLI follows the same scrape flow as `main.py`: fetch the Ubisoft maps listing, discover map pages, extract each map's blueprint ZIP link, and download the ZIP files locally.

The port is intended to make the scraper easier to run in automation and downstream R6 replay tooling: it builds as a small standalone binary, has no Python runtime/dependency requirement, and includes focused tests around parsing, retry, dry-run, and download behavior.

## What changed

- Added a Go CLI under `cmd/r6-map-scrape`.
- Added reusable scraper logic under `internal/scraper` for:
  - map-card discovery from the Ubisoft maps page;
  - blueprint ZIP link discovery from individual map pages;
  - bounded concurrent map-page fetches and ZIP downloads;
  - configurable source URLs, output directory, and concurrency;
  - dry-run discovery output without writing ZIP files;
  - HTTP 429 handling with `Retry-After` support and exponential backoff;
  - ZIP filename derivation and sanitization.
- Updated `README.md` with Go CLI usage, flags, development commands, and the retained Python CLI entry point.
- Added `blueprints/` to `.gitignore` so downloaded assets are not committed.
- Added Go unit tests and microbenchmarks for parser/path hot paths.

## CLI usage

```sh
go run ./cmd/r6-map-scrape
```

Common options:

```sh
go run ./cmd/r6-map-scrape -dry-run
go run ./cmd/r6-map-scrape -out /tmp/r6-blueprints
go run ./cmd/r6-map-scrape -map-concurrency 3 -download-concurrency 4
```

## Validation

Commands run from this repo:

```sh
gofmt -w ./cmd ./internal
GOCACHE=/tmp/go-build-cache go test ./...
GOCACHE=/tmp/go-build-cache go run ./cmd/r6-map-scrape -h
```

Coverage added for:

- map-link parsing;
- blueprint-link parsing;
- filename derivation/sanitization;
- dry-run output;
- downloads against a fake HTTP client;
- 429 retry-delay parsing for seconds and HTTP-date `Retry-After` values;
- ordered concurrent blueprint discovery while skipping maps without downloads.

## Performance notes

Local CPU microbenchmarks against equivalent Python hot paths show the Go implementation is faster for parser/path operations:

- parsing 80 map cards: ~5.0x faster;
- parsing a blueprint link: ~13.2x faster;
- deriving filenames from URLs: ~9.0x faster.

These are synthetic local benchmarks with no network or download I/O. Real end-to-end scrape time will still be dominated by Ubisoft/CDN latency, rate limits, and ZIP download throughput, but the Go port reduces local parser overhead and removes the Python environment requirement for production runs.

During benchmarking, attribute parsing was optimized to use precompiled `class` and `href` regexes, improving map-card parsing from ~1.06 ms/op to ~0.38 ms/op and blueprint-link parsing from ~148 µs/op to ~36 µs/op.

Detailed report: `/home/vqx/openclaw-workspace/research/r6-scraper-go-port-performance-report.md`.

## Notes for reviewers

- The existing Python implementation remains in `main.py` for compatibility/reference.
- The Go parser intentionally targets the current Ubisoft markup classes used by the Python scraper (`maplist__card` and `map-details__gallery__button`).
- Downloaded blueprint ZIPs are treated as generated local artifacts and ignored by git.
