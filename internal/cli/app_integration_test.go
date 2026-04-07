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
	assertNotExists(t, filepath.Join(repoRoot, ".local/share/changes/templates/repository-markdown-release.md.tmpl"))

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
	if !strings.Contains(status, "Current version: unreleased") ||
		!strings.Contains(status, "Initial release target: 0.1.0") ||
		!strings.Contains(status, "Unreleased fragments: 1") ||
		!strings.Contains(status, "Recommended next final: 0.1.0") {
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
	if !strings.Contains(rootHelp, "create") || !strings.Contains(rootHelp, "render profiles") || !strings.Contains(rootHelp, "doctor") || !strings.Contains(rootHelp, "version") {
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

func TestAppVersionSurface(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Version = "0.1.0"
	app.Commit = "abc123"
	app.Date = "2026-04-07T12:00:00Z"

	if err := app.Run(context.Background(), []string{"--version"}); err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if got := stdout.String(); got != "changes 0.1.0\n" {
		t.Fatalf("--version output = %q, want %q", got, "changes 0.1.0\n")
	}

	stdout.Reset()
	if err := app.Run(context.Background(), []string{"version"}); err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if got := stdout.String(); got != "changes 0.1.0\n" {
		t.Fatalf("version output = %q, want %q", got, "changes 0.1.0\n")
	}
}

func TestAppVersionDefaultsToDev(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)

	if err := app.Run(context.Background(), []string{"--version"}); err != nil {
		t.Fatalf("--version returned error: %v", err)
	}
	if got := stdout.String(); got != "changes dev\n" {
		t.Fatalf("--version output = %q, want %q", got, "changes dev\n")
	}
}

func TestInitHelpSurfaceIncludesLayoutFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)

	if err := app.Run(context.Background(), []string{"help", "init"}); err != nil {
		t.Fatalf("help init returned error: %v", err)
	}
	initHelp := stdout.String()
	if !strings.Contains(initHelp, "changes init [--current-version <semver|unreleased>] [--layout xdg|home] [--home PATH]") {
		t.Fatalf("init help missing layout flags:\n%s", initHelp)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"help", "init", "global"}); err != nil {
		t.Fatalf("help init global returned error: %v", err)
	}
	globalHelp := stdout.String()
	if !strings.Contains(globalHelp, "changes init global [--layout xdg|home] [--home PATH]") {
		t.Fatalf("init global help missing layout flags:\n%s", globalHelp)
	}
	if strings.Contains(globalHelp, "--current-version") {
		t.Fatalf("init global help should not mention --current-version:\n%s", globalHelp)
	}
}

