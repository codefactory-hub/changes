package templates

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/render"
)

func TestEnsureDefaultFilesCreatesBuiltinsAndIsIdempotent(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	first, err := EnsureDefaultFiles(repoRoot, cfg)
	if err != nil {
		t.Fatalf("EnsureDefaultFiles returned error: %v", err)
	}

	builtins := render.BuiltinTemplateFiles()
	if len(first.Paths) != len(builtins) {
		t.Fatalf("created %d paths, want %d", len(first.Paths), len(builtins))
	}
	if len(first.CreatedPaths) != len(builtins) {
		t.Fatalf("created %d new paths, want %d", len(first.CreatedPaths), len(builtins))
	}

	for _, path := range first.Paths {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stat %s: %v", path, err)
		}
	}

	second, err := EnsureDefaultFiles(repoRoot, cfg)
	if err != nil {
		t.Fatalf("second EnsureDefaultFiles returned error: %v", err)
	}
	if len(second.CreatedPaths) != 0 {
		t.Fatalf("second EnsureDefaultFiles created %d files, want 0", len(second.CreatedPaths))
	}
}

func TestWriteIfMissingPreservesExistingContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "template.tmpl")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("write existing file: %v", err)
	}

	written, err := writeIfMissing(path, "replacement")
	if err != nil {
		t.Fatalf("writeIfMissing returned error: %v", err)
	}
	if written {
		t.Fatalf("writeIfMissing reported written=true for existing file")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if string(raw) != "original" {
		t.Fatalf("file contents = %q, want %q", string(raw), "original")
	}
}
