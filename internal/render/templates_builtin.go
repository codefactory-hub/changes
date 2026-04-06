package render

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

const DefaultReleaseEntryTemplate = `- {{ singleLine .Body }}{{ if .Breaking }} (breaking){{ end }}`

const DefaultTesterSummaryEntryTemplate = `- {{ singleLine .Body }}`

const DefaultPackageEntryTemplate = `- {{ singleLine .Body }}{{ if .Breaking }} (breaking){{ end }}`

func BuiltinTemplateFiles() map[string]string {
	return map[string]string{
		"repository-markdown-release.md.tmpl": DefaultRepositoryMarkdownReleaseTemplate,
		"github-release.md.tmpl":              DefaultGitHubReleaseTemplate,
		"tester-summary-release.md.tmpl":      DefaultTesterSummaryReleaseTemplate,
		"debian-changelog.tmpl":               DefaultDebianChangelogTemplate,
		"rpm-changelog.tmpl":                  DefaultRPMChangelogTemplate,
		"release-entry.md.tmpl":               DefaultReleaseEntryTemplate,
		"tester-summary-entry.md.tmpl":        DefaultTesterSummaryEntryTemplate,
		"package-entry.tmpl":                  DefaultPackageEntryTemplate,
	}
}
