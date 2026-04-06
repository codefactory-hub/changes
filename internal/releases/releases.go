package releases

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/versioning"
)

type ReleaseSection struct {
	Key   string `toml:"key" json:"key"`
	Title string `toml:"title" json:"title"`
}

type ReleaseRecord struct {
	Product          string           `toml:"product" json:"product"`
	Version          string           `toml:"version" json:"version"`
	Bootstrap        bool             `toml:"bootstrap" json:"bootstrap,omitempty"`
	ParentVersion    string           `toml:"parent_version" json:"parent_version,omitempty"`
	CreatedAt        time.Time        `toml:"created_at" json:"created_at"`
	AddedFragmentIDs []string         `toml:"added_fragment_ids" json:"added_fragment_ids,omitempty"`
	DisplayTitle     string           `toml:"display_title" json:"display_title,omitempty"`
	Summary          string           `toml:"summary" json:"summary,omitempty"`
	Edition          string           `toml:"edition" json:"edition,omitempty"`
	SourceURL        string           `toml:"source_url" json:"source_url,omitempty"`
	CompanionPurpose string           `toml:"companion_purpose" json:"companion_purpose,omitempty"`
	Sections         []ReleaseSection `toml:"sections" json:"sections,omitempty"`
}

type releaseRecordFile struct {
	Product          string           `toml:"product"`
	Version          string           `toml:"version"`
	Bootstrap        bool             `toml:"bootstrap,omitempty"`
	ParentVersion    *string          `toml:"parent_version,omitempty"`
	CreatedAt        time.Time        `toml:"created_at"`
	AddedFragmentIDs []string         `toml:"added_fragment_ids,omitempty"`
	DisplayTitle     string           `toml:"display_title,omitempty"`
	Summary          string           `toml:"summary,omitempty"`
	Edition          string           `toml:"edition,omitempty"`
	SourceURL        string           `toml:"source_url,omitempty"`
	CompanionPurpose string           `toml:"companion_purpose,omitempty"`
	Sections         []ReleaseSection `toml:"sections,omitempty"`
}

func RecordPath(repoRoot string, cfg config.Config, product, version string) string {
	return filepath.Join(config.ReleasesDir(repoRoot, cfg), fmt.Sprintf("%s-%s.toml", product, version))
}

func Write(repoRoot string, cfg config.Config, record ReleaseRecord) (string, error) {
	if err := record.Validate(); err != nil {
		return "", err
	}

	path := RecordPath(repoRoot, cfg, record.Product, record.Version)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create releases directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create release record: %w", err)
	}
	defer file.Close()

	model := record.toFile()
	if err := toml.NewEncoder(file).Encode(model); err != nil {
		return "", fmt.Errorf("encode release record: %w", err)
	}

	return path, nil
}

func Load(path string) (ReleaseRecord, error) {
	var model releaseRecordFile
	meta, err := toml.DecodeFile(path, &model)
	if err != nil {
		return ReleaseRecord{}, fmt.Errorf("decode release record %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return ReleaseRecord{}, fmt.Errorf("decode release record %s: unsupported keys: %s", path, joinKeys(undecoded))
	}

	record := model.toReleaseRecord()
	if err := record.Validate(); err != nil {
		return ReleaseRecord{}, err
	}
	if err := validatePathConsistency(path, record); err != nil {
		return ReleaseRecord{}, err
	}
	return record, nil
}

func List(repoRoot string, cfg config.Config) ([]ReleaseRecord, error) {
	root := config.ReleasesDir(repoRoot, cfg)
	if _, err := os.Stat(root); err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read releases dir: %w", err)
	}

	var records []ReleaseRecord
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".toml") {
			return nil
		}
		record, err := Load(path)
		if err != nil {
			return err
		}
		records = append(records, record)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk releases dir: %w", err)
	}

	sortRecords(records)
	if err := ValidateSet(records); err != nil {
		return nil, err
	}
	return records, nil
}

