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

type GlobalInitSelectionOptions struct {
	HomeDir        string
	RequestedStyle string
	RequestedHome  string
	ChangesHome    string
	XDGConfigHome  string
	XDGDataHome    string
	XDGStateHome   string
}

type GlobalInitSelection struct {
	Style  Style
	Root   string
	Config string
	Data   string
	State  string
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
			GitignoreEntry: repoStateGitignoreEntry(repoRoot, filepath.Join(root, "state")),
		}, nil
	case StyleXDG:
		paths := repoPaths(StyleXDG, ResolveOptions{RepoRoot: repoRoot})
		return RepoInitSelection{
			Style:          StyleXDG,
			Root:           paths.Root,
			Config:         paths.Config,
			Data:           paths.Data,
			State:          paths.State,
			GitignoreEntry: repoStateGitignoreEntry(repoRoot, paths.State),
		}, nil
	default:
		return RepoInitSelection{}, fmt.Errorf("select repo init layout: unsupported style %q", style)
	}
}

func SelectGlobalInitLayout(opts GlobalInitSelectionOptions) (GlobalInitSelection, error) {
	style, root, err := selectGlobalInitStyleAndRoot(opts)
	if err != nil {
		return GlobalInitSelection{}, err
	}

	switch style {
	case StyleHome:
		paths := globalPaths(StyleHome, ResolveOptions{
			HomeDir:     strings.TrimSpace(opts.HomeDir),
			ChangesHome: root,
		})
		return GlobalInitSelection{
			Style:  StyleHome,
			Root:   paths.Root,
			Config: paths.Config,
			Data:   paths.Data,
			State:  paths.State,
		}, nil
	case StyleXDG:
		paths := globalPaths(StyleXDG, ResolveOptions{
			HomeDir:       strings.TrimSpace(opts.HomeDir),
			XDGConfigHome: strings.TrimSpace(opts.XDGConfigHome),
			XDGDataHome:   strings.TrimSpace(opts.XDGDataHome),
			XDGStateHome:  strings.TrimSpace(opts.XDGStateHome),
		})
		return GlobalInitSelection{
			Style:  StyleXDG,
			Root:   paths.Root,
			Config: paths.Config,
			Data:   paths.Data,
			State:  paths.State,
		}, nil
	default:
		return GlobalInitSelection{}, fmt.Errorf("select global init layout: unsupported style %q", style)
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

func selectGlobalInitStyleAndRoot(opts GlobalInitSelectionOptions) (Style, string, error) {
	homeDir := strings.TrimSpace(opts.HomeDir)

	if strings.TrimSpace(opts.RequestedStyle) != "" || strings.TrimSpace(opts.RequestedHome) != "" {
		style, home, err := normalizeGlobalInitPreference(opts.RequestedStyle, opts.RequestedHome, "flags")
		if err != nil {
			return "", "", err
		}
		if style == StyleHome {
			root, err := normalizeGlobalInitHome(homeDir, home, strings.TrimSpace(opts.ChangesHome))
			if err != nil {
				return "", "", err
			}
			return StyleHome, root, nil
		}
		return StyleXDG, "", nil
	}

	if strings.TrimSpace(opts.ChangesHome) != "" {
		root, err := normalizeGlobalInitHome(homeDir, "", strings.TrimSpace(opts.ChangesHome))
		if err != nil {
			return "", "", err
		}
		return StyleHome, root, nil
	}

	if strings.TrimSpace(opts.XDGConfigHome) != "" || strings.TrimSpace(opts.XDGDataHome) != "" || strings.TrimSpace(opts.XDGStateHome) != "" {
		return StyleXDG, "", nil
	}

	return StyleXDG, "", nil
}

func normalizeGlobalInitPreference(rawStyle, rawHome, source string) (Style, string, error) {
	style := Style(strings.TrimSpace(rawStyle))
	home := strings.TrimSpace(rawHome)

	switch style {
	case "":
		if home != "" {
			return "", "", fmt.Errorf("select global init layout: %s home is only valid with style home", source)
		}
		return "", "", nil
	case StyleXDG:
		if home != "" {
			return "", "", fmt.Errorf("select global init layout: %s home is only valid with style home", source)
		}
		return StyleXDG, "", nil
	case StyleHome:
		return StyleHome, home, nil
	default:
		return "", "", fmt.Errorf("select global init layout: %s style must be xdg or home", source)
	}
}

func normalizeGlobalInitHome(homeDir, rawHome, changesHome string) (string, error) {
	root := strings.TrimSpace(rawHome)
	if root == "" {
		root = strings.TrimSpace(changesHome)
	}
	if root == "" {
		if strings.TrimSpace(homeDir) == "" {
			return "", fmt.Errorf("select global init layout: home dir is required for style home")
		}
		root = filepath.Join(homeDir, ".changes")
	}

	clean := filepath.Clean(root)
	if filepath.IsAbs(clean) {
		return clean, nil
	}
	if strings.TrimSpace(homeDir) == "" {
		return "", fmt.Errorf("select global init layout: relative home path %q requires a home dir", rawHome)
	}
	return filepath.Join(homeDir, clean), nil
}

func repoStateGitignoreEntry(repoRoot, path string) string {
	rel, err := filepath.Rel(repoRoot, path)
	if err != nil {
		return "/" + filepath.ToSlash(filepath.Base(path)) + "/"
	}
	return "/" + strings.Trim(filepath.ToSlash(rel), "/") + "/"
}
