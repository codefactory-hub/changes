package changelog

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/render"
	"github.com/example/changes/internal/versioning"
)

func Rebuild(repoRoot string, cfg config.Config, allFragments []fragments.Fragment, manifests []releases.Manifest) (string, error) {
	index := make(map[string]fragments.Fragment, len(allFragments))
	for _, item := range allFragments {
		index[item.ID] = item
	}

	stable := make([]releases.Manifest, 0)
	for _, manifest := range manifests {
		if manifest.Channel == releases.ChannelStable && manifest.Consumes {
			stable = append(stable, manifest)
		}
	}

	slices.SortFunc(stable, func(a, b releases.Manifest) int {
		av := versioning.MustParse(a.Version)
		bv := versioning.MustParse(b.Version)
		if cmp := versioning.Compare(av, bv); cmp != 0 {
			return -cmp
		}
		if a.CreatedAt.Before(b.CreatedAt) {
			return 1
		}
		if a.CreatedAt.After(b.CreatedAt) {
			return -1
		}
		return strings.Compare(a.Version, b.Version)
	})

	parts := []string{"# Changelog"}
	for _, manifest := range stable {
		renderer, err := render.New(repoRoot, cfg, manifest)
		if err != nil {
			return "", err
		}

		selected := make([]fragments.Fragment, 0, len(manifest.FragmentIDs))
		for _, id := range manifest.FragmentIDs {
			item, ok := index[id]
			if !ok {
				return "", fmt.Errorf("rebuild changelog: fragment %s referenced by %s is missing", id, manifest.Version)
			}
			selected = append(selected, item)
		}

		text, err := renderer.Render(cfg, manifest, selected)
		if err != nil {
			return "", err
		}
		parts = append(parts, strings.TrimSpace(text))
	}

	return strings.Join(parts, "\n\n") + "\n", nil
}

func Write(repoRoot string, cfg config.Config, content string) error {
	path := config.ChangelogPath(repoRoot, cfg)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write changelog: %w", err)
	}
	return nil
}