func TestInitGlobalHomeReportsResolvedPaths(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("CHANGES_HOME", filepath.Join(homeDir, ".changes-home-env"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(homeDir, ".config-home"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(homeDir, ".data-home"))
	t.Setenv("XDG_STATE_HOME", filepath.Join(homeDir, ".state-home"))

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 7, 3, 15, 0, 0, time.UTC)
	}
	app.IsTTY = func() bool { return false }

	customHome := filepath.Join(homeDir, ".changes-global")
	if err := app.Run(context.Background(), []string{"init", "global", "--layout", "home", "--home", customHome}); err != nil {
		t.Fatalf("init global returned error: %v\nstderr=%s", err, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "initialized global home") {
		t.Fatalf("init global stdout missing layout summary:\n%s", output)
	}
	for _, line := range []string{
		"config: " + filepath.Join(customHome, "config"),
		"data: " + filepath.Join(customHome, "data"),
		"state: " + filepath.Join(customHome, "state"),
	} {
		if !strings.Contains(output, line) {
			t.Fatalf("init global stdout missing %q:\n%s", line, output)
		}
	}
}

func TestInitRejectsHomeFlagWithoutHomeLayout(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.IsTTY = func() bool { return false }

	err := app.Run(context.Background(), []string{"init", "--home", ".changes"})
	if err == nil {
		t.Fatalf("init --home without home layout returned nil error")
	}
	if !strings.Contains(stderr.String(), "--home is only valid when --layout home") {
		t.Fatalf("unexpected stderr for repo init:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	err = app.Run(context.Background(), []string{"init", "global", "--home", filepath.Join(t.TempDir(), ".changes-global")})
	if err == nil {
		t.Fatalf("init global --home without home layout returned nil error")
	}
	if !strings.Contains(stderr.String(), "--home is only valid when --layout home") {
		t.Fatalf("unexpected stderr for global init:\n%s", stderr.String())
	}
}

func TestAppInitUsesGlobalRepoInitDefaultsWhenPresent(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome := configureGlobalLayoutEnv(t)
	if err := writeManagedGlobalRepoInitDefaults(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, config.StyleXDG, "[repo.init]\nstyle = \"home\"\nhome = \".changes-global\"\n"); err != nil {
		t.Fatalf("write managed global defaults: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 7, 3, 40, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 2, 3, 4})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}

	body := stdout.String()
	if !strings.Contains(body, "layout: home") {
		t.Fatalf("init stdout missing selected home layout:\n%s", body)
	}
	if !strings.Contains(body, "config: .changes-global/config") {
		t.Fatalf("init stdout missing global-default config path:\n%s", body)
	}
	if stderr.Len() != 0 {
		t.Fatalf("init stderr should be empty for one authoritative global default:\n%s", stderr.String())
	}
}

func TestDoctorDefaultsToRepoScopeInsideRepository(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 6, 9, 0, 0, 0, time.UTC)
	}

	if err := app.Run(context.Background(), []string{"doctor"}); err != nil {
		t.Fatalf("doctor returned error: %v\nstderr=%s", err, stderr.String())
	}

	if got := stdout.String(); !strings.Contains(got, "repo: authoritative xdg") {
		t.Fatalf("doctor stdout missing repo default output:\n%s", got)
	}
	if strings.Contains(stdout.String(), "global:") {
		t.Fatalf("doctor should not inspect global by default inside a repo:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("doctor should not write stderr for normal inspection:\n%s", stderr.String())
	}
}

func TestDoctorDefaultOutputStaysConciseAndExplainAddsDetail(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write legacy home repo config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 6, 9, 5, 0, 0, time.UTC)
	}

	if err := app.Run(context.Background(), []string{"doctor"}); err != nil {
		t.Fatalf("doctor returned error: %v\nstderr=%s", err, stderr.String())
	}
	concise := stdout.String()
	if !strings.Contains(concise, "repo: authoritative xdg") {
		t.Fatalf("concise output missing status line:\n%s", concise)
	}
	if strings.Contains(concise, "Candidates:") || strings.Contains(concise, "Precedence inputs:") {
		t.Fatalf("concise output should stay terse:\n%s", concise)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"doctor", "--explain"}); err != nil {
		t.Fatalf("doctor --explain returned error: %v\nstderr=%s", err, stderr.String())
	}
	explain := stdout.String()
	if !strings.Contains(explain, "Precedence inputs:") || !strings.Contains(explain, "Candidates:") {
		t.Fatalf("explain output missing detail:\n%s", explain)
	}
	if !strings.Contains(explain, ".changes/config") || !strings.Contains(explain, "Warnings:") {
		t.Fatalf("explain output missing candidate or warning detail:\n%s", explain)
	}
}

func TestDoctorJSONOutputReturnsStructuredInspection(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 6, 9, 10, 0, 0, time.UTC)
	}

	if err := app.Run(context.Background(), []string{"doctor", "--json"}); err != nil {
		t.Fatalf("doctor --json returned error: %v\nstderr=%s", err, stderr.String())
	}

	body := stdout.String()
	for _, fragment := range []string{
		`"requested_scope":"repo"`,
		`"repo":`,
		`"selected_style":"xdg"`,
		`"summary":`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("json output missing %q:\n%s", fragment, body)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("doctor --json should not write stderr:\n%s", stderr.String())
	}
}

