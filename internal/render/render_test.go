package render

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
)

func TestAvailablePacksExposeBuiltInProfiles(t *testing.T) {
	packs, err := AvailablePacks(config.Default())
	if err != nil {
		t.Fatalf("AvailablePacks returned error: %v", err)
	}
	names := make([]string, 0, len(packs))
	for _, pack := range packs {
		names = append(names, pack.Name)
	}

	want := []string{
		config.RenderProfileDebianChangelog,
		config.RenderProfileGitHubRelease,
		config.RenderProfileRepositoryMarkdown,
		config.RenderProfileRPMChangelog,
		config.RenderProfileTesterSummary,
	}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("pack names = %v, want %v", names, want)
	}
}

func TestRenderBuiltInPacksProduceDistinctOutput(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	ensureBuiltinTemplates(t, repoRoot, cfg)

	record := releases.ReleaseRecord{
		Product:          "changes",
		Version:          "0.1.0",
		CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f1"},
	}
	allFragments := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC), Type: "fixed"}, Body: "first body"},
	}
	bundle, err := releases.AssembleRelease(record, []releases.ReleaseRecord{record}, allFragments)
	if err != nil {
		t.Fatalf("AssembleRelease returned error: %v", err)
	}

	githubRenderer, err := New(repoRoot, cfg, config.RenderProfileGitHubRelease)
	if err != nil {
		t.Fatalf("New github renderer returned error: %v", err)
	}
	testerRenderer, err := New(repoRoot, cfg, config.RenderProfileTesterSummary)
	if err != nil {
		t.Fatalf("New tester renderer returned error: %v", err)
	}
	debianRenderer, err := New(repoRoot, cfg, config.RenderProfileDebianChangelog)
	if err != nil {
		t.Fatalf("New debian renderer returned error: %v", err)
	}
	rpmRenderer, err := New(repoRoot, cfg, config.RenderProfileRPMChangelog)
	if err != nil {
		t.Fatalf("New rpm renderer returned error: %v", err)
	}

	doc := Document{Bundles: []releases.ReleaseBundle{bundle}}

	githubOutput, err := githubRenderer.Render(doc)
	if err != nil {
		t.Fatalf("Render github returned error: %v", err)
	}
	testerOutput, err := testerRenderer.Render(doc)
	if err != nil {
		t.Fatalf("Render tester returned error: %v", err)
	}
	debianOutput, err := debianRenderer.Render(doc)
	if err != nil {
		t.Fatalf("Render debian returned error: %v", err)
	}
	rpmOutput, err := rpmRenderer.Render(doc)
	if err != nil {
		t.Fatalf("Render rpm returned error: %v", err)
	}

	if !strings.Contains(githubOutput, "# Release 0.1.0") {
		t.Fatalf("github output missing heading: %s", githubOutput)
	}
	if !strings.Contains(testerOutput, "- first body") {
		t.Fatalf("tester output should render the body preview: %s", testerOutput)
	}
	if !strings.Contains(debianOutput, "changes (0.1.0) unstable; urgency=medium") {
		t.Fatalf("debian output missing expected header: %s", debianOutput)
	}
	if !strings.Contains(rpmOutput, "Changes Release Bot <changes@example.invalid> - 0.1.0") {
		t.Fatalf("rpm output missing expected header: %s", rpmOutput)
	}
}

func TestRenderChainDropsWholeReleaseBlocks(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	cfg.RenderProfiles = map[string]config.RenderProfile{
		config.RenderProfileRepositoryMarkdown: {MaxChars: 120},
	}
	ensureBuiltinTemplates(t, repoRoot, cfg)

	renderer, err := New(repoRoot, cfg, config.RenderProfileRepositoryMarkdown)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	doc := Document{
		Bundles: []releases.ReleaseBundle{
			{
				Release:  releases.ReleaseRecord{Product: "changes", Version: "0.2.0"},
				Sections: []releases.BundleSection{{Key: "fixed", Title: "Fixed", Entries: []releases.BundleEntry{{Fragment: fragments.Fragment{Metadata: fragments.Metadata{ID: "f1", Type: "fixed", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC)}, Body: "short"}}}}},
			},
			{
				Release:  releases.ReleaseRecord{Product: "changes", Version: "0.1.0"},
				Sections: []releases.BundleSection{{Key: "added", Title: "Added", Entries: []releases.BundleEntry{{Fragment: fragments.Fragment{Metadata: fragments.Metadata{ID: "f2", Type: "added", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC)}, Body: "long body to trigger omission over the profile max chars threshold"}}}}},
			},
		},
	}

	output, err := renderer.Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}

	if !strings.Contains(output, "## 0.2.0 (stable)") {
		t.Fatalf("output missing first release block: %s", output)
	}
	if strings.Contains(output, "long body to trigger omission") {
		t.Fatalf("output should omit the second release block entirely: %s", output)
	}
	if !strings.Contains(output, "Additional releases omitted for length.") {
		t.Fatalf("output missing omission notice: %s", output)
	}
}

