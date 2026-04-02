package render

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
	"unicode/utf8"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
)

type Renderer struct {
	entryTemplate   *template.Template
	releaseTemplate *template.Template
}

type Section struct {
	Key     string
	Title   string
	Entries []string
}

type releaseTemplateData struct {
	Release        releases.Manifest
	Sections       []Section
	OmissionNotice string
}

type renderedEntry struct {
	SectionKey string
	Section    string
	Text       string
}

var sectionOrder = []struct {
	Key   string
	Title string
}{
	{Key: "breaking", Title: "Breaking Changes"},
	{Key: "added", Title: "Added"},
	{Key: "changed", Title: "Changed"},
	{Key: "fixed", Title: "Fixed"},
	{Key: "removed", Title: "Removed"},
	{Key: "security", Title: "Security"},
	{Key: "other", Title: "Other"},
}

func New(repoRoot string, cfg config.Config, manifest releases.Manifest) (*Renderer, error) {
	funcs := template.FuncMap{
		"indent": indent,
		"join":   strings.Join,
	}

	entryBytes, err := os.ReadFile(filepath.Join(config.TemplatesDir(repoRoot, cfg), cfg.Render.EntryTemplate))
	if err != nil {
		return nil, fmt.Errorf("read entry template: %w", err)
	}

	releaseBytes, err := os.ReadFile(filepath.Join(config.TemplatesDir(repoRoot, cfg), manifest.Template))
	if err != nil {
		return nil, fmt.Errorf("read release template: %w", err)
	}

	entryTemplate, err := template.New(cfg.Render.EntryTemplate).Funcs(funcs).Parse(string(entryBytes))
	if err != nil {
		return nil, fmt.Errorf("parse entry template: %w", err)
	}

	releaseTemplate, err := template.New(manifest.Template).Funcs(funcs).Parse(string(releaseBytes))
	if err != nil {
		return nil, fmt.Errorf("parse release template: %w", err)
	}

	return &Renderer{
		entryTemplate:   entryTemplate,
		releaseTemplate: releaseTemplate,
	}, nil
}

func (r *Renderer) Render(cfg config.Config, manifest releases.Manifest, selected []fragments.Fragment) (string, error) {
	slices.SortFunc(selected, func(a, b fragments.Fragment) int {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	candidates := make([]renderedEntry, 0, len(selected))
	for _, item := range selected {
		body, err := r.renderEntry(item)
		if err != nil {
			return "", err
		}
		sectionKey, sectionTitle := classify(item)
		candidates = append(candidates, renderedEntry{
			SectionKey: sectionKey,
			Section:    sectionTitle,
			Text:       strings.TrimSpace(body),
		})
	}

	keep := len(candidates)
	for keep >= 0 {
		data := releaseTemplateData{
			Release:  manifest,
			Sections: groupEntries(candidates[:keep]),
		}
		if keep < len(candidates) {
			data.OmissionNotice = cfg.Render.OmissionNotice
		}

		out, err := r.renderRelease(data)
		if err != nil {
			return "", err
		}
		if utf8.RuneCountInString(out) <= manifest.MaxChars {
			return out, nil
		}
		keep--
	}

	return "", fmt.Errorf("render release: max_chars=%d is too small for the release header and omission notice", manifest.MaxChars)
}

func (r *Renderer) renderEntry(item fragments.Fragment) (string, error) {
	var buf bytes.Buffer
	if err := r.entryTemplate.Execute(&buf, item); err != nil {
		return "", fmt.Errorf("render entry %s: %w", item.ID, err)
	}
	return buf.String(), nil
}

func (r *Renderer) renderRelease(data releaseTemplateData) (string, error) {
	var buf bytes.Buffer
	if err := r.releaseTemplate.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render release %s: %w", data.Release.Version, err)
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func classify(item fragments.Fragment) (string, string) {
	if item.Breaking {
		return "breaking", "Breaking Changes"
	}

	switch strings.ToLower(strings.TrimSpace(item.Type)) {
	case "added":
		return "added", "Added"
	case "changed":
		return "changed", "Changed"
	case "fixed":
		return "fixed", "Fixed"
	case "removed":
		return "removed", "Removed"
	case "security":
		return "security", "Security"
	default:
		return "other", "Other"
	}
}

func groupEntries(entries []renderedEntry) []Section {
	buckets := make(map[string][]string, len(sectionOrder))
	for _, entry := range entries {
		buckets[entry.SectionKey] = append(buckets[entry.SectionKey], entry.Text)
	}

	sections := make([]Section, 0, len(sectionOrder))
	for _, meta := range sectionOrder {
		items := buckets[meta.Key]
		if len(items) == 0 {
			continue
		}
		sections = append(sections, Section{
			Key:     meta.Key,
			Title:   meta.Title,
			Entries: items,
		})
	}
	return sections
}

func indent(text string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	lines := strings.Split(strings.TrimSpace(text), "\n")
	for i := range lines {
		lines[i] = prefix + strings.TrimSpace(lines[i])
	}
	return strings.Join(lines, "\n")
}
