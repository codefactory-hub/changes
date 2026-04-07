package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	appsvc "github.com/example/changes/internal/app"
	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/render"
	"github.com/example/changes/internal/reporoot"
	"github.com/example/changes/internal/semverpolicy"
	"github.com/example/changes/internal/versioning"
)

type App struct {
	Stdout   io.Writer
	Stderr   io.Writer
	Stdin    io.Reader
	Now      func() time.Time
	Random   io.Reader
	IsTTY    func() bool
	EditFile func(path string) error
	Version  string
	Commit   string
	Date     string
}

func NewApp(stdout, stderr io.Writer) *App {
	app := &App{
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  os.Stdin,
		Now:    time.Now,
		Version: "dev",
		Commit:  "unknown",
		Date:    "unknown",
	}
	app.IsTTY = app.defaultIsTTY
	app.EditFile = app.defaultEditFile
	return app
}

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return a.fail(fmt.Errorf("usage: changes <command>"))
	}
	if isVersionArg(args[0]) && len(args) == 1 {
		a.printVersion()
		return nil
	}
	if isHelpArg(args[0]) {
		a.printHelp(args[1:])
		return nil
	}
	if isHelpArg(args[len(args)-1]) && len(args) == 1 {
		a.printHelp(nil)
		return nil
	}

	var err error
	switch args[0] {
	case "version":
		err = a.runVersion(args[1:])
	case "init":
		err = a.runInit(ctx, args[1:])
	case "create":
		err = a.runCreate(ctx, args[1:])
	case "status":
		err = a.runStatus(ctx, args[1:])
	case "doctor":
		err = a.runDoctor(ctx, args[1:])
	case "release":
		err = a.runRelease(ctx, args[1:])
	case "render":
		err = a.runRender(ctx, args[1:])
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}

	if err != nil {
		return a.fail(err)
	}
	return nil
}

func (a *App) runVersion(args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"version"})
		return nil
	}
	if len(args) > 0 {
		return fmt.Errorf("usage: changes version")
	}
	a.printVersion()
	return nil
}

func (a *App) runInit(ctx context.Context, args []string) error {
	if len(args) > 0 && args[0] == "global" {
		return a.runInitGlobal(ctx, args[1:])
	}
	if wantsHelp(args) {
		a.printHelp([]string{"init"})
		return nil
	}

	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var currentVersion string
	var layout string
	var home string
	fs.StringVar(&currentVersion, "current-version", "", "Current released version or unreleased")
	fs.StringVar(&layout, "layout", "", "Repo layout style")
	fs.StringVar(&home, "home", "", "Repo home path for home layout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("usage: changes init [--current-version <semver|unreleased>] [--layout xdg|home] [--home PATH]")
	}
	if strings.TrimSpace(home) != "" && strings.TrimSpace(layout) != "home" {
		return fmt.Errorf("init: --home is only valid when --layout home")
	}

	if strings.TrimSpace(currentVersion) == "" && a.isTTY() {
		answer, err := a.promptOptionalLine(a.newPromptReader(), "Current released version [unreleased]: ")
		if err != nil {
			return err
		}
		currentVersion = strings.TrimSpace(answer)
	}

	repoRoot, err := a.repoRoot(ctx)
	if err != nil {
		return err
	}

	result, err := appsvc.Initialize(ctx, appsvc.InitializeRequest{
		RepoRoot:        repoRoot,
		CurrentVersion:  currentVersion,
		RequestedLayout: layout,
		RequestedHome:   home,
		Now:             a.Now().UTC().Truncate(time.Second),
		Random:          a.Random,
	})
	if err != nil {
		return err
	}
	a.printAuthorityWarnings(repoRoot, result.AuthorityWarnings)

	_, _ = fmt.Fprintf(a.Stdout, "initialized %s\n", result.RepoRoot)
	_, _ = fmt.Fprintf(a.Stdout, "layout: %s\n", result.SelectedLayout)
	_, _ = fmt.Fprintf(a.Stdout, "config: %s\n", repoRelativePath(result.RepoRoot, result.ConfigPath))
	_, _ = fmt.Fprintf(a.Stdout, "data: %s\n", repoRelativePath(result.RepoRoot, result.DataPath))
	_, _ = fmt.Fprintf(a.Stdout, "state: %s\n", repoRelativePath(result.RepoRoot, result.StatePath))
	if result.GitignoreUpdated {
		_, _ = fmt.Fprintln(a.Stdout, "updated .gitignore")
	}
	if strings.TrimSpace(result.PromptPath) != "" {
		_, _ = fmt.Fprintf(a.Stdout, "next step: review %s to replace or refine the standard adoption history.\n", repoRelativePath(result.RepoRoot, result.PromptPath))
	}
	return nil
}

