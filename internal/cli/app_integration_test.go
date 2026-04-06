package cli

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
)

func TestAppEndToEnd(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}

	assertExists(t, filepath.Join(repoRoot, ".config/changes/config.toml"))
	assertNotExists(t, config.HistoryImportPromptPath(repoRoot, config.Default()))
	assertExists(t, filepath.Join(repoRoot, ".local/share/changes/templates/repository-markdown-release.md.tmpl"))

	gitignore, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), "/.local/state/") {
		t.Fatalf(".gitignore missing state dir entry: %s", gitignore)
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 15, 30, 45, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{0, 1, 2, 3})
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"--type", "fixed",
		"--scope", "release",
		"Fix release note rendering.\n\nRender whole entries only.",
	}); err != nil {
		t.Fatalf("create returned error: %v\nstderr=%s", err, stderr.String())
	}

	fragmentPath := strings.TrimSpace(stdout.String())
	assertExists(t, fragmentPath)
	if !regexp.MustCompile(`[a-z]+-[a-z]+-[a-z]+\.md$`).MatchString(fragmentPath) {
		t.Fatalf("fragment path %q did not match expected pattern", fragmentPath)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	status := stdout.String()
	if !strings.Contains(status, "Unreleased fragments: 1") || !strings.Contains(status, "Recommended next stable: 0.1.0") {
		t.Fatalf("unexpected status output:\n%s", status)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"render", "profiles"}); err != nil {
		t.Fatalf("render profiles returned error: %v", err)
	}
	profilesOut := stdout.String()
	for _, name := range []string{
		config.RenderProfileRepositoryMarkdown,
		config.RenderProfileGitHubRelease,
		config.RenderProfileTesterSummary,
		config.RenderProfileDebianChangelog,
		config.RenderProfileRPMChangelog,
	} {
		if !strings.Contains(profilesOut, name) {
			t.Fatalf("render profiles missing %s:\n%s", name, profilesOut)
		}
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"release", "create", "--pre", "rc"}); err == nil {
		t.Fatalf("release create should be rejected after the CLI refactor")
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 16, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{"release", "--accept", "--pre", "rc"}); err != nil {
		t.Fatalf("release --pre rc returned error: %v", err)
	}
	rc1Path := strings.TrimSpace(stdout.String())
	rc1Record, err := releases.Load(rc1Path)
	if err != nil {
		t.Fatalf("load rc1 release record: %v", err)
	}
	if rc1Record.ParentVersion != "" {
		t.Fatalf("first prerelease should not have a parent, got %q", rc1Record.ParentVersion)
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 16, 30, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{8, 9, 10, 11})
	if err := app.Run(context.Background(), []string{
		"create",
		"--public-api", "add",
		"--type", "added",
		"Add tester profile.\n\nIntroduce concise tester rendering.",
	}); err != nil {
		t.Fatalf("second create returned error: %v\nstderr=%s", err, stderr.String())
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 17, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{"release", "--accept", "--pre", "rc"}); err != nil {
		t.Fatalf("second release --pre rc returned error: %v", err)
	}
	rc2Path := strings.TrimSpace(stdout.String())
	rc2Record, err := releases.Load(rc2Path)
	if err != nil {
		t.Fatalf("load rc2 release record: %v", err)
	}
	if rc2Record.ParentVersion != rc1Record.Version {
		t.Fatalf("rc2 parent = %q, want %q", rc2Record.ParentVersion, rc1Record.Version)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"render", "--version", "0.1.0-rc.2", "--profile", config.RenderProfileGitHubRelease}); err != nil {
		t.Fatalf("render github returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "# Release 0.1.0-rc.2") {
		t.Fatalf("github render missing expected heading:\n%s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Add tester profile.") {
		t.Fatalf("github render should include only the second release delta:\n%s", stdout.String())
	}
	if strings.Contains(stdout.String(), "Fix release note rendering.") {
		t.Fatalf("github render should not include the parent delta:\n%s", stdout.String())
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 17, 30, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{"release", "--accept", "--pre", "beta"}); err != nil {
		t.Fatalf("release --pre beta returned error: %v", err)
	}
	beta1Path := strings.TrimSpace(stdout.String())
	beta1Record, err := releases.Load(beta1Path)
	if err != nil {
		t.Fatalf("load beta1 release record: %v", err)
	}
	if beta1Record.ParentVersion != "" {
		t.Fatalf("beta1 should not inherit the rc lineage, got parent %q", beta1Record.ParentVersion)
	}
	if got := beta1Record.AddedFragmentIDs; len(got) != 2 {
		t.Fatalf("beta1 added_fragment_ids = %#v, want both fragments", got)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"render", "--version", "0.1.0-beta.1", "--profile", config.RenderProfileGitHubRelease}); err != nil {
		t.Fatalf("render beta github returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Fix release note rendering.") || !strings.Contains(stdout.String(), "Add tester profile.") {
		t.Fatalf("beta render should include both final-unreleased fragments:\n%s", stdout.String())
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status after prereleases returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Active prerelease heads:") || !strings.Contains(stdout.String(), "0.1.0-rc.2 -> 0.1.0") || !strings.Contains(stdout.String(), "0.1.0-beta.1 -> 0.1.0") {
		t.Fatalf("unexpected prerelease status output:\n%s", stdout.String())
	}

	stdout.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{"release", "--accept"}); err != nil {
		t.Fatalf("release returned error: %v", err)
	}
	stablePath := strings.TrimSpace(stdout.String())
	stableRecord, err := releases.Load(stablePath)
	if err != nil {
		t.Fatalf("load stable release record: %v", err)
	}
	if stableRecord.ParentVersion != "" {
		t.Fatalf("first stable should not have a parent, got %q", stableRecord.ParentVersion)
	}
	if got := stableRecord.AddedFragmentIDs; len(got) != 2 {
		t.Fatalf("stable added_fragment_ids = %#v, want both fragments", got)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"render", "--version", "0.1.0", "--profile", config.RenderProfileDebianChangelog}); err != nil {
		t.Fatalf("render debian returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "changes (0.1.0) unstable; urgency=medium") {
		t.Fatalf("debian render missing expected header:\n%s", stdout.String())
	}

	changelogPath := filepath.Join(repoRoot, "CHANGELOG.rendered.md")
	if err := app.Run(context.Background(), []string{"render", "--latest", "--profile", config.RenderProfileRepositoryMarkdown, "--output", changelogPath}); err != nil {
		t.Fatalf("render latest repository markdown returned error: %v", err)
	}
	changelogBytes, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("read changelog: %v", err)
	}
	if !strings.Contains(string(changelogBytes), "# Changelog") || !strings.Contains(string(changelogBytes), "## 0.1.0 (stable)") {
		t.Fatalf("changelog missing expected content:\n%s", changelogBytes)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status after stable returned error: %v", err)
	}
	if !strings.Contains(stdout.String(), "Unreleased fragments: 0") {
		t.Fatalf("unexpected post-release status:\n%s", stdout.String())
	}
}

