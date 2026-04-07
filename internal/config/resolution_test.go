package config

import (
	"os"
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

func TestResolveRepoAmbiguousWhenDistinctOperationalCandidatesCompete(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, ".config", "changes", "layout.toml"), []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/.local/state/changes\"\n"))
	writeTestFile(t, filepath.Join(repoRoot, ".changes", "config", "layout.toml"), []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"home\"\n\n[layout]\nroot = \"$REPO_ROOT/.changes\"\nconfig = \"$layout.root/config\"\ndata = \"$layout.root/data\"\nstate = \"$layout.root/state\"\n"))

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}
	if resolution.Status != StatusAmbiguous {
		t.Fatalf("status = %q, want %q", resolution.Status, StatusAmbiguous)
	}
	if resolution.Authoritative != nil {
		t.Fatalf("Authoritative = %#v, want nil", resolution.Authoritative)
	}
}

func TestResolveRepoAllowsSingleOperationalCandidateWithLegacySiblingWarning(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, ".config", "changes", "layout.toml"), []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/.local/state/changes\"\n"))
	writeTestFile(t, filepath.Join(repoRoot, ".changes", "config", "config.toml"), []byte("[project]\nname = \"legacy\"\n"))

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}
	if resolution.Status != StatusResolved {
		t.Fatalf("status = %q, want %q", resolution.Status, StatusResolved)
	}
	if resolution.Authoritative == nil {
		t.Fatalf("Authoritative candidate is nil")
	}
	if resolution.Authoritative.Style != StyleXDG {
		t.Fatalf("authoritative style = %q, want %q", resolution.Authoritative.Style, StyleXDG)
	}
	if len(resolution.Warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(resolution.Warnings))
	}
	warning := resolution.Warnings[0]
	if warning.Scope != ScopeRepo {
		t.Fatalf("warning scope = %q, want %q", warning.Scope, ScopeRepo)
	}
	if warning.Style != StyleHome {
		t.Fatalf("warning style = %q, want %q", warning.Style, StyleHome)
	}
	if warning.Status != StatusLegacyOnly {
		t.Fatalf("warning status = %q, want %q", warning.Status, StatusLegacyOnly)
	}
	if warning.Path != filepath.Join(repoRoot, ".changes", "config") {
		t.Fatalf("warning path = %q", warning.Path)
	}
}

func TestResolveRepoAllowsSingleOperationalCandidateWithInvalidSiblingWarning(t *testing.T) {
	repoRoot := t.TempDir()
	writeTestFile(t, filepath.Join(repoRoot, ".config", "changes", "layout.toml"), []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/.local/state/changes\"\n"))
	writeTestFile(t, filepath.Join(repoRoot, ".changes", "config", "layout.toml"), []byte("schema_version = 99\nscope = \"repo\"\nstyle = \"home\"\n\n[layout]\nroot = \"$REPO_ROOT/.changes\"\nconfig = \"$layout.root/config\"\ndata = \"$layout.root/data\"\nstate = \"$layout.root/state\"\n"))

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}
	if resolution.Status != StatusResolved {
		t.Fatalf("status = %q, want %q", resolution.Status, StatusResolved)
	}
	if resolution.Authoritative == nil {
		t.Fatalf("Authoritative candidate is nil")
	}
	if resolution.Authoritative.Style != StyleXDG {
		t.Fatalf("authoritative style = %q, want %q", resolution.Authoritative.Style, StyleXDG)
	}
	if len(resolution.Warnings) != 1 {
		t.Fatalf("warnings = %d, want 1", len(resolution.Warnings))
	}
	warning := resolution.Warnings[0]
	if warning.Scope != ScopeRepo {
		t.Fatalf("warning scope = %q, want %q", warning.Scope, ScopeRepo)
	}
	if warning.Style != StyleHome {
		t.Fatalf("warning style = %q, want %q", warning.Style, StyleHome)
	}
	if warning.Status != StatusInvalid {
		t.Fatalf("warning status = %q, want %q", warning.Status, StatusInvalid)
	}
	if warning.Path != filepath.Join(repoRoot, ".changes", "config") {
		t.Fatalf("warning path = %q", warning.Path)
	}
}

func TestResolveRepoCollapsesEquivalentOperationalCandidates(t *testing.T) {
	root := t.TempDir()
	repoReal := filepath.Join(root, "repo-real")
	if err := os.MkdirAll(repoReal, 0o755); err != nil {
		t.Fatalf("mkdir repo real: %v", err)
	}
	repoAlias := filepath.Join(root, "repo-alias")
	if err := os.Symlink(repoReal, repoAlias); err != nil {
		t.Fatalf("symlink repo alias: %v", err)
	}

	xdgPaths := repoPaths(StyleXDG, ResolveOptions{RepoRoot: repoReal})
	homePaths := repoPaths(StyleHome, ResolveOptions{RepoRoot: repoAlias})

	resolution, err := resolveScopeFromCandidates(ScopeRepo, ResolveOptions{RepoRoot: repoReal}, []Candidate{
		newResolvedCandidateForTest(StyleXDG, xdgPaths),
		newResolvedCandidateForTest(StyleHome, homePaths),
	})
	if err != nil {
		t.Fatalf("resolveScopeFromCandidates returned error: %v", err)
	}
	if resolution.Status != StatusResolved {
		t.Fatalf("status = %q, want %q", resolution.Status, StatusResolved)
	}
	if resolution.Authoritative == nil {
		t.Fatalf("Authoritative candidate is nil")
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

func newResolvedCandidateForTest(style Style, paths LayoutPaths) Candidate {
	return Candidate{
		Scope:  ScopeRepo,
		Style:  style,
		Status: StatusResolved,
		Paths:  paths,
		Manifest: &LayoutManifest{
			SchemaVersion: 1,
			Scope:         ScopeRepo,
			Style:         style,
			Resolved:      paths,
		},
	}
}
