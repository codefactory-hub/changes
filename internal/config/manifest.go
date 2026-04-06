package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
)

var (
	errInvalidLayoutManifest = errors.New("invalid layout manifest")
	symbolPattern            = regexp.MustCompile(`\$[A-Za-z_][A-Za-z0-9_.]*`)
)

type layoutDocument struct {
	SchemaVersion int   `toml:"schema_version"`
	Scope         Scope `toml:"scope"`
	Style         Style `toml:"style"`
	Layout        struct {
		Root   string `toml:"root"`
		Config string `toml:"config"`
		Data   string `toml:"data"`
		State  string `toml:"state"`
	} `toml:"layout"`
}

func loadLayoutManifest(path string, scope Scope, style Style, candidatePaths LayoutPaths, opts ResolveOptions) (*LayoutManifest, error) {
	var doc layoutDocument
	meta, err := toml.DecodeFile(path, &doc)
	if err != nil {
		return nil, fmt.Errorf("%w: decode %s: %v", errInvalidLayoutManifest, path, err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return nil, fmt.Errorf("%w: unsupported keys: %s", errInvalidLayoutManifest, joinKeys(undecoded))
	}
	if doc.Scope != scope {
		return nil, fmt.Errorf("%w: scope %q does not match %q", errInvalidLayoutManifest, doc.Scope, scope)
	}
	if doc.Style != style {
		return nil, fmt.Errorf("%w: style %q does not match %q", errInvalidLayoutManifest, doc.Style, style)
	}

	symbolic := LayoutPaths{
		Root:   strings.TrimSpace(doc.Layout.Root),
		Config: strings.TrimSpace(doc.Layout.Config),
		Data:   strings.TrimSpace(doc.Layout.Data),
		State:  strings.TrimSpace(doc.Layout.State),
	}
	if symbolic.Root == "" || symbolic.Config == "" || symbolic.Data == "" || symbolic.State == "" {
		return nil, fmt.Errorf("%w: layout paths must be non-empty", errInvalidLayoutManifest)
	}

	resolved, err := expandManifestPaths(symbolic, opts)
	if err != nil {
		return nil, err
	}
	if scope == ScopeRepo {
		if err := ensureRepoLocalPaths(opts.RepoRoot, resolved); err != nil {
			return nil, err
		}
	}
	if err := ensureEquivalentCandidate(candidatePaths, resolved); err != nil {
		return nil, err
	}

	return &LayoutManifest{
		SchemaVersion: doc.SchemaVersion,
		Scope:         doc.Scope,
		Style:         doc.Style,
		Symbolic:      symbolic,
		Resolved:      resolved,
	}, nil
}

func expandManifestPaths(symbolic LayoutPaths, opts ResolveOptions) (LayoutPaths, error) {
	root, err := expandSymbolic(symbolic.Root, map[string]string{
		"$REPO_ROOT":       opts.RepoRoot,
		"$CHANGES_HOME":    opts.ChangesHome,
		"$XDG_CONFIG_HOME": opts.XDGConfigHome,
		"$XDG_DATA_HOME":   opts.XDGDataHome,
		"$XDG_STATE_HOME":  opts.XDGStateHome,
		"$HOME":            opts.HomeDir,
	})
	if err != nil {
		return LayoutPaths{}, err
	}

	vars := map[string]string{
		"$REPO_ROOT":       opts.RepoRoot,
		"$CHANGES_HOME":    opts.ChangesHome,
		"$XDG_CONFIG_HOME": opts.XDGConfigHome,
		"$XDG_DATA_HOME":   opts.XDGDataHome,
		"$XDG_STATE_HOME":  opts.XDGStateHome,
		"$HOME":            opts.HomeDir,
		"$layout.root":     root,
	}

	config, err := expandSymbolic(symbolic.Config, vars)
	if err != nil {
		return LayoutPaths{}, err
	}
	data, err := expandSymbolic(symbolic.Data, vars)
	if err != nil {
		return LayoutPaths{}, err
	}
	state, err := expandSymbolic(symbolic.State, vars)
	if err != nil {
		return LayoutPaths{}, err
	}

	return LayoutPaths{
		Root:   filepath.Clean(root),
		Config: filepath.Clean(config),
		Data:   filepath.Clean(data),
		State:  filepath.Clean(state),
	}, nil
}

func expandSymbolic(value string, vars map[string]string) (string, error) {
	expanded := value
	for _, token := range symbolPattern.FindAllString(value, -1) {
		replacement, ok := vars[token]
		if !ok {
			return "", fmt.Errorf("%w: unsupported symbolic reference %q", errInvalidLayoutManifest, token)
		}
		if strings.TrimSpace(replacement) == "" {
			return "", fmt.Errorf("%w: symbolic reference %q resolved to empty value", errInvalidLayoutManifest, token)
		}
		expanded = strings.ReplaceAll(expanded, token, replacement)
	}
	if strings.Contains(expanded, "$") {
		return "", fmt.Errorf("%w: unsupported symbolic reference in %q", errInvalidLayoutManifest, value)
	}
	return filepath.Clean(expanded), nil
}

func ensureRepoLocalPaths(repoRoot string, paths LayoutPaths) error {
	for _, candidate := range []struct {
		name string
		path string
	}{
		{name: "root", path: paths.Root},
		{name: "config", path: paths.Config},
		{name: "data", path: paths.Data},
		{name: "state", path: paths.State},
	} {
		ok, err := withinRoot(repoRoot, candidate.path)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%w: repo-local %s path escapes repo root", errInvalidLayoutManifest, candidate.name)
		}
	}
	return nil
}

func ensureEquivalentCandidate(expected LayoutPaths, actual LayoutPaths) error {
	for _, pair := range []struct {
		name     string
		expected string
		actual   string
	}{
		{name: "root", expected: expected.Root, actual: actual.Root},
		{name: "config", expected: expected.Config, actual: actual.Config},
		{name: "data", expected: expected.Data, actual: actual.Data},
		{name: "state", expected: expected.State, actual: actual.State},
	} {
		if pair.expected == "" && pair.actual == "" {
			continue
		}
		equal, err := equivalentPaths(pair.expected, pair.actual)
		if err != nil {
			return err
		}
		if !equal {
			return fmt.Errorf("%w: manifest %s path %q does not match candidate path %q", errInvalidLayoutManifest, pair.name, pair.actual, pair.expected)
		}
	}
	return nil
}

func equivalentPaths(left string, right string) (bool, error) {
	leftPath, err := canonicalPathForComparison(left)
	if err != nil {
		return false, err
	}
	rightPath, err := canonicalPathForComparison(right)
	if err != nil {
		return false, err
	}
	return leftPath == rightPath, nil
}

func canonicalPathForComparison(path string) (string, error) {
	clean := filepath.Clean(path)

	current := clean
	var suffix []string
	for {
		_, err := os.Lstat(current)
		if err == nil {
			evaluated, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			parts := append([]string{filepath.Clean(evaluated)}, suffix...)
			return filepath.Join(parts...), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return clean, nil
		}
		suffix = append([]string{filepath.Base(current)}, suffix...)
		current = parent
	}
}

func withinRoot(root string, path string) (bool, error) {
	canonicalRoot, err := canonicalPathForComparison(root)
	if err != nil {
		return false, err
	}
	canonicalPath, err := canonicalPathForComparison(path)
	if err != nil {
		return false, err
	}

	rel, err := filepath.Rel(canonicalRoot, canonicalPath)
	if err != nil {
		return false, err
	}
	if rel == "." {
		return true, nil
	}
	return filepath.IsLocal(rel), nil
}
