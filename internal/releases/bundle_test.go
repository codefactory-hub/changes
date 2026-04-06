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

func TestAssembleReleaseLineageResolvesFromCompanionAndBuildsAncestors(t *testing.T) {
	records := []ReleaseRecord{
		{
			Product:          "changes",
			Version:          "1.1.0",
			CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f1"},
		},
		{
			Product:          "changes",
			Version:          "1.2.0",
			ParentVersion:    "1.1.0",
			CreatedAt:        time.Date(2026, 4, 3, 18, 0, 0, 0, time.UTC),
			AddedFragmentIDs: []string{"f2"},
			Sections:         []ReleaseSection{{Key: "ops_updates", Title: "Ops Updates"}},
		},
		{
			Product:          "changes",
			Version:          "1.2.0+docs.1",
			CreatedAt:        time.Date(2026, 4, 3, 18, 5, 0, 0, time.UTC),
			CompanionPurpose: "docs",
		},
	}
	allFragments := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 17, 0, 0, 0, time.UTC), Type: "added"}, Body: "older"},
		{Metadata: fragments.Metadata{ID: "f2", CreatedAt: time.Date(2026, 4, 3, 17, 0, 0, 0, time.UTC), Type: "changed", SectionKey: "ops_updates"}, Body: "newer"},
	}

	bundles, err := AssembleReleaseLineage(records[2], records, allFragments)
	if err != nil {
		t.Fatalf("AssembleReleaseLineage returned error: %v", err)
	}
	if len(bundles) != 2 {
		t.Fatalf("bundles = %d, want 2", len(bundles))
	}
	if bundles[0].Release.Version != "1.2.0" || bundles[1].Release.Version != "1.1.0" {
		t.Fatalf("unexpected bundle order: %#v", bundles)
	}
	if len(bundles[0].Ancestors) != 1 || bundles[0].Ancestors[0].Version != "1.1.0" {
		t.Fatalf("ancestors = %#v, want [1.1.0]", bundles[0].Ancestors)
	}
	if len(bundles[0].Sections) != 1 || bundles[0].Sections[0].Title != "Ops Updates" {
		t.Fatalf("sections = %#v", bundles[0].Sections)
	}
}

func TestBundleHelpersClassifySortAndHumanize(t *testing.T) {
	selected := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "later", CreatedAt: time.Date(2026, 4, 3, 10, 0, 0, 0, time.UTC), Type: "fixed"}, Body: "later"},
		{Metadata: fragments.Metadata{ID: "ordered", DisplayOrder: 1, CreatedAt: time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC), Type: "security"}, Body: "ordered"},
		{Metadata: fragments.Metadata{ID: "breaking", CreatedAt: time.Date(2026, 4, 2, 10, 0, 0, 0, time.UTC), Breaking: true, RequiresAction: true}, Body: "breaking"},
		{Metadata: fragments.Metadata{ID: "custom", CreatedAt: time.Date(2026, 4, 1, 10, 0, 0, 0, time.UTC), Type: "unknown", SectionKey: "ops_updates", CustomerVisible: true, ReleaseNotesPriority: 1}, Body: "custom"},
	}

	sortBundleFragments(selected)
	if got := []string{selected[0].ID, selected[1].ID, selected[2].ID, selected[3].ID}; got[0] != "ordered" || got[1] != "custom" {
		t.Fatalf("sorted ids = %#v, want ordered then custom first", got)
	}

	base := ReleaseRecord{Sections: []ReleaseSection{{Key: "ops_updates", Title: "Ops Updates"}}}
	sections := bundleSections(base, selected)
	if len(sections) < 3 {
		t.Fatalf("sections = %#v, want at least custom/default sections", sections)
	}
	if sections[0].Key != "ops_updates" || sections[0].Title != "Ops Updates" {
		t.Fatalf("first section = %#v, want custom section", sections[0])
	}

	if key, title := classifyFragment(fragments.Fragment{Metadata: fragments.Metadata{Type: "security"}}); key != "security" || title != "Security" {
		t.Fatalf("classifyFragment(security) = (%q, %q)", key, title)
	}
	if got := mustIncludeFragmentIDs(selected); len(got) != 3 {
		t.Fatalf("mustIncludeFragmentIDs = %#v, want 3 items", got)
	}
	if got := humanizeSectionKey("ops_updates"); got != "Ops Updates" {
		t.Fatalf("humanizeSectionKey = %q, want Ops Updates", got)
	}
}
