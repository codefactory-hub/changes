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
		"patch",
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
	if err := app.Run(context.Background(), []string{"version", "next", "--pre", "rc"}); err != nil {
		t.Fatalf("version next returned error: %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "0.1.0-rc.1" {
		t.Fatalf("version next --pre rc = %q, want 0.1.0-rc.1", got)
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
	if err := app.Run(context.Background(), []string{"release", "--pre", "rc"}); err != nil {
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
		"minor",
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
	if err := app.Run(context.Background(), []string{"release", "--pre", "rc"}); err != nil {
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
	if err := app.Run(context.Background(), []string{"release", "--pre", "beta"}); err != nil {
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
	if err := app.Run(context.Background(), []string{"release"}); err != nil {
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

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"changelog", "rebuild"}); err != nil {
		t.Fatalf("changelog rebuild returned error: %v", err)
	}
	changelogPath := strings.TrimSpace(stdout.String())
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

func TestAppResolveEmitsReleaseBundleJSON(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{0, 1, 2, 3, 4, 5, 6, 7})

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"minor",
		"--public-api", "add",
		"--type", "added",
		"--section-key", "highlights",
		"--customer-visible",
		"--release-notes-priority", "2",
		"Introduce highlights section.\n\nExpose a faster path.",
	}); err != nil {
		t.Fatalf("create returned error: %v\nstderr=%s", err, stderr.String())
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"release"}); err != nil {
		t.Fatalf("release returned error: %v", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	product := cfg.Project.Name
	companion := releases.ReleaseRecord{
		Product:          product,
		Version:          "0.1.0+docs.1",
		CreatedAt:        time.Date(2026, 4, 3, 12, 30, 0, 0, time.UTC),
		CompanionPurpose: "docs",
		SourceURL:        "https://example.invalid/docs",
	}
	if _, err := releases.Write(repoRoot, cfg, companion); err != nil {
		t.Fatalf("write companion record: %v", err)
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"resolve", "--version", "0.1.0"}); err != nil {
		t.Fatalf("resolve returned error: %v", err)
	}

	body := stdout.String()
	if !strings.Contains(body, "\"version\": \"0.1.0\"") {
		t.Fatalf("resolve output missing base version:\n%s", body)
	}
	if !strings.Contains(body, "\"version\": \"0.1.0+docs.1\"") {
		t.Fatalf("resolve output missing companion version:\n%s", body)
	}
	if !strings.Contains(body, "\"must_include_fragment_ids\": [") {
		t.Fatalf("resolve output missing must_include_fragment_ids:\n%s", body)
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
	if !strings.Contains(rootHelp, "create") || !strings.Contains(rootHelp, "render profiles") || !strings.Contains(rootHelp, "changelog rebuild") {
		t.Fatalf("root help missing commands:\n%s", rootHelp)
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

	err := app.Run(context.Background(), []string{"create", "minor"})
	if err == nil {
		t.Fatalf("create without body should fail outside TTY")
	}
	if !strings.Contains(stderr.String(), "body is required") {
		t.Fatalf("unexpected stderr for missing body:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = app.Run(context.Background(), []string{"create", "minor", "--public-api", "oops", "body"})
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
	app.Stdin = strings.NewReader("did-something-cool\nThe whirly-gig no longer breaks on Thursdays.\n")

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"create", "minor"}); err != nil {
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
	app.Stdin = strings.NewReader("did-something-cool\n")
	app.EditFile = func(path string) error {
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(raw), `bump = "minor"`) || !strings.Contains(string(raw), `# public_api = "add|change|remove"`) {
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
	if err := app.Run(context.Background(), []string{"create", "minor", "--edit"}); err != nil {
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