func (r ReleaseRecord) Validate() error {
	if strings.TrimSpace(r.Product) == "" {
		return fmt.Errorf("release record product is required")
	}
	if strings.TrimSpace(r.Version) == "" {
		return fmt.Errorf("release record version is required")
	}
	version, err := versioning.Parse(r.Version)
	if err != nil {
		return err
	}
	if r.CreatedAt.IsZero() {
		return fmt.Errorf("release record created_at is required")
	}

	seenSections := make(map[string]struct{}, len(r.Sections))
	for _, section := range r.Sections {
		if strings.TrimSpace(section.Key) == "" {
			return fmt.Errorf("release record sections must not contain empty keys")
		}
		if strings.TrimSpace(section.Title) == "" {
			return fmt.Errorf("release record sections must not contain empty titles")
		}
		if _, ok := seenSections[section.Key]; ok {
			return fmt.Errorf("release record sections must be unique by key")
		}
		seenSections[section.Key] = struct{}{}
	}

	if r.IsCompanionRecord() {
		if r.Bootstrap {
			return fmt.Errorf("companion release record %s must not be marked bootstrap", r.Version)
		}
		if strings.TrimSpace(r.CompanionPurpose) == "" {
			return fmt.Errorf("companion release record %s must set companion_purpose", r.Version)
		}
		if strings.TrimSpace(r.ParentVersion) != "" {
			return fmt.Errorf("companion release record %s must not set parent_version", r.Version)
		}
		if len(r.AddedFragmentIDs) > 0 {
			return fmt.Errorf("companion release record %s must not set added_fragment_ids", r.Version)
		}
		if len(r.Sections) > 0 {
			return fmt.Errorf("companion release record %s must not define sections", r.Version)
		}
		return nil
	}

	if version.BuildMetadata != "" {
		return fmt.Errorf("base release record %s must not include build metadata", r.Version)
	}
	if len(r.AddedFragmentIDs) == 0 {
		return fmt.Errorf("release record added_fragment_ids must not be empty")
	}

	seenFragments := make(map[string]struct{}, len(r.AddedFragmentIDs))
	for _, id := range r.AddedFragmentIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("release record added_fragment_ids must not contain empty IDs")
		}
		if _, ok := seenFragments[id]; ok {
			return fmt.Errorf("release record added_fragment_ids must be unique")
		}
		seenFragments[id] = struct{}{}
	}
	return nil
}

func ValidateSet(records []ReleaseRecord) error {
	sorted := slices.Clone(records)
	sortRecords(sorted)

	groups := make(map[string][]ReleaseRecord)
	for _, record := range sorted {
		if err := record.Validate(); err != nil {
			return err
		}
		key := identityKey(record.Product, record.ReleaseIdentity())
		groups[key] = append(groups[key], record)
	}

	for key, items := range groups {
		baseCount := 0
		for _, item := range items {
			if item.IsBaseRecord() {
				baseCount++
			}
		}
		switch {
		case baseCount == 0:
			return fmt.Errorf("release identity %s is missing a base record", key)
		case baseCount > 1:
			return fmt.Errorf("release identity %s has multiple base records", key)
		}
	}

	baseByProduct := make(map[string][]ReleaseRecord)
	for _, record := range sorted {
		if !record.IsBaseRecord() {
			continue
		}
		baseByProduct[record.Product] = append(baseByProduct[record.Product], record)
	}

	for _, baseRecords := range baseByProduct {
		seen := make([]ReleaseRecord, 0, len(baseRecords))
		for _, record := range baseRecords {
			expected := expectedParentVersion(seen, record)
			switch {
			case expected == "" && strings.TrimSpace(record.ParentVersion) != "":
				return fmt.Errorf("release record %s must not set parent_version", record.Version)
			case expected != "" && record.ParentVersion != expected:
				return fmt.Errorf("release record %s parent %s must be %s", record.Version, record.ParentVersion, expected)
			}
			seen = append(seen, record)
		}
	}

	return nil
}

func Index(records []ReleaseRecord) map[string]ReleaseRecord {
	index := make(map[string]ReleaseRecord)
	for _, record := range records {
		if !record.IsBaseRecord() {
			continue
		}
		index[identityKey(record.Product, record.Version)] = record
	}
	return index
}

func LatestFinalHead(records []ReleaseRecord) *ReleaseRecord {
	return latestFinalHeadForProduct(records, "")
}

func LatestFinalHeadForProduct(records []ReleaseRecord, product string) *ReleaseRecord {
	return latestFinalHeadForProduct(records, product)
}

func latestFinalHeadForProduct(records []ReleaseRecord, product string) *ReleaseRecord {
	var best *ReleaseRecord
	for idx := range records {
		record := records[idx]
		if !record.IsBaseRecord() {
			continue
		}
		if product != "" && record.Product != product {
			continue
		}
		if record.IsPrerelease() {
			continue
		}
		if best == nil {
			copy := record
			best = &copy
			continue
		}
		current := versioning.MustParse(record.Version)
		bestVersion := versioning.MustParse(best.Version)
		if versioning.Compare(current, bestVersion) > 0 {
			copy := record
			best = &copy
		}
	}
	return best
}

func LatestStableHead(records []ReleaseRecord) *ReleaseRecord {
	return LatestFinalHead(records)
}

