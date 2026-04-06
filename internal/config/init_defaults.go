package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

type RepoInitSelectionOptions struct {
	RepoRoot        string
	RequestedStyle  string
	RequestedHome   string
	GlobalInitStyle string
	GlobalInitHome  string
	ChangesHome     string
	XDGConfigHome   string
	XDGDataHome     string
	XDGStateHome    string
}

type RepoInitSelection struct {
	Style          Style
	Root           string
	Config         string
	Data           string
	State          string
	GitignoreEntry string
}

func SelectRepoInitLayout(opts RepoInitSelectionOptions) (RepoInitSelection, error) {
	repoRoot := filepath.Clean(strings.TrimSpace(opts.RepoRoot))
	if repoRoot == "" || repoRoot == "." {
		return RepoInitSelection{}, fmt.Errorf("select repo init layout: repo root is required")
	}

	style, home, err := selectRepoInitStyleAndHome(opts)
	if err != nil {
		return RepoInitSelection{}, err
	}

	switch style {
	case StyleHome:
		root, err := normalizeRepoInitHome(repoRoot, home)
		if err != nil {
			return RepoInitSelection{}, err
		}
		return RepoInitSelection{
			Style:          StyleHome,
			Root:           root,
			Config:         filepath.Join(root, "config"),
			Data:           filepath.Join(root, "data"),
			State:          filepath.Join(root, "state"),
			GitignoreEntry: "/.changes/state/",
		}, nil
	case StyleXDG:
		paths := repoPaths(StyleXDG, ResolveOptions{RepoRoot: repoRoot})
		return RepoInitSelection{
			Style:          StyleXDG,
			Root:           paths.Root,
			Config:         paths.Config,
			Data:           paths.Data,
			State:          paths.State,
			GitignoreEntry: "/.local/state/",
		}, nil
	default:
		return RepoInitSelection{}, fmt.Errorf("select repo init layout: unsupported style %q", style)
	}
}

func selectRepoInitStyleAndHome(opts RepoInitSelectionOptions) (Style, string, error) {
	if strings.TrimSpace(opts.RequestedStyle) != "" || strings.TrimSpace(opts.RequestedHome) != "" {
		return normalizeRepoInitPreference(opts.RequestedStyle, opts.RequestedHome, "flags")
	}
	if strings.TrimSpace(opts.GlobalInitStyle) != "" || strings.TrimSpace(opts.GlobalInitHome) != "" {
		return normalizeRepoInitPreference(opts.GlobalInitStyle, opts.GlobalInitHome, "global config")
	}
	if strings.TrimSpace(opts.ChangesHome) != "" {
		return StyleHome, ".changes", nil
	}
	if strings.TrimSpace(opts.XDGConfigHome) != "" || strings.TrimSpace(opts.XDGDataHome) != "" || strings.TrimSpace(opts.XDGStateHome) != "" {
		return StyleXDG, "", nil
	}
	return StyleXDG, "", nil
}

func normalizeRepoInitPreference(rawStyle, rawHome, source string) (Style, string, error) {
	style := Style(strings.TrimSpace(rawStyle))
	home := strings.TrimSpace(rawHome)

	switch style {
	case "":
		if home != "" {
			return "", "", fmt.Errorf("select repo init layout: %s home requires style home", source)
		}
		return "", "", nil
	case StyleXDG:
		if home != "" {
			return "", "", fmt.Errorf("select repo init layout: %s home is only valid with style home", source)
		}
		return StyleXDG, "", nil
	case StyleHome:
		if home == "" {
			home = ".changes"
		}
		return StyleHome, home, nil
	default:
		return "", "", fmt.Errorf("select repo init layout: %s style must be xdg or home", source)
	}
}

func normalizeRepoInitHome(repoRoot, rawHome string) (string, error) {
	home := strings.TrimSpace(rawHome)
	if home == "" {
		home = ".changes"
	}
	clean := filepath.Clean(home)
	if !filepath.IsLocal(clean) {
		return "", fmt.Errorf("select repo init layout: home path %q must stay within the repo root", rawHome)
	}
	return filepath.Join(repoRoot, clean), nil
}
