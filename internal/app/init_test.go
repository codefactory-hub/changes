package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
)

func TestInitializeReturnsSelectedLayoutAndPaths(t *testing.T) {
	repoRoot := t.TempDir()

	result, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:        repoRoot,
		RequestedLayout: "home",
		Now:             time.Date(2026, 4, 7, 3, 0, 0, 0, time.UTC),
		Random:          bytes.NewReader([]byte{1, 2, 3, 4}),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if result.SelectedLayout != config.StyleHome {
		t.Fatalf("SelectedLayout = %q, want %q", result.SelectedLayout, config.StyleHome)
	}
	if result.ConfigPath != filepath.Join(repoRoot, ".changes", "config") {
		t.Fatalf("ConfigPath = %q", result.ConfigPath)
	}
	if result.DataPath != filepath.Join(repoRoot, ".changes", "data") {
		t.Fatalf("DataPath = %q", result.DataPath)
	}
	if result.StatePath != filepath.Join(repoRoot, ".changes", "state") {
		t.Fatalf("StatePath = %q", result.StatePath)
	}
}

func TestInitializeReportsGitignoreChangeOnlyWhenModified(t *testing.T) {
	repoRoot := t.TempDir()

	first, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:        repoRoot,
		RequestedLayout: "home",
		Now:             time.Date(2026, 4, 7, 3, 5, 0, 0, time.UTC),
		Random:          bytes.NewReader([]byte{5, 6, 7, 8}),
	})
	if err != nil {
		t.Fatalf("first Initialize returned error: %v", err)
	}
	if !first.GitignoreUpdated {
		t.Fatalf("first GitignoreUpdated = false, want true")
	}

	raw, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore after first init: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "/.changes/state/") {
		t.Fatalf(".gitignore = %q, want home state entry", body)
	}
	if strings.Contains(body, "/.local/state/") {
		t.Fatalf(".gitignore = %q, should not include xdg state entry", body)
	}

	second, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:        repoRoot,
		RequestedLayout: "home",
		Now:             time.Date(2026, 4, 7, 3, 6, 0, 0, time.UTC),
		Random:          bytes.NewReader([]byte{9, 10, 11, 12}),
	})
	if err != nil {
		t.Fatalf("second Initialize returned error: %v", err)
	}
	if second.GitignoreUpdated {
		t.Fatalf("second GitignoreUpdated = true, want false")
	}
}

