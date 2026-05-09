# r6-map-scrape

Scrape Rainbow Six Siege map pages from Ubisoft and download the available
blueprint ZIP files.

The original Python implementation is preserved in `main.py`. The Go CLI is
the preferred implementation for the R6 replay tooling ecosystem because it can
be built as a small standalone binary and run from automation without a Python
environment.

## Go CLI

```sh
go run ./cmd/r6-map-scrape
```

By default the CLI:

1. Fetches `https://www.ubisoft.com/en-us/game/rainbow-six/siege/game-info/maps`.
2. Finds links with the `maplist__card` class.
3. Opens each map page and finds the `map-details__gallery__button` blueprint
   link.
4. Downloads each blueprint ZIP into `blueprints/`.

`blueprints/` is ignored by git so scraped assets are not committed.

### Options

```sh
go run ./cmd/r6-map-scrape -dry-run
go run ./cmd/r6-map-scrape -out /tmp/r6-blueprints
go run ./cmd/r6-map-scrape -map-concurrency 3 -download-concurrency 4
```

Available flags:

- `-base-url`: base URL used to resolve relative Ubisoft links.
- `-maps-url`: maps listing URL to scrape.
- `-out`: output directory for downloaded ZIP files. Defaults to `blueprints`.
- `-map-concurrency`: concurrent map page requests. Defaults to `3`.
- `-download-concurrency`: concurrent blueprint downloads. Defaults to `4`.
- `-dry-run`: discover and print blueprint URLs without downloading files.

## Python CLI

The Python implementation remains available:

```sh
python main.py
```

Install the dependencies from `pyproject.toml` before running it.

## Development

Run the Go tests before opening a PR:

```sh
gofmt -w ./cmd ./internal
go test ./...
```
