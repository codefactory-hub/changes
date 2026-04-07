package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveManifestPreservesSymbolicLayoutWithoutRewrite(t *testing.T) {
	repoRoot := t.TempDir()
	manifestPath := filepath.Join(repoRoot, ".changes", "config", "layout.toml")
	raw := []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"home\"\n\n[layout]\nroot = \"$REPO_ROOT/.changes\"\nconfig = \"$layout.root/config\"\ndata = \"$layout.root/data\"\nstate = \"$layout.root/state\"\n")
	writeTestFile(t, manifestPath, raw)

	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest before resolve: %v", err)
	}

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}

	home := findCandidateByStyle(t, resolution, StyleHome)
	if home.Status != StatusResolved {
		t.Fatalf("candidate status = %q, want %q", home.Status, StatusResolved)
	}
	if home.Manifest == nil {
		t.Fatalf("candidate manifest is nil")
	}
	if home.Manifest.Symbolic.Root != "$REPO_ROOT/.changes" {
		t.Fatalf("symbolic root = %q", home.Manifest.Symbolic.Root)
	}
	if home.Manifest.Symbolic.Config != "$layout.root/config" {
		t.Fatalf("symbolic config = %q", home.Manifest.Symbolic.Config)
	}
	if home.Manifest.Symbolic.Data != "$layout.root/data" {
		t.Fatalf("symbolic data = %q", home.Manifest.Symbolic.Data)
	}
	if home.Manifest.Symbolic.State != "$layout.root/state" {
		t.Fatalf("symbolic state = %q", home.Manifest.Symbolic.State)
	}
	if home.Manifest.Resolved.Root != filepath.Join(repoRoot, ".changes") {
		t.Fatalf("resolved root = %q", home.Manifest.Resolved.Root)
	}
	if home.Manifest.Resolved.Config != filepath.Join(repoRoot, ".changes", "config") {
		t.Fatalf("resolved config = %q", home.Manifest.Resolved.Config)
	}
	if home.Manifest.Resolved.Data != filepath.Join(repoRoot, ".changes", "data") {
		t.Fatalf("resolved data = %q", home.Manifest.Resolved.Data)
	}
	if home.Manifest.Resolved.State != filepath.Join(repoRoot, ".changes", "state") {
		t.Fatalf("resolved state = %q", home.Manifest.Resolved.State)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after resolve: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("manifest bytes changed during resolve")
	}
}

func TestResolveManifestRejectsUnsupportedKeys(t *testing.T) {
	repoRoot := t.TempDir()
	manifestPath := filepath.Join(repoRoot, ".config", "changes", "layout.toml")
	raw := []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"xdg\"\nextra = \"nope\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/.local/state/changes\"\n")
	writeTestFile(t, manifestPath, raw)

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}

	xdg := findCandidateByStyle(t, resolution, StyleXDG)
	if xdg.Status != StatusInvalid {
		t.Fatalf("candidate status = %q, want %q", xdg.Status, StatusInvalid)
	}
	if resolution.Status != StatusInvalid {
		t.Fatalf("scope status = %q, want %q", resolution.Status, StatusInvalid)
	}
}

func TestResolveManifestRejectsUnsupportedSchemaVersion(t *testing.T) {
	repoRoot := t.TempDir()
	manifestPath := filepath.Join(repoRoot, ".config", "changes", "layout.toml")
	raw := []byte("schema_version = 2\nscope = \"repo\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/.local/state/changes\"\n")
	writeTestFile(t, manifestPath, raw)

	before, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest before resolve: %v", err)
	}

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}

	xdg := findCandidateByStyle(t, resolution, StyleXDG)
	if xdg.Status != StatusInvalid {
		t.Fatalf("candidate status = %q, want %q", xdg.Status, StatusInvalid)
	}
	if resolution.Status != StatusInvalid {
		t.Fatalf("scope status = %q, want %q", resolution.Status, StatusInvalid)
	}

	after, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest after resolve: %v", err)
	}
	if string(after) != string(before) {
		t.Fatalf("manifest bytes changed during resolve")
	}
}

func TestResolveRepoManifestRejectsEscapingPaths(t *testing.T) {
	repoRoot := t.TempDir()
	manifestPath := filepath.Join(repoRoot, ".config", "changes", "layout.toml")
	raw := []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/../escape\"\n")
	writeTestFile(t, manifestPath, raw)

	resolution, err := ResolveRepo(ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}

	xdg := findCandidateByStyle(t, resolution, StyleXDG)
	if xdg.Status != StatusInvalid {
		t.Fatalf("candidate status = %q, want %q", xdg.Status, StatusInvalid)
	}
}

func TestResolveCanonicalizesEquivalentRootsForComparison(t *testing.T) {
	root := t.TempDir()
	homeDir := filepath.Join(root, "home")
	if err := os.MkdirAll(homeDir, 0o755); err != nil {
		t.Fatalf("mkdir home dir: %v", err)
	}

	repoReal := filepath.Join(homeDir, "repo-real")
	if err := os.MkdirAll(repoReal, 0o755); err != nil {
		t.Fatalf("mkdir repo real: %v", err)
	}

	repoLink := filepath.Join(root, "repo-link")
	if err := os.Symlink(repoReal, repoLink); err != nil {
		t.Fatalf("symlink repo root: %v", err)
	}

	manifestPath := filepath.Join(repoLink, ".changes", "config", "layout.toml")
	raw := []byte("schema_version = 1\nscope = \"repo\"\nstyle = \"home\"\n\n[layout]\nroot = \"$HOME/repo-real/.changes\"\nconfig = \"$layout.root/config\"\ndata = \"$layout.root/data\"\nstate = \"$layout.root/state\"\n")
	writeTestFile(t, manifestPath, raw)

	resolution, err := ResolveRepo(ResolveOptions{
		RepoRoot: repoLink,
		HomeDir:  homeDir,
	})
	if err != nil {
		t.Fatalf("ResolveRepo returned error: %v", err)
	}

	home := findCandidateByStyle(t, resolution, StyleHome)
	if home.Status != StatusResolved {
		t.Fatalf("candidate status = %q, want %q", home.Status, StatusResolved)
	}
	if home.Manifest == nil {
		t.Fatalf("candidate manifest is nil")
	}
	if home.Manifest.Symbolic.Root != "$HOME/repo-real/.changes" {
		t.Fatalf("symbolic root = %q", home.Manifest.Symbolic.Root)
	}
	if home.Manifest.Resolved.Root != filepath.Join(repoReal, ".changes") {
		t.Fatalf("resolved root = %q", home.Manifest.Resolved.Root)
	}
	if home.Paths.Root != filepath.Join(repoLink, ".changes") {
		t.Fatalf("candidate root = %q", home.Paths.Root)
	}
}

func writeTestFile(t *testing.T, path string, contents []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