func TestDoctorHelpSurfaceIncludesInspectionAndMigrationFlags(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)

	if err := app.Run(context.Background(), []string{"help", "doctor"}); err != nil {
		t.Fatalf("help doctor returned error: %v", err)
	}

	body := stdout.String()
	for _, line := range []string{
		"changes doctor [--scope global|repo|all] [--explain] [--json]",
		"changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]",
		"--scope <global|repo|all>",
		"--migration-prompt",
		"--to <xdg|home>",
		"--output <path>",
	} {
		if !strings.Contains(body, line) {
			t.Fatalf("doctor help missing %q:\n%s", line, body)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("help doctor should not write stderr:\n%s", stderr.String())
	}
}

func TestDoctorMigrationPromptWritesStdoutByDefault(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".config", "changes", "extra.toml"), []byte("extra"), 0o644); err != nil {
		t.Fatalf("write extra config artifact: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 6, 9, 15, 0, 0, time.UTC)
	}

	if err := app.Run(context.Background(), []string{"doctor", "--migration-prompt", "--scope", "repo", "--to", "home", "--home", ".changes-next"}); err != nil {
		t.Fatalf("doctor migration prompt returned error: %v\nstderr=%s", err, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "## Requested Migration") || !strings.Contains(got, "## Safety Rules") {
		t.Fatalf("migration prompt stdout missing markdown body:\n%s", got)
	}
	if stderr.Len() != 0 {
		t.Fatalf("doctor migration prompt should not write stderr:\n%s", stderr.String())
	}
}

func TestDoctorMigrationPromptWritesFileWhenOutputIsSet(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}

	outputPath := filepath.Join(repoRoot, "doctor-brief.md")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 6, 9, 20, 0, 0, time.UTC)
	}

	if err := app.Run(context.Background(), []string{"doctor", "--migration-prompt", "--scope", "repo", "--to", "home", "--output", outputPath}); err != nil {
		t.Fatalf("doctor migration prompt returned error: %v\nstderr=%s", err, stderr.String())
	}

	body := strings.TrimSpace(stdout.String())
	if !strings.Contains(body, "wrote migration prompt to") || strings.Contains(body, "## Requested Migration") {
		t.Fatalf("stdout should contain only a concise success line:\n%s", body)
	}
	raw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}
	if !strings.Contains(string(raw), "## Requested Migration") {
		t.Fatalf("output file missing markdown body:\n%s", string(raw))
	}
	if stderr.Len() != 0 {
		t.Fatalf("doctor migration prompt should not write stderr:\n%s", stderr.String())
	}
}

