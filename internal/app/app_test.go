package app

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
)

func TestInitializeUnreleasedCreatesNoBootstrapArtifacts(t *testing.T) {
	repoRoot := t.TempDir()

	result, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      time.Date(2026, 4, 6, 12, 0, 0, 0, time.UTC),
		Random:   bytes.NewReader([]byte{1, 2, 3, 4}),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if result.AdoptionRecord != nil || result.AdoptionFragment != nil {
		t.Fatalf("unexpected adoption bootstrap: %#v", result)
	}
	if result.PromptPath != "" {
		t.Fatalf("unexpected prompt path: %q", result.PromptPath)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := os.Stat(config.HistoryImportPromptPath(repoRoot, cfg)); !os.IsNotExist(err) {
		t.Fatalf("history import prompt should not exist, err=%v", err)
	}
	if _, err := os.Stat(config.TemplatesDir(repoRoot, cfg)); !os.IsNotExist(err) {
		t.Fatalf("templates dir should not exist after init, err=%v", err)
	}
}

func TestInitializeUsesSelectedRepoLayoutDefaults(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 5, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:        repoRoot,
		RequestedLayout: "home",
		Now:             now,
		Random:          bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	homeConfigPath := filepath.Join(repoRoot, ".changes", "config", "config.toml")
	if _, err := os.Stat(homeConfigPath); err != nil {
		t.Fatalf("home config path missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".changes", "config", "layout.toml")); err != nil {
		t.Fatalf("home layout manifest missing: %v", err)
	}
	for _, dir := range []string{
		filepath.Join(repoRoot, ".changes", "data", "fragments"),
		filepath.Join(repoRoot, ".changes", "data", "releases"),
		filepath.Join(repoRoot, ".changes", "data", "prompts"),
		filepath.Join(repoRoot, ".changes", "state"),
	} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("expected directory %s to exist, err=%v", dir, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".config", "changes", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("xdg config path should not exist, err=%v", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if got := config.RepoConfigPath(repoRoot); got != homeConfigPath {
		t.Fatalf("RepoConfigPath = %q, want %q", got, homeConfigPath)
	}
	if got := config.FragmentsDir(repoRoot, cfg); got != filepath.Join(repoRoot, ".changes", "data", "fragments") {
		t.Fatalf("FragmentsDir = %q", got)
	}
	if got := config.StateDir(repoRoot, cfg); got != filepath.Join(repoRoot, ".changes", "state") {
		t.Fatalf("StateDir = %q", got)
	}
}

func TestInitializeHomeLayoutAddsStateGitignore(t *testing.T) {
	repoRoot := t.TempDir()

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:        repoRoot,
		RequestedLayout: "home",
		Now:             time.Date(2026, 4, 6, 12, 10, 0, 0, time.UTC),
		Random:          bytes.NewReader([]byte{5, 6, 7, 8}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	raw, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	body := string(raw)
	if !strings.Contains(body, "/.changes/state/") {
		t.Fatalf(".gitignore = %q, want home state entry", body)
	}
	if strings.Contains(body, "/.local/state/") {
		t.Fatalf(".gitignore = %q, should not include xdg state entry", body)
	}
}

func TestInitializeReleasedVersionCreatesBootstrapArtifacts(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 12, 30, 0, 0, time.UTC)

	result, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:       repoRoot,
		CurrentVersion: "2.7.4",
		Now:            now,
		Random:         bytes.NewReader([]byte{5, 6, 7, 8}),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if result.AdoptionRecord == nil || result.AdoptionFragment == nil {
		t.Fatalf("expected adoption bootstrap, got %#v", result)
	}
	if result.AdoptionRecord.Version != "2.7.4" {
		t.Fatalf("bootstrap version = %q, want 2.7.4", result.AdoptionRecord.Version)
	}
	if strings.TrimSpace(result.PromptPath) == "" {
		t.Fatalf("expected history import prompt path")
	}
}

func TestInitializeRollsBackAfterBootstrapFailure(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 13, 0, 0, 0, time.UTC)
	cfg := config.Default()
	cfg.Project.Name = filepath.Base(repoRoot)

	errBoom := errors.New("boom after bootstrap")
	_, err := initializeWithDeps(
		context.Background(),
		InitializeRequest{
			RepoRoot:       repoRoot,
			CurrentVersion: "2.7.4",
			Now:            now,
			Random:         bytes.NewReader([]byte{9, 10, 11, 12}),
		}, initializeDeps{
			createAdoptionBootstrap:  createAdoptionBootstrap,
			writeHistoryImportPrompt: writeHistoryImportPrompt,
			stageHook: func(stage string) error {
				if stage == "after_bootstrap" {
					return errBoom
				}
				return nil
			},
		},
	)
	if !errors.Is(err, errBoom) {
		t.Fatalf("initializeWithDeps error = %v, want %v", err, errBoom)
	}

	assertPathMissing(t, config.RepoConfigPath(repoRoot))
	assertPathMissing(t, config.HistoryImportPromptPath(repoRoot, cfg))
	assertDirEmptyOrMissing(t, config.FragmentsDir(repoRoot, cfg))
	assertDirEmptyOrMissing(t, config.ReleasesDir(repoRoot, cfg))
}

func TestInitializeRestoresGitignoreOnFailure(t *testing.T) {
	repoRoot := t.TempDir()
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	original := "node_modules/\n"
	if err := os.WriteFile(gitignorePath, []byte(original), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	errBoom := errors.New("boom after gitignore")
	_, err := initializeWithDeps(
		context.Background(),
		InitializeRequest{
			RepoRoot: repoRoot,
			Now:      time.Date(2026, 4, 6, 13, 30, 0, 0, time.UTC),
			Random:   bytes.NewReader([]byte{1, 1, 1, 1}),
		}, initializeDeps{
			createAdoptionBootstrap:  createAdoptionBootstrap,
			writeHistoryImportPrompt: writeHistoryImportPrompt,
			stageHook: func(stage string) error {
				if stage == "after_gitignore" {
					return errBoom
				}
				return nil
			},
		},
	)
	if !errors.Is(err, errBoom) {
		t.Fatalf("initializeWithDeps error = %v, want %v", err, errBoom)
	}

	raw, readErr := os.ReadFile(gitignorePath)
	if readErr != nil {
		t.Fatalf("read .gitignore: %v", readErr)
	}
	if string(raw) != original {
		t.Fatalf(".gitignore = %q, want %q", string(raw), original)
	}
}

func TestStatusUsesUnreleasedOrLatestReleaseRecord(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 14, 0, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      now,
		Random:   bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := fragments.Create(repoRoot, cfg, now.Add(time.Minute), bytes.NewReader([]byte{5, 6, 7, 8}), fragments.NewInput{
		Behavior: "fix",
		Body:     "Fix the parser edge case.",
	}); err != nil {
		t.Fatalf("create fragment: %v", err)
	}

	result, err := Status(context.Background(), StatusRequest{RepoRoot: repoRoot})
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if result.CurrentVersionSource != "unreleased" {
		t.Fatalf("current version source = %q, want unreleased", result.CurrentVersionSource)
	}
	if result.CurrentVersionLabel != "unreleased" {
		t.Fatalf("current version label = %q, want unreleased", result.CurrentVersionLabel)
	}
	if result.InitialReleaseTarget == nil {
		t.Fatalf("initial release target = nil, want 0.1.0")
	}
	if got := result.InitialReleaseTarget.String(); got != "0.1.0" {
		t.Fatalf("initial release target = %q, want 0.1.0", got)
	}

	adoptedRepo := t.TempDir()
	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:       adoptedRepo,
		CurrentVersion: "2.7.4",
		Now:            now,
		Random:         bytes.NewReader([]byte{9, 10, 11, 12}),
	}); err != nil {
		t.Fatalf("Initialize adopted repo returned error: %v", err)
	}
	adoptedStatus, err := Status(context.Background(), StatusRequest{RepoRoot: adoptedRepo})
	if err != nil {
		t.Fatalf("Status on adopted repo returned error: %v", err)
	}
	if adoptedStatus.CurrentVersionSource != "latest_release_record" {
		t.Fatalf("current version source = %q, want latest_release_record", adoptedStatus.CurrentVersionSource)
	}
	if adoptedStatus.CurrentVersionLabel != "2.7.4" {
		t.Fatalf("current version label = %q, want 2.7.4", adoptedStatus.CurrentVersionLabel)
	}
	if adoptedStatus.InitialReleaseTarget != nil {
		t.Fatalf("initial release target = %v, want nil", adoptedStatus.InitialReleaseTarget)
	}
}

func TestPlanReleaseSupportsRecommendedAndOverrideFlows(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 15, 0, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:       repoRoot,
		CurrentVersion: "2.7.4",
		Now:            now,
		Random:         bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := fragments.Create(repoRoot, cfg, now.Add(time.Minute), bytes.NewReader([]byte{5, 6, 7, 8}), fragments.NewInput{
		PublicAPI: "add",
		Body:      "Add an operator-visible command.",
	}); err != nil {
		t.Fatalf("create fragment: %v", err)
	}

	recommended, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot: repoRoot,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease returned error: %v", err)
	}
	if !recommended.RecommendedChoice {
		t.Fatalf("expected recommended choice")
	}
	if recommended.ChosenBump != "minor" || recommended.ChosenVersion.String() != "2.8.0" {
		t.Fatalf("unexpected recommended plan: bump=%s version=%s", recommended.ChosenBump, recommended.ChosenVersion.String())
	}

	prerelease, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot:     repoRoot,
		RequestedPre: "rc",
		Now:          now.Add(3 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease prerelease returned error: %v", err)
	}
	if prerelease.ChosenVersion.String() != "2.8.0-rc.1" {
		t.Fatalf("unexpected prerelease version: %s", prerelease.ChosenVersion.String())
	}

	overrideBump, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot:      repoRoot,
		RequestedBump: "patch",
		Now:           now.Add(4 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease override bump returned error: %v", err)
	}
	if overrideBump.RecommendedChoice {
		t.Fatalf("override bump should not be marked recommended")
	}
	if overrideBump.ChosenVersion.String() != "2.7.5" {
		t.Fatalf("unexpected override-by-bump version: %s", overrideBump.ChosenVersion.String())
	}

	overrideVersion, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot:         repoRoot,
		RequestedVersion: "3.0.0-beta.3",
		Now:              now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease override version returned error: %v", err)
	}
	if overrideVersion.ChosenVersion.String() != "3.0.0-beta.3" {
		t.Fatalf("unexpected override-by-version: %s", overrideVersion.ChosenVersion.String())
	}
}

