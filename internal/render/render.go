package render

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/templates"
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

type Selector struct {
	manifests     []releases.Manifest
	manifestIndex map[string]releases.Manifest
	fragmentIndex map[string]fragments.Fragment
}

type Document struct {
	Releases            []ReleaseDocument
	OmittedReleaseCount int
}

type ReleaseDocument struct {
	Release   releases.Manifest
	Sections  []Section
	Ancestors []releases.Manifest
}

type Section struct {
	Key     string
	Title   string
	Entries []Entry
}

type Entry struct {
	Fragment fragments.Fragment
}

type renderedSection struct {
	Key     string
	Title   string
	Entries []string
}

type releaseTemplateData struct {
	Pack     TemplatePack
	Release  releases.Manifest
	Sections []renderedSection
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

func New(repoRoot string, cfg config.Config, profileName string) (*Renderer, error) {
	profile, err := cfg.RenderProfile(profileName)
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

	entryBody, err := templates.LoadTemplate(repoRoot, cfg, pack.EntryTemplate)
	if err != nil {
		return nil, fmt.Errorf("load entry template: %w", err)
	}
	releaseBody, err := templates.LoadTemplate(repoRoot, cfg, pack.ReleaseTemplate)
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

func NewSelector(allFragments []fragments.Fragment, manifests []releases.Manifest) *Selector {
	fragmentIndex := make(map[string]fragments.Fragment, len(allFragments))
	for _, item := range allFragments {
		fragmentIndex[item.ID] = item
	}

	manifestIndex := make(map[string]releases.Manifest, len(manifests))
	for _, manifest := range manifests {
		manifestIndex[manifest.Version] = manifest
	}

	return &Selector{
		manifests:     slices.Clone(manifests),
		manifestIndex: manifestIndex,
		fragmentIndex: fragmentIndex,
	}
}

func AvailablePacks(cfg config.Config) []TemplatePack {
	names := make([]string, 0, len(cfg.RenderProfiles))
	for name := range cfg.RenderProfiles {
		names = append(names, name)
	}
	slices.Sort(names)

	packs := make([]TemplatePack, 0, len(names))
	for _, name := range names {
		profile := cfg.RenderProfiles[name]
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
	return packs
}

func (s *Selector) Release(manifest releases.Manifest) (Document, error) {
	releaseDoc, err := s.releaseDocument(manifest, nil)
	if err != nil {
		return Document{}, err
	}
	return Document{Releases: []ReleaseDocument{releaseDoc}}, nil
}

func (s *Selector) ReleaseChain(head releases.Manifest) (Document, error) {
	lineage, err := releases.Lineage(head, s.manifests)
	if err != nil {
		return Document{}, err
	}

	releasesOut := make([]ReleaseDocument, 0, len(lineage))
	for i, manifest := range lineage {
		ancestors := append([]releases.Manifest(nil), lineage[i+1:]...)
		releaseDoc, err := s.releaseDocument(manifest, ancestors)
		if err != nil {
			return Document{}, err
		}
		releasesOut = append(releasesOut, releaseDoc)
	}
	return Document{Releases: releasesOut}, nil
}

func (r *Renderer) Render(doc Document) (string, error) {
	switch r.pack.Mode {
	case config.RenderModeSingleRelease:
		if len(doc.Releases) != 1 {
			return "", fmt.Errorf("render pack %q expects a single release document, got %d", r.pack.Name, len(doc.Releases))
		}
		out, err := r.renderRelease(doc.Releases[0])
		if err != nil {
			return "", err
		}
		if r.pack.MaxChars > 0 && utf8.RuneCountInString(out) > r.pack.MaxChars {
			return "", fmt.Errorf("render release %s: output exceeds pack max_chars=%d", doc.Releases[0].Release.Version, r.pack.MaxChars)
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

func (s *Selector) releaseDocument(manifest releases.Manifest, ancestors []releases.Manifest) (ReleaseDocument, error) {
	selected := make([]fragments.Fragment, 0, len(manifest.AddedFragmentIDs))
	for _, id := range manifest.AddedFragmentIDs {
		item, ok := s.fragmentIndex[id]
		if !ok {
			return ReleaseDocument{}, fmt.Errorf("manifest %s references missing fragment %s", manifest.Version, id)
		}
		selected = append(selected, item)
	}

	slices.SortFunc(selected, func(a, b fragments.Fragment) int {
		if !a.CreatedAt.Equal(b.CreatedAt) {
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			return 1
		}
		return strings.Compare(a.ID, b.ID)
	})

	return ReleaseDocument{
		Release:   manifest,
		Sections:  groupEntries(selected),
		Ancestors: slices.Clone(ancestors),
	}, nil
}

func (r *Renderer) renderRelease(doc ReleaseDocument) (string, error) {
	sections := make([]renderedSection, 0, len(doc.Sections))
	for _, section := range doc.Sections {
		renderedEntries := make([]string, 0, len(section.Entries))
		for _, entry := range section.Entries {
			body, err := r.renderEntry(entry)
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
		Release:  doc.Release,
		Sections: sections,
	}); err != nil {
		return "", fmt.Errorf("render release %s: %w", doc.Release.Version, err)
	}
	return strings.TrimSpace(buf.String()) + "\n", nil
}

func (r *Renderer) renderChain(doc Document) (string, error) {
	keep := len(doc.Releases)
	for keep >= 0 {
		trimmed := Document{
			Releases:            slices.Clone(doc.Releases[:keep]),
			OmittedReleaseCount: len(doc.Releases) - keep,
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
	blocks := make([]string, 0, len(doc.Releases))
	for _, releaseDoc := range doc.Releases {
		block, err := r.renderRelease(releaseDoc)
		if err != nil {
			return "", err
		}
		blocks = append(blocks, block)
	}
	return assembleDocument(r.pack, blocks, doc.OmittedReleaseCount), nil
}

func (r *Renderer) renderEntry(entry Entry) (string, error) {
	var buf bytes.Buffer
	if err := r.entryTemplate.Execute(&buf, entry.Fragment); err != nil {
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

func groupEntries(entries []fragments.Fragment) []Section {
	buckets := make(map[string][]Entry, len(sectionOrder))
	for _, entry := range entries {
		key, _ := classify(entry)
		buckets[key] = append(buckets[key], Entry{Fragment: entry})
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

func (p TemplatePack) metadataValue(key, fallback string) string {
	if value, ok := p.Metadata[key]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}
