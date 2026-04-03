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

const (
	ChannelPreview = "preview"
	ChannelStable  = "stable"
)

type Manifest struct {
	Version          string    `toml:"version"`
	TargetVersion    string    `toml:"target_version"`
	Channel          string    `toml:"channel"`
	ParentVersion    string    `toml:"parent_version"`
	CreatedAt        time.Time `toml:"created_at"`
	AddedFragmentIDs []string  `toml:"added_fragment_ids"`
}

type manifestFile struct {
	Version          string    `toml:"version"`
	TargetVersion    string    `toml:"target_version"`
	Channel          string    `toml:"channel"`
	ParentVersion    *string   `toml:"parent_version,omitempty"`
	CreatedAt        time.Time `toml:"created_at"`
	AddedFragmentIDs []string  `toml:"added_fragment_ids"`
}

func ManifestPath(repoRoot string, cfg config.Config, version string) string {
	return filepath.Join(config.ReleasesDir(repoRoot, cfg), version+".toml")
}

func Write(repoRoot string, cfg config.Config, manifest Manifest) (string, error) {
	if err := manifest.Validate(); err != nil {
		return "", err
	}

	path := ManifestPath(repoRoot, cfg, manifest.Version)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create releases directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("create release manifest: %w", err)
	}
	defer file.Close()

	model := manifest.toFile()
	if err := toml.NewEncoder(file).Encode(model); err != nil {
		return "", fmt.Errorf("encode release manifest: %w", err)
	}

	return path, nil
}

