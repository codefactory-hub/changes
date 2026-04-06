package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/releases"
)

type TemplatePack struct {
	Name            string
	Description     string
	Mode            string
	DocumentHeader  string
	ReleaseTemplate string
	EntryTemplate   string
	MaxChars        int
	OmissionNotice  string
	Metadata        map[string]string
}

type Renderer struct {
	pack            TemplatePack
	entryTemplate   *template.Template
	releaseTemplate *template.Template
}

type Document struct {
	Bundles             []releases.ReleaseBundle
	OmittedReleaseCount int
}

type renderedSection struct {
	Key     string
	Title   string
	Entries []string
}

type releaseTemplateData struct {
	Pack     TemplatePack
	Release  releases.ReleaseRecord
	Sections []renderedSection
}

func New(repoRoot string, cfg config.Config, profileName string) (*Renderer, error) {
	profile, err := resolveProfile(cfg, profileName)
	if err != nil {
		return nil, err
	}

	pack := TemplatePack{
		Name:            profileName,
		Description:     profile.Description,
		Mode:            profile.Mode,
		DocumentHeader:  profile.DocumentHeader,
		ReleaseTemplate: profile.ReleaseTemplate,
		EntryTemplate:   profile.EntryTemplate,
		MaxChars:        profile.MaxChars,
		OmissionNotice:  profile.OmissionNotice,
		Metadata:        cloneMetadata(profile.Metadata),
	}

	funcs := template.FuncMap{
		"indent":        indent,
		"join":          strings.Join,
		"singleLine":    singleLine,
		"metadata":      pack.metadataValue,
		"formatDate":    formatDate,
		"formatDateRPM": formatDateRPM,
	}

	entryBody, err := loadTemplate(repoRoot, cfg, pack.EntryTemplate)
	if err != nil {
		return nil, fmt.Errorf("load entry template: %w", err)
	}
	releaseBody, err := loadTemplate(repoRoot, cfg, pack.ReleaseTemplate)
	if err != nil {
		return nil, fmt.Errorf("load release template: %w", err)
	}

	entryTemplate, err := template.New(pack.EntryTemplate).Funcs(funcs).Parse(entryBody)
	if err != nil {
		return nil, fmt.Errorf("parse entry template: %w", err)
	}

	releaseTemplate, err := template.New(pack.ReleaseTemplate).Funcs(funcs).Parse(releaseBody)
	if err != nil {
		return nil, fmt.Errorf("parse release template: %w", err)
	}

	return &Renderer{
		pack:            pack,
		entryTemplate:   entryTemplate,
		releaseTemplate: releaseTemplate,
	}, nil
}

func (r *Renderer) Render(doc Document) (string, error) {
	switch r.pack.Mode {
	case config.RenderModeSingleRelease:
		if len(doc.Bundles) != 1 {
			return "", fmt.Errorf("render pack %q expects a single release bundle, got %d", r.pack.Name, len(doc.Bundles))
		}
		out, err := r.renderBundle(doc.Bundles[0])
		if err != nil {
			return "", err
		}
		if r.pack.MaxChars > 0 && utf8.RuneCountInString(out) > r.pack.MaxChars {
			return "", fmt.Errorf("render release %s: output exceeds pack max_chars=%d", doc.Bundles[0].Release.Version, r.pack.MaxChars)
		}
		return out, nil
	case config.RenderModeReleaseChain:
		return r.renderChain(doc)
	default:
		return "", fmt.Errorf("render pack %q has unsupported mode %q", r.pack.Name, r.pack.Mode)
	}
}

func (r *Renderer) Pack() TemplatePack {
	return r.pack
}

