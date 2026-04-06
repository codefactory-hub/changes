package render

import (
	"fmt"
	"slices"
	"strings"

	"github.com/example/changes/internal/config"
)

func BuiltinProfiles() map[string]config.RenderProfile {
	return map[string]config.RenderProfile{
		config.RenderProfileRepositoryMarkdown: {
			Description:     "Repository changelog in Markdown generated from the stable release chain.",
			Mode:            config.RenderModeReleaseChain,
			DocumentHeader:  "# Changelog",
			ReleaseTemplate: "repository-markdown-release.md.tmpl",
			EntryTemplate:   "release-entry.md.tmpl",
			MaxChars:        0,
			OmissionNotice:  "Additional releases omitted for length.",
		},
		config.RenderProfileGitHubRelease: {
			Description:     "Markdown release body for GitHub or GitLab releases.",
			Mode:            config.RenderModeSingleRelease,
			ReleaseTemplate: "github-release.md.tmpl",
			EntryTemplate:   "release-entry.md.tmpl",
			MaxChars:        4000,
			OmissionNotice:  "Additional content omitted for length.",
		},
		config.RenderProfileTesterSummary: {
			Description:     "Concise tester-facing release summary.",
			Mode:            config.RenderModeSingleRelease,
			ReleaseTemplate: "tester-summary-release.md.tmpl",
			EntryTemplate:   "tester-summary-entry.md.tmpl",
			MaxChars:        8000,
			OmissionNotice:  "Additional content omitted for length.",
		},
		config.RenderProfileDebianChangelog: {
			Description:     "Debian-style changelog text generated from a single release record.",
			Mode:            config.RenderModeSingleRelease,
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
		config.RenderProfileRPMChangelog: {
			Description:     "RPM-style changelog text generated from a single release record.",
			Mode:            config.RenderModeSingleRelease,
			ReleaseTemplate: "rpm-changelog.tmpl",
			EntryTemplate:   "package-entry.tmpl",
			MaxChars:        0,
			OmissionNotice:  "",
			Metadata: map[string]string{
				"maintainer_name":  "Changes Release Bot",
				"maintainer_email": "changes@example.invalid",
			},
		},
	}
}

func ResolveProfiles(cfg config.Config) (map[string]config.RenderProfile, error) {
	out := make(map[string]config.RenderProfile, len(BuiltinProfiles())+len(cfg.RenderProfiles))
	for name, profile := range BuiltinProfiles() {
		out[name] = cloneProfile(profile)
	}

	for name, override := range cfg.RenderProfiles {
		base, ok := out[name]
		if ok {
			out[name] = mergeProfile(base, override)
			continue
		}
		out[name] = cloneProfile(override)
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("render: no profiles are available")
	}
	if err := validateProfiles(out); err != nil {
		return nil, err
	}
	return out, nil
}

func AvailablePacks(cfg config.Config) ([]TemplatePack, error) {
	profiles, err := ResolveProfiles(cfg)
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	slices.Sort(names)

	packs := make([]TemplatePack, 0, len(names))
	for _, name := range names {
		profile := profiles[name]
		packs = append(packs, TemplatePack{
			Name:            name,
			Description:     profile.Description,
			Mode:            profile.Mode,
			DocumentHeader:  profile.DocumentHeader,
			ReleaseTemplate: profile.ReleaseTemplate,
			EntryTemplate:   profile.EntryTemplate,
			MaxChars:        profile.MaxChars,
			OmissionNotice:  profile.OmissionNotice,
			Metadata:        cloneMetadata(profile.Metadata),
		})
	}
	return packs, nil
}

func resolveProfile(cfg config.Config, name string) (config.RenderProfile, error) {
	profiles, err := ResolveProfiles(cfg)
	if err != nil {
		return config.RenderProfile{}, err
	}
	profile, ok := profiles[name]
	if !ok {
		return config.RenderProfile{}, fmt.Errorf("render profile %q is not configured", name)
	}
	return profile, nil
}

func validateProfiles(profiles map[string]config.RenderProfile) error {
	for name, profile := range profiles {
		switch profile.Mode {
		case config.RenderModeSingleRelease, config.RenderModeReleaseChain:
		default:
			return fmt.Errorf("render profile %s has unsupported mode %q", name, profile.Mode)
		}
		if strings.TrimSpace(profile.ReleaseTemplate) == "" {
			return fmt.Errorf("render profile %s is missing release_template", name)
		}
		if strings.TrimSpace(profile.EntryTemplate) == "" {
			return fmt.Errorf("render profile %s is missing entry_template", name)
		}
		if profile.MaxChars < 0 {
			return fmt.Errorf("render profile %s max_chars must be >= 0", name)
		}
	}
	return nil
}

func mergeProfile(base, override config.RenderProfile) config.RenderProfile {
	merged := cloneProfile(base)
	if override.HasDescription() {
		merged.Description = override.Description
	}
	if override.HasMode() {
		merged.Mode = override.Mode
	}
	if override.HasDocumentHeader() {
		merged.DocumentHeader = override.DocumentHeader
	}
	if override.HasReleaseTemplate() {
		merged.ReleaseTemplate = override.ReleaseTemplate
	}
	if override.HasEntryTemplate() {
		merged.EntryTemplate = override.EntryTemplate
	}
	if override.HasMaxChars() {
		merged.MaxChars = override.MaxChars
	}
	if override.HasOmissionNotice() {
		merged.OmissionNotice = override.OmissionNotice
	}
	if override.HasMetadata() {
		if merged.Metadata == nil {
			merged.Metadata = map[string]string{}
		}
		for key, value := range override.Metadata {
			merged.Metadata[key] = value
		}
	}
	return merged
}

func cloneProfile(profile config.RenderProfile) config.RenderProfile {
	profile.Metadata = cloneMetadata(profile.Metadata)
	return profile
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
