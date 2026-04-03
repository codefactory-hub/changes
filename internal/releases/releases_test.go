package releases

import (
	"os"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
)

func TestPreviewAndStableSelectionUsesLineage(t *testing.T) {
	all := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC), Title: "One", Bump: "patch"}, Body: "body one"},
		{Metadata: fragments.Metadata{ID: "f2", CreatedAt: time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC), Title: "Two", Bump: "minor"}, Body: "body two"},
		{Metadata: fragments.Metadata{ID: "f3", CreatedAt: time.Date(2026, 4, 2, 17, 0, 0, 0, time.UTC), Title: "Three", Bump: "patch"}, Body: "body three"},
	}

	manifests := []Manifest{
		{
			Version:          "1.1.0",
			TargetVersion:    "1.1.0",
			Channel:          ChannelStable,
			CreatedAt:        time.Date(2026, 4, 1, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f1"},
		},
		{
			Version:          "1.2.0-rc.1",
			TargetVersion:    "1.2.0",
			Channel:          ChannelPreview,
			CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f2"},
		},
		{
			Version:          "1.2.0-rc.2",
			TargetVersion:    "1.2.0",
			Channel:          ChannelPreview,
			ParentVersion:    "1.2.0-rc.1",
			CreatedAt:        time.Date(2026, 4, 2, 19, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f3"},
		},
	}
	if err := ValidateSet(manifests); err != nil {
		t.Fatalf("ValidateSet returned error: %v", err)
	}

	stableUnreleased, err := UnreleasedStableFragments(all, manifests)
	if err != nil {
		t.Fatalf("UnreleasedStableFragments returned error: %v", err)
	}
	if got := fragmentIDs(stableUnreleased); !equalIDs(got, []string{"f2", "f3"}) {
		t.Fatalf("stable unreleased = %#v, want f2,f3", got)
	}

	previewUnreleased, err := UnreleasedPreviewFragments(all, manifests, "1.2.0", "rc")
	if err != nil {
		t.Fatalf("UnreleasedPreviewFragments returned error: %v", err)
	}
	if len(previewUnreleased) != 0 {
		t.Fatalf("preview unreleased = %#v, want none", fragmentIDs(previewUnreleased))
	}

	betaUnreleased, err := UnreleasedPreviewFragments(all, manifests, "1.2.0", "beta")
	if err != nil {
		t.Fatalf("UnreleasedPreviewFragments returned error: %v", err)
	}
	if got := fragmentIDs(betaUnreleased); !equalIDs(got, []string{"f2", "f3"}) {
		t.Fatalf("beta unreleased = %#v, want f2,f3", got)
	}
}

func TestValidateSetRejectsInvalidParentReference(t *testing.T) {
	manifests := []Manifest{
		{
			Version:          "1.2.0-rc.2",
			TargetVersion:    "1.2.0",
			Channel:          ChannelPreview,
			ParentVersion:    "1.2.0-rc.1",
			CreatedAt:        time.Date(2026, 4, 2, 19, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f2"},
		},
	}

	if err := ValidateSet(manifests); err == nil {
		t.Fatalf("ValidateSet should reject a missing parent reference")
	}
}

func TestListValidatesParentChannelAndTarget(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	if err := os.MkdirAll(config.ReleasesDir(repoRoot, cfg), 0o755); err != nil {
		t.Fatalf("mkdir releases dir: %v", err)
	}

	parent := Manifest{
		Version:          "1.2.0-beta.1",
		TargetVersion:    "1.2.0",
		Channel:          ChannelPreview,
		CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f1"},
	}
	child := Manifest{
		Version:          "1.2.0-rc.1",
		TargetVersion:    "1.2.0",
		Channel:          ChannelPreview,
		ParentVersion:    "1.2.0-beta.1",
		CreatedAt:        time.Date(2026, 4, 2, 19, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f2"},
	}
	if _, err := Write(repoRoot, cfg, parent); err != nil {
		t.Fatalf("Write parent returned error: %v", err)
	}
	if _, err := Write(repoRoot, cfg, child); err != nil {
		t.Fatalf("Write child returned error: %v", err)
	}

	if _, err := List(repoRoot, cfg); err == nil {
		t.Fatalf("List should reject cross-label preview parents")
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
