package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/releases"
)

func TestAppEndToEnd(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}

	assertExists(t, filepath.Join(repoRoot, ".config/changes/config.toml"))
	assertExists(t, filepath.Join(repoRoot, ".local/share/changes/templates/release.md.tmpl"))

	gitignore, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), "/.local/state/changes/") {
		t.Fatalf(".gitignore missing state dir entry: %s", gitignore)
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{0, 1, 2, 3})
	if err := app.Run(context.Background(), []string{
		"add",
		"--title", "Fix release note rendering",
		"--type", "fixed",
		"--bump", "patch",
		"--scope", "release",
		"--body", "Render whole entries only.",
	}); err != nil {
		t.Fatalf("add returned error: %v\nstderr=%s", err, stderr.String())
	}

	fragmentPath := strings.TrimSpace(stdout.String())
	assertExists(t, fragmentPath)
	if !regexp.MustCompile(`20260402T153045Z--fix-release-note-rendering--[a-z0-9]{4}\.md$`).MatchString(fragmentPath) {
		t.Fatalf("fragment path %q did not match expected pattern", fragmentPath)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	status := stdout.String()
	if !strings.Contains(status, "Unreleased fragments: 1") || !strings.Contains(status, "Recommended next stable: 0.1.0") {
		t.Fatalf("unexpected status output:\n%s", status)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"version", "next", "--pre", "rc"}); err != nil {
		t.Fatalf("version next returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "0.1.0-rc.1" {
		t.Fatalf("version next --pre rc = %q, want 0.1.0-rc.1", got)
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{"release", "create", "--channel", "preview", "--pre", "rc"}); err != nil {
		t.Fatalf("release create preview returned error: %v", err)
	}
	previewPath := strings.TrimSpace(stdout.String())
	previewManifest, err := releases.Load(previewPath)
	if err != nil {
		t.Fatalf("load preview manifest: %v", err)
	}
	if previewManifest.Consumes {
		t.Fatalf("preview manifest should not consume fragments")
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"render", "--version", "0.1.0-rc.1"}); err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Fix release note rendering") {
		t.Fatalf("render output missing fragment title:\n%s", stdout.String())
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 17, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{"release", "create", "--channel", "stable"}); err != nil {
		t.Fatalf("release create stable returned error: %v", err)
	}
	stablePath := strings.TrimSpace(stdout.String())
	stableManifest, err := releases.Load(stablePath)
	if err != nil {
		t.Fatalf("load stable manifest: %v", err)
	}
	if !stableManifest.Consumes {
		t.Fatalf("stable manifest should consume fragments")
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"changelog", "rebuild"}); err != nil {
		t.Fatalf("changelog rebuild returned error: %v", err)
	}
	changelogPath := strings.TrimSpace(stdout.String())
	changelogBytes, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("read changelog: %v", err)
	}
	if !strings.Contains(string(changelogBytes), "## 0.1.0 (stable)") {
		t.Fatalf("changelog missing release heading:\n%s", changelogBytes)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status after stable returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Unreleased fragments: 0") {
		t.Fatalf("unexpected post-release status:\n%s", stdout.String())
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}