func TestInitializeGlobalCreatesSelectedLayout(t *testing.T) {
	_, _, xdgConfigHome, xdgDataHome, xdgStateHome := configureGlobalLayoutEnv(t)

	xdgResult, err := InitializeGlobal(context.Background(), InitializeGlobalRequest{
		RequestedLayout: "xdg",
		Now:             time.Date(2026, 4, 7, 3, 10, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("InitializeGlobal xdg returned error: %v", err)
	}
	if xdgResult.SelectedLayout != config.StyleXDG {
		t.Fatalf("xdg SelectedLayout = %q, want %q", xdgResult.SelectedLayout, config.StyleXDG)
	}
	for _, path := range []string{
		filepath.Join(xdgConfigHome, "changes", "layout.toml"),
		filepath.Join(xdgDataHome, "changes"),
		filepath.Join(xdgStateHome, "changes"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("xdg path %s missing: %v", path, err)
		}
	}

	homeDir, _, _, _, _ := configureGlobalLayoutEnv(t)
	customHome := filepath.Join(homeDir, ".changes-global")
	homeResult, err := InitializeGlobal(context.Background(), InitializeGlobalRequest{
		RequestedLayout: "home",
		RequestedHome:   customHome,
		Now:             time.Date(2026, 4, 7, 3, 11, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("InitializeGlobal home returned error: %v", err)
	}
	if homeResult.SelectedLayout != config.StyleHome {
		t.Fatalf("home SelectedLayout = %q, want %q", homeResult.SelectedLayout, config.StyleHome)
	}
	for _, path := range []string{
		filepath.Join(customHome, "config", "layout.toml"),
		filepath.Join(customHome, "data"),
		filepath.Join(customHome, "state"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("home path %s missing: %v", path, err)
		}
	}
}

func TestInitializeDefaultsToRepoXDGWithoutOverrides(t *testing.T) {
	t.Setenv("CHANGES_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	repoRoot := t.TempDir()

	result, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      time.Date(2026, 4, 7, 3, 20, 0, 0, time.UTC),
		Random:   bytes.NewReader([]byte{1, 3, 5, 7}),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if result.SelectedLayout != config.StyleXDG {
		t.Fatalf("SelectedLayout = %q, want %q", result.SelectedLayout, config.StyleXDG)
	}
	if result.ConfigPath != filepath.Join(repoRoot, ".config", "changes") {
		t.Fatalf("ConfigPath = %q", result.ConfigPath)
	}
	if result.DataPath != filepath.Join(repoRoot, ".local", "share", "changes") {
		t.Fatalf("DataPath = %q", result.DataPath)
	}
	if result.StatePath != filepath.Join(repoRoot, ".local", "state", "changes") {
		t.Fatalf("StatePath = %q", result.StatePath)
	}
}

func TestInitializeGlobalDefaultsToXDGWithoutOverrides(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("CHANGES_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	result, err := InitializeGlobal(context.Background(), InitializeGlobalRequest{
		Now: time.Date(2026, 4, 7, 3, 25, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("InitializeGlobal returned error: %v", err)
	}

	if result.SelectedLayout != config.StyleXDG {
		t.Fatalf("SelectedLayout = %q, want %q", result.SelectedLayout, config.StyleXDG)
	}
	if result.ConfigPath != filepath.Join(homeDir, ".config", "changes") {
		t.Fatalf("ConfigPath = %q", result.ConfigPath)
	}
	if result.DataPath != filepath.Join(homeDir, ".local", "share", "changes") {
		t.Fatalf("DataPath = %q", result.DataPath)
	}
	if result.StatePath != filepath.Join(homeDir, ".local", "state", "changes") {
		t.Fatalf("StatePath = %q", result.StatePath)
	}
}

func TestManifestBackedXDGRepoOperatesWithoutMigration(t *testing.T) {
	t.Setenv("CHANGES_HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")

	repoRoot := t.TempDir()
	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:        repoRoot,
		RequestedLayout: "xdg",
		Now:             time.Date(2026, 4, 7, 3, 30, 0, 0, time.UTC),
		Random:          bytes.NewReader([]byte{2, 4, 6, 8}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	result, err := Status(context.Background(), StatusRequest{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if result.CurrentVersionLabel != "unreleased" {
		t.Fatalf("CurrentVersionLabel = %q, want unreleased", result.CurrentVersionLabel)
	}
}

func TestManifestBackedHomeRepoOperatesWithoutMigration(t *testing.T) {
	repoRoot := t.TempDir()
	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:        repoRoot,
		RequestedLayout: "home",
		Now:             time.Date(2026, 4, 7, 3, 35, 0, 0, time.UTC),
		Random:          bytes.NewReader([]byte{9, 7, 5, 3}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	result, err := Status(context.Background(), StatusRequest{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if result.CurrentVersionLabel != "unreleased" {
		t.Fatalf("CurrentVersionLabel = %q, want unreleased", result.CurrentVersionLabel)
	}
}

func TestLegacyRepoFailsCleanlyForOrdinaryCommands(t *testing.T) {
	repoRoot := t.TempDir()
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write legacy xdg config: %v", err)
	}

	_, err := Status(context.Background(), StatusRequest{RepoRoot: repoRoot})
	if err == nil {
		t.Fatalf("Status returned nil error")
	}

	var authorityErr *config.AuthorityError
	if !errors.As(err, &authorityErr) {
		t.Fatalf("Status error = %v, want AuthorityError", err)
	}
	if authorityErr.Scope != config.ScopeRepo || authorityErr.Status != config.StatusLegacyOnly {
		t.Fatalf("authority error = %#v, want repo legacy-only", authorityErr)
	}
}
