package releases

import (
	"fmt"
	"slices"
	"strings"

	"github.com/example/changes/internal/fragments"
)

type ReleaseBundle struct {
	Release                ReleaseRecord   `json:"release"`
	Companions             []ReleaseRecord `json:"companions,omitempty"`
	Ancestors              []ReleaseRecord `json:"ancestors,omitempty"`
	Sections               []BundleSection `json:"sections,omitempty"`
	MustIncludeFragmentIDs []string        `json:"must_include_fragment_ids,omitempty"`
}

type BundleSection struct {
	Key     string        `json:"key"`
	Title   string        `json:"title"`
	Entries []BundleEntry `json:"entries"`
}

type BundleEntry struct {
	Fragment fragments.Fragment `json:"fragment"`
}

var defaultSectionOrder = []ReleaseSection{
	{Key: "breaking", Title: "Breaking Changes"},
	{Key: "added", Title: "Added"},
	{Key: "changed", Title: "Changed"},
	{Key: "fixed", Title: "Fixed"},
	{Key: "removed", Title: "Removed"},
	{Key: "security", Title: "Security"},
	{Key: "other", Title: "Other"},
}

func AssembleRelease(head ReleaseRecord, records []ReleaseRecord, allFragments []fragments.Fragment) (ReleaseBundle, error) {
	base, err := resolveBaseRecord(head, records)
	if err != nil {
		return ReleaseBundle{}, err
	}

	fragmentIndex := make(map[string]fragments.Fragment, len(allFragments))
	for _, item := range allFragments {
		fragmentIndex[item.ID] = item
	}

	selected := make([]fragments.Fragment, 0, len(base.AddedFragmentIDs))
	for _, id := range base.AddedFragmentIDs {
		item, ok := fragmentIndex[id]
		if !ok {
			return ReleaseBundle{}, fmt.Errorf("release record %s references missing fragment %s", base.Version, id)
		}
		selected = append(selected, item)
	}
	sortBundleFragments(selected)

	lineage, err := Lineage(base, records)
	if err != nil {
		return ReleaseBundle{}, err
	}
	companions := companionsForBase(base, records)

	return ReleaseBundle{
		Release:                base,
		Companions:             companions,
		Ancestors:              append([]ReleaseRecord(nil), lineage[1:]...),
		Sections:               bundleSections(base, selected),
		MustIncludeFragmentIDs: mustIncludeFragmentIDs(selected),
	}, nil
}

func AssembleReleaseLineage(head ReleaseRecord, records []ReleaseRecord, allFragments []fragments.Fragment) ([]ReleaseBundle, error) {
	base, err := resolveBaseRecord(head, records)
	if err != nil {
		return nil, err
	}

	lineage, err := Lineage(base, records)
	if err != nil {
		return nil, err
	}

	out := make([]ReleaseBundle, 0, len(lineage))
	for _, record := range lineage {
		bundle, err := AssembleRelease(record, records, allFragments)
		if err != nil {
			return nil, err
		}
		out = append(out, bundle)
	}
	return out, nil
}

func resolveBaseRecord(record ReleaseRecord, records []ReleaseRecord) (ReleaseRecord, error) {
	if record.IsBaseRecord() {
		return record, nil
	}
	base, err := FindBaseRecord(records, record.Product, record.Version)
	if err != nil {
		return ReleaseRecord{}, err
	}
	return *base, nil
}

func companionsForBase(base ReleaseRecord, records []ReleaseRecord) []ReleaseRecord {
	out := make([]ReleaseRecord, 0)
	for _, record := range records {
		if record.Product != base.Product {
			continue
		}
		if record.ReleaseIdentity() != base.ReleaseIdentity() {
			continue
		}
		if record.IsBaseRecord() {
			continue
		}
		out = append(out, record)
	}
	slices.SortFunc(out, func(a, b ReleaseRecord) int {
		return strings.Compare(a.Version, b.Version)
	})
	return out
}

