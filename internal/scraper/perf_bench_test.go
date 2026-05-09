package scraper

import (
	"fmt"
	"strings"
	"testing"
)

var benchMapHTML = func() string {
	var b strings.Builder
	for i := 0; i < 80; i++ {
		fmt.Fprintf(&b, `<a class="teaser maplist__card featured" href="/en-us/game/rainbow-six/siege/game-info/maps/map-%d">Map %d</a>\n`, i, i)
	}
	return b.String()
}()

var benchBlueprintHTML = `<html><body>` + strings.Repeat(`<a class="x" href="/ignored">x</a>`, 25) + `<a class="button map-details__gallery__button" href="../downloads/bank%20blueprint">Download</a></body></html>`

func BenchmarkParseMapLinks80(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = ParseMapLinks(benchMapHTML, DefaultBaseURL)
	}
}

func BenchmarkParseBlueprintLink(b *testing.B) {
	pageURL := "https://www.ubisoft.com/en-us/game/rainbow-six/siege/game-info/maps/bank"
	for i := 0; i < b.N; i++ {
		_ = ParseBlueprintLink(benchBlueprintHTML, pageURL)
	}
}

func BenchmarkFilenameFromURL(b *testing.B) {
	raw := "https://cdn.example/replays/R6%20Replay.zip?token=abc"
	for i := 0; i < b.N; i++ {
		_ = FilenameFromURL(raw)
	}
}
