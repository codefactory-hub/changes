package reporoot

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDetectFindsNearestGitDirectory(t *testing.T) {
	repoRoot := t.TempDir()
	nested := filepath.Join(repoRoot, "a", "b", "c")
	if err := os.MkdirAll(filepath.Join(repoRoot, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, err := Detect(nested)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if got != repoRoot {
		t.Fatalf("Detect = %q, want %q", got, repoRoot)
	}
}

func TestDetectAcceptsGitFileWorktreeLayout(t *testing.T) {
	repoRoot := t.TempDir()
	nested := filepath.Join(repoRoot, "pkg")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".git"), []byte("gitdir: /tmp/worktree\n"), 0o644); err != nil {
		t.Fatalf("write .git file: %v", err)
	}

	got, err := Detect(nested)
	if err != nil {
		t.Fatalf("Detect returned error: %v", err)
	}
	if got != repoRoot {
		t.Fatalf("Detect = %q, want %q", got, repoRoot)
	}
}

func TestDetectRejectsEmptyPathAndMissingRepository(t *testing.T) {
	if _, err := Detect(""); err == nil {
		t.Fatalf("Detect(empty) returned nil error")
	}

	start := t.TempDir()
	if _, err := Detect(start); !errors.Is(err, ErrNotGitRepo) {
		t.Fatalf("Detect(non-repo) error = %v, want %v", err, ErrNotGitRepo)
	}
}