func (a *App) runInitGlobal(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"init", "global"})
		return nil
	}

	fs := flag.NewFlagSet("init global", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var layout string
	var home string
	fs.StringVar(&layout, "layout", "", "Global layout style")
	fs.StringVar(&home, "home", "", "Global home path for home layout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("usage: changes init global [--layout xdg|home] [--home PATH]")
	}
	if strings.TrimSpace(home) != "" && strings.TrimSpace(layout) != "home" {
		return fmt.Errorf("init global: --home is only valid when --layout home")
	}

	result, err := appsvc.InitializeGlobal(ctx, appsvc.InitializeGlobalRequest{
		RequestedLayout: layout,
		RequestedHome:   home,
		Now:             a.Now().UTC().Truncate(time.Second),
	})
	if err != nil {
		return err
	}
	a.printAuthorityWarnings("", result.AuthorityWarnings)

	_, _ = fmt.Fprintf(a.Stdout, "initialized global %s\n", result.SelectedLayout)
	_, _ = fmt.Fprintf(a.Stdout, "config: %s\n", result.ConfigPath)
	_, _ = fmt.Fprintf(a.Stdout, "data: %s\n", result.DataPath)
	_, _ = fmt.Fprintf(a.Stdout, "state: %s\n", result.StatePath)
	return nil
}

func (a *App) runStatus(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"status"})
		return nil
	}

	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var explain bool
	fs.BoolVar(&explain, "explain", false, "Show policy evidence for the pending bump")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("usage: changes status [--explain]")
	}

	repoRoot, err := a.repoRoot(ctx)
	if err != nil {
		return err
	}
	result, err := appsvc.Status(ctx, appsvc.StatusRequest{RepoRoot: repoRoot})
	if err != nil {
		return err
	}
	a.printAuthorityWarnings(repoRoot, result.AuthorityWarnings)

	_, _ = fmt.Fprintf(a.Stdout, "Current version: %s\n", result.CurrentVersionLabel)
	_, _ = fmt.Fprintf(a.Stdout, "Current version source: %s\n", result.CurrentVersionSource)
	if result.InitialReleaseTarget != nil {
		_, _ = fmt.Fprintf(a.Stdout, "Initial release target: %s\n", result.InitialReleaseTarget.String())
	}
	_, _ = fmt.Fprintf(a.Stdout, "Unreleased fragments: %d\n", len(result.PendingFragments))
	_, _ = fmt.Fprintf(a.Stdout, "Recommended bump: %s\n", result.Recommendation.SuggestedBump)
	_, _ = fmt.Fprintf(a.Stdout, "Recommended next final: %s\n", result.RecommendedNextFinal.String())
	if explain {
		renderRecommendationExplanation(a.Stdout, result.Recommendation)
	}

	if len(result.PrereleaseHeads) > 0 {
		_, _ = fmt.Fprintln(a.Stdout, "Active prerelease heads:")
		for _, head := range result.PrereleaseHeads {
			_, _ = fmt.Fprintf(a.Stdout, "- %s -> %s\n", head.Version, head.TargetVersion())
		}
	}

	if len(result.PendingFragments) > 0 {
		_, _ = fmt.Fprintln(a.Stdout, "Pending fragments:")
		for _, item := range result.PendingFragments {
			_, _ = fmt.Fprintf(a.Stdout, "- %s [%s] %s\n", item.ID, item.Type, item.BodyPreview())
		}
	}

	return nil
}

