package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/example/changes/internal/changelog"
	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/render"
	"github.com/example/changes/internal/reporoot"
	"github.com/example/changes/internal/semverpolicy"
	"github.com/example/changes/internal/templates"
	"github.com/example/changes/internal/versioning"
)

type App struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	Now        func() time.Time
	Random     io.Reader
	HTTPClient any
	IsTTY      func() bool
	EditFile   func(path string) error
	promptIn   io.Reader
}

func NewApp(stdout, stderr io.Writer) *App {
	app := &App{
		Stdout: stdout,
		Stderr: stderr,
		Stdin:  os.Stdin,
		Now:    time.Now,
	}
	app.IsTTY = app.defaultIsTTY
	app.EditFile = app.defaultEditFile
	return app
}

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return a.fail(fmt.Errorf("usage: changes <command>"))
	}
	if isHelpArg(args[0]) {
		a.printHelp(args[1:])
		return nil
	}
	if isHelpArg(args[len(args)-1]) && len(args) == 1 {
		a.printHelp(nil)
		return nil
	}

	handled, err := a.runOptionalCommand(ctx, args)
	if err != nil {
		return a.fail(err)
	}
	if handled {
		return nil
	}

	switch args[0] {
	case "init":
		err = a.runInit(ctx, args[1:])
	case "create":
		err = a.runCreate(ctx, args[1:])
	case "status":
		err = a.runStatus(ctx, args[1:])
	case "release":
		err = a.runRelease(ctx, args[1:])
	case "render":
		err = a.runRender(ctx, args[1:])
	case "resolve":
		err = a.runResolve(ctx, args[1:])
	case "changelog":
		err = a.runChangelog(ctx, args[1:])
	default:
		err = fmt.Errorf("unknown command %q", args[0])
	}

	if err != nil {
		return a.fail(err)
	}
	return nil
}

