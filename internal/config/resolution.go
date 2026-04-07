package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

func ResolveAll(opts ResolveOptions) (Resolution, error) {
	global, err := resolveScope(ScopeGlobal, opts)
	if err != nil {
		return Resolution{}, err
	}

	repo, err := resolveScope(ScopeRepo, opts)
	if err != nil {
		return Resolution{}, err
	}

	return Resolution{
		Global: global,
		Repo:   repo,
	}, nil
}

func ResolveGlobal(opts ResolveOptions) (ScopeResolution, error) {
	all, err := ResolveAll(opts)
	if err != nil {
		return ScopeResolution{}, err
	}
	return all.Global, nil
}

func ResolveRepo(opts ResolveOptions) (ScopeResolution, error) {
	all, err := ResolveAll(opts)
	if err != nil {
		return ScopeResolution{}, err
	}
	return all.Repo, nil
}

func resolveScope(scope Scope, opts ResolveOptions) (ScopeResolution, error) {
	candidates, err := inspectCandidates(scope, opts)
	if err != nil {
		return ScopeResolution{}, err
	}

	return resolveScopeFromCandidates(scope, opts, candidates)
}

func inspectCandidates(scope Scope, opts ResolveOptions) ([]Candidate, error) {
	specs := []struct {
		style Style
		paths LayoutPaths
	}{
		{style: StyleXDG, paths: supportedPaths(scope, StyleXDG, opts)},
		{style: StyleHome, paths: supportedPaths(scope, StyleHome, opts)},
	}

	candidates := make([]Candidate, 0, len(specs))
	for _, spec := range specs {
		candidate, err := inspectCandidate(scope, spec.style, spec.paths, opts)
		if err != nil {
			return nil, err
		}
		candidates = append(candidates, candidate)
	}

	return candidates, nil
}

func inspectCandidate(scope Scope, style Style, paths LayoutPaths, opts ResolveOptions) (Candidate, error) {
	candidate := Candidate{
		Scope:    scope,
		Style:    style,
		Status:   StatusUninitialized,
		Paths:    paths,
		Evidence: candidateSignals(scope, style, paths, opts),
	}

	manifestPath := filepath.Join(paths.Config, "layout.toml")
	manifest, manifestExists, err := inspectManifestCandidate(manifestPath, scope, style, paths, opts)
	if err != nil {
		return Candidate{}, err
	}
	candidate.Evidence = append(candidate.Evidence, CandidateEvidence{
		Kind:   "file",
		Name:   "layout.toml",
		Path:   manifestPath,
		Exists: manifestExists,
	})
	if manifestExists {
		if manifest == nil {
			candidate.Status = StatusInvalid
			return candidate, nil
		}
		candidate.Status = StatusResolved
		candidate.Manifest = manifest
		return candidate, nil
	}

	configPath := filepath.Join(paths.Config, "config.toml")
	configExists, err := fileExists(configPath)
	if err != nil {
		return Candidate{}, fmt.Errorf("inspect %s config artifact: %w", style, err)
	}
	candidate.Evidence = append(candidate.Evidence, CandidateEvidence{
		Kind:   "file",
		Name:   "config.toml",
		Path:   configPath,
		Exists: configExists,
	})
	if configExists {
		candidate.Status = StatusLegacyOnly
	}

	return candidate, nil
}

func supportedPaths(scope Scope, style Style, opts ResolveOptions) LayoutPaths {
	switch scope {
	case ScopeGlobal:
		return globalPaths(style, opts)
	case ScopeRepo:
		return repoPaths(style, opts)
	default:
		return LayoutPaths{}
	}
}

