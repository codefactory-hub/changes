package releases

import (
	"testing"
	"time"

	"github.com/example/changes/internal/fragments"
)

func TestPreviewAndStableConsumption(t *testing.T) {
	all := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC), Title: "One", Bump: "patch"}, Body: "body one"},
		{Metadata: fragments.Metadata{ID: "f2", CreatedAt: time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC), Title: "Two", Bump: "minor"}, Body: "body two"},
	}
	preview := Manifest{
		Version:       "1.2.0-rc.1",
		TargetVersion: "1.2.0",
		Channel:       ChannelPreview,
		Consumes:      false,
		CreatedAt:     time.Date(2026, 4, 2, 17, 0, 0, 0, time.UTC),
		MaxChars:      4000,
		Template:      "release.md.tmpl",
		FragmentIDs:   []string{"f1"},
	}

	if got := UnreleasedStableFragments(all, []Manifest{preview}); len(got) != 2 {
		t.Fatalf("stable unreleased = %d, want 2", len(got))
	}

	if got := UnreleasedPreviewFragments(all, []Manifest{preview}, "1.2.0", "rc"); len(got) != 1 || got[0].ID != "f2" {
		t.Fatalf("rc line unreleased = %#v, want only f2", got)
	}

	if got := UnreleasedPreviewFragments(all, []Manifest{preview}, "1.2.0", "beta"); len(got) != 2 {
		t.Fatalf("beta line unreleased = %d, want 2", len(got))
	}

	stable := Manifest{
		Version:       "1.2.0",
		TargetVersion: "1.2.0",
		Channel:       ChannelStable,
		Consumes:      true,
		CreatedAt:     time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		MaxChars:      4000,
		Template:      "release.md.tmpl",
		FragmentIDs:   []string{"f1", "f2"},
	}

	if got := UnreleasedStableFragments(all, []Manifest{preview, stable}); len(got) != 0 {
		t.Fatalf("stable unreleased after stable release = %d, want 0", len(got))
	}
}
