package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAcceptsStableAndUnstablePublicAPI(t *testing.T) {
	repoRoot := t.TempDir()
	path := RepoConfigPath(repoRoot)

	for _, value := range []string{"stable", "unstable"} {
		cfg := Default()
		cfg.Project.Name = "changes"
		cfg.Versioning.PublicAPI = value
		if err := Write(path, cfg); err != nil {
			t.Fatalf("write config: %v", err)
		}

		loaded, err := Load(repoRoot)
		if err != nil {
			t.Fatalf("load config for %s: %v", value, err)
		}
		if loaded.Versioning.PublicAPI != value {
			t.Fatalf("public_api = %q, want %q", loaded.Versioning.PublicAPI, value)
		}
	}
}

func TestLoadRejectsInvalidPublicAPI(t *testing.T) {
	repoRoot := t.TempDir()
	path := RepoConfigPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	raw := []byte("[project]\nname = \"changes\"\nchangelog_file = \"CHANGELOG.md\"\ninitial_version = \"0.1.0\"\n\n[paths]\ndata_dir = \".local/share/changes\"\nstate_dir = \".local/state/changes\"\ntemplates_dir = \".local/share/changes/templates\"\n\n[versioning]\npublic_api = \"managed\"\n")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(repoRoot)
	if err == nil || !strings.Contains(err.Error(), "versioning.public_api must be one of stable, unstable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsLegacyPrereleaseLabel(t *testing.T) {
	repoRoot := t.TempDir()
	path := RepoConfigPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	raw := []byte("[project]\nname = \"changes\"\nchangelog_file = \"CHANGELOG.md\"\ninitial_version = \"0.1.0\"\n\n[paths]\ndata_dir = \".local/share/changes\"\nstate_dir = \".local/state/changes\"\ntemplates_dir = \".local/share/changes/templates\"\n\n[versioning]\npublic_api = \"unstable\"\nprerelease_label = \"rc\"\n")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(repoRoot)
	if err == nil || !strings.Contains(err.Error(), "unsupported keys: versioning.prerelease_label") {
		t.Fatalf("unexpected error: %v", err)
	}
}