func globalPaths(style Style, opts ResolveOptions) LayoutPaths {
	homeDir := filepath.Clean(opts.HomeDir)
	switch style {
	case StyleHome:
		root := opts.ChangesHome
		if root == "" {
			root = filepath.Join(homeDir, ".changes")
		}
		root = filepath.Clean(root)
		return LayoutPaths{
			Root:   root,
			Config: filepath.Join(root, "config"),
			Data:   filepath.Join(root, "data"),
			State:  filepath.Join(root, "state"),
		}
	case StyleXDG:
		configRoot := opts.XDGConfigHome
		if configRoot == "" {
			configRoot = filepath.Join(homeDir, ".config")
		}
		dataRoot := opts.XDGDataHome
		if dataRoot == "" {
			dataRoot = filepath.Join(homeDir, ".local", "share")
		}
		stateRoot := opts.XDGStateHome
		if stateRoot == "" {
			stateRoot = filepath.Join(homeDir, ".local", "state")
		}
		return LayoutPaths{
			Root:   homeDir,
			Config: filepath.Join(configRoot, "changes"),
			Data:   filepath.Join(dataRoot, "changes"),
			State:  filepath.Join(stateRoot, "changes"),
		}
	default:
		return LayoutPaths{}
	}
}

func repoPaths(style Style, opts ResolveOptions) LayoutPaths {
	repoRoot := filepath.Clean(opts.RepoRoot)
	switch style {
	case StyleHome:
		root := filepath.Join(repoRoot, ".changes")
		return LayoutPaths{
			Root:   root,
			Config: filepath.Join(root, "config"),
			Data:   filepath.Join(root, "data"),
			State:  filepath.Join(root, "state"),
		}
	case StyleXDG:
		return LayoutPaths{
			Root:   repoRoot,
			Config: filepath.Join(repoRoot, ".config", "changes"),
			Data:   filepath.Join(repoRoot, ".local", "share", "changes"),
			State:  filepath.Join(repoRoot, ".local", "state", "changes"),
		}
	default:
		return LayoutPaths{}
	}
}

func summarizeScopeStatus(candidates []Candidate) ResolutionStatus {
	groupCount, _ := operationalAuthorityGroupCount(candidates)
	return summarizeScopeStatusWithGroupCount(candidates, groupCount)
}

func resolveScopeFromCandidates(scope Scope, opts ResolveOptions, candidates []Candidate) (ScopeResolution, error) {
	groupCount, err := operationalAuthorityGroupCount(candidates)
	if err != nil {
		return ScopeResolution{}, err
	}

	resolution := ScopeResolution{
		Scope:      scope,
		Candidates: candidates,
		Status:     summarizeScopeStatusWithGroupCount(candidates, groupCount),
	}

	group, err := authoritativeCandidateGroup(candidates)
	if err != nil {
		return ScopeResolution{}, err
	}

	if resolution.Status == StatusResolved && len(group) > 0 {
		authoritativeIndex := pickAuthorityIndex(scope, opts, group, candidates)
		resolution.Authoritative = &resolution.Candidates[authoritativeIndex]
		resolution.Warnings = authorityWarnings(candidates)
	}

	if resolution.Authoritative != nil {
		resolution.Preferred = resolution.Authoritative
	} else if preferred := preferredStyle(scope, opts); preferred != "" {
		for i := range resolution.Candidates {
			if resolution.Candidates[i].Style == preferred {
				resolution.Preferred = &resolution.Candidates[i]
				break
			}
		}
	}

	return resolution, nil
}

func summarizeScopeStatusWithGroupCount(candidates []Candidate, groupCount int) ResolutionStatus {
	legacy := 0
	invalid := 0

	for _, candidate := range candidates {
		switch candidate.Status {
		case StatusLegacyOnly:
			legacy++
		case StatusInvalid:
			invalid++
		}
	}

	switch {
	case groupCount > 1:
		return StatusAmbiguous
	case groupCount == 1:
		return StatusResolved
	case invalid > 0:
		return StatusInvalid
	case legacy > 0:
		return StatusLegacyOnly
	default:
		return StatusUninitialized
	}
}

func operationalAuthorityGroupCount(candidates []Candidate) (int, error) {
	groups, err := operationalAuthorityGroups(candidates)
	if err != nil {
		return 0, err
	}
	return len(groups), nil
}

