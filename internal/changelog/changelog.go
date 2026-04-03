package changelog

import (
	"fmt"
	"os"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/render"
)

func Rebuild(repoRoot string, cfg config.Config, allFragments []fragments.Fragment, manifests []releases.Manifest) (string, error) {
	head := releases.LatestStableHead(manifests)
	if head == nil {
		pack, err := cfg.RenderProfile(config.RenderProfileRepositoryMarkdown)
		if err != nil {
			return "", err
		}
		if pack.DocumentHeader == "" {
			return "", nil
		}
		return pack.DocumentHeader + "\n", nil
	}

	selector := render.NewSelector(allFragments, manifests)
	doc, err := selector.ReleaseChain(*head)
	if err != nil {
		return "", err
	}

	renderer, err := render.New(repoRoot, cfg, config.RenderProfileRepositoryMarkdown)
	if err != nil {
		return "", err
	}
	content, err := renderer.Render(doc)
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
