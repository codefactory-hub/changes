//go:build devtools

package collection

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/example/changes/internal/config"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Corpus struct {
	CatalogPath  string    `json:"catalog_path"`
	CollectedAt  time.Time `json:"collected_at"`
	SnapshotDir  string    `json:"snapshot_dir"`
	SourceCount  int       `json:"source_count"`
	SuccessCount int       `json:"success_count"`
	FailureCount int       `json:"failure_count"`
	Results      []Result  `json:"results"`
}

type Result struct {
	Source         Source    `json:"source"`
	FetchedAt      time.Time `json:"fetched_at"`
	StatusCode     int       `json:"status_code"`
	ContentType    string    `json:"content_type"`
	DetectedFormat string    `json:"detected_format"`
	Title          string    `json:"title"`
	Headings       []string  `json:"headings"`
	Excerpt        string    `json:"excerpt"`
	RawPath        string    `json:"raw_path"`
	NormalizedPath string    `json:"normalized_path"`
	Error          string    `json:"error,omitempty"`
}

type snapshotManifest struct {
	CatalogPath string    `json:"catalog_path"`
	CollectedAt time.Time `json:"collected_at"`
	Results     []Result  `json:"results"`
}

func Collect(ctx context.Context, repoRoot string, cfg config.Config, client HTTPClient, catalogPath string, catalog Catalog, now time.Time) (Corpus, error) {
	if client == nil {
		client = http.DefaultClient
	}

	sources, err := catalog.normalizedSources()
	if err != nil {
		return Corpus{}, err
	}

	collectedAt := now.UTC().Truncate(time.Second)
	snapshotDir := filepath.Join(config.CollectionsDir(repoRoot, cfg), collectedAt.Format("20060102T150405Z"))
	if err := os.MkdirAll(snapshotDir, 0o755); err != nil {
		return Corpus{}, fmt.Errorf("create collection snapshot dir: %w", err)
	}

	results := make([]Result, 0, len(sources))
	for _, source := range sources {
		result := collectOne(ctx, client, snapshotDir, source, collectedAt)
		results = append(results, result)
	}

	slices.SortFunc(results, func(a, b Result) int {
		return strings.Compare(a.Source.Name, b.Source.Name)
	})

	corpus := Corpus{
		CatalogPath: catalogPath,
		CollectedAt: collectedAt,
		SnapshotDir: snapshotDir,
		SourceCount: len(results),
		Results:     results,
	}
	for _, result := range results {
		if result.Error == "" {
			corpus.SuccessCount++
			continue
		}
		corpus.FailureCount++
	}

	if err := writeManifest(snapshotDir, snapshotManifest{
		CatalogPath: catalogPath,
		CollectedAt: collectedAt,
		Results:     results,
	}); err != nil {
		return Corpus{}, err
	}

	return corpus, nil
}

func collectOne(ctx context.Context, client HTTPClient, snapshotDir string, source Source, fetchedAt time.Time) Result {
	result := Result{
		Source:    source,
		FetchedAt: fetchedAt,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, source.URL, nil)
	if err != nil {
		result.Error = fmt.Sprintf("build request: %v", err)
		return result
	}
	req.Header.Set("User-Agent", "changes/collect")
	req.Header.Set("Accept", "text/markdown, text/plain, text/html, application/xhtml+xml;q=0.9, */*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		result.Error = fmt.Sprintf("fetch source: %v", err)
		return result
	}
	defer resp.Body.Close()

	result.StatusCode = resp.StatusCode
	result.ContentType = resp.Header.Get("Content-Type")

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		result.Error = fmt.Sprintf("read response body: %v", err)
		return result
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		result.Error = fmt.Sprintf("unexpected HTTP status %d", resp.StatusCode)
	}

	detectedFormat := detectFormat(source, result.ContentType, body)
	result.DetectedFormat = detectedFormat

	normalized := normalizeContent(detectedFormat, body)
	if normalized.Title != "" {
		result.Title = normalized.Title
	} else {
		result.Title = source.Name
	}
	result.Headings = normalized.Headings
	result.Excerpt = normalized.Excerpt

	rawExt := sourceExtension(source.URL, detectedFormat)
	rawPath := filepath.Join(snapshotDir, source.ID+rawExt)
	if err := os.WriteFile(rawPath, body, 0o644); err != nil {
		result.Error = joinError(result.Error, fmt.Sprintf("write raw snapshot: %v", err))
	} else {
		result.RawPath = rawPath
	}

	normalizedPath := filepath.Join(snapshotDir, source.ID+".txt")
	if err := os.WriteFile(normalizedPath, []byte(normalized.Body), 0o644); err != nil {
		result.Error = joinError(result.Error, fmt.Sprintf("write normalized snapshot: %v", err))
	} else {
		result.NormalizedPath = normalizedPath
	}

	return result
}

func writeManifest(snapshotDir string, manifest snapshotManifest) error {
	bytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode collection manifest: %w", err)
	}
	path := filepath.Join(snapshotDir, "manifest.json")
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		return fmt.Errorf("write collection manifest: %w", err)
	}
	return nil
}

type normalizedContent struct {
	Body     string
	Title    string
	Headings []string
	Excerpt  string
}