func (a *App) runInit(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"init"})
		return nil
	}
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, err := a.repoRoot(ctx)
	if err != nil {
		return err
	}

	cfg := config.Default()
	if cfg.Project.Name == "" {
		cfg.Project.Name = filepath.Base(repoRoot)
	}

	for _, dir := range []string{
		config.FragmentsDir(repoRoot, cfg),
		config.ReleasesDir(repoRoot, cfg),
		config.TemplatesDir(repoRoot, cfg),
		config.StateDir(repoRoot, cfg),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if _, err := os.Stat(config.RepoConfigPath(repoRoot)); os.IsNotExist(err) {
		if err := config.Write(config.RepoConfigPath(repoRoot), cfg); err != nil {
			return err
		}
	}

	if _, err := templates.EnsureDefaultFiles(repoRoot, cfg); err != nil {
		return err
	}

	changelogPath := config.ChangelogPath(repoRoot, cfg)
	if _, err := os.Stat(changelogPath); os.IsNotExist(err) {
		if err := os.WriteFile(changelogPath, []byte("# Changelog\n"), 0o644); err != nil {
			return fmt.Errorf("write starter changelog: %w", err)
		}
	}

	if err := ensureGitignore(repoRoot); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "initialized %s\n", repoRoot)
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

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	allFragments, records, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	product := cfg.Project.Name
	unreleased, err := releases.UnreleasedFinalFragments(allFragments, records, product)
	if err != nil {
		return err
	}
	highest := highestPendingBump(unreleased)
	nextStable, err := recommendedStableVersion(cfg, records, product, unreleased)
	if err != nil {
		return err
	}
	policy, err := recommendationPolicy(cfg, unreleased)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "Unreleased fragments: %d\n", len(unreleased))
	_, _ = fmt.Fprintf(a.Stdout, "Highest declared bump: %s\n", highest)
	_, _ = fmt.Fprintf(a.Stdout, "Recommended next stable: %s\n", nextStable.String())
	if explain {
		renderRecommendationExplanation(a.Stdout, policy)
	}

	prereleaseHeads := releases.PrereleaseHeads(records, product)
	if len(prereleaseHeads) > 0 {
		_, _ = fmt.Fprintln(a.Stdout, "Active prerelease heads:")
		for _, head := range prereleaseHeads {
			_, _ = fmt.Fprintf(a.Stdout, "- %s -> %s\n", head.Version, head.TargetVersion())
		}
	}

	if len(unreleased) > 0 {
		_, _ = fmt.Fprintln(a.Stdout, "Pending fragments:")
		for _, item := range unreleased {
			_, _ = fmt.Fprintf(a.Stdout, "- %s [%s/%s] %s\n", item.ID, item.Type, item.Bump, item.BodyPreview())
		}
	}

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
	var yes bool

	fs.StringVar(&version, "version", "", "Release version")
	fs.StringVar(&pre, "pre", "", "Prerelease label")
	fs.StringVar(&bump, "bump", "", "Override the recommended bump")
	fs.BoolVar(&yes, "yes", false, "Accept the current recommendation without prompting")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("usage: changes release [--version v] [--pre label] [--bump patch|minor|major] [--yes]")
	}
	if strings.TrimSpace(version) != "" && strings.TrimSpace(pre) != "" {
		return fmt.Errorf("release: --version cannot be combined with --pre")
	}
	if strings.TrimSpace(version) != "" && strings.TrimSpace(bump) != "" {
		return fmt.Errorf("release: --version cannot be combined with --bump")
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	allFragments, records, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	product := cfg.Project.Name
	unreleased, err := releases.UnreleasedFinalFragments(allFragments, records, product)
	if err != nil {
		return err
	}
	policy, err := recommendationPolicy(cfg, unreleased)
	if err != nil {
		return err
	}
	releaseVersion, releaseBump, err := selectReleaseVersion(cfg, records, product, allFragments, version, pre, bump)
	if err != nil {
		return err
	}
	if !a.isTTY() && !yes && strings.TrimSpace(version) == "" && strings.TrimSpace(bump) == "" {
		return fmt.Errorf("release: non-interactive use requires --yes, --bump, or --version")
	}
	if a.isTTY() && !yes && strings.TrimSpace(version) == "" {
		releaseVersion, releaseBump, err = a.confirmReleaseSelection(releaseVersion, releaseBump, pre, unreleased, policy, cfg, records, product)
		if err != nil {
			return err
		}
	}

	selected, err := selectReleaseFragments(allFragments, records, product, releaseVersion)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(selected))
	for _, item := range selected {
		ids = append(ids, item.ID)
	}

	parentVersion := selectParentVersion(records, product, releaseVersion)
	record := releases.ReleaseRecord{
		Product:          product,
		Version:          releaseVersion.String(),
		ParentVersion:    parentVersion,
		CreatedAt:        a.Now().UTC().Truncate(time.Second),
		AddedFragmentIDs: ids,
	}

	if err := releases.ValidateSet(append(slices.Clone(records), record)); err != nil {
		return err
	}

	path, err := releases.Write(repoRoot, cfg, record)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", path)
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

	fs.StringVar(&version, "version", "", "Release version")
	fs.StringVar(&recordPath, "record", "", "Explicit release record path")
	fs.StringVar(&profile, "profile", config.RenderProfileGitHubRelease, "Render profile")
	fs.StringVar(&outputPath, "output", "", "Output path")
	fs.StringVar(&product, "product", "", "Product name")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(version) == "" && strings.TrimSpace(recordPath) == "" {
		return fmt.Errorf("render: provide --version or --record")
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	allFragments, records, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}
	if strings.TrimSpace(product) == "" {
		product = cfg.Project.Name
	}

	var record releases.ReleaseRecord
	if strings.TrimSpace(recordPath) != "" {
		record, err = releases.Load(recordPath)
	} else {
		base, findErr := releases.FindBaseRecord(records, product, version)
		if findErr != nil {
			return findErr
		}
		record = *base
	}

	renderer, err := render.New(repoRoot, cfg, profile)
	if err != nil {
		return err
	}
	doc, err := selectRenderDocument(renderer.Pack(), record, records, allFragments)
	if err != nil {
		return err
	}
	content, err := renderer.Render(doc)
	if err != nil {
		return err
	}

	if strings.TrimSpace(outputPath) != "" {
		if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write render output: %w", err)
		}
		return nil
	}

	_, _ = fmt.Fprint(a.Stdout, content)
	return nil
}

