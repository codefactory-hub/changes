//go:build devtools

package collection

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
)

func TestLoadCatalogRejectsUnknownKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "catalog.toml")
	if err := os.WriteFile(path, []byte(`
[[sources]]
name = "Go"
url = "https://example.com/go.md"
unexpected = "nope"
`), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	_, err := LoadCatalog(path)
	if err == nil || !strings.Contains(err.Error(), "unsupported keys") {
		t.Fatalf("LoadCatalog error = %v, want unsupported keys", err)
	}
}

func TestCollectWritesSnapshotsAndNormalizesMarkdownAndHTML(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/markdown":
			w.Header().Set("Content-Type", "text/markdown")
			_, _ = w.Write([]byte("# Go\n\n## 1.22.0\n\nShipped toolchain updates.\n"))
		case "/html":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Node.js Releases</title></head><body><h1>Node.js Releases</h1><h2>v22.0.0</h2><p>Introduced runtime updates.</p></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repoRoot := t.TempDir()
	cfg := config.Default()
	now := time.Date(2026, 4, 2, 18, 30, 0, 0, time.UTC)

	catalog := Catalog{
		Sources: []Source{
			{Name: "Go", URL: server.URL + "/markdown"},
			{Name: "Node.js", URL: server.URL + "/html"},
		},
	}

	corpus, err := Collect(context.Background(), repoRoot, cfg, server.Client(), filepath.Join(repoRoot, "catalog.toml"), catalog, now)
	if err != nil {
		t.Fatalf("Collect returned error: %v", err)
	}

	if corpus.SourceCount != 2 || corpus.SuccessCount != 2 || corpus.FailureCount != 0 {
		t.Fatalf("unexpected corpus counters: %#v", corpus)
	}

	if _, err := os.Stat(filepath.Join(corpus.SnapshotDir, "manifest.json")); err != nil {
		t.Fatalf("manifest.json missing: %v", err)
	}

	goResult := corpus.Results[0]
	nodeResult := corpus.Results[1]
	if goResult.Source.Name != "Go" {
		goResult, nodeResult = nodeResult, goResult
	}

	if goResult.DetectedFormat != FormatMarkdown {
		t.Fatalf("Go detected format = %q, want markdown", goResult.DetectedFormat)
	}
	if len(goResult.Headings) == 0 || goResult.Headings[0] != "Go" {
		t.Fatalf("Go headings = %v, want extracted markdown headings", goResult.Headings)
	}

	if nodeResult.DetectedFormat != FormatHTML {
		t.Fatalf("Node detected format = %q, want html", nodeResult.DetectedFormat)
	}
	if nodeResult.Title != "Node.js Releases" {
		t.Fatalf("Node title = %q, want Node.js Releases", nodeResult.Title)
	}
	if len(nodeResult.Headings) < 2 || nodeResult.Headings[1] != "v22.0.0" {
		t.Fatalf("Node headings = %v, want HTML headings", nodeResult.Headings)
	}

	for _, result := range corpus.Results {
		if result.RawPath == "" || result.NormalizedPath == "" {
			t.Fatalf("result paths should be populated: %#v", result)
		}
		if _, err := os.Stat(result.RawPath); err != nil {
			t.Fatalf("raw snapshot missing for %s: %v", result.Source.Name, err)
		}
		if _, err := os.Stat(result.NormalizedPath); err != nil {
			t.Fatalf("normalized snapshot missing for %s: %v", result.Source.Name, err)
		}
	}
}

func TestRenderSupportsMarkdownAndJSON(t *testing.T) {
	corpus := Corpus{
		CollectedAt:  time.Date(2026, 4, 2, 19, 0, 0, 0, time.UTC),
		SnapshotDir:  "/tmp/collection",
		SourceCount:  1,
		SuccessCount: 1,
		Results: []Result{
			{
				Source:         Source{Name: "Go", URL: "https://example.com/go.md"},
				DetectedFormat: FormatMarkdown,
				Title:          "Go",
				Headings:       []string{"Go", "1.22.0"},
				Excerpt:        "Shipped toolchain updates.",
			},
		},
	}

	markdown, err := Render(corpus, OutputFormatMarkdown)
	if err != nil {
		t.Fatalf("Render markdown returned error: %v", err)
	}
	if !strings.Contains(markdown, "# Changelog Collection") || !strings.Contains(markdown, "## Go") {
		t.Fatalf("markdown render missing expected content:\n%s", markdown)
	}

	jsonOutput, err := Render(corpus, OutputFormatJSON)
	if err != nil {
		t.Fatalf("Render json returned error: %v", err)
	}
	if !strings.Contains(jsonOutput, "\"source_count\": 1") {
		t.Fatalf("json render missing expected content:\n%s", jsonOutput)
	}
}
