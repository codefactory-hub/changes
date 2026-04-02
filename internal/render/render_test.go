package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/templates"
)

func TestRenderDropsWholeEntriesOnly(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	if _, err := templates.EnsureDefaultFiles(repoRoot, cfg); err != nil {
		t.Fatalf("EnsureDefaultFiles returned error: %v", err)
	}

	manifest := releases.Manifest{
		Version:       "0.1.0",
		TargetVersion: "0.1.0",
		Channel:       releases.ChannelStable,
		Consumes:      true,
		CreatedAt:     time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		MaxChars:      120,
		Template:      cfg.Render.ReleaseTemplate,
		FragmentIDs:   []string{"f1", "f2"},
	}

	renderer, err := New(repoRoot, cfg, manifest)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	output, err := renderer.Render(cfg, manifest, []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC), Title: "Short title", Type: "fixed", Bump: "patch"}, Body: "first body"},
		{Metadata: fragments.Metadata{ID: "f2", CreatedAt: time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC), Title: "Longer title", Type: "fixed", Bump: "patch"}, Body: "second body that should drop entirely because the character limit is too small to keep every rendered entry"},
	})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(output, "Additional entries omitted for length.") {
		t.Fatalf("output missing omission notice: %s", output)
	}
	if !strings.Contains(output, "Short title") {
		t.Fatalf("output missing first entry: %s", output)
	}
	if strings.Contains(output, "second body that should drop entirely") {
		t.Fatalf("output contains dropped body fragment: %s", output)
	}
	if strings.Contains(output, "Longer title") {
		t.Fatalf("output contains dropped entry title: %s", output)
	}

	_ = os.WriteFile(filepath.Join(repoRoot, "debug.txt"), []byte(output), 0o644)
}