func TestPlanReleaseReturnsNoneWhenNoImpactIsInferred(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 16, 0, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      now,
		Random:   bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := fragments.Create(repoRoot, cfg, now.Add(time.Minute), bytes.NewReader([]byte{5, 6, 7, 8}), fragments.NewInput{
		Type: "changed",
		Body: "Refactor internal wiring without release-visible impact.",
	}); err != nil {
		t.Fatalf("create fragment: %v", err)
	}

	plan, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot: repoRoot,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease returned error: %v", err)
	}
	if plan.ChosenBump != "none" {
		t.Fatalf("chosen bump = %s, want none", plan.ChosenBump)
	}
}

func TestCommitReleasePersistsThePlannedRecord(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 16, 30, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      now,
		Random:   bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := fragments.Create(repoRoot, cfg, now.Add(time.Minute), bytes.NewReader([]byte{5, 6, 7, 8}), fragments.NewInput{
		Behavior: "fix",
		Body:     "Fix the shipped parser crash.",
	}); err != nil {
		t.Fatalf("create fragment: %v", err)
	}

	plan, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot: repoRoot,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease returned error: %v", err)
	}

	result, err := CommitRelease(context.Background(), plan)
	if err != nil {
		t.Fatalf("CommitRelease returned error: %v", err)
	}
	if _, statErr := os.Stat(result.Path); statErr != nil {
		t.Fatalf("release record path missing: %v", statErr)
	}
	if !result.Record.CreatedAt.Equal(plan.CreatedAt.UTC().Truncate(time.Second)) {
		t.Fatalf("created_at = %s, want %s", result.Record.CreatedAt, plan.CreatedAt.UTC().Truncate(time.Second))
	}
}

