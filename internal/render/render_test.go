package render

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/templates"
)

func TestAvailablePacksExposeBuiltInProfiles(t *testing.T) {
	packs := AvailablePacks(config.Default())
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

func TestSelectorProducesStableDocumentsIndependentOfPack(t *testing.T) {
	manifest := releases.Manifest{
		Version:          "0.1.0",
		TargetVersion:    "0.1.0",
		Channel:          releases.ChannelStable,
		CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f1"},
	}
	allFragments := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC), Title: "Short title", Type: "fixed", Bump: "patch"}, Body: "first body"},
	}

	selector := NewSelector(allFragments, []releases.Manifest{manifest})
	doc, err := selector.Release(manifest)
	if err != nil {
		t.Fatalf("Release returned error: %v", err)
	}

	if len(doc.Releases) != 1 {
		t.Fatalf("expected one release document, got %d", len(doc.Releases))
	}
	if got := doc.Releases[0].Sections[0].Entries[0].Fragment.Title; got != "Short title" {
		t.Fatalf("selector entry title = %q, want %q", got, "Short title")
	}
}

func TestRenderBuiltInPacksProduceDistinctOutput(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	if _, err := templates.EnsureDefaultFiles(repoRoot, cfg); err != nil {
		t.Fatalf("EnsureDefaultFiles returned error: %v", err)
	}

	manifest := releases.Manifest{
		Version:          "0.1.0",
		TargetVersion:    "0.1.0",
		Channel:          releases.ChannelStable,
		CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
		AddedFragmentIDs: []string{"f1"},
	}
	allFragments := []fragments.Fragment{
		{Metadata: fragments.Metadata{ID: "f1", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC), Title: "Short title", Type: "fixed", Bump: "patch"}, Body: "first body"},
	}
	selector := NewSelector(allFragments, []releases.Manifest{manifest})
	doc, err := selector.Release(manifest)
	if err != nil {
		t.Fatalf("Release returned error: %v", err)
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
	if strings.Contains(testerOutput, "first body") {
		t.Fatalf("tester output should use concise entry template: %s", testerOutput)
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
	profile := cfg.RenderProfiles[config.RenderProfileRepositoryMarkdown]
	profile.MaxChars = 120
	cfg.RenderProfiles[config.RenderProfileRepositoryMarkdown] = profile

	if _, err := templates.EnsureDefaultFiles(repoRoot, cfg); err != nil {
		t.Fatalf("EnsureDefaultFiles returned error: %v", err)
	}

	renderer, err := New(repoRoot, cfg, config.RenderProfileRepositoryMarkdown)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}

	doc := Document{
		Releases: []ReleaseDocument{
			{
				Release:  releases.Manifest{Version: "0.2.0", Channel: releases.ChannelStable},
				Sections: []Section{{Key: "fixed", Title: "Fixed", Entries: []Entry{{Fragment: fragments.Fragment{Metadata: fragments.Metadata{ID: "f1", Title: "Keep me", Type: "fixed", Bump: "patch", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC)}, Body: "short"}}}}},
			},
			{
				Release:  releases.Manifest{Version: "0.1.0", Channel: releases.ChannelStable},
				Sections: []Section{{Key: "added", Title: "Added", Entries: []Entry{{Fragment: fragments.Fragment{Metadata: fragments.Metadata{ID: "f2", Title: "Drop me because this block is very long", Type: "added", Bump: "minor", CreatedAt: time.Date(2026, 4, 2, 15, 0, 0, 0, time.UTC)}, Body: "long body to trigger omission over the profile max chars threshold"}}}}},
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
	if strings.Contains(output, "Drop me because") {
		t.Fatalf("output should omit the second release block entirely: %s", output)
	}
	if !strings.Contains(output, "Additional releases omitted for length.") {
		t.Fatalf("output missing omission notice: %s", output)
	}
}

func TestRepoLocalTemplateOverridesBuiltInPack(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()
	if _, err := templates.EnsureDefaultFiles(repoRoot, cfg); err != nil {
		t.Fatalf("EnsureDefaultFiles returned error: %v", err)
	}

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
		Releases: []ReleaseDocument{{Release: releases.Manifest{Version: "0.2.0"}}},
	}
	output, err := renderer.Render(doc)
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	if !strings.Contains(output, "## Overridden 0.2.0") {
		t.Fatalf("render should use repo-local template override: %s", output)
	}
}
