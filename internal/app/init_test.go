package app

import (
	"bytes"
	"context"
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