func (r *Renderer) renderBundle(bundle releases.ReleaseBundle) (string, error) {
	sections := make([]renderedSection, 0, len(bundle.Sections))
	suppressBreaking := strings.TrimSpace(bundle.Release.ParentVersion) == "" && !bundle.Release.Bootstrap
	for _, section := range bundle.Sections {
		renderedEntries := make([]string, 0, len(section.Entries))
		for _, entry := range section.Entries {
			body, err := r.renderEntry(entry, suppressBreaking)
			if err != nil {
				return "", err
			}
			renderedEntries = append(renderedEntries, strings.TrimSpace(body))
		}
		sections = append(sections, renderedSection{
			Key:     section.Key,
			Title:   section.Title,
			Entries: renderedEntries,
		})
	}

	var buf bytes.Buffer
	if err := r.releaseTemplate.Execute(&buf, releaseTemplateData{
		Pack:     r.pack,
		Release:  bundle.Release,
		Sections: sections,
	}); err != nil {
		return "", fmt.Errorf("render release %s: %w", bundle.Release.Version, err)
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func (r *Renderer) renderChain(doc Document) (string, error) {
	keep := len(doc.Bundles)
	for keep >= 0 {
		trimmed := Document{
			Bundles:             slices.Clone(doc.Bundles[:keep]),
			OmittedReleaseCount: len(doc.Bundles) - keep,
		}
		content, err := r.renderDocument(trimmed)
		if err != nil {
			return "", err
		}
		if r.pack.MaxChars == 0 || utf8.RuneCountInString(content) <= r.pack.MaxChars {
			return content, nil
		}
		keep--
	}
	return "", fmt.Errorf("render document: pack max_chars=%d is too small for the header and omission notice", r.pack.MaxChars)
}

func (r *Renderer) renderDocument(doc Document) (string, error) {
	blocks := make([]string, 0, len(doc.Bundles))
	for _, bundle := range doc.Bundles {
		block, err := r.renderBundle(bundle)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, block)
	}
	return assembleDocument(r.pack, blocks, doc.OmittedReleaseCount), nil
}

func (r *Renderer) renderEntry(entry releases.BundleEntry, suppressBreaking bool) (string, error) {
	fragment := entry.Fragment
	if suppressBreaking {
		fragment.Breaking = false
	}
	var buf bytes.Buffer
	if err := r.entryTemplate.Execute(&buf, fragment); err != nil {
		return "", fmt.Errorf("render entry %s: %w", entry.Fragment.ID, err)
	}
	return buf.String(), nil
}

func assembleDocument(pack TemplatePack, blocks []string, omittedReleaseCount int) string {
	parts := make([]string, 0, len(blocks)+2)
	if strings.TrimSpace(pack.DocumentHeader) != "" {
		parts = append(parts, strings.TrimSpace(pack.DocumentHeader))
	}
	for _, block := range blocks {
		parts = append(parts, strings.TrimSpace(block))
	}
	if omittedReleaseCount > 0 && strings.TrimSpace(pack.OmissionNotice) != "" {
		parts = append(parts, strings.TrimSpace(pack.OmissionNotice))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n\n") + "\n"
}

func indent(text string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := range lines {
		lines[i] = prefix + strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}

func singleLine(text string) string {
	fields := strings.Fields(strings.TrimSpace(text))
	return strings.Join(fields, " ")
}

func formatDate(value time.Time) string {
	return value.UTC().Format("Mon, 02 Jan 2006 15:04:05 -0700")
}

func formatDateRPM(value time.Time) string {
	return value.UTC().Format("Mon Jan 02 2006")
}

func loadTemplate(repoRoot string, cfg config.Config, name string) (string, error) {
	path := config.TemplatesDir(repoRoot, cfg)
	fullPath := filepath.Join(path, name)
	if raw, err := os.ReadFile(fullPath); err == nil {
		return string(raw), nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read template %s: %w", fullPath, err)
	}

	body, ok := BuiltinTemplateFiles()[name]
	if !ok {
		return "", fmt.Errorf("template %s is not available", name)
	}
	return body, nil
}

func (p TemplatePack) metadataValue(key, fallback string) string {
	if value, ok := p.Metadata[key]; ok {
		return value
	}
	return fallback
}
