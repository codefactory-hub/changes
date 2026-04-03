package releases

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
)

func TestFinalAndPrereleaseSelectionUsesLineage(t *testing.T) {
	all := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC), Title: "One", Bump: "patch"}, Body: "body one"},
		{Metadata: fragments.Metadata{ID: "f2", CreatedAt: time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC), Title: "Two", Bump: "minor"}, Body: "body two"},
		{Metadata: fragments.Metadata{ID: "f3", CreatedAt: time.Date(2026, 4, 2, 17, 0, 0, 0, time.UTC), Title: "Three", Bump: "patch"}, Body: "body three"},
	}

	records := []ReleaseRecord{
		{
			Product:          "changes",
			Version:          "1.1.0",
			CreatedAt:        time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f1"},
		},
		{
			Product:          "changes",
			Version:          "1.2.0-rc.1",
			ParentVersion:    "1.1.0",
			CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f2"},
		},
		{
			Product:          "changes",
			Version:          "1.2.0-rc.2",
			ParentVersion:    "1.2.0-rc.1",
			CreatedAt:        time.Date(2026, 4, 2, 19, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f3"},
		},
	}
	if err := ValidateSet(records); err != nil {
		t.Fatalf("ValidateSet returned error: %v", err)
	}

	finalUnreleased, err := UnreleasedFinalFragments(all, records, "changes")
	if err != nil {
		t.Fatalf("UnreleasedFinalFragments returned error: %v", err)
	}
	if got := fragmentIDs(finalUnreleased); !equalIDs(got, []string{"f2", "f3"}) {
		t.Fatalf("final unreleased = %#v, want f2,f3", got)
	}

	rcUnreleased, err := UnreleasedPrereleaseFragments(all, records, "changes", "1.2.0", "rc")
	if err != nil {
		t.Fatalf("UnreleasedPrereleaseFragments returned error: %v", err)
	}
	if len(rcUnreleased) != 0 {
		t.Fatalf("rc unreleased = %#v, want none", fragmentIDs(rcUnreleased))
	}

	betaUnreleased, err := UnreleasedPrereleaseFragments(all, records, "changes", "1.2.0", "beta")
	if err != nil {
		t.Fatalf("UnreleasedPrereleaseFragments returned error: %v", err)
	}
	if got := fragmentIDs(betaUnreleased); !equalIDs(got, []string{"f2", "f3"}) {
		t.Fatalf("beta unreleased = %#v, want f2,f3", got)
	}
}

func TestValidateSetRejectsMissingExpectedPrereleaseParent(t *testing.T) {
	records := []ReleaseRecord{
		{
			Product:          "changes",
			Version:          "1.1.0",
			CreatedAt:        time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f1"},
		},
		{
			Product:          "changes",
			Version:          "1.2.0-rc.1",
			ParentVersion:    "1.1.0",
			CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f2"},
		},
		{
			Product:          "changes",
			Version:          "1.2.0-rc.2",
			ParentVersion:    "1.1.0",
			CreatedAt:        time.Date(2026, 4, 2, 19, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f3"},
		},
	}

	if err := ValidateSet(records); err == nil {
		t.Fatalf("ValidateSet should reject a prerelease that skips its expected same-label parent")
	}
}

func TestListRejectsCrossLabelPrereleaseParent(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	if err := os.MkdirAll(config.ReleasesDir(repoRoot, cfg), 0o755); err != nil {
		t.Fatalf("mkdir releases dir: %v", err)
	}

	final := ReleaseRecord{
		Product:          "changes",
		Version:          "1.1.0",
		CreatedAt:        time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f1"},
	}
	rc := ReleaseRecord{
		Product:          "changes",
		Version:          "1.2.0-rc.1",
		ParentVersion:    "1.1.0",
		CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f2"},
	}
	beta := ReleaseRecord{
		Product:          "changes",
		Version:          "1.2.0-beta.1",
		ParentVersion:    "1.2.0-rc.1",
		CreatedAt:        time.Date(2026, 4, 2, 19, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f3"},
	}
	if _, err := Write(repoRoot, cfg, final); err != nil {
		t.Fatalf("Write final returned error: %v", err)
	}
	if _, err := Write(repoRoot, cfg, rc); err != nil {
		t.Fatalf("Write rc returned error: %v", err)
	}
	if _, err := Write(repoRoot, cfg, beta); err != nil {
		t.Fatalf("Write beta returned error: %v", err)
	}

	if _, err := List(repoRoot, cfg); err == nil {
		t.Fatalf("List should reject cross-label prerelease parents")
	}
}

func TestLoadRejectsMismatchedPathProductOrFilename(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	path := filepath.Join(config.ReleasesDir(repoRoot, cfg), "cli-1.2.3.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir product dir: %v", err)
	}

	body := []byte("" +
		"product = \"server\"\n" +
		"version = \"1.2.3\"\n" +
		"created_at = 2026-04-03T18:00:00Z\n" +
		"added_fragment_ids = [\"f1\"]\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write record: %v", err)
	}

	if _, err := Load(path); err == nil {
		t.Fatalf("Load should reject mismatched product/path consistency")
	}
}

func TestValidateSetRejectsCompanionWithoutBase(t *testing.T) {
	records := []ReleaseRecord{
		{
			Product:          "changes",
			Version:          "1.2.3+docs.1",
			CreatedAt:        time.Date(2026, 4, 3, 18, 0, 0, 0, time.UTC),
			CompanionPurpose: "docs",
		},
	}

	if err := ValidateSet(records); err == nil {
		t.Fatalf("ValidateSet should require a base record for companions")
	}
}

func TestLoadAcceptsFlatReleaseDirectoryLayout(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	path := filepath.Join(config.ReleasesDir(repoRoot, cfg), "cli-1.2.3.toml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir releases dir: %v", err)
	}

	body := []byte("" +
		"product = \"cli\"\n" +
		"version = \"1.2.3\"\n" +
		"created_at = 2026-04-03T18:00:00Z\n" +
		"added_fragment_ids = [\"f1\"]\n")
	if err := os.WriteFile(path, body, 0o644); err != nil {
		t.Fatalf("write record: %v", err)
	}

	record, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if record.Product != "cli" || record.Version != "1.2.3" {
		t.Fatalf("unexpected record: %#v", record)
	}
}

func fragmentIDs(items []fragments.Fragment) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, item.ID)
	}
	return out
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