func TestCLIWarnsOrFailsConsistentlyAcrossFocusedRolloutScenarios(t *testing.T) {
	t.Run("default repo init reports xdg", func(t *testing.T) {
		repoRoot := t.TempDir()
		gitInit(t, repoRoot)
		t.Chdir(repoRoot)
		t.Setenv("CHANGES_HOME", "")
		t.Setenv("XDG_CONFIG_HOME", "")
		t.Setenv("XDG_DATA_HOME", "")
		t.Setenv("XDG_STATE_HOME", "")

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		app := NewApp(&stdout, &stderr)
		app.Now = func() time.Time {
			return time.Date(2026, 4, 7, 3, 45, 0, 0, time.UTC)
		}
		app.Random = bytes.NewReader([]byte{4, 5, 6, 7})
		app.IsTTY = func() bool { return false }

		if err := app.Run(context.Background(), []string{"init"}); err != nil {
			t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
		}
		if !strings.Contains(stdout.String(), "layout: xdg") {
			t.Fatalf("init stdout missing xdg layout:\n%s", stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("init stderr should be empty:\n%s", stderr.String())
		}
	})

	t.Run("legacy repo status fails with doctor guidance", func(t *testing.T) {
		repoRoot := t.TempDir()
		gitInit(t, repoRoot)
		t.Chdir(repoRoot)

		if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
			t.Fatalf("write legacy xdg config: %v", err)
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		app := NewApp(&stdout, &stderr)
		err := app.Run(context.Background(), []string{"status"})
		if err == nil {
			t.Fatalf("status returned nil error")
		}
		if !strings.Contains(stderr.String(), "error: repo authority is legacy-only") || !strings.Contains(stderr.String(), "changes doctor --scope repo --repair") {
			t.Fatalf("status stderr missing legacy doctor guidance:\n%s", stderr.String())
		}
		if stdout.Len() != 0 {
			t.Fatalf("status stdout should be empty on failure:\n%s", stdout.String())
		}
	})

	t.Run("doctor explain remains useful for legacy repo", func(t *testing.T) {
		repoRoot := t.TempDir()
		gitInit(t, repoRoot)
		t.Chdir(repoRoot)

		if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
			t.Fatalf("write legacy home config: %v", err)
		}

		var stdout bytes.Buffer
		var stderr bytes.Buffer

		app := NewApp(&stdout, &stderr)
		if err := app.Run(context.Background(), []string{"doctor", "--scope", "repo", "--explain"}); err != nil {
			t.Fatalf("doctor --explain returned error: %v\nstderr=%s", err, stderr.String())
		}
		body := stdout.String()
		if !strings.Contains(body, "Status: legacy-detected") || !strings.Contains(body, "Repair hint:") {
			t.Fatalf("doctor explain missing legacy detail:\n%s", body)
		}
		if stderr.Len() != 0 {
			t.Fatalf("doctor --explain should not write stderr:\n%s", stderr.String())
		}
	})
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
	if got := stdout.String(); !strings.Contains(got, "Current version: 2.7.4") || !strings.Contains(got, "Recommended next final: 2.7.4") {
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
	if got := stdout.String(); !strings.Contains(got, "Recommended next final: 2.7.5") {
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

func TestRenderProfilesSurfacesInvalidEffectiveProfileConfig(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 9, 30, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 2, 3, 4})

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	cfgPath := config.RepoConfigPath(repoRoot)
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	raw = append(raw, []byte("\n[render_profiles.github_release]\nmode = \"unsupported\"\n")...)
	if err := os.WriteFile(cfgPath, raw, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	err = app.Run(context.Background(), []string{"render", "profiles"})
	if err == nil || !strings.Contains(stderr.String(), "unsupported mode") {
		t.Fatalf("render profiles should surface the profile error:\nstdout=%s\nstderr=%s", stdout.String(), stderr.String())
	}
}

func TestAppStatusPrintsAuthorityWarningToStderr(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 9, 45, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 2, 3, 4})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write legacy home config: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Current version: unreleased") {
		t.Fatalf("status stdout missing normal output:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), ".changes/config") {
		t.Fatalf("status stderr missing authority warning:\n%s", stderr.String())
	}
}

func TestAppCreatePrintsAuthorityWarningToStderr(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 9, 50, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{5, 6, 7, 8})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write legacy home config: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"create", "--behavior", "fix", "Fix the parser edge case."}); err != nil {
		t.Fatalf("create returned error: %v\nstderr=%s", err, stderr.String())
	}
	assertExists(t, strings.TrimSpace(stdout.String()))
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), ".changes/config") {
		t.Fatalf("create stderr missing authority warning:\n%s", stderr.String())
	}
}

func TestAppReleasePrintsAuthorityWarningToStderr(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 9, 55, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{9, 10, 11, 12})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{"create", "--behavior", "fix", "Fix the first parser bug."}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write legacy home config: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"release", "--accept"}); err != nil {
		t.Fatalf("release returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(strings.TrimSpace(stdout.String()), filepath.Join(repoRoot, ".local", "share", "changes", "releases")) {
		t.Fatalf("release stdout missing record path:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), ".changes/config") {
		t.Fatalf("release stderr missing authority warning:\n%s", stderr.String())
	}
}

func TestAppRenderProfilesPrintsAuthorityWarningToStderr(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 10, 5, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 3, 5, 7})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write legacy home config: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"render", "profiles"}); err != nil {
		t.Fatalf("render profiles returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), config.RenderProfileGitHubRelease) {
		t.Fatalf("render profiles stdout missing profiles:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), ".changes/config") {
		t.Fatalf("render profiles stderr missing authority warning:\n%s", stderr.String())
	}
}

func TestAmbiguousAuthorityFailurePrintsMigrationHint(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write xdg config: %v", err)
	}
	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write home config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 10, 10, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{2, 4, 6, 8})
	app.IsTTY = func() bool { return false }

	err := app.Run(context.Background(), []string{"create", "--behavior", "fix", "Fix the parser edge case."})
	if err == nil {
		t.Fatalf("create returned nil error")
	}
	if !strings.Contains(stderr.String(), "error: repo authority is ambiguous") || !strings.Contains(stderr.String(), "changes doctor --scope repo") || !strings.Contains(stderr.String(), "changes doctor --migration-prompt --scope repo --to xdg|home") {
		t.Fatalf("create stderr missing migration hint:\n%s", stderr.String())
	}
}

