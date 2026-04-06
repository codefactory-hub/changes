package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestLoadAcceptsStableAndUnstablePublicAPI(t *testing.T) {
	repoRoot := t.TempDir()

	for _, value := range []string{"stable", "unstable"} {
		cfg := Default()
		cfg.Project.Name = "changes"
		cfg.Versioning.PublicAPI = value
		if err := writeManagedRepoConfig(t, repoRoot, StyleXDG, cfg); err != nil {
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
	raw := []byte("[project]\nname = \"changes\"\nchangelog_file = \"CHANGELOG.md\"\ninitial_version = \"0.1.0\"\n\n[paths]\ndata_dir = \".local/share/changes\"\nstate_dir = \".local/state/changes\"\ntemplates_dir = \".local/share/changes/templates\"\n\n[versioning]\npublic_api = \"managed\"\n")
	if err := writeManagedRepoConfigBytes(t, repoRoot, StyleXDG, raw); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(repoRoot)
	if err == nil || !strings.Contains(err.Error(), "versioning.public_api must be one of stable, unstable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadRejectsLegacyPrereleaseLabel(t *testing.T) {
	repoRoot := t.TempDir()
	raw := []byte("[project]\nname = \"changes\"\nchangelog_file = \"CHANGELOG.md\"\ninitial_version = \"0.1.0\"\n\n[paths]\ndata_dir = \".local/share/changes\"\nstate_dir = \".local/state/changes\"\ntemplates_dir = \".local/share/changes/templates\"\n\n[versioning]\npublic_api = \"unstable\"\nprerelease_label = \"rc\"\n")
	if err := writeManagedRepoConfigBytes(t, repoRoot, StyleXDG, raw); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(repoRoot)
	if err == nil || !strings.Contains(err.Error(), "unsupported keys: versioning.prerelease_label") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPathHelpersResolveConfiguredLocations(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := Default()
	cfg.Project.ChangelogFile = "docs/CHANGELOG.md"
	cfg.Paths.DataDir = ".data/changes"
	cfg.Paths.StateDir = ".state/changes"
	cfg.Paths.TemplatesDir = ".templates/changes"

	if got := RepoConfigPath(repoRoot); got != filepath.Join(repoRoot, ".config/changes/config.toml") {
		t.Fatalf("RepoConfigPath = %q", got)
	}
	if got := UserConfigPath("/home/tester"); got != filepath.Join("/home/tester", ".config/changes/config.toml") {
		t.Fatalf("UserConfigPath = %q", got)
	}
	if got := FragmentsDir(repoRoot, cfg); got != filepath.Join(repoRoot, ".data/changes/fragments") {
		t.Fatalf("FragmentsDir = %q", got)
	}
	if got := ReleasesDir(repoRoot, cfg); got != filepath.Join(repoRoot, ".data/changes/releases") {
		t.Fatalf("ReleasesDir = %q", got)
	}
	if got := TemplatesDir(repoRoot, cfg); got != filepath.Join(repoRoot, ".templates/changes") {
		t.Fatalf("TemplatesDir = %q", got)
	}
	if got := PromptsDir(repoRoot, cfg); got != filepath.Join(repoRoot, ".data/changes/prompts") {
		t.Fatalf("PromptsDir = %q", got)
	}
	if got := HistoryImportPromptPath(repoRoot, cfg); got != filepath.Join(repoRoot, ".data/changes/prompts/release-history-import-llm-prompt.md") {
		t.Fatalf("HistoryImportPromptPath = %q", got)
	}
	if got := StateDir(repoRoot, cfg); got != filepath.Join(repoRoot, ".state/changes") {
		t.Fatalf("StateDir = %q", got)
	}
	if got := ChangelogPath(repoRoot, cfg); got != filepath.Join(repoRoot, "docs/CHANGELOG.md") {
		t.Fatalf("ChangelogPath = %q", got)
	}
}

func TestLoadUsesResolverBackedRepoConfigPath(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := Default()
	cfg.Project.Name = "resolver-home"
	if err := writeManagedRepoConfig(t, repoRoot, StyleHome, cfg); err != nil {
		t.Fatalf("write managed config: %v", err)
	}

	loaded, err := Load(repoRoot)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Project.Name != "resolver-home" {
		t.Fatalf("project name = %q, want resolver-home", loaded.Project.Name)
	}
}

func TestPathHelpersUseResolverAuthoritativePaths(t *testing.T) {
	xdgRoot := t.TempDir()
	homeRoot := t.TempDir()
	cfg := Default()

	if err := writeRepoLayoutManifest(t, xdgRoot, StyleXDG); err != nil {
		t.Fatalf("write xdg manifest: %v", err)
	}
	if err := writeRepoLayoutManifest(t, homeRoot, StyleHome); err != nil {
		t.Fatalf("write home manifest: %v", err)
	}

	if got := RepoConfigPath(xdgRoot); got != filepath.Join(xdgRoot, ".config", "changes", "config.toml") {
		t.Fatalf("RepoConfigPath xdg = %q", got)
	}
	if got := FragmentsDir(xdgRoot, cfg); got != filepath.Join(xdgRoot, ".local", "share", "changes", "fragments") {
		t.Fatalf("FragmentsDir xdg = %q", got)
	}
	if got := ReleasesDir(xdgRoot, cfg); got != filepath.Join(xdgRoot, ".local", "share", "changes", "releases") {
		t.Fatalf("ReleasesDir xdg = %q", got)
	}
	if got := TemplatesDir(xdgRoot, cfg); got != filepath.Join(xdgRoot, ".local", "share", "changes", "templates") {
		t.Fatalf("TemplatesDir xdg = %q", got)
	}
	if got := PromptsDir(xdgRoot, cfg); got != filepath.Join(xdgRoot, ".local", "share", "changes", "prompts") {
		t.Fatalf("PromptsDir xdg = %q", got)
	}
	if got := HistoryImportPromptPath(xdgRoot, cfg); got != filepath.Join(xdgRoot, ".local", "share", "changes", "prompts", "release-history-import-llm-prompt.md") {
		t.Fatalf("HistoryImportPromptPath xdg = %q", got)
	}
	if got := StateDir(xdgRoot, cfg); got != filepath.Join(xdgRoot, ".local", "state", "changes") {
		t.Fatalf("StateDir xdg = %q", got)
	}

	if got := RepoConfigPath(homeRoot); got != filepath.Join(homeRoot, ".changes", "config", "config.toml") {
		t.Fatalf("RepoConfigPath home = %q", got)
	}
	if got := FragmentsDir(homeRoot, cfg); got != filepath.Join(homeRoot, ".changes", "data", "fragments") {
		t.Fatalf("FragmentsDir home = %q", got)
	}
	if got := ReleasesDir(homeRoot, cfg); got != filepath.Join(homeRoot, ".changes", "data", "releases") {
		t.Fatalf("ReleasesDir home = %q", got)
	}
	if got := TemplatesDir(homeRoot, cfg); got != filepath.Join(homeRoot, ".changes", "data", "templates") {
		t.Fatalf("TemplatesDir home = %q", got)
	}
	if got := PromptsDir(homeRoot, cfg); got != filepath.Join(homeRoot, ".changes", "data", "prompts") {
		t.Fatalf("PromptsDir home = %q", got)
	}
	if got := HistoryImportPromptPath(homeRoot, cfg); got != filepath.Join(homeRoot, ".changes", "data", "prompts", "release-history-import-llm-prompt.md") {
		t.Fatalf("HistoryImportPromptPath home = %q", got)
	}
	if got := StateDir(homeRoot, cfg); got != filepath.Join(homeRoot, ".changes", "state") {
		t.Fatalf("StateDir home = %q", got)
	}
}

func TestLoadReturnsInitHintForUninitializedRepoLayout(t *testing.T) {
	repoRoot := t.TempDir()

	_, err := Load(repoRoot)
	if err == nil {
		t.Fatalf("Load returned nil error")
	}
	if !strings.Contains(err.Error(), "run `changes init` first") {
		t.Fatalf("Load error = %v, want init hint", err)
	}
}

func TestLoadAppliesDefaultVersioningWhenMissing(t *testing.T) {
	repoRoot := t.TempDir()
	raw := []byte("[project]\nname = \"changes\"\nchangelog_file = \"CHANGELOG.md\"\ninitial_version = \"0.1.0\"\n")
	if err := writeManagedRepoConfigBytes(t, repoRoot, StyleXDG, raw); err != nil {
		t.Fatalf("write config: %v", err)
	}

	loaded, err := Load(repoRoot)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if loaded.Versioning.PublicAPI != "unstable" {
		t.Fatalf("public_api = %q, want unstable", loaded.Versioning.PublicAPI)
	}
}

func TestRenderProfileUnmarshalTracksPresenceForEmptyValues(t *testing.T) {
	var profile RenderProfile
	if err := profile.UnmarshalTOML(map[string]any{
		"description":      "",
		"mode":             "",
		"document_header":  "",
		"release_template": "",
		"entry_template":   "",
		"max_chars":        int64(0),
		"omission_notice":  "",
		"metadata":         map[string]any{},
	}); err != nil {
		t.Fatalf("UnmarshalTOML returned error: %v", err)
	}

	if !profile.HasDescription() || !profile.HasMode() || !profile.HasDocumentHeader() ||
		!profile.HasReleaseTemplate() || !profile.HasEntryTemplate() ||
		!profile.HasMaxChars() || !profile.HasOmissionNotice() || !profile.HasMetadata() {
		t.Fatalf("presence helpers should report explicit empty values as present: %#v", profile)
	}
}

func TestRenderProfileUnmarshalRejectsUnsupportedOrWrongTypes(t *testing.T) {
	var profile RenderProfile
	if err := profile.UnmarshalTOML(map[string]any{"unknown": "value"}); err == nil {
		t.Fatalf("UnmarshalTOML unsupported key returned nil error")
	}
	if err := profile.UnmarshalTOML(map[string]any{"max_chars": "nope"}); err == nil {
		t.Fatalf("UnmarshalTOML invalid max_chars returned nil error")
	}
	if err := profile.UnmarshalTOML(map[string]any{"metadata": map[string]any{"channel": 5}}); err == nil {
		t.Fatalf("UnmarshalTOML invalid metadata returned nil error")
	}
}

func TestWriteCreatesParentDirectoriesAndRoundTripsTOML(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "config.toml")
	cfg := Default()
	cfg.Project.Name = "changes"
	cfg.RenderProfiles = map[string]RenderProfile{
		"custom": {
			Description: "Custom profile",
			Mode:        RenderModeSingleRelease,
			MaxChars:    120,
			Metadata:    map[string]string{"channel": "notes"},
		},
	}

	if err := Write(path, cfg); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	var decoded Config
	if _, err := toml.DecodeFile(path, &decoded); err != nil {
		t.Fatalf("DecodeFile returned error: %v", err)
	}
	if decoded.Project.Name != "changes" {
		t.Fatalf("decoded project name = %q, want changes", decoded.Project.Name)
	}
	if decoded.RenderProfiles["custom"].Metadata["channel"] != "notes" {
		t.Fatalf("decoded metadata = %#v", decoded.RenderProfiles["custom"].Metadata)
	}
}

func writeManagedRepoConfig(t *testing.T, repoRoot string, style Style, cfg Config) error {
	t.Helper()
	path := repoConfigPathForStyle(repoRoot, style)
	if err := writeRepoLayoutManifest(t, repoRoot, style); err != nil {
		return err
	}
	return Write(path, cfg)
}

func writeManagedRepoConfigBytes(t *testing.T, repoRoot string, style Style, raw []byte) error {
	t.Helper()
	path := repoConfigPathForStyle(repoRoot, style)
	if err := writeRepoLayoutManifest(t, repoRoot, style); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

func writeRepoLayoutManifest(t *testing.T, repoRoot string, style Style) error {
	t.Helper()
	path := repoLayoutManifestPath(repoRoot, style)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var raw string
	switch style {
	case StyleHome:
		raw = "schema_version = 1\nscope = \"repo\"\nstyle = \"home\"\n\n[layout]\nroot = \"$REPO_ROOT/.changes\"\nconfig = \"$layout.root/config\"\ndata = \"$layout.root/data\"\nstate = \"$layout.root/state\"\n"
	case StyleXDG:
		raw = "schema_version = 1\nscope = \"repo\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/.local/state/changes\"\n"
	default:
		return os.ErrInvalid
	}

	return os.WriteFile(path, []byte(raw), 0o644)
}

func repoLayoutManifestPath(repoRoot string, style Style) string {
	return filepath.Join(filepath.Dir(repoConfigPathForStyle(repoRoot, style)), "layout.toml")
}

func repoConfigPathForStyle(repoRoot string, style Style) string {
	switch style {
	case StyleHome:
		return filepath.Join(repoRoot, ".changes", "config", "config.toml")
	default:
		return filepath.Join(repoRoot, ".config", "changes", "config.toml")
	}
}