func PrereleaseHead(records []ReleaseRecord, product, targetVersion, label string) *ReleaseRecord {
	var best *ReleaseRecord
	for idx := range records {
		record := records[idx]
		if !record.IsBaseRecord() {
			continue
		}
		if product != "" && record.Product != product {
			continue
		}
		if !record.IsPrerelease() {
			continue
		}
		version := versioning.MustParse(record.Version)
		recordLabel, _, ok := version.PrereleaseLabelNumber()
		if !ok || version.Stable().String() != targetVersion || recordLabel != label {
			continue
		}
		if best == nil {
			copy := record
			best = &copy
			continue
		}
		if versioning.Compare(version, versioning.MustParse(best.Version)) > 0 {
			copy := record
			best = &copy
		}
	}
	return best
}

func PrereleaseHeads(records []ReleaseRecord, product string) []ReleaseRecord {
	heads := make(map[string]ReleaseRecord)
	for _, record := range records {
		if !record.IsBaseRecord() {
			continue
		}
		if product != "" && record.Product != product {
			continue
		}
		if !record.IsPrerelease() {
			continue
		}
		version := versioning.MustParse(record.Version)
		label, _, ok := version.PrereleaseLabelNumber()
		if !ok {
			continue
		}
		line := version.Stable().String() + "-" + label
		current, hasCurrent := heads[line]
		if !hasCurrent || versioning.Compare(version, versioning.MustParse(current.Version)) > 0 {
			heads[line] = record
		}
	}

	out := make([]ReleaseRecord, 0, len(heads))
	for _, record := range heads {
		out = append(out, record)
	}
	sortRecords(out)
	slices.Reverse(out)
	return out
}

func PrereleaseVersionsForLine(records []ReleaseRecord, product, targetVersion, label string) []versioning.Version {
	heads := make([]versioning.Version, 0)
	for _, record := range records {
		if !record.IsBaseRecord() {
			continue
		}
		if product != "" && record.Product != product {
			continue
		}
		version := versioning.MustParse(record.Version)
		recordLabel, _, ok := version.PrereleaseLabelNumber()
		if !ok || version.Stable().String() != targetVersion || recordLabel != label {
			continue
		}
		heads = append(heads, version)
	}
	return heads
}

func Lineage(head ReleaseRecord, records []ReleaseRecord) ([]ReleaseRecord, error) {
	if !head.IsBaseRecord() {
		return nil, fmt.Errorf("release record %s is not a base record", head.Version)
	}

	index := Index(records)
	current := head
	lineage := []ReleaseRecord{current}

	for strings.TrimSpace(current.ParentVersion) != "" {
		parent, ok := index[identityKey(current.Product, current.ParentVersion)]
		if !ok {
			return nil, fmt.Errorf("release record %s references missing parent %s", current.Version, current.ParentVersion)
		}
		lineage = append(lineage, parent)
		current = parent
	}

	return lineage, nil
}