func TestAppInitPrintsGlobalAuthorityWarningToStderr(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)
	homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome := configureGlobalLayoutEnv(t)

	if err := writeManagedGlobalRepoInitDefaults(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, config.StyleXDG, "[repo.init]\nstyle = \"home\"\nhome = \".changes-global\"\n"); err != nil {
		t.Fatalf("write xdg global defaults: %v", err)
	}
	if err := writeLegacyGlobalRepoInitDefaults(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, config.StyleHome, "[repo.init]\nstyle = \"home\"\nhome = \".changes-legacy\"\n"); err != nil {
		t.Fatalf("write legacy home defaults: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 10, 15, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{1, 1, 2, 3})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "initialized ") {
		t.Fatalf("init stdout missing success output:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "warning:") || !strings.Contains(stderr.String(), changesHome) {
		t.Fatalf("init stderr missing global authority warning:\n%s", stderr.String())
	}
}

func TestAppInitFailsWithTerseAmbiguousGlobalDoctorHint(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)
	homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome := configureGlobalLayoutEnv(t)

	if err := writeManagedGlobalRepoInitDefaults(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, config.StyleXDG, "[repo.init]\nstyle = \"xdg\"\n"); err != nil {
		t.Fatalf("write xdg global defaults: %v", err)
	}
	if err := writeManagedGlobalRepoInitDefaults(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, config.StyleHome, "[repo.init]\nstyle = \"home\"\nhome = \".changes-home\"\n"); err != nil {
		t.Fatalf("write home global defaults: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 10, 20, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{4, 3, 2, 1})
	app.IsTTY = func() bool { return false }

	err := app.Run(context.Background(), []string{"init"})
	if err == nil {
		t.Fatalf("init returned nil error")
	}
	if !strings.Contains(stderr.String(), "error: global authority is ambiguous") || !strings.Contains(stderr.String(), "changes doctor --scope global") {
		t.Fatalf("init stderr missing global doctor hint:\n%s", stderr.String())
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

func TestReleasePromptRequiresOverrideWhenNoBumpIsInferred(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 13, 15, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{2, 3, 4})
	app.IsTTY = func() bool { return true }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"Refactor parser wiring without release-visible changes.",
	}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	app.Stdin = strings.NewReader("\n")
	err := app.Run(context.Background(), []string{"release"})
	if err == nil || !strings.Contains(err.Error(), "choose patch, minor, major, or cancel") {
		t.Fatalf("unexpected no-bump release error: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "No version bump was inferred. Choose patch/minor/major to override, or type cancel: ") {
		t.Fatalf("missing no-bump prompt:\n%s", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 13, 20, 0, 0, time.UTC)
	}
	app.Stdin = strings.NewReader("patch\n")
	if err := app.Run(context.Background(), []string{"release"}); err != nil {
		t.Fatalf("interactive release override returned error: %v\nstderr=%s", err, stderr.String())
	}
	recordPath := strings.TrimSpace(strings.Split(stdout.String(), "\n")[len(strings.Split(strings.TrimSpace(stdout.String()), "\n"))-1])
	record, err := releases.Load(recordPath)
	if err != nil {
		t.Fatalf("load release record: %v", err)
	}
	if record.Version != "0.1.0" {
		t.Fatalf("override release version = %q, want 0.1.0", record.Version)
	}
}

func TestRenderRejectsInvalidSelectorsAndCanRenderByRecordPath(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 5, 14, 45, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{8, 7, 6, 5})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}
	if err := app.Run(context.Background(), []string{
		"create",
		"--behavior", "fix",
		"Fix a parser output issue.",
	}); err != nil {
		t.Fatalf("create returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	err := app.Run(context.Background(), []string{"render"})
	if err == nil || !strings.Contains(stderr.String(), "provide exactly one of --version, --record, or --latest") {
		t.Fatalf("render without selector should fail:\nstdout=%s\nstderr=%s\nerr=%v", stdout.String(), stderr.String(), err)
	}

	stdout.Reset()
	stderr.Reset()
	err = app.Run(context.Background(), []string{"render", "--version", "0.1.0", "--latest"})
	if err == nil || !strings.Contains(stderr.String(), "provide exactly one of --version, --record, or --latest") {
		t.Fatalf("render with conflicting selectors should fail:\nstdout=%s\nstderr=%s\nerr=%v", stdout.String(), stderr.String(), err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"release", "--accept"}); err != nil {
		t.Fatalf("release returned error: %v", err)
	}
	recordPath := strings.TrimSpace(stdout.String())

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"render", "--record", recordPath}); err != nil {
		t.Fatalf("render by record returned error: %v\nstderr=%s", err, stderr.String())
	}
	if got := stdout.String(); !strings.Contains(got, "# Release 0.1.0") {
		t.Fatalf("render by record missing release heading:\n%s", got)
	}
}

func TestCreateSupportsBodyFlagAndRejectsMixedBodyInputs(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)
	}
	app.Random = bytes.NewReader([]byte{9, 9, 9})
	app.IsTTY = func() bool { return false }

	if err := app.Run(context.Background(), []string{"init"}); err != nil {
		t.Fatalf("init returned error: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"create", "--body", "Body from flag"}); err != nil {
		t.Fatalf("create --body returned error: %v\nstderr=%s", err, stderr.String())
	}
	fragmentPath := strings.TrimSpace(stdout.String())
	raw, err := os.ReadFile(fragmentPath)
	if err != nil {
		t.Fatalf("read fragment: %v", err)
	}
	if !strings.Contains(string(raw), "Body from flag") {
		t.Fatalf("fragment missing --body content:\n%s", raw)
	}

	stdout.Reset()
	stderr.Reset()
	err = app.Run(context.Background(), []string{"create", "--body", "flag body", "trailing body"})
	if err == nil || !strings.Contains(stderr.String(), "pass the body either with --body or as the trailing argument") {
		t.Fatalf("create mixed body inputs should fail:\nstdout=%s\nstderr=%s\nerr=%v", stdout.String(), stderr.String(), err)
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

func configureGlobalLayoutEnv(t *testing.T) (string, string, string, string, string) {
	t.Helper()
	homeDir := t.TempDir()
	changesHome := filepath.Join(homeDir, ".changes-home")
	xdgConfigHome := filepath.Join(homeDir, ".config-home")
	xdgDataHome := filepath.Join(homeDir, ".data-home")
	xdgStateHome := filepath.Join(homeDir, ".state-home")

	t.Setenv("HOME", homeDir)
	t.Setenv("CHANGES_HOME", changesHome)
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)
	t.Setenv("XDG_DATA_HOME", xdgDataHome)
	t.Setenv("XDG_STATE_HOME", xdgStateHome)

	return homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome
}

func writeManagedRepoConfigForStyle(t *testing.T, repoRoot string, style config.Style, cfg config.Config) error {
	t.Helper()
	if err := writeRepoLayoutManifestForStyle(t, repoRoot, style); err != nil {
		return err
	}
	return config.Write(repoConfigPathForStyle(repoRoot, style), cfg)
}

func writeLegacyRepoConfigForStyle(t *testing.T, repoRoot string, style config.Style, cfg config.Config) error {
	t.Helper()
	path := repoConfigPathForStyle(repoRoot, style)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return config.Write(path, cfg)
}

func writeManagedGlobalRepoInitDefaults(t *testing.T, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome string, style config.Style, raw string) error {
	t.Helper()
	if err := writeGlobalLayoutManifestForStyle(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, style); err != nil {
		return err
	}
	return writeGlobalConfigFileForStyle(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, style, raw)
}

func writeLegacyGlobalRepoInitDefaults(t *testing.T, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome string, style config.Style, raw string) error {
	t.Helper()
	return writeGlobalConfigFileForStyle(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, style, raw)
}

func writeGlobalConfigFileForStyle(t *testing.T, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome string, style config.Style, raw string) error {
	t.Helper()
	path := globalConfigPathForStyle(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, style)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(raw), 0o644)
}

