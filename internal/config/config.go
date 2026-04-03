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
		RenderProfiles: map[string]RenderProfile{
			RenderProfileRepositoryMarkdown: {
				Description:     "Repository changelog in Markdown generated from the stable release chain.",
				Mode:            RenderModeReleaseChain,
				DocumentHeader:  "# Changelog",
				ReleaseTemplate: "repository-markdown-release.md.tmpl",
				EntryTemplate:   "release-entry.md.tmpl",
				MaxChars:        0,
				OmissionNotice:  "Additional releases omitted for length.",
			},
			RenderProfileGitHubRelease: {
				Description:     "Markdown release body for GitHub or GitLab releases.",
				Mode:            RenderModeSingleRelease,
				ReleaseTemplate: "github-release.md.tmpl",
				EntryTemplate:   "release-entry.md.tmpl",
				MaxChars:        4000,
				OmissionNotice:  "Additional content omitted for length.",
			},
			RenderProfileTesterSummary: {
				Description:     "Concise tester-facing release summary.",
				Mode:            RenderModeSingleRelease,
				ReleaseTemplate: "tester-summary-release.md.tmpl",
				EntryTemplate:   "tester-summary-entry.md.tmpl",
				MaxChars:        8000,
				OmissionNotice:  "Additional content omitted for length.",
			},
			RenderProfileDebianChangelog: {
				Description:     "Debian-style changelog text generated from a single release record.",
				Mode:            RenderModeSingleRelease,
				ReleaseTemplate: "debian-changelog.tmpl",
				EntryTemplate:   "package-entry.tmpl",
				MaxChars:        0,
				OmissionNotice:  "",
				Metadata: map[string]string{
					"distribution":     "unstable",
					"urgency":          "medium",
					"maintainer_name":  "Changes Release Bot",
					"maintainer_email": "changes@example.invalid",
				},
			},
			RenderProfileRPMChangelog: {
				Description:     "RPM-style changelog text generated from a single release record.",
				Mode:            RenderModeSingleRelease,
				ReleaseTemplate: "rpm-changelog.tmpl",
				EntryTemplate:   "package-entry.tmpl",
				MaxChars:        0,
				OmissionNotice:  "",
				Metadata: map[string]string{
					"maintainer_name":  "Changes Release Bot",
					"maintainer_email": "changes@example.invalid",
				},
			},
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
	if c.RenderProfiles == nil {
		c.RenderProfiles = map[string]RenderProfile{}
	}

	for name, profile := range defaults.RenderProfiles {
		current, ok := c.RenderProfiles[name]
		if !ok {
			c.RenderProfiles[name] = profile
			continue
		}
		if current.Mode == "" {
			current.Mode = profile.Mode
		}
		if current.Description == "" {
			current.Description = profile.Description
		}
		if current.DocumentHeader == "" {
			current.DocumentHeader = profile.DocumentHeader
		}
		if current.ReleaseTemplate == "" {
			current.ReleaseTemplate = profile.ReleaseTemplate
		}
		if current.EntryTemplate == "" {
			current.EntryTemplate = profile.EntryTemplate
		}
		if current.OmissionNotice == "" {
			current.OmissionNotice = profile.OmissionNotice
		}
		if current.Metadata == nil && profile.Metadata != nil {
			current.Metadata = cloneMetadata(profile.Metadata)
		}
		for key, value := range profile.Metadata {
			if current.Metadata == nil {
				current.Metadata = map[string]string{}
			}
			if _, ok := current.Metadata[key]; !ok {
				current.Metadata[key] = value
			}
		}
		c.RenderProfiles[name] = current
	}
}

func (c Config) Validate() error {
	if c.RenderProfiles == nil || len(c.RenderProfiles) == 0 {
		return fmt.Errorf("config: render_profiles must define at least one profile")
	}

	for name, profile := range c.RenderProfiles {
		switch profile.Mode {
		case RenderModeSingleRelease, RenderModeReleaseChain:
		default:
			return fmt.Errorf("config: render profile %s has unsupported mode %q", name, profile.Mode)
		}
		if strings.TrimSpace(profile.ReleaseTemplate) == "" {
			return fmt.Errorf("config: render profile %s is missing release_template", name)
		}
		if strings.TrimSpace(profile.EntryTemplate) == "" {
			return fmt.Errorf("config: render profile %s is missing entry_template", name)
		}
		if profile.MaxChars < 0 {
			return fmt.Errorf("config: render profile %s max_chars must be >= 0", name)
		}
	}

	return nil
}

func (c Config) RenderProfile(name string) (RenderProfile, error) {
	profile, ok := c.RenderProfiles[name]
	if !ok {
		return RenderProfile{}, fmt.Errorf("render profile %q is not configured", name)
	}
	return profile, nil
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

func cloneMetadata(in map[string]string) map[string]string {
	if in == nil {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
