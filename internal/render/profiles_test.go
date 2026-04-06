package render

import (
	"strings"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/example/changes/internal/config"
)

func TestResolveProfilesIncludesBuiltInsWithoutOverrides(t *testing.T) {
	profiles, err := ResolveProfiles(config.Default())
	if err != nil {
		t.Fatalf("ResolveProfiles returned error: %v", err)
	}

	for _, name := range []string{
		config.RenderProfileRepositoryMarkdown,
		config.RenderProfileGitHubRelease,
		config.RenderProfileTesterSummary,
		config.RenderProfileDebianChangelog,
		config.RenderProfileRPMChangelog,
	} {
		if _, ok := profiles[name]; !ok {
			t.Fatalf("missing built-in profile %q", name)
		}
	}
}

func TestResolveProfilesMergesOverridesIntoBuiltIns(t *testing.T) {
	cfg := config.Default()
	cfg.RenderProfiles = map[string]config.RenderProfile{
		config.RenderProfileGitHubRelease: {
			Description: "Custom release body",
			MaxChars:    1234,
			Metadata: map[string]string{
				"channel": "nightly",
			},
		},
	}

	profiles, err := ResolveProfiles(cfg)
	if err != nil {
		t.Fatalf("ResolveProfiles returned error: %v", err)
	}

	profile := profiles[config.RenderProfileGitHubRelease]
	if profile.Description != "Custom release body" {
		t.Fatalf("description = %q", profile.Description)
	}
	if profile.MaxChars != 1234 {
		t.Fatalf("max_chars = %d, want 1234", profile.MaxChars)
	}
	if profile.ReleaseTemplate != "github-release.md.tmpl" {
		t.Fatalf("release template = %q, want built-in value", profile.ReleaseTemplate)
	}
	if got := profile.Metadata["channel"]; got != "nightly" {
		t.Fatalf("metadata[channel] = %q, want nightly", got)
	}
}

func TestResolveProfilesAllowsClearingBuiltInFields(t *testing.T) {
	cfg := config.Default()
	raw := `
[render_profiles.repository_markdown]
document_header = ""
max_chars = 0
omission_notice = ""

[render_profiles.debian_changelog.metadata]
distribution = ""
`
	if _, err := toml.Decode(raw, &cfg); err != nil {
		t.Fatalf("decode override config: %v", err)
	}

	profiles, err := ResolveProfiles(cfg)
	if err != nil {
		t.Fatalf("ResolveProfiles returned error: %v", err)
	}

	repoMarkdown := profiles[config.RenderProfileRepositoryMarkdown]
	if repoMarkdown.DocumentHeader != "" {
		t.Fatalf("document_header = %q, want empty", repoMarkdown.DocumentHeader)
	}
	if repoMarkdown.MaxChars != 0 {
		t.Fatalf("max_chars = %d, want 0", repoMarkdown.MaxChars)
	}
	if repoMarkdown.OmissionNotice != "" {
		t.Fatalf("omission_notice = %q, want empty", repoMarkdown.OmissionNotice)
	}

	debian := profiles[config.RenderProfileDebianChangelog]
	if got := debian.Metadata["distribution"]; got != "" {
		t.Fatalf("metadata[distribution] = %q, want empty", got)
	}
}

func TestResolveProfilesAllowsNewCustomProfiles(t *testing.T) {
	cfg := config.Default()
	cfg.RenderProfiles = map[string]config.RenderProfile{
		"ops_summary": {
			Description:     "Ops summary",
			Mode:            config.RenderModeSingleRelease,
			ReleaseTemplate: "github-release.md.tmpl",
			EntryTemplate:   "release-entry.md.tmpl",
		},
	}

	profiles, err := ResolveProfiles(cfg)
	if err != nil {
		t.Fatalf("ResolveProfiles returned error: %v", err)
	}
	if _, ok := profiles["ops_summary"]; !ok {
		t.Fatalf("custom profile missing from effective set")
	}
}

func TestResolveProfilesRejectsInvalidEffectiveProfiles(t *testing.T) {
	cfg := config.Default()
	cfg.RenderProfiles = map[string]config.RenderProfile{
		config.RenderProfileGitHubRelease: {
			Mode: "unsupported",
		},
	}

	_, err := ResolveProfiles(cfg)
	if err == nil || !strings.Contains(err.Error(), "unsupported mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveProfileAndAvailablePacksCloneMetadata(t *testing.T) {
	cfg := config.Default()

	profile, err := resolveProfile(cfg, config.RenderProfileDebianChangelog)
	if err != nil {
		t.Fatalf("resolveProfile returned error: %v", err)
	}
	profile.Metadata["distribution"] = "mutated"

	again, err := resolveProfile(cfg, config.RenderProfileDebianChangelog)
	if err != nil {
		t.Fatalf("resolveProfile second call returned error: %v", err)
	}
	if again.Metadata["distribution"] != "unstable" {
		t.Fatalf("resolveProfile should clone built-in metadata, got %#v", again.Metadata)
	}

	packs, err := AvailablePacks(cfg)
	if err != nil {
		t.Fatalf("AvailablePacks returned error: %v", err)
	}
	for _, pack := range packs {
		if pack.Name == config.RenderProfileDebianChangelog {
			pack.Metadata["distribution"] = "mutated"
		}
	}
	resolved, err := resolveProfile(cfg, config.RenderProfileDebianChangelog)
	if err != nil {
		t.Fatalf("resolveProfile after packs returned error: %v", err)
	}
	if resolved.Metadata["distribution"] != "unstable" {
		t.Fatalf("AvailablePacks should clone metadata, got %#v", resolved.Metadata)
	}
}

func TestResolveProfilesRejectsMissingTemplatesAndUnknownProfiles(t *testing.T) {
	cfg := config.Default()
	cfg.RenderProfiles = map[string]config.RenderProfile{
		"broken": {
			Mode:          config.RenderModeSingleRelease,
			EntryTemplate: "release-entry.md.tmpl",
		},
	}
	if _, err := ResolveProfiles(cfg); err == nil || !strings.Contains(err.Error(), "missing release_template") {
		t.Fatalf("unexpected ResolveProfiles error: %v", err)
	}

	if _, err := resolveProfile(config.Default(), "missing"); err == nil || !strings.Contains(err.Error(), "is not configured") {
		t.Fatalf("unexpected resolveProfile error: %v", err)
	}
}
