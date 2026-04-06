package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	repoConfigPath = ".config/changes/config.toml"
	userConfigPath = ".config/changes/config.toml"

	RenderModeSingleRelease = "single_release"
	RenderModeReleaseChain  = "release_chain"

	RenderProfileRepositoryMarkdown = "repository_markdown"
	RenderProfileChangelog          = RenderProfileRepositoryMarkdown
	RenderProfileGitHubRelease      = "github_release"
	RenderProfileTesterSummary      = "tester_summary"
	RenderProfileTester             = RenderProfileTesterSummary
	RenderProfileDebianChangelog    = "debian_changelog"
	RenderProfileRPMChangelog       = "rpm_changelog"
)

type Config struct {
	Project        ProjectConfig            `toml:"project"`
	Paths          PathsConfig              `toml:"paths"`
	RenderProfiles map[string]RenderProfile `toml:"render_profiles"`
	Versioning     VersioningConfig         `toml:"versioning"`
}

type ProjectConfig struct {
	Name           string `toml:"name"`
	ChangelogFile  string `toml:"changelog_file"`
	InitialVersion string `toml:"initial_version"`
}

type PathsConfig struct {
	DataDir      string `toml:"data_dir"`
	StateDir     string `toml:"state_dir"`
	TemplatesDir string `toml:"templates_dir"`
}

type RenderProfile struct {
	Description     string            `toml:"description"`
	Mode            string            `toml:"mode"`
	DocumentHeader  string            `toml:"document_header"`
	ReleaseTemplate string            `toml:"release_template"`
	EntryTemplate   string            `toml:"entry_template"`
	MaxChars        int               `toml:"max_chars"`
	OmissionNotice  string            `toml:"omission_notice"`
	Metadata        map[string]string `toml:"metadata"`
}

type VersioningConfig struct {
	PublicAPI string `toml:"public_api"`
}

func Default() Config {
	return Config{
		Project: ProjectConfig{
			ChangelogFile:  "CHANGELOG.md",
			InitialVersion: "0.1.0",
		},
		Paths: PathsConfig{
			DataDir:      ".local/share/changes",
			StateDir:     ".local/state/changes",
			TemplatesDir: ".local/share/changes/templates",
		},
		Versioning: VersioningConfig{
			PublicAPI: "unstable",
		},
	}
}

func RepoConfigPath(repoRoot string) string {
	return filepath.Join(repoRoot, repoConfigPath)
}

func UserConfigPath(home string) string {
	return filepath.Join(home, userConfigPath)
}

func FragmentsDir(repoRoot string, cfg Config) string {
	return filepath.Join(repoRoot, cfg.Paths.DataDir, "fragments")
}

func ReleasesDir(repoRoot string, cfg Config) string {
	return filepath.Join(repoRoot, cfg.Paths.DataDir, "releases")
}

func TemplatesDir(repoRoot string, cfg Config) string {
	return filepath.Join(repoRoot, cfg.Paths.TemplatesDir)
}

func PromptsDir(repoRoot string, cfg Config) string {
	return filepath.Join(repoRoot, cfg.Paths.DataDir, "prompts")
}

func HistoryImportPromptPath(repoRoot string, cfg Config) string {
	return filepath.Join(PromptsDir(repoRoot, cfg), "release-history-import-llm-prompt.md")
}

func StateDir(repoRoot string, cfg Config) string {
	return filepath.Join(repoRoot, cfg.Paths.StateDir)
}

func ChangelogPath(repoRoot string, cfg Config) string {
	return filepath.Join(repoRoot, cfg.Project.ChangelogFile)
}

func Load(repoRoot string) (Config, error) {
	path := RepoConfigPath(repoRoot)
	cfg := Default()
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("load config: %s does not exist; run `changes init` first", path)
		}
		return Config{}, fmt.Errorf("load config: %w", err)
	}

	meta, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if undecoded := meta.Undecoded(); len(undecoded) > 0 {
		return Config{}, fmt.Errorf("decode config: unsupported keys: %s", joinKeys(undecoded))
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	defaults := Default()

	if strings.TrimSpace(c.Versioning.PublicAPI) == "" {
		c.Versioning.PublicAPI = defaults.Versioning.PublicAPI
	}
}

func (c Config) Validate() error {
	for name, profile := range c.RenderProfiles {
		switch profile.Mode {
		case RenderModeSingleRelease, RenderModeReleaseChain:
		case "":
			// Empty fields are allowed in repo overrides and resolved later.
		default:
			return fmt.Errorf("config: render profile %s has unsupported mode %q", name, profile.Mode)
		}
		if profile.MaxChars < 0 {
			return fmt.Errorf("config: render profile %s max_chars must be >= 0", name)
		}
	}

	switch strings.TrimSpace(c.Versioning.PublicAPI) {
	case "stable", "unstable":
	default:
		return fmt.Errorf("config: versioning.public_api must be one of stable, unstable")
	}

	return nil
}

func Write(path string, cfg Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	defer file.Close()

	if err := toml.NewEncoder(file).Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return nil
}

func joinKeys(keys []toml.Key) string {
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key.String())
	}
	return strings.Join(parts, ", ")
}
