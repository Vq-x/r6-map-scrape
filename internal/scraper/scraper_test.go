package scraper

import (
	"context"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseMapLinks(t *testing.T) {
	html := `
		<a class="teaser maplist__card featured" href="/en-us/game/rainbow-six/siege/game-info/maps/bank">Bank</a>
		<a class='maplist__card' href='https://example.test/maps/chalet'>Chalet</a>
		<a class="not-maplist__card" href="/ignored">Ignored</a>
	`
	got := ParseMapLinks(html, "https://www.ubisoft.com")
	want := []string{
		"https://www.ubisoft.com/en-us/game/rainbow-six/siege/game-info/maps/bank",
		"https://example.test/maps/chalet",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ParseMapLinks() = %#v, want %#v", got, want)
	}
}

func TestParseBlueprintLink(t *testing.T) {
	html := `<a href="../downloads/bank%20blueprint" class="button map-details__gallery__button">Download</a>`
	got := ParseBlueprintLink(html, "https://www.ubisoft.com/en-us/game/rainbow-six/siege/game-info/maps/bank")
	want := "https://www.ubisoft.com/en-us/game/rainbow-six/siege/game-info/downloads/bank%20blueprint"
	if got != want {
		t.Fatalf("ParseBlueprintLink() = %q, want %q", got, want)
	}
}

func TestParseBlueprintLinkMissing(t *testing.T) {
	if got := ParseBlueprintLink(`<a class="other" href="/file.zip">Download</a>`, "https://example.test/maps/bank"); got != "" {
		t.Fatalf("ParseBlueprintLink() = %q, want empty string", got)
	}
}

func TestFilenameFromURL(t *testing.T) {
	tests := map[string]string{
		"https://example.test/files/bank%20blueprint.zip?token=abc": "bank blueprint.zip",
		"https://example.test/files/chalet":                         "chalet.zip",
		"https://example.test/files/unsafe:name?.zip":               "unsafe_name.zip",
		"https://example.test/":                                     "blueprint.zip",
	}
	for input, want := range tests {
		if got := FilenameFromURL(input); got != want {
			t.Fatalf("FilenameFromURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestGetBlueprintLinksPreservesInputOrderAndSkipsMissing(t *testing.T) {
	baseURL := "https://example.test"
	s := New(testClient(map[string]string{
		"/map-one":   `<a class="map-details__gallery__button" href="/one.zip">One</a>`,
		"/map-two":   `<main>No download</main>`,
		"/map-three": `<a class="map-details__gallery__button" href="/three.zip">Three</a>`,
	}), Config{MapConcurrency: 2})
	got, err := s.GetBlueprintLinks(context.Background(), []string{
		baseURL + "/map-one",
		baseURL + "/map-two",
		baseURL + "/map-three",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{baseURL + "/one.zip", baseURL + "/three.zip"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("GetBlueprintLinks() = %#v, want %#v", got, want)
	}
}

func TestRunDryRun(t *testing.T) {
	baseURL := "https://example.test"

	s := New(testClient(map[string]string{
		"/maps": `<a class="maplist__card" href="/bank">Bank</a>`,
		"/bank": `<a class="map-details__gallery__button" href="/bank.zip">Download</a>`,
	}), Config{
		BaseURL:        baseURL,
		MapsURL:        baseURL + "/maps",
		DownloadDir:    filepath.Join(t.TempDir(), "blueprints"),
		MapConcurrency: 1,
		DryRun:         true,
	})
	var out testWriter
	if err := s.Run(context.Background(), &out); err != nil {
		t.Fatal(err)
	}
	if got, want := out.String(), "Found 1 maps\nFound 1 blueprint zips\n"+baseURL+"/bank.zip\n"; got != want {
		t.Fatalf("Run() output = %q, want %q", got, want)
	}
}

func TestDownloadBlueprints(t *testing.T) {
	dir := t.TempDir()
	s := New(testClient(map[string]string{
		"/bank.zip": "zip bytes",
	}), Config{DownloadDir: dir, DownloadConcurrency: 1})
	files, err := s.DownloadBlueprints(context.Background(), []string{"https://example.test/bank.zip"})
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("downloaded %d files, want 1", len(files))
	}
	body, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "zip bytes" {
		t.Fatalf("downloaded body = %q", body)
	}
}

func TestRetryDelay(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC) }
	if got := retryDelay("2.5", 0, now); got != 2500*time.Millisecond {
		t.Fatalf("retryDelay seconds = %s", got)
	}
	if got := retryDelay("Sat, 09 May 2026 12:00:03 GMT", 0, now); got != 3*time.Second {
		t.Fatalf("retryDelay date = %s", got)
	}
	if got := retryDelay("bad", 2, now); got != 4*time.Second {
		t.Fatalf("retryDelay fallback = %s", got)
	}
}

type testWriter struct {
	buf []byte
}

func (w *testWriter) Write(p []byte) (int, error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

func (w *testWriter) String() string {
	return string(w.buf)
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func testClient(routes map[string]string) *http.Client {
	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			body, ok := routes[req.URL.Path]
			if !ok {
				return &http.Response{
					StatusCode: http.StatusNotFound,
					Status:     "404 Not Found",
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("not found")),
					Request:    req,
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Status:     "200 OK",
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
				Request:    req,
			}, nil
		}),
	}
}
