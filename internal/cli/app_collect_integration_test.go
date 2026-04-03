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
