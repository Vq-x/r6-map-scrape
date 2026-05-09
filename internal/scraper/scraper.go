package scraper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL             = "https://www.ubisoft.com"
	DefaultMapsURL             = DefaultBaseURL + "/en-us/game/rainbow-six/siege/game-info/maps"
	DefaultDownloadDir         = "blueprints"
	DefaultMapConcurrency      = 3
	DefaultDownloadConcurrency = 4
	MaxRetries                 = 5
	UserAgent                  = "r6-maps-scrape-go/0.1.0"
)

type Config struct {
	BaseURL             string
	MapsURL             string
	DownloadDir         string
	MapConcurrency      int
	DownloadConcurrency int
	DryRun              bool
}

type Scraper struct {
	client *http.Client
	cfg    Config
}

func New(client *http.Client, cfg Config) *Scraper {
	if client == nil {
		client = http.DefaultClient
	}
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.MapsURL == "" {
		cfg.MapsURL = DefaultMapsURL
	}
	if cfg.DownloadDir == "" {
		cfg.DownloadDir = DefaultDownloadDir
	}
	if cfg.MapConcurrency < 1 {
		cfg.MapConcurrency = DefaultMapConcurrency
	}
	if cfg.DownloadConcurrency < 1 {
		cfg.DownloadConcurrency = DefaultDownloadConcurrency
	}
	return &Scraper{client: client, cfg: cfg}
}

func (s *Scraper) Run(ctx context.Context, out io.Writer) error {
	mapLinks, err := s.GetMapLinks(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found %d maps\n", len(mapLinks))

	blueprintLinks, err := s.GetBlueprintLinks(ctx, mapLinks)
	if err != nil {
		return err
	}
	fmt.Fprintf(out, "Found %d blueprint zips\n", len(blueprintLinks))

	if s.cfg.DryRun {
		for _, link := range blueprintLinks {
			fmt.Fprintln(out, link)
		}
		return nil
	}

	downloaded, err := s.DownloadBlueprints(ctx, blueprintLinks)
	if err != nil {
		return err
	}
	for _, file := range downloaded {
		fmt.Fprintf(out, "Downloaded %s\n", file)
	}
	return nil
}

func (s *Scraper) GetMapLinks(ctx context.Context) ([]string, error) {
	body, err := s.requestWithRetries(ctx, http.MethodGet, s.cfg.MapsURL)
	if err != nil {
		return nil, err
	}
	return ParseMapLinks(string(body), s.cfg.BaseURL), nil
}

func (s *Scraper) GetBlueprintLinks(ctx context.Context, mapLinks []string) ([]string, error) {
	return parallelMap(ctx, s.cfg.MapConcurrency, mapLinks, func(ctx context.Context, mapLink string) (string, bool, error) {
		body, err := s.requestWithRetries(ctx, http.MethodGet, mapLink)
		if err != nil {
			return "", false, err
		}
		link := ParseBlueprintLink(string(body), mapLink)
		return link, link != "", nil
	})
}

func (s *Scraper) DownloadBlueprints(ctx context.Context, blueprintLinks []string) ([]string, error) {
	if err := os.MkdirAll(s.cfg.DownloadDir, 0o755); err != nil {
		return nil, err
	}

	return parallelMap(ctx, s.cfg.DownloadConcurrency, blueprintLinks, func(ctx context.Context, link string) (string, bool, error) {
		body, err := s.requestWithRetries(ctx, http.MethodGet, link)
		if err != nil {
			return "", false, err
		}
		destination := filepath.Join(s.cfg.DownloadDir, FilenameFromURL(link))
		if err := os.WriteFile(destination, body, 0o644); err != nil {
			return "", false, err
		}
		return destination, true, nil
	})
}

func (s *Scraper) requestWithRetries(ctx context.Context, method, rawURL string) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt <= MaxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", UserAgent)

		resp, err := s.client.Do(req)
		if err != nil {
			lastErr = err
		} else {
			body, readErr := io.ReadAll(resp.Body)
			closeErr := resp.Body.Close()
			if readErr != nil {
				return nil, readErr
			}
			if closeErr != nil {
				return nil, closeErr
			}
			if resp.StatusCode != http.StatusTooManyRequests {
				if resp.StatusCode < 200 || resp.StatusCode > 299 {
					return nil, fmt.Errorf("%s %s: unexpected status %s", method, rawURL, resp.Status)
				}
				return body, nil
			}
			lastErr = fmt.Errorf("%s %s: rate limited", method, rawURL)
			if attempt == MaxRetries {
				return nil, lastErr
			}
			delay := retryDelay(resp.Header.Get("Retry-After"), attempt, time.Now)
			if err := sleepContext(ctx, delay); err != nil {
				return nil, err
			}
			continue
		}

		if attempt == MaxRetries {
			break
		}
		if err := sleepContext(ctx, backoff(attempt)); err != nil {
			return nil, err
		}
	}
	if lastErr == nil {
		lastErr = errors.New("request failed")
	}
	return nil, lastErr
}

