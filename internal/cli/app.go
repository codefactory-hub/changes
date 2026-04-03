package cli

import (
	"context"
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
	"github.com/example/changes/internal/templates"
	"github.com/example/changes/internal/versioning"
)

type App struct {
	Stdout     io.Writer
	Stderr     io.Writer
	Now        func() time.Time
	Random     io.Reader
	HTTPClient any
}

func NewApp(stdout, stderr io.Writer) *App {
	return &App{
		Stdout: stdout,
		Stderr: stderr,
		Now:    time.Now,
	}
}

func (a *App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return a.fail(fmt.Errorf("usage: changes <command>"))
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
	case "add":
		err = a.runAdd(ctx, args[1:])
	case "status":
		err = a.runStatus(ctx, args[1:])
	case "version":
		err = a.runVersion(ctx, args[1:])
	case "release":
		err = a.runRelease(ctx, args[1:])
	case "render":
		err = a.runRender(ctx, args[1:])
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

func (a *App) runAdd(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var title string
	var kind string
	var bump string
	var body string
	var breaking bool
	var scopes stringSliceFlag

	fs.StringVar(&title, "title", "", "Fragment title")
	fs.StringVar(&kind, "type", "changed", "Fragment type")
	fs.StringVar(&bump, "bump", "patch", "Version bump (patch|minor|major)")
	fs.StringVar(&body, "body", "", "Fragment body")
	fs.BoolVar(&breaking, "breaking", false, "Mark entry as breaking")
	fs.Var(&scopes, "scope", "Fragment scope (repeatable)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(title) == "" {
		return fmt.Errorf("add: --title is required")
	}
	if strings.TrimSpace(body) == "" {
		return fmt.Errorf("add: --body is required in the first layer; editor-based authoring is a documented follow-up")
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	normalizedBump, err := versioning.NormalizeBump(bump)
	if err != nil {
		return err
	}

	item, err := fragments.Create(repoRoot, cfg, a.Now(), a.Random, fragments.NewInput{
		Title:    title,
		Type:     kind,
		Bump:     normalizedBump,
		Breaking: breaking,
		Scopes:   scopes,
		Body:     body,
	})
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", item.Path)
	return nil
}

func (a *App) runStatus(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	if err := fs.Parse(args); err != nil {
		return err
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	allFragments, manifests, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	unreleased, err := releases.UnreleasedStableFragments(allFragments, manifests)
	if err != nil {
		return err
	}
	highest := highestPendingBump(unreleased)
	nextStable, err := recommendedStableVersion(cfg, manifests, unreleased)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "Unreleased fragments: %d\n", len(unreleased))
	_, _ = fmt.Fprintf(a.Stdout, "Highest pending bump: %s\n", highest)
	_, _ = fmt.Fprintf(a.Stdout, "Recommended next stable: %s\n", nextStable.String())

	previewHeads := releases.PreviewHeads(manifests)
	if len(previewHeads) > 0 {
		_, _ = fmt.Fprintln(a.Stdout, "Active preview heads:")
		for _, head := range previewHeads {
			_, _ = fmt.Fprintf(a.Stdout, "- %s -> %s\n", head.Version, head.TargetVersion)
		}
	}

	if len(unreleased) > 0 {
		_, _ = fmt.Fprintln(a.Stdout, "Pending fragments:")
		for _, item := range unreleased {
			_, _ = fmt.Fprintf(a.Stdout, "- %s [%s/%s] %s\n", item.ID, item.Type, item.Bump, item.Title)
		}
	}

	return nil
}

func (a *App) runVersion(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "next" {
		return fmt.Errorf("usage: changes version next [--pre label]")
	}

	fs := flag.NewFlagSet("version next", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var pre string
	fs.StringVar(&pre, "pre", "", "Prerelease label")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	allFragments, manifests, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	unreleased, err := releases.UnreleasedStableFragments(allFragments, manifests)
	if err != nil {
		return err
	}
	nextStable, err := recommendedStableVersion(cfg, manifests, unreleased)
	if err != nil {
		return err
	}

	if strings.TrimSpace(pre) == "" {
		_, _ = fmt.Fprintln(a.Stdout, nextStable.String())
		return nil
	}

	nextPreview := recommendedPreviewVersion(cfg, manifests, nextStable, pre)
	_, _ = fmt.Fprintln(a.Stdout, nextPreview.String())
	return nil
}

func (a *App) runRelease(ctx context.Context, args []string) error {
	if len(args) == 0 || args[0] != "create" {
		return fmt.Errorf("usage: changes release create [--version v] [--pre label] [--channel preview|stable]")
	}

	fs := flag.NewFlagSet("release create", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var version string
	var pre string
	var channel string

	fs.StringVar(&version, "version", "", "Release version")
	fs.StringVar(&pre, "pre", "", "Prerelease label")
	fs.StringVar(&channel, "channel", releases.ChannelStable, "Release channel")

	if err := fs.Parse(args[1:]); err != nil {
		return err
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	allFragments, manifests, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	releaseVersion, targetVersion, err := selectReleaseVersion(cfg, manifests, allFragments, version, pre, channel)
	if err != nil {
		return err
	}

	selected, err := selectReleaseFragments(allFragments, manifests, releaseVersion, channel)
	if err != nil {
		return err
	}

	ids := make([]string, 0, len(selected))
	for _, item := range selected {
		ids = append(ids, item.ID)
	}

	parentVersion := selectParentVersion(manifests, releaseVersion, channel)
	manifest := releases.Manifest{
		Version:          releaseVersion.String(),
		TargetVersion:    targetVersion.String(),
		Channel:          channel,
		ParentVersion:    parentVersion,
		CreatedAt:        a.Now().UTC().Truncate(time.Second),
		AddedFragmentIDs: ids,
	}

	if err := releases.ValidateSet(append(slices.Clone(manifests), manifest)); err != nil {
		return err
	}

	path, err := releases.Write(repoRoot, cfg, manifest)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(a.Stdout, "%s\n", path)
	return nil
}

func (a *App) runRender(ctx context.Context, args []string) error {
	if len(args) > 0 && args[0] == "profiles" {
		return a.runRenderProfiles(ctx, args[1:])
	}

	fs := flag.NewFlagSet("render", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	var version string
	var manifestPath string
	var profile string
	var outputPath string

	fs.StringVar(&version, "version", "", "Manifest version")
	fs.StringVar(&manifestPath, "manifest", "", "Explicit manifest path")
	fs.StringVar(&profile, "profile", config.RenderProfileGitHubRelease, "Render profile")
	fs.StringVar(&outputPath, "output", "", "Output path")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if strings.TrimSpace(version) == "" && strings.TrimSpace(manifestPath) == "" {
		return fmt.Errorf("render: provide --version or --manifest")
	}

	repoRoot, cfg, err := a.loadConfig(ctx)
	if err != nil {
		return err
	}

	_, manifests, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	var manifest releases.Manifest
	if strings.TrimSpace(manifestPath) != "" {
		manifest, err = releases.Load(manifestPath)
	} else {
		manifest, err = releases.Load(releases.ManifestPath(repoRoot, cfg, version))
	}
	if err != nil {
		return err
	}

	allFragments, err := fragments.List(repoRoot, cfg)
	if err != nil {
		return err
	}

	renderer, err := render.New(repoRoot, cfg, profile)
	if err != nil {
		return err
	}
	selector := render.NewSelector(allFragments, manifests)
	doc, err := selectRenderDocument(renderer.Pack(), selector, manifest)
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

func (a *App) runRenderProfiles(ctx context.Context, args []string) error {
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

	allFragments, manifests, err := a.loadState(repoRoot, cfg)
	if err != nil {
		return err
	}

	content, err := changelog.Rebuild(repoRoot, cfg, allFragments, manifests)
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

func (a *App) loadState(repoRoot string, cfg config.Config) ([]fragments.Fragment, []releases.Manifest, error) {
	allFragments, err := fragments.List(repoRoot, cfg)
	if err != nil {
		return nil, nil, err
	}
	manifests, err := releases.List(repoRoot, cfg)
	if err != nil {
		return nil, nil, err
	}
	return allFragments, manifests, nil
}

func (a *App) fail(err error) error {
	_, _ = fmt.Fprintf(a.Stderr, "error: %v\n", err)
	return err
}

func recommendedStableVersion(cfg config.Config, manifests []releases.Manifest, pending []fragments.Fragment) (versioning.Version, error) {
	initial, err := versioning.Parse(cfg.Project.InitialVersion)
	if err != nil {
		return versioning.Version{}, fmt.Errorf("parse initial version: %w", err)
	}

	var latestStable *versioning.Version
	if manifest := releases.LatestStableHead(manifests); manifest != nil {
		value := versioning.MustParse(manifest.Version)
		latestStable = &value
	}

	return versioning.NextStable(latestStable, initial, highestPendingBump(pending)), nil
}

func recommendedPreviewVersion(cfg config.Config, manifests []releases.Manifest, target versioning.Version, label string) versioning.Version {
	if strings.TrimSpace(label) == "" {
		label = cfg.Versioning.PrereleaseLabel
	}
	current := releases.PreviewVersionsForLine(manifests, target.String(), label)
	return versioning.NextPrerelease(target, label, current)
}

func selectReleaseVersion(cfg config.Config, manifests []releases.Manifest, allFragments []fragments.Fragment, requestedVersion, requestedPre, channel string) (versioning.Version, versioning.Version, error) {
	switch channel {
	case releases.ChannelPreview, releases.ChannelStable:
	default:
		return versioning.Version{}, versioning.Version{}, fmt.Errorf("unsupported release channel %q", channel)
	}

	if strings.TrimSpace(requestedVersion) != "" {
		version, err := versioning.Parse(requestedVersion)
		if err != nil {
			return versioning.Version{}, versioning.Version{}, err
		}
		if channel == releases.ChannelStable && version.IsPrerelease() {
			return versioning.Version{}, versioning.Version{}, fmt.Errorf("stable release requires a stable version")
		}
		if channel == releases.ChannelPreview && !version.IsPrerelease() {
			return versioning.Version{}, versioning.Version{}, fmt.Errorf("preview release requires a prerelease version")
		}
		return version, version.Stable(), nil
	}

	pending, err := releases.UnreleasedStableFragments(allFragments, manifests)
	if err != nil {
		return versioning.Version{}, versioning.Version{}, err
	}
	target, err := recommendedStableVersion(cfg, manifests, pending)
	if err != nil {
		return versioning.Version{}, versioning.Version{}, err
	}

	if channel == releases.ChannelStable {
		return target, target, nil
	}

	pre := requestedPre
	if strings.TrimSpace(pre) == "" {
		pre = cfg.Versioning.PrereleaseLabel
	}
	version := recommendedPreviewVersion(cfg, manifests, target, pre)
	return version, target, nil
}

func selectReleaseFragments(allFragments []fragments.Fragment, manifests []releases.Manifest, version versioning.Version, channel string) ([]fragments.Fragment, error) {
	var (
		selected []fragments.Fragment
		err      error
	)

	switch channel {
	case releases.ChannelStable:
		selected, err = releases.UnreleasedStableFragments(allFragments, manifests)
	case releases.ChannelPreview:
		selected, err = releases.UnreleasedPreviewFragments(allFragments, manifests, version.Stable().String(), version.PreLabel)
	default:
		return nil, fmt.Errorf("unsupported release channel %q", channel)
	}
	if err != nil {
		return nil, err
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no fragments available for %s release %s", channel, version.String())
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

func selectParentVersion(manifests []releases.Manifest, releaseVersion versioning.Version, channel string) string {
	switch channel {
	case releases.ChannelStable:
		if head := releases.LatestStableHead(manifests); head != nil {
			return head.Version
		}
	case releases.ChannelPreview:
		if head := releases.PreviewHead(manifests, releaseVersion.Stable().String(), releaseVersion.PreLabel); head != nil {
			return head.Version
		}
	}
	return ""
}

func selectRenderDocument(pack render.TemplatePack, selector *render.Selector, head releases.Manifest) (render.Document, error) {
	switch pack.Mode {
	case config.RenderModeSingleRelease:
		return selector.Release(head)
	case config.RenderModeReleaseChain:
		return selector.ReleaseChain(head)
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

func ensureGitignore(repoRoot string) error {
	path := filepath.Join(repoRoot, ".gitignore")
	entry := "/.local/state/changes/"

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
