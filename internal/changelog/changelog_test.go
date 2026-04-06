package changelog

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/templates"
)

func TestRebuildDeterministic(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	if _, err := templates.EnsureDefaultFiles(repoRoot, cfg); err != nil {
		t.Fatalf("EnsureDefaultFiles returned error: %v", err)
	}

	frags := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC), Type: "added"}, Body: "Bootstrap fragments.\n\nAdded fragment storage."},
		{Metadata: fragments.Metadata{ID: "f2", CreatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC), Type: "fixed"}, Body: "Fix render limits.\n\nDrops whole entries only."},
	}
	records := []releases.ReleaseRecord{
		{
			Product:          "changes",
			Version:          "0.2.0",
			ParentVersion:    "0.1.0",
			CreatedAt:        time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f2"},
		},
		{
			Product:          "changes",
			Version:          "0.1.0",
			CreatedAt:        time.Date(2026, 4, 2, 12, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f1"},
		},
	}

	got, err := Rebuild(repoRoot, cfg, frags, records)
	if err != nil {
		t.Fatalf("Rebuild returned error: %v", err)
	}

	wantBytes, err := os.ReadFile(filepath.Join("testdata", "rebuild.golden"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if got != string(wantBytes) {
		t.Fatalf("Rebuild mismatch:\n--- got ---\n%s\n--- want ---\n%s", got, string(wantBytes))
	}
}