func TestAppHelpSurface(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)

	if err := app.Run(context.Background(), []string{"--help"}); err != nil {
		t.Fatalf("root --help returned error: %v", err)
	}
	rootHelp := stdout.String()
	if !strings.Contains(rootHelp, "Usage:") || !strings.Contains(rootHelp, "changes <command> [options]") {
		t.Fatalf("root help missing usage:\n%s", rootHelp)
	}
	if !strings.Contains(rootHelp, "create") || !strings.Contains(rootHelp, "render profiles") {
		t.Fatalf("root help missing commands:\n%s", rootHelp)
	}
	if strings.Contains(rootHelp, "version next") {
		t.Fatalf("root help should not mention version next:\n%s", rootHelp)
	}
	if strings.Contains(rootHelp, "resolve") || strings.Contains(rootHelp, "changelog rebuild") {
		t.Fatalf("root help should not expose resolve or changelog rebuild:\n%s", rootHelp)
	}
	if stderr.Len() != 0 {
		t.Fatalf("root --help should not write stderr:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"help", "render"}); err != nil {
		t.Fatalf("help render returned error: %v", err)
	}
	renderHelp := stdout.String()
	if !strings.Contains(renderHelp, "changes render --version <version>") {
		t.Fatalf("render help missing version usage:\n%s", renderHelp)
	}
	if !strings.Contains(renderHelp, "changes render --latest") {
		t.Fatalf("render help missing latest usage:\n%s", renderHelp)
	}
	if !strings.Contains(renderHelp, "changes render --record <path>") {
		t.Fatalf("render help missing record usage:\n%s", renderHelp)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"render", "--help"}); err != nil {
		t.Fatalf("render --help returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Render one release or a release lineage") {
		t.Fatalf("render --help missing description:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("render --help should not write stderr:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err := app.Run(context.Background(), []string{"resolve"})
	if err == nil || !strings.Contains(stderr.String(), `unknown command "resolve"`) {
		t.Fatalf("resolve should be rejected:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = app.Run(context.Background(), []string{"changelog", "rebuild"})
	if err == nil || !strings.Contains(stderr.String(), `unknown command "changelog"`) {
		t.Fatalf("changelog should be rejected:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestStatusExplainShowsPolicyEvidence(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{7, 8, 9})

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--public-api", "change",
		"Shift the parser API to a new shape.",
	}); err != nil {
		t.Fatalf("create returned error: %v\nstderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"status", "--explain"}); err != nil {
		t.Fatalf("status --explain returned error: %v\nstderr=%s", err, stderr.String())
	}

	body := stdout.String()
	if !strings.Contains(body, "Policy evidence:") {
		t.Fatalf("missing policy explanation:\n%s", body)
	}
	if !strings.Contains(body, "Public API policy: unstable") {
		t.Fatalf("missing configured public API policy:\n%s", body)
	}
	if !strings.Contains(body, "Recommended bump: minor") {
		t.Fatalf("missing suggested bump:\n%s", body)
	}
	if !strings.Contains(body, "public_api=change implies minor while public API policy is unstable") {
		t.Fatalf("missing fragment reason:\n%s", body)
	}
}

func TestInitDefaultsToUnreleasedWithoutBootstrapPrompt(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 8, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 2, 3, 4})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	promptPath := config.HistoryImportPromptPath(repoRoot, cfg)
	assertNotExists(t, promptPath)

	records, err := releases.List(repoRoot, cfg)
	if err != nil {
		t.Fatalf("list release records: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("unexpected release records: %#v", records)
	}
}

func TestInitCurrentVersionCreatesAdoptionBootstrap(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 8, 30, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{5, 6, 7, 8, 9, 10})

	if err := app.Run(context.Background(), []string{"init", "--current-version", "2.7.4"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	promptPath := config.HistoryImportPromptPath(repoRoot, cfg)
	assertExists(t, promptPath)

	promptBytes, err := os.ReadFile(promptPath)
	if err != nil {
		t.Fatalf("read prompt: %v", err)
	}
	prompt := string(promptBytes)
	if !strings.Contains(prompt, "Current version supplied during `changes init`: `2.7.4`") {
		t.Fatalf("prompt missing current version context:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Standard adoption release exists: yes (`2.7.4`)") {
		t.Fatalf("prompt missing adoption bootstrap context:\n%s", prompt)
	}

	records, err := releases.List(repoRoot, cfg)
	if err != nil {
		t.Fatalf("list release records: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("release record count = %d, want 1", len(records))
	}
	record := records[0]
	if !record.Bootstrap || record.Version != "2.7.4" {
		t.Fatalf("unexpected bootstrap record: %#v", record)
	}
	if len(record.AddedFragmentIDs) != 1 {
		t.Fatalf("bootstrap added_fragment_ids = %#v, want 1", record.AddedFragmentIDs)
	}

	allFragments, err := fragments.List(repoRoot, cfg)
	if err != nil {
		t.Fatalf("list fragments: %v", err)
	}
	if len(allFragments) != 1 {
		t.Fatalf("fragment count = %d, want 1", len(allFragments))
	}
	if !allFragments[0].Bootstrap {
		t.Fatalf("expected bootstrap fragment, got %#v", allFragments[0].Metadata)
	}
	if !strings.Contains(allFragments[0].Body, "adopted `changes` at version 2.7.4") {
		t.Fatalf("unexpected bootstrap fragment body:\n%s", allFragments[0].Body)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Unreleased fragments: 0") || !strings.Contains(got, "Recommended next stable: 2.7.4") {
		t.Fatalf("unexpected status output:\n%s", got)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"render", "--version", "2.7.4", "--profile", config.RenderProfileGitHubRelease}); err != nil {
		t.Fatalf("render returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "adopted `changes` at version 2.7.4") {
		t.Fatalf("render missing adoption text:\n%s", got)
	}

	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 9, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix the first post-adoption bug.",
	}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status after create returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Recommended next stable: 2.7.5") {
		t.Fatalf("unexpected post-adoption status:\n%s", got)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"release", "--accept"}); err != nil {
		t.Fatalf("release returned error: %v", err)
	}
	releasePath := strings.TrimSpace(strings.Split(stdout.String(), "\n")[0])
	released, err := releases.Load(releasePath)
	if err != nil {
		t.Fatalf("load post-adoption release: %v", err)
	}
	if released.Version != "2.7.5" || released.ParentVersion != "2.7.4" {
		t.Fatalf("unexpected post-adoption release record: %#v", released)
	}
}

func TestInitTreatsZeroVersionAsUnreleased(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 8, 45, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{3, 4, 5, 6})

	if err := app.Run(context.Background(), []string{"init", "--current-version", "0.0.0"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	records, err := releases.List(repoRoot, cfg)
	if err != nil {
		t.Fatalf("list release records: %v", err)
	}
	if len(records) != 0 {
		t.Fatalf("0.0.0 should behave like unreleased; got %#v", records)
	}
}

func TestInitPromptsForCurrentVersionInTTY(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 8, 50, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{7, 7, 7})
	app.IsTTY = func() bool { return true }
	app.Stdin = strings.NewReader("\n")

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if !strings.Contains(stderr.String(), "Current released version [unreleased]: ") {
		t.Fatalf("missing init prompt:\n%s", stderr.String())
	}
}

func TestInitFailsWhenBootstrapArtifactsAlreadyExist(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 9, 15, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 1, 1, 1})

	if err := app.Run(context.Background(), []string{"init", "--current-version", "2.7.4"}); err != nil {
		t.Fatalf("first init returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	err := app.Run(context.Background(), []string{"init"})
	if err == nil {
		t.Fatalf("re-init should fail once bootstrap artifacts exist")
	}
	if !strings.Contains(stderr.String(), "bootstrap adoption artifacts already exist") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestReleaseRequiresDecisionOutsideTTY(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 10, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{9, 8, 7})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix the parser edge case.",
	}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	err := app.Run(context.Background(), []string{"release"})
	if err == nil {
		t.Fatalf("release should require an explicit decision outside TTY")
	}
	if !strings.Contains(stderr.String(), "non-interactive use requires --accept or --override") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestReleaseRequiresExplicitBumpWhenNoImpactIsInferred(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 10, 30, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{5, 4, 3})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{"create", "Document an internal refactor."}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"status", "--explain"}); err != nil {
		t.Fatalf("status --explain returned error: %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Recommended bump: none") || !strings.Contains(got, "no semantic levers present; no release impact was inferred") {
		t.Fatalf("unexpected status output:\n%s", got)
	}

	stdout.Reset()
	stderr.Reset()
	err := app.Run(context.Background(), []string{"release", "--accept"})
	if err == nil {
		t.Fatalf("release --accept should fail when no bump is inferred")
	}
	if !strings.Contains(stderr.String(), "no version bump was inferred; use --override --bump or --override --version") {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func TestReleasePromptsInTTYAndAllowsOverride(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 11, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 3, 5, 7, 9, 11})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix the first parser bug.",
	}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{"release", "--accept"}); err != nil {
		t.Fatalf("initial release returned error: %v", err)
	}

	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 12, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix the second parser bug.",
	}); err != nil {
		t.Fatalf("second create returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	app.IsTTY = func() bool { return true }
	app.Stdin = strings.NewReader("minor\n")
	if err := app.Run(context.Background(), []string{"release"}); err != nil {
		t.Fatalf("interactive release returned error: %v\nstderr=%s", err, stderr.String())
	}

	body := stdout.String()
	if !strings.Contains(body, "Policy evidence:") || !strings.Contains(body, "Default release: 0.1.1") {
		t.Fatalf("interactive release missing evidence:\n%s", body)
	}
	if !strings.Contains(stderr.String(), "Press Enter to accept the recommendation, choose patch/minor/major to override, or type cancel: ") {
		t.Fatalf("interactive release missing prompt wording:\n%s", stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(body), "\n")
	releasePath := lines[len(lines)-1]
	record, err := releases.Load(releasePath)
	if err != nil {
		t.Fatalf("load release record: %v", err)
	}
	if record.Version != "0.2.0" {
		t.Fatalf("interactive override version = %q, want 0.2.0", record.Version)
	}
}

func TestReleasePromptCanCancel(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 13, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{2, 4, 6})
	app.IsTTY = func() bool { return true }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix the cancel test bug.",
	}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	app.Stdin = strings.NewReader("cancel\n")
	err := app.Run(context.Background(), []string{"release"})
	if err == nil || !strings.Contains(err.Error(), "release canceled") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestReleaseValidatesAcceptAndOverrideFlags(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 13, 30, 0, 0, time.UTC)
	}
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "override requires detail",
			args: []string{"release", "--override"},
			want: "release: --override requires --bump or --version",
		},
		{
			name: "accept cannot combine with override",
			args: []string{"release", "--accept", "--override", "--bump", "minor"},
			want: "release: --accept and --override cannot be combined",
		},
		{
			name: "accept cannot combine with bump",
			args: []string{"release", "--accept", "--bump", "minor"},
			want: "release: --accept cannot be combined with --bump",
		},
		{
			name: "accept cannot combine with version",
			args: []string{"release", "--accept", "--version", "1.2.0"},
			want: "release: --accept cannot be combined with --version",
		},
		{
			name: "override cannot combine bump and version",
			args: []string{"release", "--override", "--bump", "minor", "--version", "1.2.0"},
			want: "release: --version cannot be combined with --bump",
		},
		{
			name: "pre cannot combine with override version",
			args: []string{"release", "--override", "--version", "1.2.0-beta.3", "--pre", "beta"},
			want: "release: --pre cannot be combined with --override --version",
		},
		{
			name: "bump requires override",
			args: []string{"release", "--bump", "minor"},
			want: "release: --bump requires --override",
		},
		{
			name: "version requires override",
			args: []string{"release", "--version", "1.2.0"},
			want: "release: --version requires --override",
		},
	}

	for _, tc := range cases {
		stdout.Reset()
		stderr.Reset()
		err := app.Run(context.Background(), tc.args)
		if err == nil || !strings.Contains(stderr.String(), tc.want) {
			t.Fatalf("%s: unexpected error\nstdout=%s\nstderr=%s\nerr=%v", tc.name, stdout.String(), stderr.String(), err)
		}
	}
}