func TestCommitReleaseRejectsStaleOrDuplicatePlans(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 16, 45, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      now,
		Random:   bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := fragments.Create(repoRoot, cfg, now.Add(time.Minute), bytes.NewReader([]byte{5, 6, 7, 8}), fragments.NewInput{
		Behavior: "fix",
		Body:     "Fix the shipped parser crash.",
	}); err != nil {
		t.Fatalf("create fragment: %v", err)
	}

	plan, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot: repoRoot,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease returned error: %v", err)
	}

	if _, err := CommitRelease(context.Background(), plan); err != nil {
		t.Fatalf("first CommitRelease returned error: %v", err)
	}
	if _, err := CommitRelease(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("second CommitRelease error = %v, want stale-plan failure", err)
	}
}

func TestRenderSupportsLatestVersionAndRecordSelectors(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 17, 0, 0, 0, time.UTC)

	initResult, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot:       repoRoot,
		CurrentVersion: "2.7.4",
		Now:            now,
		Random:         bytes.NewReader([]byte{1, 2, 3, 4}),
	})
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	latest, err := Render(context.Background(), RenderRequest{
		RepoRoot: repoRoot,
		Latest:   true,
		Profile:  config.RenderProfileGitHubRelease,
	})
	if err != nil {
		t.Fatalf("Render latest returned error: %v", err)
	}
	if !strings.Contains(latest.Content, "2.7.4") {
		t.Fatalf("latest render missing version: %s", latest.Content)
	}

	byVersion, err := Render(context.Background(), RenderRequest{
		RepoRoot: repoRoot,
		Version:  "2.7.4",
		Profile:  config.RenderProfileGitHubRelease,
	})
	if err != nil {
		t.Fatalf("Render by version returned error: %v", err)
	}
	if byVersion.Record.Version != "2.7.4" {
		t.Fatalf("rendered record version = %s, want 2.7.4", byVersion.Record.Version)
	}

	recordPath := releases.RecordPath(repoRoot, configMustLoad(t, repoRoot), initResult.AdoptionRecord.Product, initResult.AdoptionRecord.Version)
	byRecord, err := Render(context.Background(), RenderRequest{
		RepoRoot:   repoRoot,
		RecordPath: recordPath,
		Profile:    config.RenderProfileGitHubRelease,
	})
	if err != nil {
		t.Fatalf("Render by record returned error: %v", err)
	}
	if byRecord.Record.Version != "2.7.4" {
		t.Fatalf("rendered record version = %s, want 2.7.4", byRecord.Record.Version)
	}
}