func ReachableAddedFragmentIDs(head ReleaseRecord, records []ReleaseRecord) (map[string]struct{}, error) {
	lineage, err := Lineage(head, records)
	if err != nil {
		return nil, err
	}

	out := make(map[string]struct{})
	for _, record := range lineage {
		for _, id := range record.AddedFragmentIDs {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

func UnreleasedFinalFragments(all []fragments.Fragment, records []ReleaseRecord, product string) ([]fragments.Fragment, error) {
	var referenced map[string]struct{}
	if head := LatestFinalHeadForProduct(records, product); head != nil {
		var err error
		referenced, err = ReachableAddedFragmentIDs(*head, records)
		if err != nil {
			return nil, err
		}
	}
	if referenced == nil {
		referenced = map[string]struct{}{}
	}

	out := make([]fragments.Fragment, 0, len(all))
	for _, item := range all {
		if _, ok := referenced[item.ID]; ok {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func UnreleasedStableFragments(all []fragments.Fragment, records []ReleaseRecord, product string) ([]fragments.Fragment, error) {
	return UnreleasedFinalFragments(all, records, product)
}

func UnreleasedPrereleaseFragments(all []fragments.Fragment, records []ReleaseRecord, product, targetVersion, label string) ([]fragments.Fragment, error) {
	finalUnreleased, err := UnreleasedFinalFragments(all, records, product)
	if err != nil {
		return nil, err
	}

	var referenced map[string]struct{}
	if head := PrereleaseHead(records, product, targetVersion, label); head != nil {
		referenced, err = ReachableAddedFragmentIDs(*head, records)
		if err != nil {
			return nil, err
		}
	}
	if referenced == nil {
		referenced = map[string]struct{}{}
	}

	out := make([]fragments.Fragment, 0, len(finalUnreleased))
	for _, item := range finalUnreleased {
		if _, ok := referenced[item.ID]; ok {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func UnreleasedPreviewFragments(all []fragments.Fragment, records []ReleaseRecord, product, targetVersion, label string) ([]fragments.Fragment, error) {
	return UnreleasedPrereleaseFragments(all, records, product, targetVersion, label)
}

func FindBaseRecord(records []ReleaseRecord, product, version string) (*ReleaseRecord, error) {
	parsed, err := versioning.Parse(version)
	if err != nil {
		return nil, err
	}
	identityVersion := parsed.WithoutBuildMetadata().String()
	for idx := range records {
		record := records[idx]
		if !record.IsBaseRecord() {
			continue
		}
		if record.Product != product {
			continue
		}
		if record.ReleaseIdentity() != identityVersion {
			continue
		}
		copy := record
		return &copy, nil
	}
	return nil, fmt.Errorf("release record %s/%s not found", product, identityVersion)
}

func (r ReleaseRecord) toFile() releaseRecordFile {
	file := releaseRecordFile{
		Product:          r.Product,
		Version:          r.Version,
		Bootstrap:        r.Bootstrap,
		CreatedAt:        r.CreatedAt,
		AddedFragmentIDs: slices.Clone(r.AddedFragmentIDs),
		DisplayTitle:     r.DisplayTitle,
		Summary:          r.Summary,
		Edition:          r.Edition,
		SourceURL:        r.SourceURL,
		CompanionPurpose: r.CompanionPurpose,
		Sections:         slices.Clone(r.Sections),
	}
	if strings.TrimSpace(r.ParentVersion) != "" {
		parent := r.ParentVersion
		file.ParentVersion = &parent
	}
	return file
}

func (r releaseRecordFile) toReleaseRecord() ReleaseRecord {
	record := ReleaseRecord{
		Product:          r.Product,
		Version:          r.Version,
		Bootstrap:        r.Bootstrap,
		CreatedAt:        r.CreatedAt,
		AddedFragmentIDs: slices.Clone(r.AddedFragmentIDs),
		DisplayTitle:     r.DisplayTitle,
		Summary:          r.Summary,
		Edition:          r.Edition,
		SourceURL:        r.SourceURL,
		CompanionPurpose: r.CompanionPurpose,
		Sections:         slices.Clone(r.Sections),
	}
	if r.ParentVersion != nil {
		record.ParentVersion = *r.ParentVersion
	}
	return record
}

func (r ReleaseRecord) ParsedVersion() versioning.Version {
	return versioning.MustParse(r.Version)
}

func (r ReleaseRecord) ReleaseIdentity() string {
	return r.ParsedVersion().WithoutBuildMetadata().String()
}

func (r ReleaseRecord) IsPrerelease() bool {
	return r.ParsedVersion().IsPrerelease()
}

func (r ReleaseRecord) IsBaseRecord() bool {
	return r.ParsedVersion().BuildMetadata == ""
}

func (r ReleaseRecord) IsCompanionRecord() bool {
	return !r.IsBaseRecord()
}

func (r ReleaseRecord) TargetVersion() string {
	return r.ParsedVersion().Stable().String()
}

func (r ReleaseRecord) EffectiveDisplayTitle() string {
	if strings.TrimSpace(r.DisplayTitle) != "" {
		return r.DisplayTitle
	}
	return r.Version
}

func expectedParentVersion(previous []ReleaseRecord, record ReleaseRecord) string {
	version := record.ParsedVersion()
	if version.IsPrerelease() {
		label, _, ok := version.PrereleaseLabelNumber()
		if !ok {
			return ""
		}
		if head := PrereleaseHead(previous, record.Product, version.Stable().String(), label); head != nil {
			return head.Version
		}
		if head := LatestFinalHeadForProduct(previous, record.Product); head != nil {
			return head.Version
		}
		return ""
	}

	if head := LatestFinalHeadForProduct(previous, record.Product); head != nil {
		return head.Version
	}
	return ""
}

func sortRecords(records []ReleaseRecord) {
	slices.SortFunc(records, func(a, b ReleaseRecord) int {
		if cmp := strings.Compare(a.Product, b.Product); cmp != 0 {
			return cmp
		}
		av := versioning.MustParse(a.Version)
		bv := versioning.MustParse(b.Version)
		if cmp := versioning.Compare(av, bv); cmp != 0 {
			return cmp
		}
		if cmp := strings.Compare(a.Version, b.Version); cmp != 0 {
			return cmp
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return strings.Compare(a.SourceURL, b.SourceURL)
	})
}

func validatePathConsistency(path string, record ReleaseRecord) error {
	expectedName := fmt.Sprintf("%s-%s.toml", record.Product, record.Version)
	if filepath.Base(path) != expectedName {
		return fmt.Errorf("release record %s filename must be %s", path, expectedName)
	}
	return nil
}

func identityKey(product, version string) string {
	return product + "\x00" + version
}

func joinKeys(keys []toml.Key) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key.String())
	}
	return strings.Join(parts, ", ")
}