func (a *App) runDoctor(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"doctor"})
		return nil
	}

	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var scope string
	var explain bool
	var jsonOutput bool
	var repair bool
	var migrationPrompt bool
	var to string
	var home string
	var outputPath string

	fs.StringVar(&scope, "scope", "", "Inspection scope")
	fs.BoolVar(&explain, "explain", false, "Show candidate and warning detail")
	fs.BoolVar(&jsonOutput, "json", false, "Emit structured inspection JSON")
	fs.BoolVar(&repair, "repair", false, "Repair one repo-local legacy layout by restoring its manifest")
	fs.BoolVar(&migrationPrompt, "migration-prompt", false, "Generate a migration brief")
	fs.StringVar(&to, "to", "", "Destination layout style")
	fs.StringVar(&home, "home", "", "Destination home path for home layouts")
	fs.StringVar(&outputPath, "output", "", "Output path for migration prompt")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("usage: changes doctor [--scope global|repo|all] [--explain] [--json]")
	}
	if repair && explain {
		return fmt.Errorf("doctor: --repair cannot be combined with --explain")
	}
	if repair && jsonOutput {
		return fmt.Errorf("doctor: --repair cannot be combined with --json")
	}
	if repair && migrationPrompt {
		return fmt.Errorf("doctor: --repair cannot be combined with --migration-prompt")
	}
	if repair && strings.TrimSpace(to) != "" {
		return fmt.Errorf("doctor: --repair cannot be combined with --to")
	}
	if repair && strings.TrimSpace(home) != "" {
		return fmt.Errorf("doctor: --repair cannot be combined with --home")
	}
	if repair && strings.TrimSpace(outputPath) != "" {
		return fmt.Errorf("doctor: --repair cannot be combined with --output")
	}
	if explain && jsonOutput {
		return fmt.Errorf("doctor: --explain and --json cannot be combined")
	}
	if migrationPrompt && (explain || jsonOutput) {
		return fmt.Errorf("doctor: --migration-prompt cannot be combined with --explain or --json")
	}
	if !migrationPrompt && strings.TrimSpace(to) != "" {
		return fmt.Errorf("doctor: --to requires --migration-prompt")
	}
	if !migrationPrompt && strings.TrimSpace(home) != "" {
		return fmt.Errorf("doctor: --home requires --migration-prompt")
	}
	if !migrationPrompt && strings.TrimSpace(outputPath) != "" {
		return fmt.Errorf("doctor: --output requires --migration-prompt")
	}
	if migrationPrompt && strings.TrimSpace(to) == "" {
		return fmt.Errorf("doctor: --migration-prompt requires --to xdg|home")
	}

	cwd, repoRoot, insideRepo, err := a.currentLocation()
	if err != nil {
		return err
	}
	_ = cwd

	requestedScope := scope
	if strings.TrimSpace(requestedScope) == "" {
		if insideRepo {
			requestedScope = string(appsvc.DoctorScopeRepo)
		} else {
			requestedScope = string(appsvc.DoctorScopeGlobal)
		}
	}

	doctorScope, err := parseDoctorScope(requestedScope)
	if err != nil {
		return err
	}
	if !insideRepo && (doctorScope == appsvc.DoctorScopeRepo || doctorScope == appsvc.DoctorScopeAll) {
		return fmt.Errorf("doctor: %s scope requires running inside a Git repository", doctorScope)
	}
	if repair && doctorScope != appsvc.DoctorScopeRepo {
		return fmt.Errorf("doctor: --repair requires --scope repo")
	}
	if migrationPrompt && doctorScope == appsvc.DoctorScopeAll {
		return fmt.Errorf("doctor: --migration-prompt requires --scope global or --scope repo")
	}

	var destinationStyle config.Style
	if migrationPrompt {
		destinationStyle, err = parseDoctorStyle(to)
		if err != nil {
			return err
		}
		if destinationStyle == config.StyleXDG && strings.TrimSpace(home) != "" {
			return fmt.Errorf("doctor: --home is valid only when --to home")
		}
	}

	result, err := appsvc.Doctor(ctx, appsvc.DoctorRequest{
		RepoRoot:                repoRoot,
		Scope:                   doctorScope,
		Repair:                  repair,
		GenerateMigrationPrompt: migrationPrompt,
		DestinationStyle:        destinationStyle,
		DestinationHome:         home,
		Now:                     a.Now(),
	})
	if err != nil {
		return err
	}

	if migrationPrompt {
		if strings.TrimSpace(outputPath) != "" {
			if err := os.WriteFile(outputPath, []byte(result.MigrationPrompt), 0o644); err != nil {
				return fmt.Errorf("write doctor migration prompt: %w", err)
			}
			_, _ = fmt.Fprintf(a.Stdout, "wrote migration prompt to %s\n", outputPath)
			return nil
		}
		_, _ = fmt.Fprint(a.Stdout, result.MigrationPrompt)
		return nil
	}

	if jsonOutput {
		encoder := json.NewEncoder(a.Stdout)
		encoder.SetEscapeHTML(false)
		return encoder.Encode(result)
	}
	if explain {
		a.renderDoctorExplain(repoRoot, result)
		return nil
	}

	a.renderDoctorConcise(repoRoot, result)
	return nil
}