func Load(path string) (Manifest, error) {
	var model manifestFile
	meta, err := toml.DecodeFile(path, &model)
	if err != nil {
		return Manifest{}, fmt.Errorf("decode release manifest %s: %w", path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return Manifest{}, fmt.Errorf("decode release manifest %s: unsupported keys: %s", path, joinKeys(undecoded))
	}

	manifest := model.toManifest()
	if err := manifest.Validate(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func List(repoRoot string, cfg config.Config) ([]Manifest, error) {
	dir := config.ReleasesDir(repoRoot, cfg)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read releases dir: %w", err)
	}

	out := make([]Manifest, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".toml") {
			continue
		}
		item, err := Load(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		out = append(out, item)
	}

	slices.SortFunc(out, func(a, b Manifest) int {
		av := versioning.MustParse(a.Version)
		bv := versioning.MustParse(b.Version)
		if cmp := versioning.Compare(av, bv); cmp != 0 {
			return cmp
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return -1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return 1
		}
		return strings.Compare(a.Version, b.Version)
	})

	if err := ValidateSet(out); err != nil {
		return nil, err
	}

	return out, nil
}

func (m Manifest) Validate() error {
	if strings.TrimSpace(m.Version) == "" {
		return fmt.Errorf("manifest version is required")
	}
	version, err := versioning.Parse(m.Version)
	if err != nil {
		return err
	}

	if strings.TrimSpace(m.TargetVersion) == "" {
		return fmt.Errorf("manifest target_version is required")
	}
	target, err := versioning.Parse(m.TargetVersion)
	if err != nil {
		return err
	}
	if target.IsPrerelease() {
		return fmt.Errorf("target_version must be stable")
	}
	if version.Stable().String() != target.String() {
		return fmt.Errorf("manifest version %s must target %s", m.Version, m.TargetVersion)
	}

	switch m.Channel {
	case ChannelPreview:
		if !version.IsPrerelease() {
			return fmt.Errorf("preview manifest version must be a prerelease")
		}
	case ChannelStable:
		if version.IsPrerelease() {
			return fmt.Errorf("stable manifest version must be stable")
		}
	default:
		return fmt.Errorf("unsupported manifest channel %q", m.Channel)
	}

	if m.CreatedAt.IsZero() {
		return fmt.Errorf("manifest created_at is required")
	}
	if len(m.AddedFragmentIDs) == 0 {
		return fmt.Errorf("manifest added_fragment_ids must not be empty")
	}

	seen := make(map[string]struct{}, len(m.AddedFragmentIDs))
	for _, id := range m.AddedFragmentIDs {
		if strings.TrimSpace(id) == "" {
			return fmt.Errorf("manifest added_fragment_ids must not contain empty IDs")
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("manifest added_fragment_ids must be unique")
		}
		seen[id] = struct{}{}
	}

	return nil
}

func ValidateSet(manifests []Manifest) error {
	index := Index(manifests)
	for _, manifest := range manifests {
		if strings.TrimSpace(manifest.ParentVersion) == "" {
			continue
		}
		parent, ok := index[manifest.ParentVersion]
		if !ok {
			return fmt.Errorf("manifest %s references missing parent %s", manifest.Version, manifest.ParentVersion)
		}
		if err := validateParent(manifest, parent); err != nil {
			return err
		}
	}

	for _, manifest := range manifests {
		if err := detectCycle(manifest, index); err != nil {
			return err
		}
	}

	return nil
}

func Index(manifests []Manifest) map[string]Manifest {
	index := make(map[string]Manifest, len(manifests))
	for _, manifest := range manifests {
		index[manifest.Version] = manifest
	}
	return index
}

func LatestStableHead(manifests []Manifest) *Manifest {
	var best *Manifest
	for idx := range manifests {
		item := manifests[idx]
		if item.Channel != ChannelStable {
			continue
		}
		if best == nil {
			copy := item
			best = &copy
			continue
		}
		current := versioning.MustParse(item.Version)
		bestVersion := versioning.MustParse(best.Version)
		if versioning.Compare(current, bestVersion) > 0 {
			copy := item
			best = &copy
		}
	}
	return best
}

func PreviewHead(manifests []Manifest, targetVersion, label string) *Manifest {
	var best *Manifest
	line := targetVersion + "-" + label
	for idx := range manifests {
		item := manifests[idx]
		if item.Channel != ChannelPreview {
			continue
		}
		version := versioning.MustParse(item.Version)
		if version.ReleaseLine() != line {
			continue
		}
		if best == nil {
			copy := item
			best = &copy
			continue
		}
		current := version
		bestVersion := versioning.MustParse(best.Version)
		if versioning.Compare(current, bestVersion) > 0 {
			copy := item
			best = &copy
		}
	}
	return best
}

func PreviewHeads(manifests []Manifest) []Manifest {
	heads := make(map[string]Manifest)
	for _, manifest := range manifests {
		if manifest.Channel != ChannelPreview {
			continue
		}
		version := versioning.MustParse(manifest.Version)
		line := version.ReleaseLine()
		current, ok := heads[line]
		if !ok || versioning.Compare(version, versioning.MustParse(current.Version)) > 0 {
			heads[line] = manifest
		}
	}

	out := make([]Manifest, 0, len(heads))
	for _, manifest := range heads {
		out = append(out, manifest)
	}
	slices.SortFunc(out, func(a, b Manifest) int {
		return -versioning.Compare(versioning.MustParse(a.Version), versioning.MustParse(b.Version))
	})
	return out
}

func PreviewVersionsForLine(manifests []Manifest, targetVersion, label string) []versioning.Version {
	heads := make([]versioning.Version, 0)
	for _, manifest := range manifests {
		version := versioning.MustParse(manifest.Version)
		if manifest.Channel != ChannelPreview || version.ReleaseLine() != targetVersion+"-"+label {
			continue
		}
		heads = append(heads, version)
	}
	return heads
}

func Lineage(head Manifest, manifests []Manifest) ([]Manifest, error) {
	index := Index(manifests)
	current := head
	lineage := []Manifest{current}

	for strings.TrimSpace(current.ParentVersion) != "" {
		parent, ok := index[current.ParentVersion]
		if !ok {
			return nil, fmt.Errorf("manifest %s references missing parent %s", current.Version, current.ParentVersion)
		}
		lineage = append(lineage, parent)
		current = parent
	}

	return lineage, nil
}

func ReachableAddedFragmentIDs(head Manifest, manifests []Manifest) (map[string]struct{}, error) {
	lineage, err := Lineage(head, manifests)
	if err != nil {
		return nil, err
	}

	out := make(map[string]struct{})
	for _, manifest := range lineage {
		for _, id := range manifest.AddedFragmentIDs {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

func UnreleasedStableFragments(all []fragments.Fragment, manifests []Manifest) ([]fragments.Fragment, error) {
	var referenced map[string]struct{}
	if head := LatestStableHead(manifests); head != nil {
		var err error
		referenced, err = ReachableAddedFragmentIDs(*head, manifests)
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

func UnreleasedPreviewFragments(all []fragments.Fragment, manifests []Manifest, targetVersion, label string) ([]fragments.Fragment, error) {
	stableUnreleased, err := UnreleasedStableFragments(all, manifests)
	if err != nil {
		return nil, err
	}

	var referenced map[string]struct{}
	if head := PreviewHead(manifests, targetVersion, label); head != nil {
		referenced, err = ReachableAddedFragmentIDs(*head, manifests)
		if err != nil {
			return nil, err
		}
	}
	if referenced == nil {
		referenced = map[string]struct{}{}
	}

	out := make([]fragments.Fragment, 0, len(stableUnreleased))
	for _, item := range stableUnreleased {
		if _, ok := referenced[item.ID]; ok {
			continue
		}
		out = append(out, item)
	}
	return out, nil
}

func (m Manifest) toFile() manifestFile {
	model := manifestFile{
		Version:          m.Version,
		TargetVersion:    m.TargetVersion,
		Channel:          m.Channel,
		CreatedAt:        m.CreatedAt,
		AddedFragmentIDs: slices.Clone(m.AddedFragmentIDs),
	}
	if strings.TrimSpace(m.ParentVersion) != "" {
		parent := m.ParentVersion
		model.ParentVersion = &parent
	}
	return model
}

func (m manifestFile) toManifest() Manifest {
	manifest := Manifest{
		Version:          m.Version,
		TargetVersion:    m.TargetVersion,
		Channel:          m.Channel,
		CreatedAt:        m.CreatedAt,
		AddedFragmentIDs: slices.Clone(m.AddedFragmentIDs),
	}
	if m.ParentVersion != nil {
		manifest.ParentVersion = *m.ParentVersion
	}
	return manifest
}

func validateParent(child, parent Manifest) error {
	if child.Channel != parent.Channel {
		return fmt.Errorf("manifest %s parent %s must have the same channel", child.Version, parent.Version)
	}
	if versioning.Compare(versioning.MustParse(child.Version), versioning.MustParse(parent.Version)) <= 0 {
		return fmt.Errorf("manifest %s parent %s must be older than the child", child.Version, parent.Version)
	}

	if child.Channel == ChannelPreview {
		childVersion := versioning.MustParse(child.Version)
		parentVersion := versioning.MustParse(parent.Version)
		if child.TargetVersion != parent.TargetVersion {
			return fmt.Errorf("preview manifest %s parent %s must have the same target_version", child.Version, parent.Version)
		}
		if childVersion.PreLabel != parentVersion.PreLabel {
			return fmt.Errorf("preview manifest %s parent %s must use the same prerelease label", child.Version, parent.Version)
		}
	}

	return nil
}

func detectCycle(manifest Manifest, index map[string]Manifest) error {
	seen := map[string]struct{}{manifest.Version: {}}
	current := manifest
	for strings.TrimSpace(current.ParentVersion) != "" {
		parent, ok := index[current.ParentVersion]
		if !ok {
			return fmt.Errorf("manifest %s references missing parent %s", current.Version, current.ParentVersion)
		}
		if _, exists := seen[parent.Version]; exists {
			return fmt.Errorf("manifest lineage cycle detected at %s", parent.Version)
		}
		seen[parent.Version] = struct{}{}
		current = parent
	}
	return nil
}

func joinKeys(keys []toml.Key) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key.String())
	}
	return strings.Join(parts, ", ")
}
