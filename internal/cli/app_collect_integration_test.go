//go:build devtools

package cli

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAppCollectsConfiguredChangelogSources(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/go.md":
			w.Header().Set("Content-Type", "text/markdown")
			_, _ = w.Write([]byte("# Go\n\n## 1.22.0\n\nShipped toolchain updates.\n"))
		case "/node.html":
			w.Header().Set("Content-Type", "text/html")
			_, _ = w.Write([]byte(`<!doctype html><html><head><title>Node.js Releases</title></head><body><h1>Node.js Releases</h1><p>Introduced runtime updates.</p></body></html>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 20, 0, 0, 0, time.UTC)
	}
	app.HTTPClient = server.Client()

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}

	catalogPath := filepath.Join(repoRoot, "catalog.toml")
	if err := os.WriteFile(catalogPath, []byte(`
[[sources]]
name = "Go"
url = "`+server.URL+`/go.md"

[[sources]]
name = "Node.js"
url = "`+server.URL+`/node.html"
format = "html"
`), 0o644); err != nil {
		t.Fatalf("write catalog: %v", err)
	}

	outputPath := filepath.Join(repoRoot, ".local/state/changes/collection-report.md")
	stdout.Reset()
	if err := app.Run(context.Background(), []string{"collect", "--catalog", "catalog.toml", "--output", outputPath}); err != nil {
		t.Fatalf("collect returned error: %v\nstderr=%s", err, stderr.String())
	}

	if got := strings.TrimSpace(stdout.String()); got != outputPath {
		t.Fatalf("collect stdout = %q, want %q", got, outputPath)
	}

	report, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	if !strings.Contains(string(report), "# Changelog Collection") {
		t.Fatalf("report missing heading:\n%s", report)
	}
	if !strings.Contains(string(report), "## Go") || !strings.Contains(string(report), "## Node.js") {
		t.Fatalf("report missing collected products:\n%s", report)
	}

	snapshotDir := filepath.Join(repoRoot, ".local/state/changes/collections/20260402T200000Z")
	assertExists(t, filepath.Join(snapshotDir, "manifest.json"))
	assertExists(t, filepath.Join(snapshotDir, "go.md"))
	assertExists(t, filepath.Join(snapshotDir, "go.txt"))
	assertExists(t, filepath.Join(snapshotDir, "node-js.html"))
	assertExists(t, filepath.Join(snapshotDir, "node-js.txt"))
}

func TestAppCollectDraftsWritesImportedFragmentsToState(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 3, 3, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}

	normalizedPath := filepath.Join(repoRoot, ".local/state/changes/go.txt")
	if err := os.WriteFile(normalizedPath, []byte("## 1.25.0\n\n- Added profile-guided optimization support.\n\n## 1.24.3\n\n- Fixed runtime regressions.\n"), 0o644); err != nil {
		t.Fatalf("write normalized file: %v", err)
	}

	inputPath := filepath.Join(repoRoot, ".local/state/changes/collection.json")
	if err := os.WriteFile(inputPath, []byte(`{
  "catalog_path": "/tmp/catalog.toml",
  "collected_at": "2026-04-03T02:30:00Z",
  "results": [
    {
      "source": {
        "ID": "go",
        "Name": "Go",
        "Product": "Go",
        "URL": "https://go.dev/doc/devel/release",
        "Format": "html"
      },
      "fetched_at": "2026-04-03T02:30:00Z",
      "status_code": 200,
      "detected_format": "html",
      "title": "Go Release History",
      "headings": ["Go 1.25"],
      "excerpt": "Go 1.25 includes runtime and toolchain updates.",
      "raw_path": "/tmp/go.html",
      "normalized_path": "`+normalizedPath+`"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write input json: %v", err)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"collect", "drafts", "--input", ".local/state/changes/collection.json"}); err != nil {
		t.Fatalf("collect drafts returned error: %v\nstderr=%s", err, stderr.String())
	}

	outputDir := strings.TrimSpace(stdout.String())
	wantDir := filepath.Join(repoRoot, ".local/state/collect-changes")
	if outputDir != wantDir {
		t.Fatalf("stdout output dir = %q, want %q", outputDir, wantDir)
	}

	fragmentsDir := filepath.Join(outputDir, "go", "changes", "fragments")
	entries, err := os.ReadDir(fragmentsDir)
	if err != nil {
		t.Fatalf("read fragments dir: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("draft count = %d, want 2", len(entries))
	}

	for _, dir := range []string{
		filepath.Join(outputDir, "go", "changes", "fragments"),
		filepath.Join(outputDir, "go", "changes", "releases"),
		filepath.Join(outputDir, "go", "changes", "templates"),
	} {
		assertExists(t, dir)
	}

	draftPath := filepath.Join(fragmentsDir, entries[0].Name())
	raw, err := os.ReadFile(draftPath)
	if err != nil {
		t.Fatalf("read draft: %v", err)
	}
	if !strings.Contains(string(raw), "Added profile-guided optimization support.") && !strings.Contains(string(raw), "Fixed runtime regressions.") {
		t.Fatalf("draft missing extracted section text:\n%s", raw)
	}
}

func TestAppCollectReconstructWritesReport(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 3, 4, 5, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}

	normalizedPath := filepath.Join(repoRoot, ".local/state/changes/go.txt")
	if err := os.WriteFile(normalizedPath, []byte("## 1.25.0\n\n- Added profile-guided optimization support.\n\n## 1.24.3\n\n- Fixed runtime regressions.\n"), 0o644); err != nil {
		t.Fatalf("write normalized file: %v", err)
	}

	inputPath := filepath.Join(repoRoot, ".local/state/changes/collection.json")
	if err := os.WriteFile(inputPath, []byte(`{
  "catalog_path": "/tmp/catalog.toml",
  "collected_at": "2026-04-03T02:30:00Z",
  "results": [
    {
      "source": {
        "ID": "go",
        "Name": "Go",
        "Product": "Go",
        "URL": "https://go.dev/doc/devel/release",
        "Format": "html"
      },
      "normalized_path": "`+normalizedPath+`"
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write input json: %v", err)
	}

	if err := app.Run(context.Background(), []string{"collect", "drafts", "--input", ".local/state/changes/collection.json"}); err != nil {
		t.Fatalf("collect drafts returned error: %v\nstderr=%s", err, stderr.String())
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"collect", "reconstruct", "--input", ".local/state/changes/collection.json"}); err != nil {
		t.Fatalf("collect reconstruct returned error: %v\nstderr=%s", err, stderr.String())
	}

	reportPath := strings.TrimSpace(stdout.String())
	wantReportPath := filepath.Join(repoRoot, ".local/state/collect-changes", "reconstruction-report.json")
	if reportPath != wantReportPath {
		t.Fatalf("stdout report path = %q, want %q", reportPath, wantReportPath)
	}
	assertExists(t, reportPath)
	assertExists(t, filepath.Join(repoRoot, ".local/state/collect-changes", "go", "changes", "rendered", "go", "repository_markdown.md"))
	assertExists(t, filepath.Join(repoRoot, ".local/state/collect-changes", "go", "changes", "releases", "go-0.0.1.toml"))
}