func (a *App) runRelease(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"release"})
		return nil
	}

	fs := flag.NewFlagSet("release", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var version string
	var pre string
	var bump string
	var accept bool
	var override bool

	fs.StringVar(&version, "version", "", "Release version")
	fs.StringVar(&pre, "pre", "", "Prerelease label")
	fs.StringVar(&bump, "bump", "", "Override detail: choose patch, minor, or major")
	fs.BoolVar(&accept, "accept", false, "Accept the current recommendation without prompting")
	fs.BoolVar(&override, "override", false, "Override the current recommendation")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("usage: changes release [--accept] [--accept --pre label] [--override --bump patch|minor|major] [--override --bump patch|minor|major --pre label] [--override --version version]")
	}
	if accept && override {
		return fmt.Errorf("release: --accept and --override cannot be combined")
	}
	if accept && strings.TrimSpace(bump) != "" {
		return fmt.Errorf("release: --accept cannot be combined with --bump")
	}
	if accept && strings.TrimSpace(version) != "" {
		return fmt.Errorf("release: --accept cannot be combined with --version")
	}
	if !override && strings.TrimSpace(bump) != "" {
		return fmt.Errorf("release: --bump requires --override")
	}
	if !override && strings.TrimSpace(version) != "" {
		return fmt.Errorf("release: --version requires --override")
	}
	if override && strings.TrimSpace(version) != "" && strings.TrimSpace(pre) != "" {
		return fmt.Errorf("release: --pre cannot be combined with --override --version")
	}
	if override && strings.TrimSpace(version) != "" && strings.TrimSpace(bump) != "" {
		return fmt.Errorf("release: --version cannot be combined with --bump")
	}
	if override && strings.TrimSpace(version) == "" && strings.TrimSpace(bump) == "" {
		return fmt.Errorf("release: --override requires --bump or --version")
	}
	if !a.isTTY() && !accept && !override {
		return fmt.Errorf("release: non-interactive use requires --accept or --override")
	}

	repoRoot, err := a.repoRoot(ctx)
	if err != nil {
		return err
	}

	plan, err := appsvc.PlanRelease(ctx, appsvc.ReleasePlanRequest{
		RepoRoot:         repoRoot,
		RequestedVersion: version,
		RequestedPre:     pre,
		RequestedBump:    bump,
		Now:              a.Now(),
	})
	if err != nil {
		return err
	}
	a.printAuthorityWarnings(repoRoot, plan.AuthorityWarnings)

	if accept && plan.ChosenBump == versioning.BumpNone {
		return fmt.Errorf("release: no version bump was inferred; use --override --bump or --override --version")
	}

	if a.isTTY() && !accept && !override {
		plan, err = a.confirmReleaseSelection(ctx, plan, pre)
		if err != nil {
			return err
		}
	}

	result, err := appsvc.CommitRelease(ctx, plan)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", result.Path)
	return nil
}

func (a *App) runRender(ctx context.Context, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		a.printHelp([]string{"render"})
		return nil
	}
	if len(args) > 0 && args[0] == "profiles" && wantsHelp(args[1:]) {
		a.printHelp([]string{"render", "profiles"})
		return nil
	}
	if len(args) > 0 && args[0] == "profiles" {
		return a.runRenderProfiles(ctx, args[1:])
	}
	if wantsHelp(args) {
		a.printHelp([]string{"render"})
		return nil
	}

	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var version string
	var recordPath string
	var profile string
	var outputPath string
	var product string
	var latest bool

	fs.StringVar(&version, "version", "", "Release version")
	fs.StringVar(&recordPath, "record", "", "Explicit release record path")
	fs.StringVar(&profile, "profile", config.RenderProfileGitHubRelease, "Render profile")
	fs.StringVar(&outputPath, "output", "", "Output path")
	fs.StringVar(&product, "product", "", "Product name")
	fs.BoolVar(&latest, "latest", false, "Render from the latest final release for the selected product")

	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := a.repoRoot(ctx)
	if err != nil {
		return err
	}
	result, err := appsvc.Render(ctx, appsvc.RenderRequest{
		RepoRoot:   repoRoot,
		Version:    version,
		RecordPath: recordPath,
		Profile:    profile,
		Product:    product,
		Latest:     latest,
	})
	if err != nil {
		return err
	}
	a.printAuthorityWarnings(repoRoot, result.AuthorityWarnings)

	if strings.TrimSpace(outputPath) != "" {
		if err := os.WriteFile(outputPath, []byte(result.Content), 0o644); err != nil {
			return fmt.Errorf("write render output: %w", err)
		}
		return nil
	}

	_, _ = fmt.Fprint(a.Stdout, result.Content)
	return nil
}