func TestReleaseSupportsOverrideBumpAndExactVersion(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 13, 45, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{6, 5, 4, 3, 2, 1, 7, 8, 9, 10, 11, 12, 13, 14, 15})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix parser stability.",
	}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"release", "--override", "--bump", "minor"}); err != nil {
		t.Fatalf("override bump release returned error: %v\nstderr=%s", err, stderr.String())
	}
	recordPath := strings.TrimSpace(stdout.String())
	record, err := releases.Load(recordPath)
	if err != nil {
		t.Fatalf("load override bump release: %v", err)
	}
	if record.Version != "0.1.0" {
		t.Fatalf("override bump version = %q, want 0.1.0", record.Version)
	}

	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 14, 0, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--public-api", "add",
		"Add parser extension point.",
	}); err != nil {
		t.Fatalf("second create returned error: %v", err)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"release", "--override", "--bump", "minor", "--pre", "beta"}); err != nil {
		t.Fatalf("override prerelease returned error: %v\nstderr=%s", err, stderr.String())
	}
	recordPath = strings.TrimSpace(stdout.String())
	record, err = releases.Load(recordPath)
	if err != nil {
		t.Fatalf("load override prerelease: %v", err)
	}
	if record.Version != "0.2.0-beta.1" {
		t.Fatalf("override prerelease version = %q, want 0.2.0-beta.1", record.Version)
	}

	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 14, 15, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix parser docs formatting.",
	}); err != nil {
		t.Fatalf("third create returned error: %v", err)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"release", "--override", "--version", "1.2.0"}); err != nil {
		t.Fatalf("override exact version returned error: %v\nstderr=%s", err, stderr.String())
	}
	recordPath = strings.TrimSpace(stdout.String())
	record, err = releases.Load(recordPath)
	if err != nil {
		t.Fatalf("load exact version release: %v", err)
	}
	if record.Version != "1.2.0" {
		t.Fatalf("exact version release = %q, want 1.2.0", record.Version)
	}

	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 14, 30, 0, 0, time.UTC)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix parser prerelease formatting.",
	}); err != nil {
		t.Fatalf("fourth create returned error: %v", err)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"release", "--override", "--version", "1.3.0-beta.3"}); err != nil {
		t.Fatalf("override exact prerelease returned error: %v\nstderr=%s", err, stderr.String())
	}
	recordPath = strings.TrimSpace(stdout.String())
	record, err = releases.Load(recordPath)
	if err != nil {
		t.Fatalf("load exact prerelease release: %v", err)
	}
	if record.Version != "1.3.0-beta.3" {
		t.Fatalf("exact prerelease release = %q, want 1.3.0-beta.3", record.Version)
	}
}