func (a *App) runResolve(ctx context.Context, args []string) error {
	if wantsHelp(args) {
		a.printHelp([]string{"resolve"})
		return nil
	}
	fs := flag.NewFlagSet("resolve", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var product string
	var version string
	var format string
	var outputPath string

	fs.StringVar(&product, "product", "", "Product name")
	fs.StringVar(&version, "version", "", "Release version")
	fs.StringVar(&format, "format", "json", "Output format")
	fs.StringVar(&outputPath, "output", "", "Output path")

	if err := fs.Parse(args); err != nil {
		return err
	}
	if strings.TrimSpace(version) == "" {
		return fmt.Errorf("resolve: provide --version")
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(product) == "" {
		product = cfg.Project.Name
	}

	allFragments, records, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	base, err := releases.FindBaseRecord(records, product, version)
	if err != nil {
		return err
	}
	bundle, err := releases.AssembleRelease(*base, records, allFragments)
	if err != nil {
		return err
	}

	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "json":
	default:
		return fmt.Errorf("resolve: unsupported format %q", format)
	}

	body, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		return fmt.Errorf("resolve: marshal bundle: %w", err)
	}
	body = append(body, '\n')

	if strings.TrimSpace(outputPath) != "" {
		if err := os.WriteFile(outputPath, body, 0o644); err != nil {
			return fmt.Errorf("resolve: write output: %w", err)
		}
		return nil
	}

	_, _ = a.Stdout.Write(body)
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

	_, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	for _, pack := range render.AvailablePacks(cfg) {
		_, _ = fmt.Fprintf(a.Stdout, "%s\t%s\t%s\n", pack.Name, pack.Mode, pack.Description)
	}
	return nil
}

