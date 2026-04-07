package app

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/example/changes/internal/config"
)

func Doctor(ctx context.Context, req DoctorRequest) (DoctorResult, error) {
	if err := checkContext(ctx); err != nil {
		return DoctorResult{}, err
	}

	scope := req.Scope
	if scope == "" {
		scope = DoctorScopeAll
	}

	opts, err := doctorResolveOptions(req.RepoRoot)
	if err != nil {
		return DoctorResult{}, err
	}

	result := DoctorResult{
		RequestedScope: string(scope),
		GeneratedAt:    doctorGeneratedAt(req.Now),
		Summary: DoctorSummary{
			StatusCounts: map[string]int{},
		},
	}

	if req.Repair {
		if scope != DoctorScopeRepo {
			return DoctorResult{}, fmt.Errorf("doctor: repair is supported only with --scope repo")
		}
		if req.GenerateMigrationPrompt {
			return DoctorResult{}, fmt.Errorf("doctor: repair cannot be combined with migration prompt generation")
		}
		scopeResult, err := doctorRepairRepo(ctx, req.RepoRoot, opts)
		if err != nil {
			return DoctorResult{}, err
		}
		result.Repo = &scopeResult
		result.Summary.StatusCounts[scopeResult.Status]++
		return result, nil
	}

	switch scope {
	case DoctorScopeGlobal:
		scopeResult, err := doctorInspectScope(config.ScopeGlobal, opts)
		if err != nil {
			return DoctorResult{}, err
		}
		result.Global = &scopeResult
		result.Summary.StatusCounts[scopeResult.Status]++
	case DoctorScopeRepo:
		scopeResult, err := doctorInspectScope(config.ScopeRepo, opts)
		if err != nil {
			return DoctorResult{}, err
		}
		result.Repo = &scopeResult
		result.Summary.StatusCounts[scopeResult.Status]++
	case DoctorScopeAll:
		globalResult, err := doctorInspectScope(config.ScopeGlobal, opts)
		if err != nil {
			return DoctorResult{}, err
		}
		repoResult, err := doctorInspectScope(config.ScopeRepo, opts)
		if err != nil {
			return DoctorResult{}, err
		}
		result.Global = &globalResult
		result.Repo = &repoResult
		result.Summary.StatusCounts[globalResult.Status]++
		result.Summary.StatusCounts[repoResult.Status]++
	default:
		return DoctorResult{}, fmt.Errorf("doctor: unsupported scope %q", req.Scope)
	}

	if req.GenerateMigrationPrompt {
		scopeResult, err := doctorMigrationScopeResult(result, scope)
		if err != nil {
			return DoctorResult{}, err
		}
		destination, err := doctorDestinationLayout(scope, req, opts)
		if err != nil {
			return DoctorResult{}, err
		}
		prompt, err := buildDoctorMigrationPrompt(scope, *scopeResult, destination)
		if err != nil {
			return DoctorResult{}, err
		}
		result.MigrationPrompt = prompt
	}

	return result, nil
}