func TestCreateRequiresExplicitBodyOutsideTTY(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 3, 9, 0, 0, 0, time.UTC)
	}
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	err := app.Run(context.Background(), []string{"create"})
	if err == nil {
		t.Fatalf("create without body should fail outside TTY")
	}
	if !strings.Contains(stderr.String(), "body is required") {
		t.Fatalf("unexpected stderr for missing body:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = app.Run(context.Background(), []string{"create", "--public-api", "oops", "body"})
	if err == nil {
		t.Fatalf("create with invalid public-api should fail")
	}
	if !strings.Contains(stderr.String(), "public_api must be one of") {
		t.Fatalf("unexpected stderr for invalid public-api:\n%s", stderr.String())
	}
}

func TestCreatePromptsForMissingFieldsInTTY(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 3, 10, 15, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 2, 3})
	app.IsTTY = func() bool { return true }
	app.Stdin = strings.NewReader("\n")

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = strings.NewReader("did-something-cool\nThe whirly-gig no longer breaks on Thursdays.\n")
	if err := app.Run(context.Background(), []string{"create"}); err != nil {
		t.Fatalf("interactive create returned error: %v\nstderr=%s", err, stderr.String())
	}

	fragmentPath := strings.TrimSpace(stdout.String())
	assertExists(t, fragmentPath)
	if !regexp.MustCompile(`20260403-101500--did-something-cool--[a-z]+-[a-z]+-[a-z]+\.md$`).MatchString(fragmentPath) {
		t.Fatalf("fragment path %q did not include timestamp/name/suffix", fragmentPath)
	}

	raw, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Fatalf("read fragment: %v", err)
	}
	if !strings.Contains(string(raw), `type = "changed"`) || !strings.Contains(string(raw), "The whirly-gig no longer breaks on Thursdays.") {
		t.Fatalf("interactive fragment missing prompted content:\n%s", raw)
	}
}