func (a *App) runRenderProfiles(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"render", "profiles"})
		return nil
	}
	fs := flag.NewFlagSet("render profiles", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := a.repoRoot(ctx)
	if err != nil {
		return err
	}
	cfg, authorityCheck, err := config.LoadWithAuthority(repoRoot)
	if err != nil {
		return err
	}
	a.printAuthorityWarnings(repoRoot, authorityCheck.Warnings)

	packs, err := render.AvailablePacks(cfg)
	if err != nil {
		return err
	}
	for _, pack := range packs {
		_, _ = fmt.Fprintf(a.Stdout, "%s\t%s\t%s\n", pack.Name, pack.Mode, pack.Description)
	}
	return nil
}

func (a *App) repoRoot(ctx context.Context) (string, error) {
	_ = ctx
	_, root, insideRepo, err := a.currentLocation()
	if err != nil {
		return "", err
	}
	if !insideRepo {
		return "", fmt.Errorf("detect repo root: %w", reporoot.ErrNotGitRepo)
	}
	return root, nil
}

func (a *App) currentLocation() (string, string, bool, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", false, fmt.Errorf("get working directory: %w", err)
	}
	root, err := reporoot.Detect(cwd)
	if err != nil {
		if errors.Is(err, reporoot.ErrNotGitRepo) {
			return cwd, "", false, nil
		}
		return "", "", false, fmt.Errorf("detect repo root: %w", err)
	}
	return cwd, root, true, nil
}

func (a *App) fail(err error) error {
	var authorityErr *config.AuthorityError
	if errors.As(err, &authorityErr) && authorityErr.Status == config.StatusAmbiguous {
		_, _ = fmt.Fprintf(
			a.Stderr,
			"error: %v; next: changes doctor --migration-prompt --scope %s --to xdg|home [--home PATH]\n",
			err,
			authorityErr.Scope,
		)
		return err
	}
	_, _ = fmt.Fprintf(a.Stderr, "error: %v\n", err)
	return err
}

func (a *App) printAuthorityWarnings(repoRoot string, warnings []config.AuthorityWarning) {
	for _, warning := range warnings {
		path := warning.Path
		if warning.Scope == config.ScopeRepo && strings.TrimSpace(repoRoot) != "" {
			path = repoRelativePath(repoRoot, warning.Path)
		}
		_, _ = fmt.Fprintf(
			a.Stderr,
			"warning: %s authority found %s %s sibling at %s\n",
			warning.Scope,
			authorityWarningStatusText(warning.Status),
			warning.Style,
			path,
		)
	}
}

func authorityWarningStatusText(status config.ResolutionStatus) string {
	switch status {
	case config.StatusLegacyOnly:
		return "legacy-only"
	case config.StatusInvalid:
		return "invalid-manifest"
	default:
		return string(status)
	}
}

func isHelpArg(arg string) bool {
	return arg == "--help" || arg == "-h" || arg == "help"
}

func wantsHelp(args []string) bool {
	for _, arg := range args {
		if isHelpArg(arg) {
			return true
		}
	}
	return false
}

func isVersionArg(arg string) bool {
	return arg == "--version" || arg == "-version"
}

