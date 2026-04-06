package changelog

import (
	"fmt"
	"os"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/render"
)

func Rebuild(repoRoot string, cfg config.Config, allFragments []fragments.Fragment, records []releases.ReleaseRecord) (string, error) {
	head := releases.LatestFinalHeadForProduct(records, cfg.Project.Name)
	if head == nil {
		profiles, err := render.ResolveProfiles(cfg)
		if err != nil {
			return "", err
		}
		pack, ok := profiles[config.RenderProfileRepositoryMarkdown]
		if !ok {
			return "", fmt.Errorf("render profile %q is not configured", config.RenderProfileRepositoryMarkdown)
		}
		if pack.DocumentHeader == "" {
			return "", nil
		}
		return pack.DocumentHeader + "\n", nil
	}

	bundles, err := releases.AssembleReleaseLineage(*head, records, allFragments)
	if err != nil {
		return "", err
	}

	renderer, err := render.New(repoRoot, cfg, config.RenderProfileRepositoryMarkdown)
	if err != nil {
		return "", err
	}
	content, err := renderer.Render(render.Document{Bundles: bundles})
	if err != nil {
		return "", fmt.Errorf("rebuild changelog: %w", err)
	}
	return content, nil
}

func Write(repoRoot string, cfg config.Config, content string) error {
	path := config.ChangelogPath(repoRoot, cfg)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write changelog: %w", err)
	}
	return nil
}