func TestCreateEditUsesScaffoldedFrontMatter(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 3, 11, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{4, 5, 6})
	app.IsTTY = func() bool { return true }
	app.Stdin = strings.NewReader("\n")
	app.EditFile = func(path string) error {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(raw), `# public_api = "add|change|remove"`) {
			t.Fatalf("scaffold missing expected front matter:\n%s", raw)
		}
		edited := strings.TrimSpace(string(raw)) + "\n\nAdd the whirly-gig fix.\n\nIt now behaves on Thursdays.\n"
		return os.WriteFile(path, []byte(edited), 0o644)
	}

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = strings.NewReader("did-something-cool\n")
	if err := app.Run(context.Background(), []string{"create", "--edit"}); err != nil {
		t.Fatalf("create --edit returned error: %v\nstderr=%s", err, stderr.String())
	}

	fragmentPath := strings.TrimSpace(stdout.String())
	assertExists(t, fragmentPath)
	raw, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Fatalf("read fragment: %v", err)
	}
	if !strings.Contains(string(raw), "Add the whirly-gig fix.") {
		t.Fatalf("edited fragment missing body:\n%s", raw)
	}
}

func TestAddCommandIsRejected(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	err := app.Run(context.Background(), []string{"add"})
	if err == nil {
		t.Fatalf("add should be rejected")
	}
	if !strings.Contains(stderr.String(), `unknown command "add"`) {
		t.Fatalf("unexpected stderr:\n%s", stderr.String())
	}
}

func gitInit(t *testing.T, dir string) {
	t.Helper()

	cmd := exec.Command("git", "init")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v\n%s", err, out)
	}
}

func assertExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertNotExists(t *testing.T, path string) {
	t.Helper()

	if _, err := os.Stat(path); err == nil {
		t.Fatalf("expected %s not to exist", path)
	} else if !os.IsNotExist(err) {
		t.Fatalf("stat %s: %v", path, err)
	}
}
