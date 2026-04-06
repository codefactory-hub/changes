package config

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolveGlobalPrefersChangesHomeOverXDG(t *testing.T) {
	root := t.TempDir()
	opts := ResolveOptions{
		HomeDir:       filepath.Join(root, "home"),
		ChangesHome:   filepath.Join(root, "changes-home"),
		XDGConfigHome: filepath.Join(root, "xdg-config"),
		XDGDataHome:   filepath.Join(root, "xdg-data"),
		XDGStateHome:  filepath.Join(root, "xdg-state"),
	}

	resolution, err := ResolveGlobal(opts)
	if err != nil {
		t.Fatalf("ResolveGlobal returned error: %v", err)
	}
	if resolution.Status != StatusUninitialized {
		t.Fatalf("status = %q, want %q", resolution.Status, StatusUninitialized)
	}
	if resolution.Preferred == nil {
		t.Fatalf("Preferred candidate is nil")
	}
	if resolution.Preferred.Style != StyleHome {
		t.Fatalf("preferred style = %q, want %q", resolution.Preferred.Style, StyleHome)
	}
	if resolution.Preferred.Paths.Root != opts.ChangesHome {
		t.Fatalf("preferred root = %q, want %q", resolution.Preferred.Paths.Root, opts.ChangesHome)
	}
}

func TestResolveGlobalPathsForSupportedStyles(t *testing.T) {
	root := t.TempDir()
	opts := ResolveOptions{
		HomeDir:       filepath.Join(root, "home"),
		ChangesHome:   filepath.Join(root, "changes-home"),
		XDGConfigHome: filepath.Join(root, "xdg-config"),
		XDGDataHome:   filepath.Join(root, "xdg-data"),
		XDGStateHome:  filepath.Join(root, "xdg-state"),
	}

	resolution, err := ResolveGlobal(opts)
	if err != nil {
		t.Fatalf("ResolveGlobal returned error: %v", err)
	}

	xdg := findCandidateByStyle(t, resolution, StyleXDG)
	if xdg.Paths.Config != filepath.Join(opts.XDGConfigHome, "changes") {
		t.Fatalf("xdg config = %q", xdg.Paths.Config)
	}
	if xdg.Paths.Data != filepath.Join(opts.XDGDataHome, "changes") {
		t.Fatalf("xdg data = %q", xdg.Paths.Data)
	}
	if xdg.Paths.State != filepath.Join(opts.XDGStateHome, "changes") {
		t.Fatalf("xdg state = %q", xdg.Paths.State)
	}

	home := findCandidateByStyle(t, resolution, StyleHome)
	if home.Paths.Config != filepath.Join(opts.ChangesHome, "config") {
		t.Fatalf("home config = %q", home.Paths.Config)
	}
	if home.Paths.Data != filepath.Join(opts.ChangesHome, "data") {
		t.Fatalf("home data = %q", home.Paths.Data)
	}
	if home.Paths.State != filepath.Join(opts.ChangesHome, "state") {
		t.Fatalf("home state = %q", home.Paths.State)
	}
}

func TestResolveRepoPathsForSupportedStyles(t *testing.T) {
	repoRoot := t.TempDir()

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}

	xdg := findCandidateByStyle(t, resolution, StyleXDG)
	if xdg.Paths.Config != filepath.Join(repoRoot, ".config", "changes") {
		t.Fatalf("xdg config = %q", xdg.Paths.Config)
	}
	if xdg.Paths.Data != filepath.Join(repoRoot, ".local", "share", "changes") {
		t.Fatalf("xdg data = %q", xdg.Paths.Data)
	}
	if xdg.Paths.State != filepath.Join(repoRoot, ".local", "state", "changes") {
		t.Fatalf("xdg state = %q", xdg.Paths.State)
	}

	home := findCandidateByStyle(t, resolution, StyleHome)
	if home.Paths.Root != filepath.Join(repoRoot, ".changes") {
		t.Fatalf("home root = %q", home.Paths.Root)
	}
	if home.Paths.Config != filepath.Join(repoRoot, ".changes", "config") {
		t.Fatalf("home config = %q", home.Paths.Config)
	}
	if home.Paths.Data != filepath.Join(repoRoot, ".changes", "data") {
		t.Fatalf("home data = %q", home.Paths.Data)
	}
	if home.Paths.State != filepath.Join(repoRoot, ".changes", "state") {
		t.Fatalf("home state = %q", home.Paths.State)
	}
}

func TestResolveAllReturnsThinScopeWrappers(t *testing.T) {
	root := t.TempDir()
	opts := ResolveOptions{
		RepoRoot:      filepath.Join(root, "repo"),
		HomeDir:       filepath.Join(root, "home"),
		ChangesHome:   filepath.Join(root, "changes-home"),
		XDGConfigHome: filepath.Join(root, "xdg-config"),
		XDGDataHome:   filepath.Join(root, "xdg-data"),
		XDGStateHome:  filepath.Join(root, "xdg-state"),
	}

	all, err := ResolveAll(opts)
	if err != nil {
		t.Fatalf("ResolveAll returned error: %v", err)
	}

	global, err := ResolveGlobal(opts)
	if err != nil {
		t.Fatalf("ResolveGlobal returned error: %v", err)
	}
	if !reflect.DeepEqual(global, all.Global) {
		t.Fatalf("ResolveGlobal != ResolveAll.Global")
	}

	repo, err := ResolveRepo(opts)
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}
	if !reflect.DeepEqual(repo, all.Repo) {
		t.Fatalf("ResolveRepo != ResolveAll.Repo")
	}
}

func findCandidateByStyle(t *testing.T, resolution ScopeResolution, style Style) Candidate {
	t.Helper()

	for _, candidate := range resolution.Candidates {
		if candidate.Style == style {
			return candidate
		}
	}

	t.Fatalf("candidate %q not found", style)
	return Candidate{}
}