func authoritativeCandidateGroup(candidates []Candidate) ([]int, error) {
	groups, err := operationalAuthorityGroups(candidates)
	if err != nil {
		return nil, err
	}
	if len(groups) != 1 {
		return nil, nil
	}
	for _, indexes := range groups {
		return indexes, nil
	}
	return nil, nil
}

func operationalAuthorityGroups(candidates []Candidate) (map[string][]int, error) {
	groups := make(map[string][]int)
	for i, candidate := range candidates {
		if candidate.Status != StatusResolved {
			continue
		}

		key, err := candidateAuthorityKey(candidate)
		if err != nil {
			return nil, err
		}
		groups[key] = append(groups[key], i)
	}
	return groups, nil
}

func candidateAuthorityKey(candidate Candidate) (string, error) {
	root := candidate.Paths.Root
	if candidate.Manifest != nil && candidate.Manifest.Resolved.Root != "" {
		root = candidate.Manifest.Resolved.Root
	}
	return canonicalPathForComparison(root)
}

func pickAuthorityIndex(scope Scope, opts ResolveOptions, group []int, candidates []Candidate) int {
	if preferred := preferredStyle(scope, opts); preferred != "" {
		for _, index := range group {
			if candidates[index].Style == preferred {
				return index
			}
		}
	}
	return group[0]
}

func authorityWarnings(candidates []Candidate) []AuthorityWarning {
	warnings := make([]AuthorityWarning, 0)
	for _, candidate := range candidates {
		if candidate.Status != StatusLegacyOnly && candidate.Status != StatusInvalid {
			continue
		}
		warnings = append(warnings, AuthorityWarning{
			Scope:  candidate.Scope,
			Style:  candidate.Style,
			Status: candidate.Status,
			Path:   candidate.Paths.Config,
		})
	}
	return warnings
}

func preferredStyle(scope Scope, opts ResolveOptions) Style {
	switch scope {
	case ScopeGlobal:
		if opts.ChangesHome != "" {
			return StyleHome
		}
		return StyleXDG
	case ScopeRepo:
		return StyleXDG
	default:
		return ""
	}
}

func candidateSignals(scope Scope, style Style, paths LayoutPaths, opts ResolveOptions) []CandidateEvidence {
	evidence := []CandidateEvidence{{
		Kind:   "candidate",
		Name:   string(style),
		Path:   paths.Config,
		Exists: true,
	}}

	if scope == ScopeGlobal && style == StyleHome && opts.ChangesHome != "" {
		evidence = append(evidence, CandidateEvidence{
			Kind:   "env",
			Name:   "CHANGES_HOME",
			Path:   opts.ChangesHome,
			Exists: true,
		})
	}

	if scope == ScopeGlobal && style == StyleXDG {
		if opts.XDGConfigHome != "" {
			evidence = append(evidence, CandidateEvidence{Kind: "env", Name: "XDG_CONFIG_HOME", Path: opts.XDGConfigHome, Exists: true})
		}
		if opts.XDGDataHome != "" {
			evidence = append(evidence, CandidateEvidence{Kind: "env", Name: "XDG_DATA_HOME", Path: opts.XDGDataHome, Exists: true})
		}
		if opts.XDGStateHome != "" {
			evidence = append(evidence, CandidateEvidence{Kind: "env", Name: "XDG_STATE_HOME", Path: opts.XDGStateHome, Exists: true})
		}
	}

	return evidence
}

func inspectManifestCandidate(path string, scope Scope, style Style, resolved LayoutPaths, opts ResolveOptions) (*LayoutManifest, bool, error) {
	exists, err := fileExists(path)
	if err != nil {
		return nil, false, fmt.Errorf("inspect manifest: %w", err)
	}
	if !exists {
		return nil, false, nil
	}

	manifest, err := loadLayoutManifest(path, scope, style, resolved, opts)
	if err != nil {
		if errors.Is(err, errInvalidLayoutManifest) {
			return nil, true, nil
		}
		return nil, true, err
	}

	return manifest, true, nil
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}