func TestRenderValidatesSelectorsAndLatestAvailability(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 17, 15, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      now,
		Random:   bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	if _, err := Render(context.Background(), RenderRequest{RepoRoot: repoRoot}); err == nil || !strings.Contains(err.Error(), "provide exactly one") {
		t.Fatalf("Render without selector error = %v, want selector validation", err)
	}
	if _, err := Render(context.Background(), RenderRequest{
		RepoRoot: repoRoot,
		Version:  "0.1.0",
		Latest:   true,
	}); err == nil || !strings.Contains(err.Error(), "provide exactly one") {
		t.Fatalf("Render with conflicting selectors error = %v, want selector validation", err)
	}
	if _, err := Render(context.Background(), RenderRequest{
		RepoRoot: repoRoot,
		Latest:   true,
	}); err == nil || !strings.Contains(err.Error(), "no final release records exist") {
		t.Fatalf("Render latest before release error = %v, want missing latest record failure", err)
	}
}

func TestCommitReleaseRejectsPlanWhenPendingFragmentsChange(t *testing.T) {
	repoRoot := t.TempDir()
	now := time.Date(2026, 4, 6, 17, 30, 0, 0, time.UTC)

	if _, err := Initialize(context.Background(), InitializeRequest{
		RepoRoot: repoRoot,
		Now:      now,
		Random:   bytes.NewReader([]byte{1, 2, 3, 4}),
	}); err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}
	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, err := fragments.Create(repoRoot, cfg, now.Add(time.Minute), bytes.NewReader([]byte{5, 6, 7, 8}), fragments.NewInput{
		Behavior: "fix",
		Body:     "Fix the first shipped parser crash.",
	}); err != nil {
		t.Fatalf("create fragment: %v", err)
	}

	plan, err := PlanRelease(context.Background(), ReleasePlanRequest{
		RepoRoot: repoRoot,
		Now:      now.Add(2 * time.Minute),
	})
	if err != nil {
		t.Fatalf("PlanRelease returned error: %v", err)
	}

	if _, err := fragments.Create(repoRoot, cfg, now.Add(3*time.Minute), bytes.NewReader([]byte{9, 10, 11, 12}), fragments.NewInput{
		PublicAPI: "add",
		Body:      "Add a new parser extension point.",
	}); err != nil {
		t.Fatalf("create second fragment: %v", err)
	}

	if _, err := CommitRelease(context.Background(), plan); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("CommitRelease stale-plan error = %v, want stale", err)
	}
}

func TestApplicationLayerHonorsCanceledContext(t *testing.T) {
	repoRoot := t.TempDir()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := Initialize(ctx, InitializeRequest{RepoRoot: repoRoot}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Initialize error = %v, want %v", err, context.Canceled)
	}
	if _, err := Status(ctx, StatusRequest{RepoRoot: repoRoot}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Status error = %v, want %v", err, context.Canceled)
	}
	if _, err := PlanRelease(ctx, ReleasePlanRequest{RepoRoot: repoRoot}); !errors.Is(err, context.Canceled) {
		t.Fatalf("PlanRelease error = %v, want %v", err, context.Canceled)
	}
	if _, err := CommitRelease(ctx, ReleasePlan{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("CommitRelease error = %v, want %v", err, context.Canceled)
	}
	if _, err := Render(ctx, RenderRequest{RepoRoot: repoRoot}); !errors.Is(err, context.Canceled) {
		t.Fatalf("Render error = %v, want %v", err, context.Canceled)
	}
}

func configMustLoad(t *testing.T, repoRoot string) config.Config {
	t.Helper()
	cfg, err := config.Load(repoRoot)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	return cfg
}

func assertPathMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be absent, err=%v", path, err)
	}
}

func assertDirEmptyOrMissing(t *testing.T, path string) {
	t.Helper()
	entries, err := os.ReadDir(path)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		t.Fatalf("read dir %s: %v", path, err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected %s to be empty, found %d entries", path, len(entries))
	}
}
