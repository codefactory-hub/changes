package templates

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/example/changes/internal/config"
)

const DefaultReleaseTemplate = `## {{ .Release.Version }}
{{- if .Release.Channel }} ({{ .Release.Channel }}){{ end }}

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
{{- if .OmissionNotice }}
{{ .OmissionNotice }}
{{- end }}
`

const DefaultEntryTemplate = `- {{ .Title }}{{ if .Breaking }} (breaking){{ end }}{{ if .Scopes }}{{ printf "\n%s" (indent (printf "Scope: %s" (join .Scopes ", ")) 2) }}{{ end }}{{ if .Body }}{{ printf "\n%s" (indent .Body 2) }}{{ end }}`

type FileSet struct {
	ReleaseTemplatePath string
	EntryTemplatePath   string
}

func EnsureDefaultFiles(repoRoot string, cfg config.Config) (FileSet, error) {
	dir := config.TemplatesDir(repoRoot, cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return FileSet{}, fmt.Errorf("create templates directory: %w", err)
	}

	releasePath := filepath.Join(dir, cfg.Render.ReleaseTemplate)
	entryPath := filepath.Join(dir, cfg.Render.EntryTemplate)

	if err := writeIfMissing(releasePath, DefaultReleaseTemplate); err != nil {
		return FileSet{}, err
	}
	if err := writeIfMissing(entryPath, DefaultEntryTemplate); err != nil {
		return FileSet{}, err
	}

	return FileSet{
		ReleaseTemplatePath: releasePath,
		EntryTemplatePath:   entryPath,
	}, nil
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
