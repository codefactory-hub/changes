//go:build devtools

package collection

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
)

func TestReconstructWritesManifestsRenderedOutputAndReport(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	now := time.Date(2026, 4, 3, 4, 0, 0, 0, time.UTC)

	normalizedPath := filepath.Join(repoRoot, ".local/state/changes/go.txt")
	if err := os.MkdirAll(filepath.Dir(normalizedPath), 0o755); err != nil {
		t.Fatalf("mkdir normalized dir: %v", err)
	}
	if err := os.WriteFile(normalizedPath, []byte("## 1.25.0\n\n- Added profile-guided optimization support.\n\n## 1.24.3\n\n- Fixed runtime regressions.\n"), 0o644); err != nil {
		t.Fatalf("write normalized input: %v", err)
	}

	resultSet := ResultSet{
		CatalogPath: "/tmp/catalog.toml",
		CollectedAt: time.Date(2026, 4, 3, 2, 30, 0, 0, time.UTC),
		Results: []Result{
			{
				Source: Source{
					ID:      "go",
					Name:    "Go",
					Product: "Go",
					URL:     "https://go.dev/doc/devel/release",
				},
				NormalizedPath: normalizedPath,
			},
		},
	}

	if _, err := WriteDraftBatch(repoRoot, cfg, "/tmp/collection.json", resultSet, now, bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7}), ""); err != nil {
		t.Fatalf("WriteDraftBatch returned error: %v", err)
	}

	report, err := Reconstruct(repoRoot, "/tmp/collection.json", resultSet)
	if err != nil {
		t.Fatalf("Reconstruct returned error: %v", err)
	}
	if len(report.Products) != 1 || len(report.Products[0].Sources) != 1 {
		t.Fatalf("unexpected report shape: %#v", report)
	}

	workspace := filepath.Join(config.CollectChangesDir(repoRoot), "go")
	for _, path := range []string{
		filepath.Join(workspace, "changes", "releases", "go-0.0.1.toml"),
		filepath.Join(workspace, "changes", "releases", "go-0.0.2.toml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
	renderedPath := filepath.Join(workspace, "changes", "rendered", "go", "repository_markdown.md")
	if _, err := os.Stat(renderedPath); err != nil {
		t.Fatalf("expected rendered output %s: %v", renderedPath, err)
	}
	rendered, err := os.ReadFile(renderedPath)
	if err != nil {
		t.Fatalf("read rendered output: %v", err)
	}
	if !strings.Contains(string(rendered), "Go 1.25.0") {
		t.Fatalf("rendered output missing fragment title:\n%s", rendered)
	}

	reportPath := filepath.Join(config.CollectChangesDir(repoRoot), "reconstruction-report.json")
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report %s: %v", reportPath, err)
	}
	rawReport, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report: %v", err)
	}
	var decoded ReconstructionReport
	if err := json.Unmarshal(rawReport, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.Products[0].Sources[0].ManifestCount != 2 {
		t.Fatalf("manifest count = %d, want 2", decoded.Products[0].Sources[0].ManifestCount)
	}
}

func TestReconstructBuildsSingleManifestForStructuredReleaseDocument(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	now := time.Date(2026, 4, 3, 4, 0, 0, 0, time.UTC)

	normalizedPath := filepath.Join(repoRoot, ".local/state/changes/vscode.txt")
	if err := os.MkdirAll(filepath.Dir(normalizedPath), 0o755); err != nil {
		t.Fatalf("mkdir normalized dir: %v", err)
	}
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
		t.Fatalf("write normalized input: %v", err)
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

	if _, err := WriteDraftBatch(repoRoot, cfg, "/tmp/collection.json", resultSet, now, bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7}), ""); err != nil {
		t.Fatalf("WriteDraftBatch returned error: %v", err)
	}

	report, err := Reconstruct(repoRoot, "/tmp/collection.json", resultSet)
	if err != nil {
		t.Fatalf("Reconstruct returned error: %v", err)
	}

	workspace := filepath.Join(config.CollectChangesDir(repoRoot), "visual-studio-code-repo")
	assertExists := func(path string) {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected file %s: %v", path, err)
		}
	}
	assertExists(filepath.Join(workspace, "changes", "releases", "visual-studio-code-repo-1-110-1.110.1.toml"))

	renderedPath := filepath.Join(workspace, "changes", "rendered", "visual-studio-code-repo-1-110", "repository_markdown.md")
	assertExists(renderedPath)
	rendered, err := os.ReadFile(renderedPath)
	if err != nil {
		t.Fatalf("read rendered output: %v", err)
	}
	if !strings.Contains(string(rendered), "## 1.110.1") || !strings.Contains(string(rendered), "Agent controls") || !strings.Contains(string(rendered), "Terminal") {
		t.Fatalf("rendered output missing expected release metadata or section entries:\n%s", rendered)
	}

	if got := report.Products[0].Sources[0].ManifestCount; got != 1 {
		t.Fatalf("manifest count = %d, want 1", got)
	}
}
