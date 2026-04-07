package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/render"
	"github.com/example/changes/internal/semverpolicy"
	"github.com/example/changes/internal/versioning"
)

type InitializeRequest struct {
	RepoRoot        string
	CurrentVersion  string
	RequestedLayout string
	RequestedHome   string
	Now             time.Time
	Random          io.Reader
}

type InitializeResult struct {
	RepoRoot         string
	AuthorityWarnings []config.AuthorityWarning
	AdoptionFragment *fragments.Fragment
	AdoptionRecord   *releases.ReleaseRecord
	PromptPath       string
}

type StatusRequest struct {
	RepoRoot string
}

type StatusResult struct {
	RepoRoot             string
	AuthorityWarnings    []config.AuthorityWarning
	Config               config.Config
	CurrentVersionLabel  string
	CurrentVersionSource string
	InitialReleaseTarget *versioning.Version
	PendingFragments     []fragments.Fragment
	PrereleaseHeads      []releases.ReleaseRecord
	Recommendation       semverpolicy.Recommendation
	RecommendedNextFinal versioning.Version
}

type ReleasePlanRequest struct {
	RepoRoot         string
	RequestedVersion string
	RequestedPre     string
	RequestedBump    string
	Now              time.Time
}

type ReleasePlan struct {
	RepoRoot          string
	AuthorityWarnings []config.AuthorityWarning
	Product           string
	PendingFragments  []fragments.Fragment
	Recommendation    semverpolicy.Recommendation
	ChosenVersion     versioning.Version
	ChosenBump        versioning.Bump
	CreatedAt         time.Time
	SelectedFragments []fragments.Fragment
	ParentVersion     string
	RecommendedChoice bool
}

type CommitReleaseResult struct {
	Path   string
	Record releases.ReleaseRecord
}

type RenderRequest struct {
	RepoRoot   string
	Version    string
	RecordPath string
	Profile    string
	Product    string
	Latest     bool
}

type RenderResult struct {
	RepoRoot          string
	AuthorityWarnings []config.AuthorityWarning
	Config            config.Config
	Record            releases.ReleaseRecord
	Document          render.Document
	Content           string
}

func Status(ctx context.Context, req StatusRequest) (StatusResult, error) {
	if err := checkContext(ctx); err != nil {
		return StatusResult{}, err
	}
	cfg, authorityCheck, err := config.LoadWithAuthority(req.RepoRoot)
	if err != nil {
		return StatusResult{}, err
	}
	allFragments, records, err := loadState(req.RepoRoot, cfg)
	if err != nil {
		return StatusResult{}, err
	}
	if err := checkContext(ctx); err != nil {
		return StatusResult{}, err
	}

	product := cfg.Project.Name
	pending, err := releases.UnreleasedFinalFragments(allFragments, records, product)
	if err != nil {
		return StatusResult{}, err
	}
	recommendation := recommendationPolicy(cfg, pending)
	nextStable, err := nextStableForBump(cfg, records, product, recommendation.SuggestedBump)
	if err != nil {
		return StatusResult{}, err
	}
	currentLabel, source, initialTarget, err := currentVersionStatus(cfg, records, product)
	if err != nil {
		return StatusResult{}, err
	}

	return StatusResult{
		RepoRoot:             req.RepoRoot,
		AuthorityWarnings:    append([]config.AuthorityWarning(nil), authorityCheck.Warnings...),
		Config:               cfg,
		CurrentVersionLabel:  currentLabel,
		CurrentVersionSource: source,
		InitialReleaseTarget: initialTarget,
		PendingFragments:     pending,
		PrereleaseHeads:      releases.PrereleaseHeads(records, product),
		Recommendation:       recommendation,
		RecommendedNextFinal: nextStable,
	}, nil
}

func PlanRelease(ctx context.Context, req ReleasePlanRequest) (ReleasePlan, error) {
	if err := checkContext(ctx); err != nil {
		return ReleasePlan{}, err
	}
	cfg, authorityCheck, err := config.LoadWithAuthority(req.RepoRoot)
	if err != nil {
		return ReleasePlan{}, err
	}
	allFragments, records, err := loadState(req.RepoRoot, cfg)
	if err != nil {
		return ReleasePlan{}, err
	}
	if err := checkContext(ctx); err != nil {
		return ReleasePlan{}, err
	}

	product := cfg.Project.Name
	pending, err := releases.UnreleasedFinalFragments(allFragments, records, product)
	if err != nil {
		return ReleasePlan{}, err
	}
	recommendation := recommendationPolicy(cfg, pending)
	chosenVersion, chosenBump, recommendedChoice, err := selectReleaseVersion(cfg, records, product, allFragments, req.RequestedVersion, req.RequestedPre, req.RequestedBump)
	if err != nil {
		return ReleasePlan{}, err
	}
	selected, err := selectReleaseFragments(allFragments, records, product, chosenVersion)
	if err != nil {
		return ReleasePlan{}, err
	}

	return ReleasePlan{
		RepoRoot:          req.RepoRoot,
		AuthorityWarnings: append([]config.AuthorityWarning(nil), authorityCheck.Warnings...),
		Product:           product,
		PendingFragments:  pending,
		Recommendation:    recommendation,
		ChosenVersion:     chosenVersion,
		ChosenBump:        chosenBump,
		CreatedAt:         releasePlanTime(req.Now),
		SelectedFragments: selected,
		ParentVersion:     selectParentVersion(records, product, chosenVersion),
		RecommendedChoice: recommendedChoice,
	}, nil
}