func doctorRepairRepo(ctx context.Context, repoRoot string, opts config.ResolveOptions) (result DoctorScopeResult, err error) {
	if strings.TrimSpace(repoRoot) == "" {
		return DoctorScopeResult{}, fmt.Errorf("doctor: repo root is required for repair")
	}
	if err := checkContext(ctx); err != nil {
		return DoctorScopeResult{}, err
	}

	resolution, err := config.ResolveRepo(opts)
	if err != nil {
		return DoctorScopeResult{}, err
	}

	candidate, err := doctorRepairCandidate(resolution)
	if err != nil {
		return DoctorScopeResult{}, err
	}
	selection := selectionFromCandidate(candidate)
	manifestPath := filepath.Join(selection.Config, "layout.toml")

	tx := newInitTxn()
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
			}
		}
	}()

	raw, err := config.WriteRepoLayoutManifest(selection, repoRoot)
	if err != nil {
		return DoctorScopeResult{}, err
	}
	if err := tx.WriteFileExclusive(manifestPath, raw, 0o644); err != nil {
		return DoctorScopeResult{}, fmt.Errorf("doctor: write layout manifest: %w", err)
	}
	gitignoreUpdated, err := ensureGitignoreWithTx(tx, repoRoot, selection.GitignoreEntry)
	if err != nil {
		return DoctorScopeResult{}, err
	}
	if err := checkContext(ctx); err != nil {
		return DoctorScopeResult{}, err
	}

	check, err := config.RequireRepoWriteAuthority(repoRoot)
	if err != nil {
		return DoctorScopeResult{}, fmt.Errorf("doctor: validate repaired repo authority: %w", err)
	}
	if check.Authoritative == nil || check.Authoritative.Style != candidate.Style || check.Authoritative.Paths.Root != candidate.Paths.Root {
		return DoctorScopeResult{}, fmt.Errorf("doctor: repaired repo did not become authoritative at %s", candidate.Paths.Root)
	}
	if _, _, err := config.LoadWithAuthority(repoRoot); err != nil {
		return DoctorScopeResult{}, fmt.Errorf("doctor: validate repaired repo operation: %w", err)
	}

	result, err = doctorInspectScope(config.ScopeRepo, opts)
	if err != nil {
		return DoctorScopeResult{}, err
	}
	result.Repair = &DoctorRepair{
		Changed:            true,
		ManifestPath:       manifestPath,
		GitignoreUpdated:   gitignoreUpdated,
		AuthoritativeStyle: string(check.Authoritative.Style),
		AuthoritativeRoot:  check.Authoritative.Paths.Root,
	}
	return result, nil
}

func doctorRepairCandidate(resolution config.ScopeResolution) (config.Candidate, error) {
	legacyCandidates := make([]config.Candidate, 0, len(resolution.Candidates))
	resolvedCount := 0
	for _, candidate := range resolution.Candidates {
		switch candidate.Status {
		case config.StatusResolved:
			resolvedCount++
		case config.StatusLegacyOnly:
			legacyCandidates = append(legacyCandidates, candidate)
		}
	}

	if resolvedCount > 0 {
		return config.Candidate{}, fmt.Errorf("doctor: repo repair is only for legacy repo-local layouts without an authoritative manifest")
	}
	if len(legacyCandidates) == 0 {
		return config.Candidate{}, fmt.Errorf("doctor: repo repair requires exactly one legacy repo-local candidate")
	}
	if len(legacyCandidates) > 1 {
		return config.Candidate{}, fmt.Errorf("doctor: repo repair is ambiguous because multiple legacy repo-local candidates could be stamped; use changes doctor --migration-prompt --scope repo --to xdg|home [--home PATH]")
	}

	return legacyCandidates[0], nil
}

func doctorResolveOptions(repoRoot string) (config.ResolveOptions, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return config.ResolveOptions{}, fmt.Errorf("doctor: determine home directory: %w", err)
	}

	return config.ResolveOptions{
		RepoRoot:      repoRoot,
		HomeDir:       homeDir,
		ChangesHome:   os.Getenv("CHANGES_HOME"),
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:   os.Getenv("XDG_DATA_HOME"),
		XDGStateHome:  os.Getenv("XDG_STATE_HOME"),
	}, nil
}

func doctorGeneratedAt(now time.Time) time.Time {
	if now.IsZero() {
		now = time.Now()
	}
	return now.UTC().Truncate(time.Second)
}