func (a *App) printHelp(path []string) {
	var body string
	switch strings.Join(path, " ") {
	case "":
		body = strings.TrimSpace(`
	changes is a language-agnostic changelog and release-notes tool for Git repositories.

	Usage:
	  changes <command> [options]

Commands:
  version
  init
  create
  status
  doctor
  release
  render
  render profiles

Use "changes help <command>" or "changes <command> --help" for details.
`)
	case "init":
		body = strings.TrimSpace(`
Usage:
  changes init [--current-version <semver|unreleased>] [--layout xdg|home] [--home PATH]

Initialize repo-local config, changelog, prompts, and state directories.
`)
	case "init global":
		body = strings.TrimSpace(`
Usage:
  changes init global [--layout xdg|home] [--home PATH]

Initialize global config, data, and state directories plus the layout manifest.
`)
	case "version":
		body = strings.TrimSpace(`
Usage:
  changes --version
  changes version

Show the CLI build version.
`)
	case "create":
		body = strings.TrimSpace(`
Usage:
  changes create [body] [--public-api <add|change|remove>] [--behavior <new|fix|redefine>] [--dependency <refresh|relax|restrict>] [--runtime <expand|reduce>] [--edit] [options]

Options:
  --body <text>                   Script-friendly body flag
  --public-api <value>            Public API impact: add, change, or remove
  --behavior <value>              Behavior impact: new, fix, or redefine
  --dependency <value>            Dependency compatibility: refresh, relax, or restrict
  --runtime <value>               Runtime support: expand or reduce
  --type <value>                  Optional render grouping: added, changed, or fixed
  --name <value>                  Optional filename stem hint
  --edit                          Open the configured editor with a scaffolded fragment
  --scope <value>                 Repeatable fragment scope
  --section-key <value>           Section key for rendering
  --area <value>                  Product area hint
  --platform <value>              Repeatable platform hint
  --audience <value>              Repeatable audience hint
  --customer-visible              Mark entry as customer visible
  --support-relevance             Mark entry as support relevant
  --requires-action               Mark entry as requiring operator action
  --release-notes-priority <n>    Priority for release-note inclusion
  --display-order <n>             Order within a section
  --breaking                      Mark entry as breaking
`)
	case "status":
		body = strings.TrimSpace(`
Usage:
  changes status [--explain]

Show the current version state, unreleased fragment counts, the policy-derived recommended bump, the recommended next final version, and active prerelease heads.
`)
	case "doctor":
	body = strings.TrimSpace(`
Usage:
  changes doctor [--scope global|repo|all] [--explain] [--json]
  changes doctor --scope repo --repair
  changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]

Inspect layout authority and migration state without mutating any layout.
Repair one legacy repo-local layout by restoring its authoritative manifest without moving data.

Options:
  --scope <global|repo|all>       Scope to inspect
  --explain                       Show candidate and warning detail
  --json                          Emit structured inspection JSON
  --repair                        Restore one repo-local legacy layout manifest
  --migration-prompt              Generate an advisory Markdown migration brief
  --to <xdg|home>                 Destination layout style for migration prompts
  --home <path>                   Destination home path when --to home
  --output <path>                 Write the migration prompt to a file
`)
	case "release":
		body = strings.TrimSpace(`
Usage:
  changes release
  changes release --accept
  changes release --accept --pre <label>
  changes release --override --bump <patch|minor|major>
  changes release --override --bump <patch|minor|major> --pre <label>
  changes release --override --version <version>

Create a base release record for the selected release. In a TTY, the command shows the release evidence and lets you accept or override the recommendation. In non-interactive use, choose either --accept or --override.
`)
	case "render":
		body = strings.TrimSpace(`
Usage:
  changes render --version <version> [--product <name>] [--profile <name>] [--output <path>]
  changes render --latest [--product <name>] [--profile <name>] [--output <path>]
  changes render --record <path> [--profile <name>] [--output <path>]
  changes render profiles

Render one release or a release lineage through the selected template pack.
`)
	case "render profiles":
		body = strings.TrimSpace(`
Usage:
  changes render profiles

List the available render profiles.
`)
	default:
		body = strings.TrimSpace(`
Usage:
  changes <command> [options]

Use "changes help" to see the available commands.
`)
	}
	_, _ = fmt.Fprintln(a.Stdout, body)
}

func (a *App) printVersion() {
	version := strings.TrimSpace(a.Version)
	if version == "" {
		version = "dev"
	}
	_, _ = fmt.Fprintf(a.Stdout, "changes %s\n", version)
}

func renderRecommendationExplanation(w io.Writer, recommendation semverpolicy.Recommendation) {
	_, _ = fmt.Fprintln(w, "Policy evidence:")
	_, _ = fmt.Fprintf(w, "- Public API policy: %s\n", recommendation.Stability)
	_, _ = fmt.Fprintf(w, "- Recommended bump: %s\n", recommendation.SuggestedBump)
	if len(recommendation.Assessments) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "Fragment evidence:")
	for _, item := range recommendation.Assessments {
		_, _ = fmt.Fprintf(w, "- %s => %s\n", item.FragmentID, item.SuggestedBump)
		for _, reason := range item.Reasons {
			_, _ = fmt.Fprintf(w, "  %s\n", reason)
		}
	}
}