func CommitRelease(ctx context.Context, plan ReleasePlan) (CommitReleaseResult, error) {
	if err := checkContext(ctx); err != nil {
		return CommitReleaseResult{}, err
	}
	return commitReleaseWithTimestamp(plan, releasePlanTime(plan.CreatedAt))
}

func commitReleaseWithTimestamp(plan ReleasePlan, createdAt time.Time) (CommitReleaseResult, error) {
	currentCfg, _, err := config.LoadWithAuthority(plan.RepoRoot)
	if err != nil {
		return CommitReleaseResult{}, err
	}
	allFragments, currentRecords, err := loadState(plan.RepoRoot, currentCfg)
	if err != nil {
		return CommitReleaseResult{}, err
	}
	currentSelection, err := selectReleaseFragments(allFragments, currentRecords, plan.Product, plan.ChosenVersion)
	if err != nil {
		if len(plan.SelectedFragments) > 0 {
			return CommitReleaseResult{}, fmt.Errorf("release plan is stale; refresh the release plan before committing")
		}
		return CommitReleaseResult{}, err
	}
	if !slices.Equal(fragmentIDs(currentSelection), fragmentIDs(plan.SelectedFragments)) {
		return CommitReleaseResult{}, fmt.Errorf("release plan is stale; refresh the release plan before committing")
	}
	parentVersion := selectParentVersion(currentRecords, plan.Product, plan.ChosenVersion)
	if parentVersion != plan.ParentVersion {
		return CommitReleaseResult{}, fmt.Errorf("release plan is stale; refresh the release plan before committing")
	}

	record := releases.ReleaseRecord{
		Product:          plan.Product,
		Version:          plan.ChosenVersion.String(),
		ParentVersion:    parentVersion,
		CreatedAt:        createdAt.UTC().Truncate(time.Second),
		AddedFragmentIDs: fragmentIDs(plan.SelectedFragments),
	}

	if err := releases.ValidateSet(append(cloneRecords(currentRecords), record)); err != nil {
		return CommitReleaseResult{}, err
	}
	if _, err := config.RequireRepoWriteAuthority(plan.RepoRoot); err != nil {
		return CommitReleaseResult{}, err
	}

	path, err := releases.Write(plan.RepoRoot, currentCfg, record)
	if err != nil {
		return CommitReleaseResult{}, err
	}

	return CommitReleaseResult{Path: path, Record: record}, nil
}

func Render(ctx context.Context, req RenderRequest) (RenderResult, error) {
	if err := checkContext(ctx); err != nil {
		return RenderResult{}, err
	}
	cfg, authorityCheck, err := config.LoadWithAuthority(req.RepoRoot)
	if err != nil {
		return RenderResult{}, err
	}
	allFragments, records, err := loadState(req.RepoRoot, cfg)
	if err != nil {
		return RenderResult{}, err
	}
	if err := checkContext(ctx); err != nil {
		return RenderResult{}, err
	}

	product := strings.TrimSpace(req.Product)
	if product == "" {
		product = cfg.Project.Name
	}

	selectorCount := 0
	if strings.TrimSpace(req.Version) != "" {
		selectorCount++
	}
	if strings.TrimSpace(req.RecordPath) != "" {
		selectorCount++
	}
	if req.Latest {
		selectorCount++
	}
	if selectorCount != 1 {
		return RenderResult{}, fmt.Errorf("render: provide exactly one of --version, --record, or --latest")
	}

	var record releases.ReleaseRecord
	if strings.TrimSpace(req.RecordPath) != "" {
		record, err = releases.Load(req.RecordPath)
	} else if req.Latest {
		head := releases.LatestFinalHeadForProduct(records, product)
		if head == nil {
			return RenderResult{}, fmt.Errorf("render: no final release records exist for product %q", product)
		}
		record = *head
	} else {
		base, findErr := releases.FindBaseRecord(records, product, req.Version)
		if findErr != nil {
			return RenderResult{}, findErr
		}
		record = *base
	}

	profile := req.Profile
	if strings.TrimSpace(profile) == "" {
		profile = config.RenderProfileGitHubRelease
	}
	renderer, err := render.New(req.RepoRoot, cfg, profile)
	if err != nil {
		return RenderResult{}, err
	}
	doc, err := selectRenderDocument(renderer.Pack(), record, records, allFragments)
	if err != nil {
		return RenderResult{}, err
	}
	content, err := renderer.Render(doc)
	if err != nil {
		return RenderResult{}, err
	}

	return RenderResult{
		RepoRoot:          req.RepoRoot,
		AuthorityWarnings: append([]config.AuthorityWarning(nil), authorityCheck.Warnings...),
		Config:            cfg,
		Record:            record,
		Document:          doc,
		Content:           content,
	}, nil
}