func doctorInspectScope(scope config.Scope, opts config.ResolveOptions) (DoctorScopeResult, error) {
	var (
		resolution config.ScopeResolution
		err        error
	)

	switch scope {
	case config.ScopeGlobal:
		resolution, err = config.ResolveGlobal(opts)
	case config.ScopeRepo:
		if strings.TrimSpace(opts.RepoRoot) == "" {
			return DoctorScopeResult{}, fmt.Errorf("doctor: repo root is required for repo scope")
		}
		resolution, err = config.ResolveRepo(opts)
	default:
		return DoctorScopeResult{}, fmt.Errorf("doctor: unsupported scope %q", scope)
	}
	if err != nil {
		return DoctorScopeResult{}, err
	}

	result := DoctorScopeResult{
		Scope:            string(scope),
		Status:           doctorScopeStatus(resolution.Status),
		PrecedenceInputs: doctorPrecedenceInputs(scope, opts),
		Candidates:       make([]DoctorCandidate, 0, len(resolution.Candidates)),
		Warnings:         make([]DoctorWarning, 0, len(resolution.Warnings)),
		RepairHint:       doctorRepairHint(scope, resolution.Status),
	}

	if resolution.Preferred != nil {
		result.PreferredStyle = string(resolution.Preferred.Style)
		result.PreferredRoot = resolution.Preferred.Paths.Root
	}
	if resolution.Authoritative != nil {
		result.SelectedStyle = string(resolution.Authoritative.Style)
		result.SelectedRoot = resolution.Authoritative.Paths.Root
		result.AuthoritativeStyle = string(resolution.Authoritative.Style)
		result.AuthoritativeRoot = resolution.Authoritative.Paths.Root
	}

	for _, warning := range resolution.Warnings {
		result.Warnings = append(result.Warnings, DoctorWarning{
			Scope:  string(warning.Scope),
			Style:  string(warning.Style),
			Status: string(warning.Status),
			Path:   warning.Path,
		})
	}

	for _, candidate := range resolution.Candidates {
		doctorCandidate := DoctorCandidate{
			Scope:    string(candidate.Scope),
			Style:    string(candidate.Style),
			Status:   string(candidate.Status),
			Paths:    candidate.Paths,
			Evidence: make([]DoctorEvidence, 0, len(candidate.Evidence)),
		}
		if resolution.Preferred != nil && candidate.Style == resolution.Preferred.Style && candidate.Paths.Root == resolution.Preferred.Paths.Root {
			doctorCandidate.IsPreferred = true
		}
		if resolution.Authoritative != nil && candidate.Style == resolution.Authoritative.Style && candidate.Paths.Root == resolution.Authoritative.Paths.Root {
			doctorCandidate.IsAuthoritative = true
		}
		if candidate.Manifest != nil {
			doctorCandidate.Manifest = &DoctorManifest{
				SchemaVersion: candidate.Manifest.SchemaVersion,
				Scope:         string(candidate.Manifest.Scope),
				Style:         string(candidate.Manifest.Style),
				Symbolic:      candidate.Manifest.Symbolic,
				Resolved:      candidate.Manifest.Resolved,
			}
		}
		for _, evidence := range candidate.Evidence {
			doctorCandidate.Evidence = append(doctorCandidate.Evidence, DoctorEvidence{
				Kind:   evidence.Kind,
				Name:   evidence.Name,
				Path:   evidence.Path,
				Exists: evidence.Exists,
				Detail: evidence.Detail,
			})
		}
		result.Candidates = append(result.Candidates, doctorCandidate)
	}

	return result, nil
}

func doctorScopeStatus(status config.ResolutionStatus) string {
	switch status {
	case config.StatusResolved:
		return DoctorStatusAuthoritative
	case config.StatusLegacyOnly:
		return DoctorStatusLegacyOnly
	case config.StatusInvalid:
		return DoctorStatusInvalid
	case config.StatusAmbiguous:
		return DoctorStatusAmbiguous
	default:
		return DoctorStatusUninitialized
	}
}

func doctorPrecedenceInputs(scope config.Scope, opts config.ResolveOptions) []string {
	inputs := make([]string, 0, 3)
	if scope == config.ScopeGlobal && strings.TrimSpace(opts.ChangesHome) != "" {
		inputs = append(inputs, "CHANGES_HOME")
	}
	if strings.TrimSpace(opts.XDGConfigHome) != "" || strings.TrimSpace(opts.XDGDataHome) != "" || strings.TrimSpace(opts.XDGStateHome) != "" {
		inputs = append(inputs, "XDG env vars")
	}
	inputs = append(inputs, "built-in default locations")
	return inputs
}