func (a *App) confirmReleaseSelection(ctx context.Context, plan appsvc.ReleasePlan, pre string) (appsvc.ReleasePlan, error) {
	renderReleaseDecisionSummary(a.Stdout, plan)

	prompt := "Press Enter to accept the recommendation, choose patch/minor/major to override, or type cancel: "
	if plan.ChosenBump == versioning.BumpNone {
		prompt = "No version bump was inferred. Choose patch/minor/major to override, or type cancel: "
	}

	answer, err := a.promptOptionalLine(a.newPromptReader(), prompt)
	if err != nil {
		return appsvc.ReleasePlan{}, err
	}

	switch strings.TrimSpace(strings.ToLower(answer)) {
	case "":
		if plan.ChosenBump == versioning.BumpNone {
			return appsvc.ReleasePlan{}, fmt.Errorf("release: choose patch, minor, major, or cancel")
		}
		return plan, nil
	case "cancel":
		return appsvc.ReleasePlan{}, fmt.Errorf("release canceled")
	case string(versioning.BumpPatch), string(versioning.BumpMinor), string(versioning.BumpMajor):
		return appsvc.PlanRelease(ctx, appsvc.ReleasePlanRequest{
			RepoRoot:      plan.RepoRoot,
			RequestedPre:  pre,
			RequestedBump: answer,
			Now:           a.Now(),
		})
	default:
		return appsvc.ReleasePlan{}, fmt.Errorf("release: choose patch, minor, major, or cancel")
	}
}

func renderReleaseDecisionSummary(w io.Writer, plan appsvc.ReleasePlan) {
	_, _ = fmt.Fprintf(w, "Pending fragments: %d\n", len(plan.PendingFragments))
	for _, item := range plan.PendingFragments {
		_, _ = fmt.Fprintf(w, "- %s [%s] %s\n", item.ID, item.Type, item.BodyPreview())
	}
	renderRecommendationExplanation(w, plan.Recommendation)
	if plan.Recommendation.SuggestedBump == versioning.BumpNone {
		_, _ = fmt.Fprintln(w, "Default release: none inferred")
		return
	}
	_, _ = fmt.Fprintf(w, "Default release: %s\n", plan.ChosenVersion.String())
}

func repoRelativePath(repoRoot, path string) string {
	if rel, err := filepath.Rel(repoRoot, path); err == nil {
		return rel
	}
	return path
}

func parseDoctorScope(raw string) (appsvc.DoctorScope, error) {
	switch strings.TrimSpace(raw) {
	case string(appsvc.DoctorScopeGlobal):
		return appsvc.DoctorScopeGlobal, nil
	case string(appsvc.DoctorScopeRepo):
		return appsvc.DoctorScopeRepo, nil
	case string(appsvc.DoctorScopeAll):
		return appsvc.DoctorScopeAll, nil
	default:
		return "", fmt.Errorf("doctor: --scope must be global, repo, or all")
	}
}

func parseDoctorStyle(raw string) (config.Style, error) {
	switch strings.TrimSpace(raw) {
	case string(config.StyleXDG):
		return config.StyleXDG, nil
	case string(config.StyleHome):
		return config.StyleHome, nil
	default:
		return "", fmt.Errorf("doctor: --to must be xdg or home")
	}
}