func writeRepoLayoutManifestForStyle(t *testing.T, repoRoot string, style config.Style) error {
	t.Helper()
	path := repoLayoutManifestPathForStyle(repoRoot, style)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var raw string
	switch style {
	case config.StyleHome:
		raw = "schema_version = 1\nscope = \"repo\"\nstyle = \"home\"\n\n[layout]\nroot = \"$REPO_ROOT/.changes\"\nconfig = \"$layout.root/config\"\ndata = \"$layout.root/data\"\nstate = \"$layout.root/state\"\n"
	default:
		raw = "schema_version = 1\nscope = \"repo\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$REPO_ROOT\"\nconfig = \"$REPO_ROOT/.config/changes\"\ndata = \"$REPO_ROOT/.local/share/changes\"\nstate = \"$REPO_ROOT/.local/state/changes\"\n"
	}
	return os.WriteFile(path, []byte(raw), 0o644)
}

func writeGlobalLayoutManifestForStyle(t *testing.T, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome string, style config.Style) error {
	t.Helper()
	path := globalLayoutManifestPathForStyle(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, style)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var raw string
	switch style {
	case config.StyleHome:
		raw = "schema_version = 1\nscope = \"global\"\nstyle = \"home\"\n\n[layout]\nroot = \"$CHANGES_HOME\"\nconfig = \"$layout.root/config\"\ndata = \"$layout.root/data\"\nstate = \"$layout.root/state\"\n"
	default:
		raw = "schema_version = 1\nscope = \"global\"\nstyle = \"xdg\"\n\n[layout]\nroot = \"$HOME\"\nconfig = \"$XDG_CONFIG_HOME/changes\"\ndata = \"$XDG_DATA_HOME/changes\"\nstate = \"$XDG_STATE_HOME/changes\"\n"
	}
	return os.WriteFile(path, []byte(raw), 0o644)
}