func bundleSections(base ReleaseRecord, selected []fragments.Fragment) []BundleSection {
	buckets := make(map[string][]BundleEntry)
	titles := make(map[string]string)
	defined := make([]string, 0, len(base.Sections))
	suppressBreaking := strings.TrimSpace(base.ParentVersion) == "" && !base.Bootstrap
	for _, section := range base.Sections {
		defined = append(defined, section.Key)
		titles[section.Key] = section.Title
	}

	for _, item := range selected {
		key, title := classifyFragment(item, suppressBreaking)
		if strings.TrimSpace(item.SectionKey) != "" {
			key = item.SectionKey
		}
		if definedTitle, ok := titles[key]; ok {
			title = definedTitle
		}
		titles[key] = title
		buckets[key] = append(buckets[key], BundleEntry{Fragment: item})
	}

	out := make([]BundleSection, 0, len(buckets))
	seen := make(map[string]struct{}, len(buckets))
	for _, key := range defined {
		entries := buckets[key]
		if len(entries) == 0 {
			continue
		}
		out = append(out, BundleSection{Key: key, Title: titles[key], Entries: entries})
		seen[key] = struct{}{}
	}

	for _, section := range defaultSectionOrder {
		if _, ok := seen[section.Key]; ok {
			continue
		}
		entries := buckets[section.Key]
		if len(entries) == 0 {
			continue
		}
		title := titles[section.Key]
		if strings.TrimSpace(title) == "" {
			title = section.Title
		}
		out = append(out, BundleSection{Key: section.Key, Title: title, Entries: entries})
		seen[section.Key] = struct{}{}
	}

	var leftovers []string
	for key, entries := range buckets {
		if len(entries) == 0 {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		leftovers = append(leftovers, key)
	}
	slices.Sort(leftovers)
	for _, key := range leftovers {
		title := titles[key]
		if strings.TrimSpace(title) == "" {
			title = humanizeSectionKey(key)
		}
		out = append(out, BundleSection{Key: key, Title: title, Entries: buckets[key]})
	}

	return out
}

func classifyFragment(item fragments.Fragment, suppressBreaking bool) (string, string) {
	if item.Breaking && !suppressBreaking {
		return "breaking", "Breaking Changes"
	}

	switch strings.ToLower(strings.TrimSpace(item.Type)) {
	case "added":
		return "added", "Added"
	case "changed":
		return "changed", "Changed"
	case "fixed":
		return "fixed", "Fixed"
	case "removed":
		return "removed", "Removed"
	case "security":
		return "security", "Security"
	default:
		return "other", "Other"
	}
}

func mustIncludeFragmentIDs(selected []fragments.Fragment) []string {
	var out []string
	seen := make(map[string]struct{})
	for _, item := range selected {
		if !item.Breaking &&
			!item.RequiresAction &&
			strings.ToLower(strings.TrimSpace(item.Type)) != "security" &&
			!(item.CustomerVisible && item.ReleaseNotesPriority > 0) {
			continue
		}
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		out = append(out, item.ID)
	}
	return out
}

func sortBundleFragments(items []fragments.Fragment) {
	slices.SortFunc(items, func(a, b fragments.Fragment) int {
		switch {
		case a.DisplayOrder > 0 && b.DisplayOrder > 0 && a.DisplayOrder != b.DisplayOrder:
			if a.DisplayOrder < b.DisplayOrder {
				return -1
			}
			return 1
		case a.DisplayOrder > 0 && b.DisplayOrder == 0:
			return -1
		case a.DisplayOrder == 0 && b.DisplayOrder > 0:
			return 1
		}

		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})
}

func humanizeSectionKey(key string) string {
	value := strings.TrimSpace(strings.ReplaceAll(key, "_", " "))
	if value == "" {
		return "Other"
	}
	parts := strings.Fields(value)
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, " ")
}