func TestRepoLocalTemplateOverridesBuiltInPack(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	ensureBuiltinTemplates(t, repoRoot, cfg)

	override := "## Overridden {{ .Release.Version }}\n"
	overridePath := filepath.Join(config.TemplatesDir(repoRoot, cfg), "github-release.md.tmpl")
	if err := os.WriteFile(overridePath, []byte(override), 0o644); err != nil {
		t.Fatalf("write override template: %v", err)
	}

	renderer, err := New(repoRoot, cfg, config.RenderProfileGitHubRelease)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	doc := Document{
		Bundles: []releases.ReleaseBundle{{Release: releases.ReleaseRecord{Product: "changes", Version: "0.2.0"}}},
	}
	output, err := renderer.Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(output, "## Overridden 0.2.0") {
		t.Fatalf("render should use repo-local template override: %s", output)
	}
}

func TestRendererHelpersAndValidationPaths(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	ensureBuiltinTemplates(t, repoRoot, cfg)

	renderer, err := New(repoRoot, cfg, config.RenderProfileGitHubRelease)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if renderer.Pack().Name != config.RenderProfileGitHubRelease {
		t.Fatalf("Pack().Name = %q", renderer.Pack().Name)
	}

	if _, err := renderer.Render(Document{}); err == nil || !strings.Contains(err.Error(), "expects a single release bundle") {
		t.Fatalf("unexpected Render error: %v", err)
	}

	if got := assembleDocument(TemplatePack{DocumentHeader: "# Header", OmissionNotice: "Trimmed"}, []string{" one \n", "two\n"}, 1); got != "# Header\n\none\n\ntwo\n\nTrimmed\n" {
		t.Fatalf("assembleDocument = %q", got)
	}
	if got := indent(" one \n two ", 2); got != "  one\n  two" {
		t.Fatalf("indent = %q", got)
	}
	if got := singleLine(" one \n two "); got != "one two" {
		t.Fatalf("singleLine = %q", got)
	}
	if got := (TemplatePack{Metadata: map[string]string{"channel": "stable"}}).metadataValue("channel", "fallback"); got != "stable" {
		t.Fatalf("metadataValue = %q", got)
	}
	if got := (TemplatePack{}).metadataValue("channel", "fallback"); got != "fallback" {
		t.Fatalf("metadataValue fallback = %q", got)
	}
}

func TestRenderLoadTemplateFallbackAndErrors(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	if got, err := loadTemplate(repoRoot, cfg, "release-entry.md.tmpl"); err != nil || !strings.Contains(got, "{{") {
		t.Fatalf("loadTemplate built-in fallback = (%q, %v)", got, err)
	}
	if _, err := loadTemplate(repoRoot, cfg, "missing.tmpl"); err == nil || !strings.Contains(err.Error(), "is not available") {
		t.Fatalf("unexpected missing template error: %v", err)
	}
}

func ensureBuiltinTemplates(t *testing.T, repoRoot string, cfg config.Config) {
	t.Helper()

	dir := config.TemplatesDir(repoRoot, cfg)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir templates dir: %v", err)
	}

	names := make([]string, 0, len(BuiltinTemplateFiles()))
	for name := range BuiltinTemplateFiles() {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(BuiltinTemplateFiles()[name]), 0o644); err != nil {
			t.Fatalf("write template %s: %v", name, err)
		}
	}
}