func doctorRepairHint(scope config.Scope, status config.ResolutionStatus) string {
	switch status {
	case config.StatusResolved:
		return ""
	case config.StatusAmbiguous:
		return fmt.Sprintf("Run changes doctor --migration-prompt --scope %s --to xdg|home [--home PATH].", scope)
	case config.StatusLegacyOnly, config.StatusInvalid:
		return fmt.Sprintf("Inspect %s with changes doctor --scope %s and migrate or repair one authoritative layout.", scope, scope)
	case config.StatusUninitialized:
		if scope == config.ScopeGlobal {
			return "Run changes init global [--layout xdg|home] [--home PATH]."
		}
		return "Run changes init [--layout xdg|home] [--home PATH]."
	default:
		return ""
	}
}

func doctorMigrationScopeResult(result DoctorResult, scope DoctorScope) (*DoctorScopeResult, error) {
	switch scope {
	case DoctorScopeGlobal:
		if result.Global == nil {
			return nil, fmt.Errorf("doctor: global inspection result is unavailable")
		}
		return result.Global, nil
	case DoctorScopeRepo:
		if result.Repo == nil {
			return nil, fmt.Errorf("doctor: repo inspection result is unavailable")
		}
		return result.Repo, nil
	default:
		return nil, fmt.Errorf("doctor: migration prompt requires --scope global or --scope repo")
	}
}

type doctorDestination struct {
	Scope         string
	Style         string
	RequestedHome string
	Paths         config.LayoutPaths
	SchemaVersion int
}

func doctorDestinationLayout(scope DoctorScope, req DoctorRequest, opts config.ResolveOptions) (doctorDestination, error) {
	switch req.DestinationStyle {
	case config.StyleXDG, config.StyleHome:
	default:
		return doctorDestination{}, fmt.Errorf("doctor: unsupported destination style %q", req.DestinationStyle)
	}

	if req.DestinationStyle == config.StyleXDG && strings.TrimSpace(req.DestinationHome) != "" {
		return doctorDestination{}, fmt.Errorf("doctor: --home is valid only when --to home")
	}

	if scope == DoctorScopeRepo {
		selection, err := config.SelectRepoInitLayout(config.RepoInitSelectionOptions{
			RepoRoot:       req.RepoRoot,
			RequestedStyle: string(req.DestinationStyle),
			RequestedHome:  req.DestinationHome,
		})
		if err != nil {
			return doctorDestination{}, fmt.Errorf("doctor: resolve repo destination: %w", err)
		}
		return doctorDestination{
			Scope:         string(scope),
			Style:         string(selection.Style),
			RequestedHome: req.DestinationHome,
			Paths: config.LayoutPaths{
				Root:   selection.Root,
				Config: selection.Config,
				Data:   selection.Data,
				State:  selection.State,
			},
			SchemaVersion: 1,
		}, nil
	}

	if scope != DoctorScopeGlobal {
		return doctorDestination{}, fmt.Errorf("doctor: migration prompt requires --scope global or --scope repo")
	}

	var paths config.LayoutPaths
	if req.DestinationStyle == config.StyleHome {
		root := strings.TrimSpace(req.DestinationHome)
		if root == "" {
			root = strings.TrimSpace(opts.ChangesHome)
		}
		if root == "" {
			root = filepath.Join(opts.HomeDir, ".changes")
		}
		root = filepath.Clean(root)
		paths = config.LayoutPaths{
			Root:   root,
			Config: filepath.Join(root, "config"),
			Data:   filepath.Join(root, "data"),
			State:  filepath.Join(root, "state"),
		}
	} else {
		configRoot := opts.XDGConfigHome
		if configRoot == "" {
			configRoot = filepath.Join(opts.HomeDir, ".config")
		}
		dataRoot := opts.XDGDataHome
		if dataRoot == "" {
			dataRoot = filepath.Join(opts.HomeDir, ".local", "share")
		}
		stateRoot := opts.XDGStateHome
		if stateRoot == "" {
			stateRoot = filepath.Join(opts.HomeDir, ".local", "state")
		}
		paths = config.LayoutPaths{
			Root:   filepath.Clean(opts.HomeDir),
			Config: filepath.Join(configRoot, "changes"),
			Data:   filepath.Join(dataRoot, "changes"),
			State:  filepath.Join(stateRoot, "changes"),
		}
	}

	return doctorDestination{
		Scope:         string(scope),
		Style:         string(req.DestinationStyle),
		RequestedHome: req.DestinationHome,
		Paths:         paths,
		SchemaVersion: 1,
	}, nil
}