func detectFormat(source Source, contentType string, body []byte) string {
	if source.Format != "" && source.Format != FormatAuto {
		return source.Format
	}

	mediaType, _, err := mime.ParseMediaType(contentType)
	if err == nil {
		switch mediaType {
		case "text/html", "application/xhtml+xml":
			return FormatHTML
		case "text/markdown":
			return FormatMarkdown
		case "text/plain":
			return FormatText
		}
	}

	path := ""
	if parsed, err := url.Parse(source.URL); err == nil {
		path = parsed.Path
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".html", ".htm":
		return FormatHTML
	case ".md", ".markdown":
		return FormatMarkdown
	case ".txt":
		return FormatText
	}

	content := strings.TrimSpace(string(body))
	switch {
	case strings.HasPrefix(strings.ToLower(content), "<!doctype html"), strings.HasPrefix(strings.ToLower(content), "<html"), strings.Contains(strings.ToLower(content), "<body"):
		return FormatHTML
	case strings.Contains(content, "\n# "), strings.HasPrefix(content, "#"):
		return FormatMarkdown
	default:
		return FormatText
	}
}

func sourceExtension(rawURL, format string) string {
	path := ""
	if parsed, err := url.Parse(rawURL); err == nil {
		path = parsed.Path
	}
	if ext := filepath.Ext(path); ext != "" && len(ext) <= 8 {
		return ext
	}

	switch format {
	case FormatHTML:
		return ".html"
	case FormatMarkdown:
		return ".md"
	default:
		return ".txt"
	}
}

func normalizeContent(format string, body []byte) normalizedContent {
	switch format {
	case FormatHTML:
		return normalizeHTML(body)
	case FormatMarkdown:
		return normalizeMarkdown(body)
	default:
		return normalizeText(body)
	}
}

func normalizeMarkdown(body []byte) normalizedContent {
	text := normalizeWhitespace(string(body))
	headings := extractMarkdownHeadings(text)
	title := ""
	if len(headings) > 0 {
		title = headings[0]
	}
	return normalizedContent{
		Body:     text,
		Title:    title,
		Headings: headings,
		Excerpt:  buildExcerpt(text),
	}
}

func normalizeText(body []byte) normalizedContent {
	text := normalizeWhitespace(string(body))
	return normalizedContent{
		Body:    text,
		Excerpt: buildExcerpt(text),
	}
}

var (
	scriptPattern      = regexp.MustCompile(`(?is)<script\b[^>]*>.*?</script>`)
	stylePattern       = regexp.MustCompile(`(?is)<style\b[^>]*>.*?</style>`)
	tagPattern         = regexp.MustCompile(`(?s)<[^>]+>`)
	htmlTitlePattern   = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	htmlHeadingPattern = regexp.MustCompile(`(?is)<h[1-3][^>]*>(.*?)</h[1-3]>`)
	blockBreakPattern  = regexp.MustCompile(`(?i)</(p|div|section|article|li|ul|ol|h1|h2|h3|h4|h5|h6|tr|table)>|<br\s*/?>`)
)

func normalizeHTML(body []byte) normalizedContent {
	raw := string(body)
	title := cleanupText(extractFirstSubmatch(htmlTitlePattern, raw))

	headings := make([]string, 0)
	for _, match := range htmlHeadingPattern.FindAllStringSubmatch(raw, -1) {
		if len(match) < 2 {
			continue
		}
		heading := cleanupText(match[1])
		if heading != "" {
			headings = append(headings, heading)
		}
	}

	withoutScripts := scriptPattern.ReplaceAllString(raw, " ")
	withoutStyles := stylePattern.ReplaceAllString(withoutScripts, " ")
	withBreaks := blockBreakPattern.ReplaceAllString(withoutStyles, "\n")
	text := cleanupText(tagPattern.ReplaceAllString(withBreaks, " "))

	if title == "" && len(headings) > 0 {
		title = headings[0]
	}

	return normalizedContent{
		Body:     text,
		Title:    title,
		Headings: headings,
		Excerpt:  buildExcerpt(text),
	}
}

func extractFirstSubmatch(pattern *regexp.Regexp, value string) string {
	match := pattern.FindStringSubmatch(value)
	if len(match) < 2 {
		return ""
	}
	return match[1]
}

func cleanupText(value string) string {
	return normalizeWhitespace(html.UnescapeString(value))
}

func normalizeWhitespace(value string) string {
	lines := strings.Split(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines))
	lastBlank := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			if lastBlank {
				continue
			}
			out = append(out, "")
			lastBlank = true
			continue
		}

		trimmed = strings.Join(strings.Fields(trimmed), " ")
		out = append(out, trimmed)
		lastBlank = false
	}
	return strings.TrimSpace(strings.Join(out, "\n"))
}

func extractMarkdownHeadings(value string) []string {
	lines := strings.Split(value, "\n")
	headings := make([]string, 0)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "#") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if heading != "" {
			headings = append(headings, heading)
		}
	}
	return headings
}

func buildExcerpt(value string) string {
	const maxRunes = 280
	if value == "" {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxRunes {
		return value
	}
	return strings.TrimSpace(string(runes[:maxRunes])) + "..."
}

func joinError(current, next string) string {
	if current == "" {
		return next
	}
	return current + "; " + next
}
