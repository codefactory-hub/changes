package config

import (
	"path/filepath"
	"testing"
)

func TestSelectRepoInitLayoutDefaultsToXDG(t *testing.T) {
	repoRoot := t.TempDir()

	selection, err := SelectRepoInitLayout(RepoInitSelectionOptions{
		RepoRoot: repoRoot,
	})
	if err != nil {
		t.Fatalf("SelectRepoInitLayout returned error: %v", err)
	}
	if selection.Style != StyleXDG {
		t.Fatalf("style = %q, want %q", selection.Style, StyleXDG)
	}
	if selection.Root != repoRoot {
		t.Fatalf("root = %q, want %q", selection.Root, repoRoot)
	}
	if selection.Config != filepath.Join(repoRoot, ".config", "changes") {
		t.Fatalf("config = %q", selection.Config)
	}
	if selection.Data != filepath.Join(repoRoot, ".local", "share", "changes") {
		t.Fatalf("data = %q", selection.Data)
	}
	if selection.State != filepath.Join(repoRoot, ".local", "state", "changes") {
		t.Fatalf("state = %q", selection.State)
	}
	if selection.GitignoreEntry != "/.local/state/" {
		t.Fatalf("gitignore entry = %q", selection.GitignoreEntry)
	}
}

func TestSelectRepoInitLayoutUsesRepoInitHomeDefault(t *testing.T) {
	repoRoot := t.TempDir()

	selection, err := SelectRepoInitLayout(RepoInitSelectionOptions{
		RepoRoot:        repoRoot,
		GlobalInitStyle: "home",
	})
	if err != nil {
		t.Fatalf("SelectRepoInitLayout returned error: %v", err)
	}
	if selection.Style != StyleHome {
		t.Fatalf("style = %q, want %q", selection.Style, StyleHome)
	}
	if selection.Root != filepath.Join(repoRoot, ".changes") {
		t.Fatalf("root = %q", selection.Root)
	}
	if selection.Config != filepath.Join(repoRoot, ".changes", "config") {
		t.Fatalf("config = %q", selection.Config)
	}
	if selection.Data != filepath.Join(repoRoot, ".changes", "data") {
		t.Fatalf("data = %q", selection.Data)
	}
	if selection.State != filepath.Join(repoRoot, ".changes", "state") {
		t.Fatalf("state = %q", selection.State)
	}
	if selection.GitignoreEntry != "/.changes/state/" {
		t.Fatalf("gitignore entry = %q", selection.GitignoreEntry)
	}
}

func TestSelectRepoInitLayoutPrefersChangesHomeSignalOverXDGSignal(t *testing.T) {
	repoRoot := t.TempDir()

	selection, err := SelectRepoInitLayout(RepoInitSelectionOptions{
		RepoRoot:      repoRoot,
		ChangesHome:   filepath.Join(t.TempDir(), "changes-home"),
		XDGConfigHome: filepath.Join(t.TempDir(), "xdg-config"),
		XDGDataHome:   filepath.Join(t.TempDir(), "xdg-data"),
		XDGStateHome:  filepath.Join(t.TempDir(), "xdg-state"),
	})
	if err != nil {
		t.Fatalf("SelectRepoInitLayout returned error: %v", err)
	}
	if selection.Style != StyleHome {
		t.Fatalf("style = %q, want %q", selection.Style, StyleHome)
	}
	if selection.Root != filepath.Join(repoRoot, ".changes") {
		t.Fatalf("root = %q", selection.Root)
	}
	if selection.GitignoreEntry != "/.changes/state/" {
		t.Fatalf("gitignore entry = %q", selection.GitignoreEntry)
	}
}