func (a *App) runChangelog(ctx context.Context, args []string) error {
	if len(args) == 1 && isHelpArg(args[0]) {
		a.printHelp([]string{"changelog"})
		return nil
	}
	if len(args) >= 2 && args[0] == "rebuild" && wantsHelp(args[1:]) {
		a.printHelp([]string{"changelog", "rebuild"})
		return nil
	}
	if len(args) == 0 || args[0] != "rebuild" {
		return fmt.Errorf("usage: changes changelog rebuild")
	}

	fs := flag.NewFlagSet("changelog rebuild", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var outputPath string
	fs.StringVar(&outputPath, "output", "", "Output path")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	allFragments, records, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	content, err := changelog.Rebuild(repoRoot, cfg, allFragments, records)
	if err != nil {
		return err
	}

	path := config.ChangelogPath(repoRoot, cfg)
	if err := changelog.Write(repoRoot, cfg, content); err != nil {
		return err
	}

	if strings.TrimSpace(outputPath) != "" {
		if err := os.WriteFile(outputPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("write changelog output: %w", err)
		}
		path = outputPath
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", path)
	return nil
}

func (a *App) repoRoot(ctx context.Context) (string, error) {
	_ = ctx
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	root, err := reporoot.Detect(cwd)
	if err != nil {
		return "", fmt.Errorf("detect repo root: %w", err)
	}
	return root, nil
}

func (a *App) loadConfig(ctx context.Context) (string, config.Config, error) {
	repoRoot, err := a.repoRoot(ctx)
	if err != nil {
		return "", config.Config{}, err
	}

	cfg, err := config.Load(repoRoot)
	if err != nil {
		return "", config.Config{}, err
	}

	return repoRoot, cfg, nil
}

func (a *App) loadState(repoRoot string, cfg config.Config) ([]fragments.Fragment, []releases.ReleaseRecord, error) {
	allFragments, err := fragments.List(repoRoot, cfg)
	if err != nil {
		return nil, nil, err
	}
	records, err := releases.List(repoRoot, cfg)
	if err != nil {
		return nil, nil, err
	}
	return allFragments, records, nil
}

func (a *App) fail(err error) error {
	_, _ = fmt.Fprintf(a.Stderr, "error: %v\n", err)
	return err
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

func (a *App) printHelp(path []string) {
	var body string
	switch strings.Join(path, " ") {
	case "":
		body = strings.TrimSpace(`
changes is a fragment-driven changelog and release-notes CLI.

Usage:
  changes <command> [options]

Commands:
  init
  create
  status
  release
  resolve
  render
  render profiles
  changelog rebuild

Use "changes help <command>" or "changes <command> --help" for details.
`)
	case "init":
		body = strings.TrimSpace(`
Usage:
  changes init

Initialize repo-local config, templates, changelog, and state directories.
`)
	case "create":
		body = strings.TrimSpace(`
Usage:
  changes create <patch|minor|major> [body] [--public-api <add|change|remove>] [--behavior <new|fix|redefine>] [--dependency <refresh|relax|restrict>] [--runtime <expand|reduce>] [--edit] [options]

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

Show unreleased fragment counts, the highest pending bump, the recommended next stable version, and active prerelease heads.
`)
	case "release":
		body = strings.TrimSpace(`
Usage:
  changes release [--version <version>] [--pre <label>] [--bump <patch|minor|major>] [--yes]

Create a base release record for the selected release. In a TTY, the command shows the release evidence and lets you accept or override the default recommendation unless you pass --yes.
`)
	case "resolve":
		body = strings.TrimSpace(`
Usage:
  changes resolve --version <version> [--product <name>] [--format json] [--output <path>]

Assemble and emit the ReleaseBundle for one release.
`)
	case "render":
		body = strings.TrimSpace(`
Usage:
  changes render --version <version> [--product <name>] [--profile <name>] [--output <path>]
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
	case "changelog":
		body = strings.TrimSpace(`
Usage:
  changes changelog rebuild [--output <path>]

Rebuild the repository changelog from the current final release lineage.
`)
	case "changelog rebuild":
		body = strings.TrimSpace(`
Usage:
  changes changelog rebuild [--output <path>]

Rebuild the repository changelog from the current final release lineage.
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

func recommendedStableVersion(cfg config.Config, records []releases.ReleaseRecord, product string, pending []fragments.Fragment) (versioning.Version, error) {
	policy, err := recommendationPolicy(cfg, pending)
	if err != nil {
		return versioning.Version{}, err
	}
	return nextStableForBump(cfg, records, product, policy.SuggestedBump)
}

func recommendationPolicy(cfg config.Config, pending []fragments.Fragment) (semverpolicy.Recommendation, error) {
	return semverpolicy.Evaluate(publicAPIStability(cfg.Versioning.PublicAPI), pending), nil
}

func recommendedPreviewVersion(records []releases.ReleaseRecord, product string, target versioning.Version, label string) versioning.Version {
	current := releases.PrereleaseVersionsForLine(records, product, target.String(), label)
	return versioning.NextPrerelease(target, label, current)
}

func selectReleaseVersion(cfg config.Config, records []releases.ReleaseRecord, product string, allFragments []fragments.Fragment, requestedVersion, requestedPre, requestedBump string) (versioning.Version, versioning.Bump, error) {
	if strings.TrimSpace(requestedVersion) != "" {
		version, err := versioning.Parse(requestedVersion)
		if err != nil {
			return versioning.Version{}, versioning.BumpNone, err
		}
		return version, versioning.BumpNone, nil
	}

	pending, err := releases.UnreleasedFinalFragments(allFragments, records, product)
	if err != nil {
		return versioning.Version{}, versioning.BumpNone, err
	}
	policy, err := recommendationPolicy(cfg, pending)
	if err != nil {
		return versioning.Version{}, versioning.BumpNone, err
	}
	chosenBump := policy.SuggestedBump
	if strings.TrimSpace(requestedBump) != "" {
		chosenBump, err = versioning.NormalizeBump(requestedBump)
		if err != nil {
			return versioning.Version{}, versioning.BumpNone, err
		}
	}
	target, err := nextStableForBump(cfg, records, product, chosenBump)
	if err != nil {
		return versioning.Version{}, versioning.BumpNone, err
	}

	if strings.TrimSpace(requestedPre) == "" {
		return target, chosenBump, nil
	}

	return recommendedPreviewVersion(records, product, target, requestedPre), chosenBump, nil
}

func selectReleaseFragments(allFragments []fragments.Fragment, records []releases.ReleaseRecord, product string, version versioning.Version) ([]fragments.Fragment, error) {
	var (
		selected []fragments.Fragment
		err      error
	)

	if version.IsPrerelease() {
		label, _, ok := version.PrereleaseLabelNumber()
		if !ok {
			return nil, fmt.Errorf("unsupported prerelease format %q", version.String())
		}
		selected, err = releases.UnreleasedPrereleaseFragments(allFragments, records, product, version.Stable().String(), label)
	} else {
		selected, err = releases.UnreleasedFinalFragments(allFragments, records, product)
	}
	if err != nil {
		return nil, err
	}

	if len(selected) == 0 {
		releaseKind := "final"
		if version.IsPrerelease() {
			releaseKind = "prerelease"
		}
		return nil, fmt.Errorf("no fragments available for %s release %s", releaseKind, version.String())
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
	return selected, nil
}

func selectParentVersion(records []releases.ReleaseRecord, product string, releaseVersion versioning.Version) string {
	if releaseVersion.IsPrerelease() {
		label, _, ok := releaseVersion.PrereleaseLabelNumber()
		if !ok {
			return ""
		}
		if head := releases.PrereleaseHead(records, product, releaseVersion.Stable().String(), label); head != nil {
			return head.Version
		}
		if head := releases.LatestFinalHeadForProduct(records, product); head != nil {
			return head.Version
		}
		return ""
	}

	if head := releases.LatestFinalHeadForProduct(records, product); head != nil {
		return head.Version
	}
	return ""
}

func selectRenderDocument(pack render.TemplatePack, head releases.ReleaseRecord, records []releases.ReleaseRecord, allFragments []fragments.Fragment) (render.Document, error) {
	switch pack.Mode {
	case config.RenderModeSingleRelease:
		bundle, err := releases.AssembleRelease(head, records, allFragments)
		if err != nil {
			return render.Document{}, err
		}
		return render.Document{Bundles: []releases.ReleaseBundle{bundle}}, nil
	case config.RenderModeReleaseChain:
		bundles, err := releases.AssembleReleaseLineage(head, records, allFragments)
		if err != nil {
			return render.Document{}, err
		}
		return render.Document{Bundles: bundles}, nil
	default:
		return render.Document{}, fmt.Errorf("render pack %q has unsupported mode %q", pack.Name, pack.Mode)
	}
}

func highestPendingBump(items []fragments.Fragment) versioning.Bump {
	best := versioning.BumpNone
	for _, item := range items {
		bump, err := versioning.NormalizeBump(item.Bump)
		if err != nil {
			continue
		}
		best = versioning.HighestBump(best, bump)
	}
	return best
}

func renderRecommendationExplanation(w io.Writer, recommendation semverpolicy.Recommendation) {
	_, _ = fmt.Fprintln(w, "Policy evidence:")
	_, _ = fmt.Fprintf(w, "- Public API policy: %s\n", recommendation.Stability)
	_, _ = fmt.Fprintf(w, "- Declared bump: %s\n", recommendation.DeclaredBump)
	_, _ = fmt.Fprintf(w, "- Policy-suggested bump: %s\n", recommendation.SuggestedBump)
	if len(recommendation.Assessments) == 0 {
		return
	}
	_, _ = fmt.Fprintln(w, "Fragment evidence:")
	for _, item := range recommendation.Assessments {
		_, _ = fmt.Fprintf(w, "- %s declared=%s policy=%s\n", item.FragmentID, item.DeclaredBump, item.SuggestedBump)
		for _, reason := range item.Reasons {
			_, _ = fmt.Fprintf(w, "  %s\n", reason)
		}
	}
}

func latestStableVersion(records []releases.ReleaseRecord, product string) *versioning.Version {
	if record := releases.LatestFinalHeadForProduct(records, product); record != nil {
		value := versioning.MustParse(record.Version)
		return &value
	}
	return nil
}

func nextStableForBump(cfg config.Config, records []releases.ReleaseRecord, product string, bump versioning.Bump) (versioning.Version, error) {
	initial, err := versioning.Parse(cfg.Project.InitialVersion)
	if err != nil {
		return versioning.Version{}, fmt.Errorf("parse initial version: %w", err)
	}
	return versioning.NextStable(latestStableVersion(records, product), initial, bump), nil
}

func publicAPIStability(raw string) semverpolicy.Stability {
	if strings.TrimSpace(raw) == "stable" {
		return semverpolicy.StabilityStable
	}
	return semverpolicy.StabilityUnstable
}

func (a *App) confirmReleaseSelection(releaseVersion versioning.Version, releaseBump versioning.Bump, pre string, pending []fragments.Fragment, policy semverpolicy.Recommendation, cfg config.Config, records []releases.ReleaseRecord, product string) (versioning.Version, versioning.Bump, error) {
	a.promptIn = nil
	renderReleaseDecisionSummary(a.Stdout, pending, policy, releaseVersion)

	if strings.TrimSpace(pre) != "" {
		answer, err := a.promptOptionalLine("Press Enter to accept, choose patch/minor/major for a different target, or type cancel: ")
		if err != nil {
			return versioning.Version{}, versioning.BumpNone, err
		}
		switch strings.TrimSpace(strings.ToLower(answer)) {
		case "":
			return releaseVersion, releaseBump, nil
		case "cancel":
			return versioning.Version{}, versioning.BumpNone, fmt.Errorf("release canceled")
		case string(versioning.BumpPatch), string(versioning.BumpMinor), string(versioning.BumpMajor):
			override, err := versioning.NormalizeBump(answer)
			if err != nil {
				return versioning.Version{}, versioning.BumpNone, err
			}
			target, err := nextStableForBump(cfg, records, product, override)
			if err != nil {
				return versioning.Version{}, versioning.BumpNone, err
			}
			return recommendedPreviewVersion(records, product, target, pre), override, nil
		default:
			return versioning.Version{}, versioning.BumpNone, fmt.Errorf("release: choose patch, minor, major, or cancel")
		}
	}

	answer, err := a.promptOptionalLine("Press Enter to accept, choose patch/minor/major, or type cancel: ")
	if err != nil {
		return versioning.Version{}, versioning.BumpNone, err
	}
	switch strings.TrimSpace(strings.ToLower(answer)) {
	case "":
		return releaseVersion, releaseBump, nil
	case "cancel":
		return versioning.Version{}, versioning.BumpNone, fmt.Errorf("release canceled")
	case string(versioning.BumpPatch), string(versioning.BumpMinor), string(versioning.BumpMajor):
		override, err := versioning.NormalizeBump(answer)
		if err != nil {
			return versioning.Version{}, versioning.BumpNone, err
		}
		target, err := nextStableForBump(cfg, records, product, override)
		if err != nil {
			return versioning.Version{}, versioning.BumpNone, err
		}
		return target, override, nil
	default:
		return versioning.Version{}, versioning.BumpNone, fmt.Errorf("release: choose patch, minor, major, or cancel")
	}
}

func renderReleaseDecisionSummary(w io.Writer, pending []fragments.Fragment, recommendation semverpolicy.Recommendation, releaseVersion versioning.Version) {
	_, _ = fmt.Fprintf(w, "Pending fragments: %d\n", len(pending))
	for _, item := range pending {
		_, _ = fmt.Fprintf(w, "- %s [%s/%s] %s\n", item.ID, item.Type, item.Bump, item.BodyPreview())
	}
	renderRecommendationExplanation(w, recommendation)
	_, _ = fmt.Fprintf(w, "Default release: %s\n", releaseVersion.String())
}

func ensureGitignore(repoRoot string) error {
	path := filepath.Join(repoRoot, ".gitignore")
	entry := "/.local/state/"

	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read .gitignore: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return nil
		}
	}

	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	lines = append(lines, entry)
	body := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
}