func loadState(repoRoot string, cfg config.Config) ([]fragments.Fragment, []releases.ReleaseRecord, error) {
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

func loadExistingOrDefaultConfig(repoRoot string) (config.Config, error) {
	resolution, err := config.ResolveRepo(config.ResolveOptions{RepoRoot: repoRoot})
	if err != nil {
		return config.Config{}, fmt.Errorf("resolve repo layout: %w", err)
	}

	if _, err := config.CheckScopeAuthority(resolution); err != nil {
		var authorityErr *config.AuthorityError
		if errors.As(err, &authorityErr) && authorityErr.Status == config.StatusUninitialized {
			return config.Default(), nil
		}
		return config.Config{}, err
	}

	cfg, _, err := config.LoadWithAuthority(repoRoot)
	if err != nil {
		return config.Config{}, err
	}
	return cfg, nil
}

func recommendationPolicy(cfg config.Config, pending []fragments.Fragment) semverpolicy.Recommendation {
	return semverpolicy.Evaluate(publicAPIStability(cfg.Versioning.PublicAPI), pending)
}

func currentVersionStatus(cfg config.Config, records []releases.ReleaseRecord, product string) (string, string, *versioning.Version, error) {
	if record := releases.LatestFinalHeadForProduct(records, product); record != nil {
		return record.Version, "latest_release_record", nil, nil
	}
	initial, err := versioning.Parse(cfg.Project.InitialVersion)
	if err != nil {
		return "", "", nil, fmt.Errorf("parse initial version: %w", err)
	}
	return "unreleased", "unreleased", &initial, nil
}

func recommendedPreviewVersion(records []releases.ReleaseRecord, product string, target versioning.Version, label string) versioning.Version {
	current := releases.PrereleaseVersionsForLine(records, product, target.String(), label)
	return versioning.NextPrerelease(target, label, current)
}

func selectReleaseVersion(cfg config.Config, records []releases.ReleaseRecord, product string, allFragments []fragments.Fragment, requestedVersion, requestedPre, requestedBump string) (versioning.Version, versioning.Bump, bool, error) {
	if strings.TrimSpace(requestedVersion) != "" {
		version, err := versioning.Parse(requestedVersion)
		if err != nil {
			return versioning.Version{}, versioning.BumpNone, false, err
		}
		return version, versioning.BumpNone, false, nil
	}

	pending, err := releases.UnreleasedFinalFragments(allFragments, records, product)
	if err != nil {
		return versioning.Version{}, versioning.BumpNone, false, err
	}
	recommendation := recommendationPolicy(cfg, pending)
	chosenBump := recommendation.SuggestedBump
	recommendedChoice := true
	if strings.TrimSpace(requestedBump) != "" {
		chosenBump, err = versioning.NormalizeBump(requestedBump)
		if err != nil {
			return versioning.Version{}, versioning.BumpNone, false, err
		}
		recommendedChoice = false
	}
	target, err := nextStableForBump(cfg, records, product, chosenBump)
	if err != nil {
		return versioning.Version{}, versioning.BumpNone, false, err
	}

	if strings.TrimSpace(requestedPre) == "" {
		return target, chosenBump, recommendedChoice, nil
	}

	return recommendedPreviewVersion(records, product, target, requestedPre), chosenBump, recommendedChoice, nil
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

func nextStableForBump(cfg config.Config, records []releases.ReleaseRecord, product string, bump versioning.Bump) (versioning.Version, error) {
	initial, err := versioning.Parse(cfg.Project.InitialVersion)
	if err != nil {
		return versioning.Version{}, fmt.Errorf("parse initial version: %w", err)
	}
	return versioning.NextStable(latestStableVersion(records, product), initial, bump), nil
}

func latestStableVersion(records []releases.ReleaseRecord, product string) *versioning.Version {
	if record := releases.LatestFinalHeadForProduct(records, product); record != nil {
		value := versioning.MustParse(record.Version)
		return &value
	}
	return nil
}

func publicAPIStability(raw string) semverpolicy.Stability {
	if strings.TrimSpace(raw) == "stable" {
		return semverpolicy.StabilityStable
	}
	return semverpolicy.StabilityUnstable
}

func fragmentIDs(items []fragments.Fragment) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	return ids
}

func cloneRecords(records []releases.ReleaseRecord) []releases.ReleaseRecord {
	return slices.Clone(records)
}

func releasePlanTime(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	return now.UTC().Truncate(time.Second)
}

func checkContext(ctx context.Context) error {
	if ctx == nil {
		return nil
	}
	return ctx.Err()
}