func buildDoctorMigrationPrompt(scope DoctorScope, scopeResult DoctorScopeResult, destination doctorDestination) (string, error) {
	originCandidates := make([]DoctorCandidate, len(scopeResult.Candidates))
	copy(originCandidates, scopeResult.Candidates)
	slices.SortFunc(originCandidates, func(left, right DoctorCandidate) int {
		if left.IsAuthoritative != right.IsAuthoritative {
			if left.IsAuthoritative {
				return -1
			}
			return 1
		}
		if left.Style != right.Style {
			return strings.Compare(left.Style, right.Style)
		}
		return strings.Compare(left.Paths.Root, right.Paths.Root)
	})

	var builder strings.Builder
	builder.WriteString("## Requested Migration\n\n")
	builder.WriteString(fmt.Sprintf("- Requested scope: %s\n", scope))
	builder.WriteString(fmt.Sprintf("- Requested destination style: %s\n", destination.Style))
	if strings.TrimSpace(destination.RequestedHome) != "" {
		builder.WriteString(fmt.Sprintf("- Requested destination home path: %s\n", destination.RequestedHome))
	}
	builder.WriteString("\n## Origin Layout\n\n")
	builder.WriteString(fmt.Sprintf("- Status: %s\n", scopeResult.Status))
	if scopeResult.AuthoritativeStyle != "" {
		builder.WriteString(fmt.Sprintf("- Selected style: %s\n", scopeResult.AuthoritativeStyle))
		builder.WriteString(fmt.Sprintf("- Selected root: %s\n", scopeResult.AuthoritativeRoot))
	}
	if scopeResult.AuthoritativeStyle == "" {
		builder.WriteString("- No authoritative origin candidate exists.\n")
	}
	for _, candidate := range originCandidates {
		builder.WriteString(fmt.Sprintf("- Candidate %s: status=%s root=%s\n", candidate.Style, candidate.Status, candidate.Paths.Root))
		if candidate.Manifest != nil {
			builder.WriteString(fmt.Sprintf("  - Manifest schema_version: %d\n", candidate.Manifest.SchemaVersion))
			builder.WriteString(fmt.Sprintf("  - Manifest scope/style: %s/%s\n", candidate.Manifest.Scope, candidate.Manifest.Style))
		} else {
			builder.WriteString("  - Manifest schema_version: none\n")
		}
	}

	builder.WriteString("\n## Destination Layout\n\n")
	builder.WriteString(fmt.Sprintf("- Destination style: %s\n", destination.Style))
	builder.WriteString(fmt.Sprintf("- Destination root: %s\n", destination.Paths.Root))
	builder.WriteString(fmt.Sprintf("- Destination config path: %s\n", destination.Paths.Config))
	builder.WriteString(fmt.Sprintf("- Destination data path: %s\n", destination.Paths.Data))
	builder.WriteString(fmt.Sprintf("- Destination state path: %s\n", destination.Paths.State))
	builder.WriteString(fmt.Sprintf("- Destination manifest schema_version: %d\n", destination.SchemaVersion))

	builder.WriteString("\n## Artifact Inventory\n\n")
	if scopeResult.AuthoritativeRoot != "" {
		authoritative := doctorAuthoritativeCandidate(scopeResult.Candidates)
		if authoritative != nil {
			writeDoctorInventorySection(&builder, "config inventory", authoritative.Paths.Config)
			writeDoctorInventorySection(&builder, "data inventory", authoritative.Paths.Data)
			writeDoctorInventorySection(&builder, "state inventory", authoritative.Paths.State)
		}
	} else {
		builder.WriteString("No authoritative origin inventory is available because the inspected scope is not operational.\n")
		for _, candidate := range originCandidates {
			builder.WriteString(fmt.Sprintf("- Candidate %s root: %s\n", candidate.Style, candidate.Paths.Root))
		}
	}

	builder.WriteString("\n## Ambiguity and Conflict Notes\n\n")
	if scopeResult.Status == DoctorStatusAmbiguous {
		builder.WriteString("Competing candidates:\n")
		for _, candidate := range originCandidates {
			builder.WriteString(fmt.Sprintf("- %s at %s (%s)\n", candidate.Style, candidate.Paths.Root, candidate.Status))
		}
	} else {
		builder.WriteString("No competing authoritative candidates detected.\n")
	}
	for _, warning := range scopeResult.Warnings {
		builder.WriteString(fmt.Sprintf("- Warning: %s %s sibling at %s\n", warning.Status, warning.Style, warning.Path))
	}

	builder.WriteString("\n## Required Verification\n\n")
	builder.WriteString(fmt.Sprintf("- Confirm the final layout resolves through changes doctor --scope %s.\n", scope))
	builder.WriteString("- Confirm the resulting layout.toml is present, parseable, and matches the destination scope/style.\n")
	builder.WriteString("- Confirm only one authoritative layout remains after migration.\n")
	if scope == DoctorScopeRepo {
		ignoreRule := "/.local/state/changes/"
		if destination.Style == string(config.StyleHome) {
			ignoreRule = "/.changes/state/"
			if strings.TrimSpace(destination.RequestedHome) != "" {
				ignoreRule = "/" + strings.Trim(strings.TrimSpace(destination.RequestedHome), "/") + "/state/"
			}
		}
		builder.WriteString(fmt.Sprintf("- Confirm .gitignore keeps the authoritative repo state path ignored with %s.\n", ignoreRule))
	}

	builder.WriteString("\n## Safety Rules\n\n")
	builder.WriteString("Preserve exactly one authoritative destination.\n\n")
	builder.WriteString("Do not dual-write or keep two live authoritative layouts.\n\n")
	builder.WriteString("Do not convert this brief into destructive automation without explicit operator review.\n")

	return builder.String(), nil
}

func doctorAuthoritativeCandidate(candidates []DoctorCandidate) *DoctorCandidate {
	for i := range candidates {
		if candidates[i].IsAuthoritative {
			return &candidates[i]
		}
	}
	return nil
}

func writeDoctorInventorySection(builder *strings.Builder, title string, path string) {
	items, err := doctorInventory(path)
	builder.WriteString(fmt.Sprintf("### %s\n\n", title))
	builder.WriteString(fmt.Sprintf("- path: %s\n", path))
	if err != nil {
		builder.WriteString(fmt.Sprintf("- inventory error: %v\n\n", err))
		return
	}
	builder.WriteString(fmt.Sprintf("- item count: %d\n", len(items)))
	if len(items) == 0 {
		builder.WriteString("- files: (none)\n\n")
		return
	}
	builder.WriteString("- files:\n")
	for _, item := range items {
		builder.WriteString(fmt.Sprintf("  - %s\n", item))
	}
	builder.WriteString("\n")
}

func doctorInventory(root string) ([]string, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return []string{filepath.Base(root)}, nil
	}

	items := make([]string, 0)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		items = append(items, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(items)
	return items, nil
}
