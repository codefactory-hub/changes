package templates

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/example/changes/internal/config"
)

const DefaultRepositoryMarkdownReleaseTemplate = `## {{ .Release.Version }}
{{- if .Release.IsPrerelease }} (prerelease){{ else }} (stable){{ end }}

{{- if .Sections }}
{{- range .Sections }}
### {{ .Title }}

{{- range .Entries }}
{{ . }}

{{- end }}
{{- end }}
{{- else }}
No entries selected for this release.
{{- end }}
`

const DefaultGitHubReleaseTemplate = `# Release {{ .Release.Version }}
{{- if .Release.IsPrerelease }} (targets {{ .Release.TargetVersion }}){{ end }}
{{- if .Sections }}

{{- range .Sections }}
## {{ .Title }}

{{- range .Entries }}
{{ . }}

{{- end }}
{{- end }}
{{- else }}
No entries selected for this release.
{{- end }}
`

const DefaultTesterSummaryReleaseTemplate = `## Tester Summary For {{ .Release.Version }}
{{- if .Sections }}

{{- range .Sections }}
### {{ .Title }}

{{- range .Entries }}
{{ . }}

{{- end }}
{{- end }}
{{- else }}
No entries selected for testers.
{{- end }}
`

const DefaultDebianChangelogTemplate = `changes ({{ .Release.Version }}) {{ metadata "distribution" "unstable" }}; urgency={{ metadata "urgency" "medium" }}

{{- if .Sections }}
{{- range .Sections }}
  * {{ .Title }}
{{- range .Entries }}
{{ indent . 4 }}
{{- end }}

{{- end }}
{{- else }}
  * No entries selected for this release.

{{- end }}
 -- {{ metadata "maintainer_name" "Changes Release Bot" }} <{{ metadata "maintainer_email" "changes@example.invalid" }}>  {{ formatDate .Release.CreatedAt }}
`

const DefaultRPMChangelogTemplate = `* {{ formatDateRPM .Release.CreatedAt }} {{ metadata "maintainer_name" "Changes Release Bot" }} <{{ metadata "maintainer_email" "changes@example.invalid" }}> - {{ .Release.Version }}
{{- if .Sections }}
{{- range .Sections }}
- {{ .Title }}
{{- range .Entries }}
{{ indent . 2 }}
{{- end }}
{{- end }}
{{- else }}
- No entries selected for this release.
{{- end }}
`

const DefaultReleaseEntryTemplate = `- {{ .Title }}{{ if .Breaking }} (breaking){{ end }}{{ if .Scopes }}{{ printf "\n%s" (indent (printf "Scope: %s" (join .Scopes ", ")) 2) }}{{ end }}{{ if .Body }}{{ printf "\n%s" (indent .Body 2) }}{{ end }}`

const DefaultTesterSummaryEntryTemplate = `- {{ .Title }}`

const DefaultPackageEntryTemplate = `- {{ .Title }}{{ if .Breaking }} (breaking){{ end }}{{ if .Body }}: {{ singleLine .Body }}{{ end }}`

type BuiltInPack struct {
	Name        string
	Description string
	Profile     config.RenderProfile
	Templates   map[string]string
}

type FileSet struct {
	Paths []string
}

func EnsureDefaultFiles(repoRoot string, cfg config.Config) (FileSet, error) {
	dir := config.TemplatesDir(repoRoot, cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileSet{}, fmt.Errorf("create templates directory: %w", err)
	}

	files := BuiltInTemplateFiles()

	paths := make([]string, 0, len(files))
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		body := files[name]
		path := filepath.Join(dir, name)
		if err := writeIfMissing(path, body); err != nil {
			return FileSet{}, err
		}
		paths = append(paths, path)
	}

	return FileSet{Paths: paths}, nil
}

func BuiltInPacks() map[string]BuiltInPack {
	defaults := config.Default()
	return map[string]BuiltInPack{
		config.RenderProfileRepositoryMarkdown: {
			Name:        config.RenderProfileRepositoryMarkdown,
			Description: defaults.RenderProfiles[config.RenderProfileRepositoryMarkdown].Description,
			Profile:     defaults.RenderProfiles[config.RenderProfileRepositoryMarkdown],
			Templates: map[string]string{
				"repository-markdown-release.md.tmpl": DefaultRepositoryMarkdownReleaseTemplate,
				"release-entry.md.tmpl":               DefaultReleaseEntryTemplate,
			},
		},
		config.RenderProfileGitHubRelease: {
			Name:        config.RenderProfileGitHubRelease,
			Description: defaults.RenderProfiles[config.RenderProfileGitHubRelease].Description,
			Profile:     defaults.RenderProfiles[config.RenderProfileGitHubRelease],
			Templates: map[string]string{
				"github-release.md.tmpl": DefaultGitHubReleaseTemplate,
				"release-entry.md.tmpl":  DefaultReleaseEntryTemplate,
			},
		},
		config.RenderProfileTesterSummary: {
			Name:        config.RenderProfileTesterSummary,
			Description: defaults.RenderProfiles[config.RenderProfileTesterSummary].Description,
			Profile:     defaults.RenderProfiles[config.RenderProfileTesterSummary],
			Templates: map[string]string{
				"tester-summary-release.md.tmpl": DefaultTesterSummaryReleaseTemplate,
				"tester-summary-entry.md.tmpl":   DefaultTesterSummaryEntryTemplate,
			},
		},
		config.RenderProfileDebianChangelog: {
			Name:        config.RenderProfileDebianChangelog,
			Description: defaults.RenderProfiles[config.RenderProfileDebianChangelog].Description,
			Profile:     defaults.RenderProfiles[config.RenderProfileDebianChangelog],
			Templates: map[string]string{
				"debian-changelog.tmpl": DefaultDebianChangelogTemplate,
				"package-entry.tmpl":    DefaultPackageEntryTemplate,
			},
		},
		config.RenderProfileRPMChangelog: {
			Name:        config.RenderProfileRPMChangelog,
			Description: defaults.RenderProfiles[config.RenderProfileRPMChangelog].Description,
			Profile:     defaults.RenderProfiles[config.RenderProfileRPMChangelog],
			Templates: map[string]string{
				"rpm-changelog.tmpl": DefaultRPMChangelogTemplate,
				"package-entry.tmpl": DefaultPackageEntryTemplate,
			},
		},
	}
}

func BuiltInTemplateFiles() map[string]string {
	files := map[string]string{}
	for _, pack := range BuiltInPacks() {
		for name, body := range pack.Templates {
			files[name] = body
		}
	}
	return files
}

func LoadTemplate(repoRoot string, cfg config.Config, name string) (string, error) {
	path := filepath.Join(config.TemplatesDir(repoRoot, cfg), name)
	if raw, err := os.ReadFile(path); err == nil {
		return string(raw), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read template %s: %w", path, err)
	}

	body, ok := BuiltInTemplateFiles()[name]
	if !ok {
		return "", fmt.Errorf("template %s is not available", name)
	}
	return body, nil
}

func writeIfMissing(path, body string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat template %s: %w", path, err)
	}

	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write template %s: %w", path, err)
	}
	return nil
}
