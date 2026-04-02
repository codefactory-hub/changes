package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const (
	repoConfigPath = ".config/changes/config.toml"
	userConfigPath = ".config/changes/config.toml"
)

type Config struct {
	Project    ProjectConfig    `toml:"project"`
	Paths      PathsConfig      `toml:"paths"`
	Render     RenderConfig     `toml:"render"`
	Versioning VersioningConfig `toml:"versioning"`
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

type RenderConfig struct {
	MaxChars        int    `toml:"max_chars"`
	DropPolicy      string `toml:"drop_policy"`
	OmissionNotice  string `toml:"omission_notice"`
	ReleaseTemplate string `toml:"release_template"`
	EntryTemplate   string `toml:"entry_template"`
}

type VersioningConfig struct {
	PrereleaseLabel string `toml:"prerelease_label"`
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
		Render: RenderConfig{
			MaxChars:        4000,
			DropPolicy:      "drop_whole_entries_from_bottom",
			OmissionNotice:  "Additional entries omitted for length.",
			ReleaseTemplate: "release.md.tmpl",
			EntryTemplate:   "entry.md.tmpl",
		},
		Versioning: VersioningConfig{
			PrereleaseLabel: "rc",
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

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	return cfg, nil
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
