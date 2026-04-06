package releases

import (
	"testing"
	"time"

	"github.com/example/changes/internal/fragments"
)

func TestAssembleReleaseIncludesCompanionsSectionsAndMustIncludeFragments(t *testing.T) {
	records := []ReleaseRecord{
		{
			Product:          "changes",
			Version:          "1.2.3",
			CreatedAt:        time.Date(2026, 4, 3, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f1", "f2"},
			Sections: []ReleaseSection{
				{Key: "highlights", Title: "Highlights"},
				{Key: "fixes", Title: "Fixes"},
			},
		},
		{
			Product:          "changes",
			Version:          "1.2.3+docs.1",
			CreatedAt:        time.Date(2026, 4, 3, 18, 5, 0, 0, time.UTC),
			CompanionPurpose: "docs",
			SourceURL:        "https://example.invalid/docs",
		},
	}
	allFragments := []fragments.Fragment{
		{
			Metadata: fragments.Metadata{
				ID:                   "f1",
				CreatedAt:            time.Date(2026, 4, 3, 17, 0, 0, 0, time.UTC),
				Title:                "Fast path",
				Type:                 "added",
				SectionKey:           "highlights",
				CustomerVisible:      true,
				ReleaseNotesPriority: 2,
				DisplayOrder:         1,
			},
			Body: "Improves the common path.",
		},
		{
			Metadata: fragments.Metadata{
				ID:             "f2",
				CreatedAt:      time.Date(2026, 4, 3, 17, 5, 0, 0, time.UTC),
				Title:          "Fix retries",
				Type:           "fixed",
				SectionKey:     "fixes",
				RequiresAction: true,
			},
			Body: "Operators should recycle the worker.",
		},
	}

	bundle, err := AssembleRelease(records[0], records, allFragments)
	if err != nil {
		t.Fatalf("AssembleRelease returned error: %v", err)
	}

	if got := len(bundle.Companions); got != 1 {
		t.Fatalf("bundle companions = %d, want 1", got)
	}
	if bundle.Companions[0].Version != "1.2.3+docs.1" {
		t.Fatalf("unexpected companion version: %s", bundle.Companions[0].Version)
	}
	if got := len(bundle.Sections); got != 2 {
		t.Fatalf("bundle sections = %d, want 2", got)
	}
	if bundle.Sections[0].Key != "highlights" || bundle.Sections[1].Key != "fixes" {
		t.Fatalf("unexpected section order: %#v", bundle.Sections)
	}
	if got := bundle.MustIncludeFragmentIDs; len(got) != 2 || got[0] != "f1" || got[1] != "f2" {
		t.Fatalf("must include fragment ids = %#v, want [f1 f2]", got)
	}
}
