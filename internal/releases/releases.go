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
	Version       string    `toml:"version"`
	TargetVersion string    `toml:"target_version"`
	Channel       string    `toml:"channel"`
	Consumes      bool      `toml:"consumes"`
	CreatedAt     time.Time `toml:"created_at"`
	MaxChars      int       `toml:"max_chars"`
	Template      string    `toml:"template"`
	FragmentIDs   []string  `toml:"fragment_ids"`
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

	if err := toml.NewEncoder(file).Encode(manifest); err != nil {
		return "", fmt.Errorf("encode release manifest: %w", err)
	}

	return path, nil
}

func Load(path string) (Manifest, error) {
	var manifest Manifest
	if _, err := toml.DecodeFile(path, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("decode release manifest %s: %w", path, err)
	}
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
	if m.MaxChars <= 0 {
		return fmt.Errorf("manifest max_chars must be positive")
	}
	if strings.TrimSpace(m.Template) == "" {
		return fmt.Errorf("manifest template is required")
	}
	return nil
}

func LatestStableConsuming(manifests []Manifest) (*Manifest, error) {
	var best *Manifest
	for idx := range manifests {
		item := manifests[idx]
		if item.Channel != ChannelStable || !item.Consumes {
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
	return best, nil
}

func StableConsumedFragmentIDs(manifests []Manifest) map[string]struct{} {
	out := make(map[string]struct{})
	for _, manifest := range manifests {
		if manifest.Channel != ChannelStable || !manifest.Consumes {
			continue
		}
		for _, id := range manifest.FragmentIDs {
			out[id] = struct{}{}
		}
	}
	return out
}

func PreviewLineConsumedIDs(manifests []Manifest, targetVersion, label string) map[string]struct{} {
	line := targetVersion + "-" + label
	out := make(map[string]struct{})
	for _, manifest := range manifests {
		version := versioning.MustParse(manifest.Version)
		if manifest.Channel != ChannelPreview || version.ReleaseLine() != line {
			continue
		}
		for _, id := range manifest.FragmentIDs {
			out[id] = struct{}{}
		}
	}
	return out
}

func PreviewVersionsForLine(manifests []Manifest, targetVersion, label string) []versioning.Version {
	line := targetVersion + "-" + label
	out := make([]versioning.Version, 0)
	for _, manifest := range manifests {
		version := versioning.MustParse(manifest.Version)
		if manifest.Channel != ChannelPreview || version.ReleaseLine() != line {
			continue
		}
		out = append(out, version)
	}
	return out
}

func LatestPreview(manifests []Manifest) *Manifest {
	var best *Manifest
	for idx := range manifests {
		item := manifests[idx]
		if item.Channel != ChannelPreview {
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

func UnreleasedStableFragments(all []fragments.Fragment, manifests []Manifest) []fragments.Fragment {
	consumed := StableConsumedFragmentIDs(manifests)
	out := make([]fragments.Fragment, 0, len(all))
	for _, item := range all {
		if _, ok := consumed[item.ID]; ok {
			continue
		}
		out = append(out, item)
	}
	return out
}

func UnreleasedPreviewFragments(all []fragments.Fragment, manifests []Manifest, targetVersion, label string) []fragments.Fragment {
	stableUnreleased := UnreleasedStableFragments(all, manifests)
	if len(stableUnreleased) == 0 {
		return nil
	}

	referenced := PreviewLineConsumedIDs(manifests, targetVersion, label)
	out := make([]fragments.Fragment, 0, len(stableUnreleased))
	for _, item := range stableUnreleased {
		if _, ok := referenced[item.ID]; ok {
			continue
		}
		out = append(out, item)
	}
	return out
}