var (
	anchorRE    = regexp.MustCompile(`(?is)<a\b[^>]*>`)
	classAttrRE = regexp.MustCompile(`(?is)\sclass\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
	hrefAttrRE  = regexp.MustCompile(`(?is)\shref\s*=\s*("([^"]*)"|'([^']*)'|([^\s>]+))`)
)

func ParseMapLinks(html, baseURL string) []string {
	var links []string
	for _, anchor := range anchorRE.FindAllString(html, -1) {
		if !hasClass(attr(anchor, "class"), "maplist__card") {
			continue
		}
		href := attr(anchor, "href")
		if href == "" {
			continue
		}
		if resolved := ResolveURL(baseURL, href); resolved != "" {
			links = append(links, resolved)
		}
	}
	return links
}

func ParseBlueprintLink(html, pageURL string) string {
	for _, anchor := range anchorRE.FindAllString(html, -1) {
		if !hasClass(attr(anchor, "class"), "map-details__gallery__button") {
			continue
		}
		href := attr(anchor, "href")
		if href == "" {
			return ""
		}
		return ResolveURL(pageURL, href)
	}
	return ""
}

func FilenameFromURL(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	filename := ""
	if err == nil {
		if unescaped, unescapeErr := url.PathUnescape(parsed.Path); unescapeErr == nil {
			filename = path.Base(unescaped)
		} else {
			filename = path.Base(parsed.Path)
		}
	}
	if filename == "." || filename == "/" || filename == "" {
		filename = "blueprint.zip"
	}
	if !strings.HasSuffix(strings.ToLower(filename), ".zip") {
		filename += ".zip"
	}
	return sanitizeFilename(filename)
}

func ResolveURL(baseURL, ref string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}
	parsedRef, err := url.Parse(ref)
	if err != nil {
		return ""
	}
	return base.ResolveReference(parsedRef).String()
}

func attr(tag, name string) string {
	var re *regexp.Regexp
	switch name {
	case "class":
		re = classAttrRE
	case "href":
		re = hrefAttrRE
	default:
		return ""
	}
	match := re.FindStringSubmatch(tag)
	if len(match) == 0 {
		return ""
	}
	for i := 2; i <= 4; i++ {
		if match[i] != "" {
			return htmlUnescape(match[i])
		}
	}
	return ""
}

func hasClass(classes, want string) bool {
	for _, class := range strings.Fields(classes) {
		if class == want {
			return true
		}
	}
	return false
}

func htmlUnescape(s string) string {
	replacer := strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
		"&quot;", `"`,
		"&#39;", "'",
	)
	return replacer.Replace(s)
}

func sanitizeFilename(filename string) string {
	var b strings.Builder
	for _, r := range filename {
		if strings.ContainsRune(`<>:"/\|?*`, r) {
			b.WriteRune('_')
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func retryDelay(retryAfter string, attempt int, now func() time.Time) time.Duration {
	if seconds, err := strconv.ParseFloat(retryAfter, 64); err == nil && seconds >= 0 {
		return time.Duration(seconds * float64(time.Second))
	}
	if retryAt, err := http.ParseTime(retryAfter); err == nil {
		delay := retryAt.Sub(now().UTC())
		if delay > 0 {
			return delay
		}
		return 0
	}
	return backoff(attempt)
}

func backoff(attempt int) time.Duration {
	delay := time.Duration(1<<attempt) * time.Second
	if delay > 30*time.Second {
		return 30 * time.Second
	}
	return delay
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type mappedValue struct {
	index int
	value string
	keep  bool
	err   error
}

func parallelMap(ctx context.Context, concurrency int, values []string, fn func(context.Context, string) (string, bool, error)) ([]string, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	jobs := make(chan int)
	results := make(chan mappedValue, len(values))
	var wg sync.WaitGroup

	for range concurrency {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for index := range jobs {
				value, keep, err := fn(ctx, values[index])
				if err != nil {
					cancel()
				}
				results <- mappedValue{index: index, value: value, keep: keep, err: err}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for index := range values {
			select {
			case <-ctx.Done():
				return
			case jobs <- index:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]string, len(values))
	keep := make([]bool, len(values))
	var firstErr error
	for result := range results {
		if result.err != nil && firstErr == nil {
			firstErr = result.err
		}
		ordered[result.index] = result.value
		keep[result.index] = result.keep
	}
	if firstErr != nil {
		return nil, firstErr
	}

	output := make([]string, 0, len(values))
	for index, value := range ordered {
		if keep[index] {
			output = append(output, value)
		}
	}
	return output, nil
}

func bytesReader(body []byte) io.Reader {
	return bytes.NewReader(body)
}