func (a *App) renderDoctorConcise(repoRoot string, result appsvc.DoctorResult) {
	for _, scopeResult := range doctorScopeResults(result) {
		if scopeResult.Repair != nil {
			_, _ = fmt.Fprintf(
				a.Stdout,
				"%s: repaired %s\n",
				scopeResult.Scope,
				scopeResult.Repair.AuthoritativeStyle,
			)
			_, _ = fmt.Fprintf(a.Stdout, "manifest: %s\n", doctorDisplayPath(repoRoot, scopeResult.Scope, scopeResult.Repair.ManifestPath))
			_, _ = fmt.Fprintf(a.Stdout, "authoritative: %s\n", scopeResult.Repair.AuthoritativeStyle)
			_, _ = fmt.Fprintf(a.Stdout, "root: %s\n", doctorDisplayPath(repoRoot, scopeResult.Scope, scopeResult.Repair.AuthoritativeRoot))
			if scopeResult.Repair.GitignoreUpdated {
				_, _ = fmt.Fprintln(a.Stdout, "updated .gitignore")
			}
			continue
		}
		line := fmt.Sprintf("%s: %s", scopeResult.Scope, scopeResult.Status)
		if scopeResult.SelectedStyle != "" {
			line = fmt.Sprintf("%s %s %s", line, scopeResult.SelectedStyle, doctorDisplayPath(repoRoot, scopeResult.Scope, scopeResult.SelectedRoot))
		}
		if len(scopeResult.Warnings) > 0 {
			line = fmt.Sprintf("%s (%d warning%s)", line, len(scopeResult.Warnings), pluralSuffix(len(scopeResult.Warnings)))
		}
		_, _ = fmt.Fprintln(a.Stdout, line)
	}
}

func (a *App) renderDoctorExplain(repoRoot string, result appsvc.DoctorResult) {
	for index, scopeResult := range doctorScopeResults(result) {
		if index > 0 {
			_, _ = fmt.Fprintln(a.Stdout)
		}
		_, _ = fmt.Fprintf(a.Stdout, "%s\n", strings.ToUpper(scopeResult.Scope))
		_, _ = fmt.Fprintf(a.Stdout, "Status: %s\n", scopeResult.Status)
		if scopeResult.SelectedStyle != "" {
			_, _ = fmt.Fprintf(a.Stdout, "Selected: %s at %s\n", scopeResult.SelectedStyle, doctorDisplayPath(repoRoot, scopeResult.Scope, scopeResult.SelectedRoot))
		}
		if len(scopeResult.PrecedenceInputs) > 0 {
			_, _ = fmt.Fprintf(a.Stdout, "Precedence inputs: %s\n", strings.Join(scopeResult.PrecedenceInputs, ", "))
		}
		_, _ = fmt.Fprintln(a.Stdout, "Candidates:")
		for _, candidate := range scopeResult.Candidates {
			flags := make([]string, 0, 2)
			if candidate.IsAuthoritative {
				flags = append(flags, "authoritative")
			}
			if candidate.IsPreferred && !candidate.IsAuthoritative {
				flags = append(flags, "preferred")
			}
			label := candidate.Status
			if len(flags) > 0 {
				label = fmt.Sprintf("%s (%s)", label, strings.Join(flags, ", "))
			}
			_, _ = fmt.Fprintf(
				a.Stdout,
				"- %s: %s at %s\n",
				candidate.Style,
				label,
				doctorDisplayPath(repoRoot, scopeResult.Scope, candidate.Paths.Root),
			)
			for _, evidence := range candidate.Evidence {
				path := evidence.Path
				if evidence.Kind == "file" && scopeResult.Scope == string(config.ScopeRepo) {
					path = doctorDisplayPath(repoRoot, scopeResult.Scope, evidence.Path)
				}
				_, _ = fmt.Fprintf(a.Stdout, "  - %s %s exists=%t path=%s\n", evidence.Kind, evidence.Name, evidence.Exists, path)
			}
		}
		_, _ = fmt.Fprintln(a.Stdout, "Warnings:")
		if len(scopeResult.Warnings) == 0 {
			_, _ = fmt.Fprintln(a.Stdout, "- none")
		}
		for _, warning := range scopeResult.Warnings {
			_, _ = fmt.Fprintf(
				a.Stdout,
				"- %s %s at %s\n",
				warning.Status,
				warning.Style,
				doctorDisplayPath(repoRoot, scopeResult.Scope, warning.Path),
			)
		}
		if scopeResult.RepairHint != "" {
			_, _ = fmt.Fprintf(a.Stdout, "Repair hint: %s\n", scopeResult.RepairHint)
		}
	}
}

func doctorScopeResults(result appsvc.DoctorResult) []appsvc.DoctorScopeResult {
	items := make([]appsvc.DoctorScopeResult, 0, 2)
	if result.Global != nil {
		items = append(items, *result.Global)
	}
	if result.Repo != nil {
		items = append(items, *result.Repo)
	}
	return items
}

func doctorDisplayPath(repoRoot string, scope string, path string) string {
	if scope == string(config.ScopeRepo) && strings.TrimSpace(repoRoot) != "" {
		return repoRelativePath(repoRoot, path)
	}
	return path
}

func pluralSuffix(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
