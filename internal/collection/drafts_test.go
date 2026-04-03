//go:build devtools

package collection

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
)

func TestLoadResultSetReadsCollectionJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "collection.json")
	if err := os.WriteFile(path, []byte(`{
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
      "excerpt": "Go 1.25 includes runtime and toolchain updates."
    }
  ]
}`), 0o644); err != nil {
		t.Fatalf("write collection json: %v", err)
	}

	got, err := LoadResultSet(path)
	if err != nil {
		t.Fatalf("LoadResultSet returned error: %v", err)
	}
	if got.CatalogPath != "/tmp/catalog.toml" || len(got.Results) != 1 {
		t.Fatalf("unexpected result set: %#v", got)
	}
}

func TestWriteDraftBatchWritesFragmentShapedDraftsToState(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	now := time.Date(2026, 4, 3, 2, 45, 0, 0, time.UTC)
	normalizedPath := filepath.Join(repoRoot, "slack-mac.txt")
	if err := os.WriteFile(normalizedPath, []byte("## 4.48.100\n\n- Fixed login issue.\n\n## 4.48.0\n\n- Added workflow shortcuts.\n"), 0o644); err != nil {
		t.Fatalf("write normalized file: %v", err)
	}

	resultSet := ResultSet{
		CatalogPath: "/tmp/catalog.toml",
		CollectedAt: time.Date(2026, 4, 3, 2, 30, 0, 0, time.UTC),
		Results: []Result{
			{
				Source: Source{
					ID:      "slack-mac",
					Name:    "Slack for Mac",
					Product: "Slack",
					URL:     "https://slack.com/release-notes/mac",
				},
				DetectedFormat: "html",
				Title:          "Slack for Mac - Release Notes",
				Headings:       []string{"Slack 4.48.100", "Bug Fixes"},
				Excerpt:        "This release includes small security improvements.",
				RawPath:        "/tmp/raw.html",
				NormalizedPath: normalizedPath,
			},
			{
				Source: Source{
					ID:   "bad-source",
					Name: "Bad Source",
					URL:  "https://example.com",
				},
				Error: "fetch source: timeout",
			},
		},
	}

	batch, err := WriteDraftBatch(repoRoot, cfg, "/tmp/collection.json", resultSet, now, bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7}), "")
	if err != nil {
		t.Fatalf("WriteDraftBatch returned error: %v", err)
	}

	wantDir := config.CollectChangesDir(repoRoot)
	if batch.OutputDir != wantDir {
		t.Fatalf("output dir = %q, want %q", batch.OutputDir, wantDir)
	}
	if len(batch.Drafts) != 2 {
		t.Fatalf("draft count = %d, want 2", len(batch.Drafts))
	}

	wantDraftPathPrefix := filepath.Join(wantDir, "slack", "changes", "fragments") + string(filepath.Separator)
	if !strings.HasPrefix(batch.Drafts[0].Path, wantDraftPathPrefix) {
		t.Fatalf("draft path = %q, want prefix %q", batch.Drafts[0].Path, wantDraftPathPrefix)
	}

	for _, dir := range []string{
		filepath.Join(wantDir, "slack", "changes", "fragments"),
		filepath.Join(wantDir, "slack", "changes", "releases"),
		filepath.Join(wantDir, "slack", "changes", "templates"),
	} {
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("expected product workspace dir %s: %v", dir, err)
		}
	}

	raw, err := os.ReadFile(batch.Drafts[0].Path)
	if err != nil {
		t.Fatalf("read draft file: %v", err)
	}
	parsed, err := fragments.Parse(raw)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if parsed.Title != "Slack for Mac 4.48.100" {
		t.Fatalf("title = %q", parsed.Title)
	}
	if !strings.Contains(parsed.Body, "- Fixed login issue.") {
		t.Fatalf("draft body missing extracted section:\n%s", parsed.Body)
	}
	if strings.Contains(parsed.Body, "Imported upstream changelog draft") {
		t.Fatalf("draft body should not contain draft warning text:\n%s", parsed.Body)
	}
}

func TestWriteDraftBatchSplitsStructuredReleaseDocumentsOnH2(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	now := time.Date(2026, 4, 3, 2, 45, 0, 0, time.UTC)
	normalizedPath := filepath.Join(repoRoot, "vscode.txt")
	content := `---
DownloadVersion: 1.110.1
ProductEdition: Stable
---
# February 2026 (version 1.110)

Intro paragraph.

## Agent controls

- Added session controls.

## Terminal

- Fixed terminal rendering.
`
	if err := os.WriteFile(normalizedPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write normalized file: %v", err)
	}

	resultSet := ResultSet{
		CatalogPath: "/tmp/catalog.toml",
		CollectedAt: time.Date(2026, 4, 3, 2, 30, 0, 0, time.UTC),
		Results: []Result{
			{
				Source: Source{
					ID:      "visual-studio-code-repo-1-110",
					Name:    "Visual Studio Code 1.110",
					Product: "Visual Studio Code Repo",
					URL:     "https://raw.githubusercontent.com/microsoft/vscode-docs/main/release-notes/v1_110.md",
				},
				NormalizedPath: normalizedPath,
			},
		},
	}

	batch, err := WriteDraftBatch(repoRoot, cfg, "/tmp/collection.json", resultSet, now, bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7}), "")
	if err != nil {
		t.Fatalf("WriteDraftBatch returned error: %v", err)
	}
	if len(batch.Drafts) != 2 {
		t.Fatalf("draft count = %d, want 2", len(batch.Drafts))
	}

	titles := []string{batch.Drafts[0].Title, batch.Drafts[1].Title}
	if !slices.Equal(titles, []string{"Agent controls", "Terminal"}) {
		t.Fatalf("titles = %#v", titles)
	}
	if strings.Contains(batch.Drafts[0].Body, "DownloadVersion") || strings.Contains(batch.Drafts[0].Body, "# February 2026") {
		t.Fatalf("structured release fragment body should exclude front matter and H1:\n%s", batch.Drafts[0].Body)
	}
}