func globalLayoutManifestPathForStyle(t *testing.T, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome string, style config.Style) string {
	t.Helper()
	return filepath.Join(filepath.Dir(globalConfigPathForStyle(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, style)), "layout.toml")
}

func globalConfigPathForStyle(t *testing.T, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome string, style config.Style) string {
	t.Helper()
	resolution, err := config.ResolveGlobal(config.ResolveOptions{
		HomeDir:       homeDir,
		ChangesHome:   changesHome,
		XDGConfigHome: xdgConfigHome,
		XDGDataHome:   xdgDataHome,
		XDGStateHome:  xdgStateHome,
	})
	if err != nil {
		t.Fatalf("resolve global candidate paths: %v", err)
	}
	for _, candidate := range resolution.Candidates {
		if candidate.Style == style {
			return filepath.Join(candidate.Paths.Config, "config.toml")
		}
	}
	t.Fatalf("missing global candidate for style %s", style)
	return ""
}

func repoLayoutManifestPathForStyle(repoRoot string, style config.Style) string {
	return filepath.Join(filepath.Dir(repoConfigPathForStyle(repoRoot, style)), "layout.toml")
}

func repoConfigPathForStyle(repoRoot string, style config.Style) string {
	if style == config.StyleHome {
		return filepath.Join(repoRoot, ".changes", "config", "config.toml")
	}
	return filepath.Join(repoRoot, ".config", "changes", "config.toml")
}
