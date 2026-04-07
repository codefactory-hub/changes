package app

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/example/changes/internal/config"
	"github.com/example/changes/internal/fragments"
	"github.com/example/changes/internal/releases"
	"github.com/example/changes/internal/versioning"
)

type historyImportPromptData struct {
	Product             string
	CurrentVersionLabel string
	ChangelogPath       string
	ConfigPath          string
	FragmentsDir        string
	ReleasesDir         string
	TemplatesDir        string
	PromptPath          string
	AdoptionRecordPath  string
	AdoptionFragment    *fragments.Fragment
	AdoptionRecord      *releases.ReleaseRecord
}

type initializeDeps struct {
	createAdoptionBootstrap  func(string, config.Config, versioning.Version, []releases.ReleaseRecord, time.Time, io.Reader) (fragments.Fragment, releases.ReleaseRecord, string, error)
	writeHistoryImportPrompt func(*initTxn, string, config.Config, historyImportPromptData) (string, error)
	stageHook                func(string) error
}

func Initialize(ctx context.Context, req InitializeRequest) (InitializeResult, error) {
	deps := initializeDeps{
		createAdoptionBootstrap:  createAdoptionBootstrap,
		writeHistoryImportPrompt: writeHistoryImportPrompt,
	}
	return initializeWithDeps(ctx, req, deps)
}

func InitializeGlobal(ctx context.Context, req InitializeGlobalRequest) (result InitializeGlobalResult, err error) {
	if err := checkContext(ctx); err != nil {
		return InitializeGlobalResult{}, err
	}

	selection, warnings, err := selectGlobalInitializeLayout(req)
	if err != nil {
		return InitializeGlobalResult{}, err
	}

	tx := newInitTxn()
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
			}
		}
	}()

	if err := tx.MkdirAll(selection.Config, 0o755); err != nil {
		return InitializeGlobalResult{}, fmt.Errorf("create global config directory: %w", err)
	}
	for _, dir := range []string{selection.Data, selection.State} {
		if err := tx.MkdirAll(dir, 0o755); err != nil {
			return InitializeGlobalResult{}, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	if err := checkContext(ctx); err != nil {
		return InitializeGlobalResult{}, err
	}

	layoutManifestPath := filepath.Join(selection.Config, "layout.toml")
	if _, statErr := os.Stat(layoutManifestPath); os.IsNotExist(statErr) {
		raw, encodeErr := encodeTOML(globalLayoutManifest(selection))
		if encodeErr != nil {
			return InitializeGlobalResult{}, encodeErr
		}
		if err := tx.WriteFileExclusive(layoutManifestPath, raw, 0o644); err != nil {
			return InitializeGlobalResult{}, fmt.Errorf("write global layout manifest: %w", err)
		}
	} else if statErr != nil {
		return InitializeGlobalResult{}, fmt.Errorf("stat global layout manifest: %w", statErr)
	}

	return InitializeGlobalResult{
		SelectedLayout:    selection.Style,
		ConfigPath:        selection.Config,
		DataPath:          selection.Data,
		StatePath:         selection.State,
		AuthorityWarnings: append([]config.AuthorityWarning(nil), warnings...),
	}, nil
}

func initializeWithDeps(ctx context.Context, req InitializeRequest, deps initializeDeps) (result InitializeResult, err error) {
	if err := checkContext(ctx); err != nil {
		return InitializeResult{}, err
	}
	selection, selectionFromDefaults, authorityWarnings, err := selectInitializeLayout(req)
	if err != nil {
		return InitializeResult{}, err
	}
	cfg, err := loadExistingOrDefaultConfig(req.RepoRoot)
	if err != nil {
		return InitializeResult{}, err
	}
	if selectionFromDefaults {
		cfg.Paths.DataDir = repoRelativeConfigPath(req.RepoRoot, selection.Data)
		cfg.Paths.StateDir = repoRelativeConfigPath(req.RepoRoot, selection.State)
		cfg.Paths.TemplatesDir = repoRelativeConfigPath(req.RepoRoot, filepath.Join(selection.Data, "templates"))
	}
	if strings.TrimSpace(cfg.Project.Name) == "" {
		cfg.Project.Name = filepath.Base(req.RepoRoot)
	}
	currentVersionLabel, currentVersion, err := resolveCurrentVersion(req.CurrentVersion)
	if err != nil {
		return InitializeResult{}, err
	}
	if err := ensureBootstrapArtifactsAbsent(req.RepoRoot, cfg); err != nil {
		return InitializeResult{}, err
	}
	if err := checkContext(ctx); err != nil {
		return InitializeResult{}, err
	}

	tx := newInitTxn()
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
			}
		}
	}()

	configPath := filepath.Join(selection.Config, "config.toml")
	layoutManifestPath := filepath.Join(selection.Config, "layout.toml")
	templatesDir := filepath.Join(selection.Data, "templates")
	promptsDir := filepath.Join(selection.Data, "prompts")
	fragmentsDir := filepath.Join(selection.Data, "fragments")
	releasesDir := filepath.Join(selection.Data, "releases")

	if err := tx.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return InitializeResult{}, err
	}
	for _, dir := range []string{
		fragmentsDir,
		releasesDir,
		promptsDir,
		selection.State,
	} {
		if err := tx.MkdirAll(dir, 0o755); err != nil {
			return InitializeResult{}, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}
	if err := checkContext(ctx); err != nil {
		return InitializeResult{}, err
	}

	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		raw, encodeErr := encodeTOML(cfg)
		if encodeErr != nil {
			return InitializeResult{}, encodeErr
		}
		if err := tx.WriteFileExclusive(configPath, raw, 0o644); err != nil {
			return InitializeResult{}, fmt.Errorf("write config: %w", err)
		}
	} else if statErr != nil {
		return InitializeResult{}, fmt.Errorf("stat config: %w", statErr)
	}
	if _, statErr := os.Stat(layoutManifestPath); os.IsNotExist(statErr) {
		raw, encodeErr := config.WriteRepoLayoutManifest(selection, req.RepoRoot)
		if encodeErr != nil {
			return InitializeResult{}, encodeErr
		}
		if err := tx.WriteFileExclusive(layoutManifestPath, raw, 0o644); err != nil {
			return InitializeResult{}, fmt.Errorf("write layout manifest: %w", err)
		}
	} else if statErr != nil {
		return InitializeResult{}, fmt.Errorf("stat layout manifest: %w", statErr)
	}

	changelogPath := config.ChangelogPath(req.RepoRoot, cfg)
	if _, err := tx.WriteFileIfMissing(changelogPath, []byte("# Changelog\n"), 0o644); err != nil {
		return InitializeResult{}, fmt.Errorf("write starter changelog: %w", err)
	}

	gitignoreUpdated, err := ensureGitignoreWithTx(tx, req.RepoRoot, selection.GitignoreEntry)
	if err != nil {
		return InitializeResult{}, err
	}
	if deps.stageHook != nil {
		if err := deps.stageHook("after_gitignore"); err != nil {
			return InitializeResult{}, err
		}
	}

	records, err := releases.List(req.RepoRoot, cfg)
	if err != nil {
		return InitializeResult{}, err
	}
	if err := checkContext(ctx); err != nil {
		return InitializeResult{}, err
	}

	result = InitializeResult{
		RepoRoot:          req.RepoRoot,
		SelectedLayout:    selection.Style,
		ConfigPath:        selection.Config,
		DataPath:          selection.Data,
		StatePath:         selection.State,
		GitignoreUpdated:  gitignoreUpdated,
		AuthorityWarnings: append([]config.AuthorityWarning(nil), authorityWarnings...),
	}
	if currentVersion != nil && releases.LatestFinalHeadForProduct(records, cfg.Project.Name) == nil {
		fragment, record, recordPath, err := deps.createAdoptionBootstrap(req.RepoRoot, cfg, *currentVersion, records, req.Now, req.Random)
		if err != nil {
			return InitializeResult{}, err
		}
		tx.RecordCreatedFile(fragment.Path)
		tx.RecordCreatedFile(recordPath)
		result.AdoptionFragment = &fragment
		result.AdoptionRecord = &record
		if err := checkContext(ctx); err != nil {
			return InitializeResult{}, err
		}

		if deps.stageHook != nil {
			if err := deps.stageHook("after_bootstrap"); err != nil {
				return InitializeResult{}, err
			}
		}

		promptPath, err := deps.writeHistoryImportPrompt(tx, req.RepoRoot, cfg, historyImportPromptData{
			Product:             cfg.Project.Name,
			CurrentVersionLabel: currentVersionLabel,
			ChangelogPath:       config.ChangelogPath(req.RepoRoot, cfg),
			ConfigPath:          configPath,
			FragmentsDir:        fragmentsDir,
			ReleasesDir:         releasesDir,
			TemplatesDir:        templatesDir,
			PromptPath:          filepath.Join(promptsDir, "release-history-import-llm-prompt.md"),
			AdoptionRecordPath:  recordPath,
			AdoptionFragment:    result.AdoptionFragment,
			AdoptionRecord:      result.AdoptionRecord,
		})
		if err != nil {
			return InitializeResult{}, err
		}
		result.PromptPath = promptPath
		if err := checkContext(ctx); err != nil {
			return InitializeResult{}, err
		}

		if deps.stageHook != nil {
			if err := deps.stageHook("after_prompt"); err != nil {
				return InitializeResult{}, err
			}
		}
	}

	return result, nil
}

func createAdoptionBootstrap(repoRoot string, cfg config.Config, currentVersion versioning.Version, existing []releases.ReleaseRecord, now time.Time, random io.Reader) (fragments.Fragment, releases.ReleaseRecord, string, error) {
	body := fmt.Sprintf(
		"This repository adopted `changes` at version %s.\n\nThis entry establishes the release-history boundary for `changes`. Historical releases before %s may be reconstructed or refined later.",
		currentVersion.String(),
		currentVersion.String(),
	)
	item, err := fragments.Create(repoRoot, cfg, now.UTC().Truncate(time.Second), random, fragments.NewInput{
		NameStem:  "changes-adoption",
		Type:      "changed",
		Bootstrap: true,
		Body:      body,
	})
	if err != nil {
		return fragments.Fragment{}, releases.ReleaseRecord{}, "", err
	}

	record := releases.ReleaseRecord{
		Product:          cfg.Project.Name,
		Version:          currentVersion.String(),
		Bootstrap:        true,
		CreatedAt:        now.UTC().Truncate(time.Second),
		AddedFragmentIDs: []string{item.ID},
		Summary:          fmt.Sprintf("Adopted `changes` metadata at version %s.", currentVersion.String()),
	}
	if err := releases.ValidateSet(append(slices.Clone(existing), record)); err != nil {
		_ = os.Remove(item.Path)
		return fragments.Fragment{}, releases.ReleaseRecord{}, "", err
	}
	path, err := releases.Write(repoRoot, cfg, record)
	if err != nil {
		_ = os.Remove(item.Path)
		return fragments.Fragment{}, releases.ReleaseRecord{}, "", err
	}
	return item, record, path, nil
}

func writeHistoryImportPrompt(tx *initTxn, repoRoot string, cfg config.Config, data historyImportPromptData) (string, error) {
	path := config.HistoryImportPromptPath(repoRoot, cfg)
	if err := tx.WriteFileExclusive(path, []byte(renderHistoryImportPrompt(repoRoot, data)), 0o644); err != nil {
		return "", fmt.Errorf("write history-import prompt: %w", err)
	}
	return path, nil
}

func renderHistoryImportPrompt(repoRoot string, data historyImportPromptData) string {
	var builder strings.Builder
	builder.WriteString("# Release History Import Prompt for `changes`\n\n")
	builder.WriteString("You are helping migrate historical release history into the `changes` data model for this repository.\n\n")
	builder.WriteString("## Repo-specific context\n")
	builder.WriteString(fmt.Sprintf("- Product name: `%s`\n", data.Product))
	builder.WriteString(fmt.Sprintf("- Current version supplied during `changes init`: `%s`\n", data.CurrentVersionLabel))
	builder.WriteString(fmt.Sprintf("- Config file: `%s`\n", repoRelativePath(repoRoot, data.ConfigPath)))
	builder.WriteString(fmt.Sprintf("- Fragments directory: `%s`\n", repoRelativePath(repoRoot, data.FragmentsDir)))
	builder.WriteString(fmt.Sprintf("- Releases directory: `%s`\n", repoRelativePath(repoRoot, data.ReleasesDir)))
	builder.WriteString(fmt.Sprintf("- Templates directory: `%s`\n", repoRelativePath(repoRoot, data.TemplatesDir)))
	builder.WriteString(fmt.Sprintf("- Changelog path: `%s`\n", repoRelativePath(repoRoot, data.ChangelogPath)))
	builder.WriteString(fmt.Sprintf("- Prompt file: `%s`\n", repoRelativePath(repoRoot, data.PromptPath)))
	if data.AdoptionRecord != nil && data.AdoptionFragment != nil {
		builder.WriteString(fmt.Sprintf("- Standard adoption release exists: yes (`%s`)\n", data.AdoptionRecord.Version))
		builder.WriteString(fmt.Sprintf("- Adoption release record path: `%s`\n", repoRelativePath(repoRoot, data.AdoptionRecordPath)))
		builder.WriteString(fmt.Sprintf("- Adoption fragment path: `%s`\n", repoRelativePath(repoRoot, data.AdoptionFragment.Path)))
	} else {
		builder.WriteString("- Standard adoption release exists: no\n")
	}

	builder.WriteString("\n## `changes` model\n")
	builder.WriteString("- Fragments are durable source records under `.local/share/changes/fragments/`.\n")
	builder.WriteString("- Release records are canonical per-release records under `.local/share/changes/releases/`.\n")
	builder.WriteString("- `CHANGELOG.md` and release-note bodies are rendered views generated from fragments plus release records.\n")
	builder.WriteString("- Historical reconstruction should preserve valid fragment front matter and release lineage.\n")

	builder.WriteString("\n## Instructions\n")
	builder.WriteString("- Inspect the repository's existing evidence as needed: changelog content, Git tags, Git history, prior release notes, package metadata, or other repo-local release markers.\n")
	if data.AdoptionRecord != nil && data.AdoptionFragment != nil {
		builder.WriteString("- A standard adoption release and fragment already exist. Replace or refine that bootstrap history intentionally instead of duplicating it.\n")
	}
	builder.WriteString("- Produce `changes` release records and fragments that match this repository's current file formats and layout.\n")
	builder.WriteString("- Do not invent unsupported semantics or rewrite history casually.\n")
	builder.WriteString("- Preserve valid `changes` file formats and release lineage.\n")
	builder.WriteString("- Prefer explicit release records and fragments over vague summaries.\n")
	builder.WriteString("- If evidence is incomplete, be clear about uncertainty instead of fabricating precision.\n")
	builder.WriteString("- Do not change templates or config unless they are directly relevant to representing release history.\n")
	return builder.String()
}

func resolveCurrentVersion(raw string) (string, *versioning.Version, error) {
	value := strings.TrimSpace(raw)
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "", "unreleased", "0.0.0":
		return "unreleased", nil, nil
	}

	parsed, err := versioning.Parse(value)
	if err != nil {
		return "", nil, fmt.Errorf("init: --current-version must be a stable semver or unreleased: %w", err)
	}
	if parsed.IsPrerelease() || parsed.BuildMetadata != "" {
		return "", nil, fmt.Errorf("init: --current-version must be a stable semver or unreleased")
	}
	return parsed.String(), &parsed, nil
}

func ensureBootstrapArtifactsAbsent(repoRoot string, cfg config.Config) error {
	paths := make([]string, 0, 3)
	promptPath := config.HistoryImportPromptPath(repoRoot, cfg)
	if _, err := os.Stat(promptPath); err == nil {
		paths = append(paths, repoRelativePath(repoRoot, promptPath))
	} else if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("check bootstrap prompt artifact: %w", err)
	}

	records, err := releases.List(repoRoot, cfg)
	if err != nil {
		return err
	}
	for _, record := range records {
		if record.Bootstrap {
			paths = append(paths, repoRelativePath(repoRoot, releases.RecordPath(repoRoot, cfg, record.Product, record.Version)))
		}
	}

	allFragments, err := fragments.List(repoRoot, cfg)
	if err != nil {
		return err
	}
	for _, item := range allFragments {
		if item.Bootstrap {
			paths = append(paths, repoRelativePath(repoRoot, item.Path))
		}
	}

	if len(paths) == 0 {
		return nil
	}
	slices.Sort(paths)
	return fmt.Errorf("init: bootstrap adoption artifacts already exist; review or remove them intentionally before re-running init: %s", strings.Join(paths, ", "))
}

func ensureGitignoreWithTx(tx *initTxn, repoRoot, entry string) (bool, error) {
	path := filepath.Join(repoRoot, ".gitignore")

	raw, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return false, fmt.Errorf("read .gitignore: %w", err)
	}

	lines := strings.Split(strings.TrimRight(string(raw), "\n"), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == entry {
			return false, nil
		}
	}

	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}
	lines = append(lines, entry)
	body := strings.Join(lines, "\n") + "\n"
	if err := tx.WriteFile(path, []byte(body), 0o644); err != nil {
		return false, fmt.Errorf("write .gitignore: %w", err)
	}
	return true, nil
}

type repoInitDefaultsDoc struct {
	Repo struct {
		Init struct {
			Style string `toml:"style"`
			Home  string `toml:"home"`
		} `toml:"init"`
	} `toml:"repo"`
}

type layoutManifestDoc struct {
	SchemaVersion int    `toml:"schema_version"`
	Scope         string `toml:"scope"`
	Style         string `toml:"style"`
	Layout        struct {
		Root   string `toml:"root"`
		Config string `toml:"config"`
		Data   string `toml:"data"`
		State  string `toml:"state"`
	} `toml:"layout"`
}

func selectInitializeLayout(req InitializeRequest) (config.RepoInitSelection, bool, []config.AuthorityWarning, error) {
	resolution, err := config.ResolveRepo(config.ResolveOptions{RepoRoot: req.RepoRoot})
	if err != nil {
		return config.RepoInitSelection{}, false, nil, fmt.Errorf("init: resolve repo layout: %w", err)
	}

	check, err := config.CheckScopeAuthority(resolution)
	if err == nil {
		return selectionFromCandidate(*check.Authoritative), false, append([]config.AuthorityWarning(nil), check.Warnings...), nil
	}

	var authorityErr *config.AuthorityError
	if !errors.As(err, &authorityErr) {
		return config.RepoInitSelection{}, false, nil, err
	}
	if authorityErr.Status != config.StatusUninitialized {
		return config.RepoInitSelection{}, false, nil, err
	}

	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		homeDir = ""
	}
	globalStyle, globalHome, globalWarnings, err := loadGlobalRepoInitDefaults(homeDir)
	if err != nil {
		return config.RepoInitSelection{}, false, nil, err
	}
	selection, err := config.SelectRepoInitLayout(config.RepoInitSelectionOptions{
		RepoRoot:        req.RepoRoot,
		RequestedStyle:  req.RequestedLayout,
		RequestedHome:   req.RequestedHome,
		GlobalInitStyle: globalStyle,
		GlobalInitHome:  globalHome,
		ChangesHome:     os.Getenv("CHANGES_HOME"),
		XDGConfigHome:   os.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:     os.Getenv("XDG_DATA_HOME"),
		XDGStateHome:    os.Getenv("XDG_STATE_HOME"),
	})
	if err != nil {
		return config.RepoInitSelection{}, false, nil, err
	}
	return selection, true, globalWarnings, nil
}

func selectionFromCandidate(candidate config.Candidate) config.RepoInitSelection {
	return config.RepoInitSelection{
		Style:          candidate.Style,
		Root:           candidate.Paths.Root,
		Config:         candidate.Paths.Config,
		Data:           candidate.Paths.Data,
		State:          candidate.Paths.State,
		GitignoreEntry: candidateGitignoreEntry(candidate),
	}
}

func candidateGitignoreEntry(candidate config.Candidate) string {
	switch candidate.Style {
	case config.StyleHome:
		return "/.changes/state/"
	default:
		rel, err := filepath.Rel(candidate.Paths.Root, candidate.Paths.State)
		if err != nil {
			return "/.local/state/changes/"
		}
		return "/" + strings.Trim(filepath.ToSlash(rel), "/") + "/"
	}
}

func selectGlobalInitializeLayout(req InitializeGlobalRequest) (config.GlobalInitSelection, []config.AuthorityWarning, error) {
	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		homeDir = ""
	}

	resolution, err := config.ResolveGlobal(config.ResolveOptions{
		HomeDir:       homeDir,
		ChangesHome:   os.Getenv("CHANGES_HOME"),
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:   os.Getenv("XDG_DATA_HOME"),
		XDGStateHome:  os.Getenv("XDG_STATE_HOME"),
	})
	if err != nil {
		return config.GlobalInitSelection{}, nil, fmt.Errorf("init global: resolve global layout: %w", err)
	}

	check, err := config.CheckScopeAuthority(resolution)
	if err == nil {
		return selectionFromGlobalCandidate(*check.Authoritative), append([]config.AuthorityWarning(nil), check.Warnings...), nil
	}

	var authorityErr *config.AuthorityError
	if !errors.As(err, &authorityErr) {
		return config.GlobalInitSelection{}, nil, err
	}
	if authorityErr.Status != config.StatusUninitialized {
		return config.GlobalInitSelection{}, nil, err
	}

	selection, err := config.SelectGlobalInitLayout(config.GlobalInitSelectionOptions{
		HomeDir:        homeDir,
		RequestedStyle: req.RequestedLayout,
		RequestedHome:  req.RequestedHome,
		ChangesHome:    os.Getenv("CHANGES_HOME"),
		XDGConfigHome:  os.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:    os.Getenv("XDG_DATA_HOME"),
		XDGStateHome:   os.Getenv("XDG_STATE_HOME"),
	})
	if err != nil {
		return config.GlobalInitSelection{}, nil, err
	}
	return selection, nil, nil
}

func selectionFromGlobalCandidate(candidate config.Candidate) config.GlobalInitSelection {
	return config.GlobalInitSelection{
		Style:  candidate.Style,
		Root:   candidate.Paths.Root,
		Config: candidate.Paths.Config,
		Data:   candidate.Paths.Data,
		State:  candidate.Paths.State,
	}
}

func loadGlobalRepoInitDefaults(homeDir string) (string, string, []config.AuthorityWarning, error) {
	opts := config.ResolveOptions{
		HomeDir:       homeDir,
		ChangesHome:   os.Getenv("CHANGES_HOME"),
		XDGConfigHome: os.Getenv("XDG_CONFIG_HOME"),
		XDGDataHome:   os.Getenv("XDG_DATA_HOME"),
		XDGStateHome:  os.Getenv("XDG_STATE_HOME"),
	}
	resolution, err := config.ResolveGlobal(opts)
	if err != nil {
		return "", "", nil, fmt.Errorf("init: resolve global config: %w", err)
	}

	check, err := config.CheckScopeAuthority(resolution)
	if err != nil {
		var authorityErr *config.AuthorityError
		if errors.As(err, &authorityErr) && authorityErr.Status == config.StatusUninitialized {
			return "", "", nil, nil
		}
		return "", "", nil, err
	}

	path := filepath.Join(check.Authoritative.Paths.Config, "config.toml")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", "", append([]config.AuthorityWarning(nil), check.Warnings...), nil
		}
		return "", "", nil, fmt.Errorf("init: read global config: %w", err)
	}

	var doc repoInitDefaultsDoc
	if _, err := toml.Decode(string(raw), &doc); err != nil {
		return "", "", nil, fmt.Errorf("init: decode global config repo.init defaults: %w", err)
	}
	return doc.Repo.Init.Style, doc.Repo.Init.Home, append([]config.AuthorityWarning(nil), check.Warnings...), nil
}

func globalLayoutManifest(selection config.GlobalInitSelection) layoutManifestDoc {
	doc := layoutManifestDoc{
		SchemaVersion: 1,
		Scope:         string(config.ScopeGlobal),
		Style:         string(selection.Style),
	}

	switch selection.Style {
	case config.StyleHome:
		doc.Layout.Root = globalHomeSymbolicRoot(selection.Root)
		doc.Layout.Config = "$layout.root/config"
		doc.Layout.Data = "$layout.root/data"
		doc.Layout.State = "$layout.root/state"
	default:
		doc.Layout.Root = "$HOME"
		doc.Layout.Config = globalXDGSymbolicPath(selection.Config, "XDG_CONFIG_HOME", os.Getenv("XDG_CONFIG_HOME"), ".config", "changes")
		doc.Layout.Data = globalXDGSymbolicPath(selection.Data, "XDG_DATA_HOME", os.Getenv("XDG_DATA_HOME"), ".local", "share", "changes")
		doc.Layout.State = globalXDGSymbolicPath(selection.State, "XDG_STATE_HOME", os.Getenv("XDG_STATE_HOME"), ".local", "state", "changes")
	}

	return doc
}

func globalHomeSymbolicRoot(root string) string {
	changesHome := strings.TrimSpace(os.Getenv("CHANGES_HOME"))
	if changesHome != "" && filepath.Clean(changesHome) == filepath.Clean(root) {
		return "$CHANGES_HOME"
	}

	homeDir, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(homeDir) != "" {
		if rel, relErr := filepath.Rel(homeDir, root); relErr == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if rel == "." {
				return "$HOME"
			}
			return "$HOME/" + filepath.ToSlash(rel)
		}
	}

	return filepath.ToSlash(root)
}

func globalXDGSymbolicPath(path string, envName string, envValue string, fallback ...string) string {
	clean := filepath.Clean(path)
	envValue = strings.TrimSpace(envValue)
	if envValue != "" {
		prefix := filepath.Clean(envValue)
		if rel, err := filepath.Rel(prefix, clean); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			if rel == "." {
				return "$" + envName
			}
			return "$" + envName + "/" + filepath.ToSlash(rel)
		}
	}

	homeDir, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(homeDir) != "" {
		defaultPath := filepath.Join(append([]string{homeDir}, fallback...)...)
		if filepath.Clean(defaultPath) == clean {
			return "$HOME/" + filepath.ToSlash(filepath.Join(fallback...))
		}
	}

	return filepath.ToSlash(clean)
}

func encodeTOML(v any) ([]byte, error) {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(v); err != nil {
		return nil, fmt.Errorf("encode toml: %w", err)
	}
	return buf.Bytes(), nil
}

func repoRelativePath(repoRoot, path string) string {
	if rel, err := filepath.Rel(repoRoot, path); err == nil {
		return rel
	}
	return path
}

func repoRelativeConfigPath(repoRoot, path string) string {
	return filepath.ToSlash(repoRelativePath(repoRoot, path))
}

type initTxn struct {
	createdDirs   []string
	createdFiles  []string
	modifiedFiles map[string]fileBackup
}

type fileBackup struct {
	contents []byte
	mode     os.FileMode
}

func newInitTxn() *initTxn {
	return &initTxn{modifiedFiles: map[string]fileBackup{}}
}

func (tx *initTxn) MkdirAll(path string, perm os.FileMode) error {
	clean := filepath.Clean(path)
	missing := make([]string, 0)
	for current := clean; current != "." && current != string(filepath.Separator); current = filepath.Dir(current) {
		if _, err := os.Stat(current); err == nil {
			break
		} else if !os.IsNotExist(err) {
			return err
		}
		missing = append(missing, current)
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
	}
	if err := os.MkdirAll(clean, perm); err != nil {
		return err
	}
	slices.Reverse(missing)
	tx.createdDirs = append(tx.createdDirs, missing...)
	return nil
}

func (tx *initTxn) WriteFileIfMissing(path string, body []byte, perm os.FileMode) (bool, error) {
	if err := tx.WriteFileExclusive(path, body, perm); err == nil {
		return true, nil
	} else if os.IsExist(err) {
		return false, nil
	} else {
		return false, err
	}
}

func (tx *initTxn) WriteFile(path string, body []byte, perm os.FileMode) error {
	if err := tx.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		if _, ok := tx.modifiedFiles[path]; !ok {
			raw, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			info, statErr := os.Stat(path)
			if statErr != nil {
				return statErr
			}
			tx.modifiedFiles[path] = fileBackup{contents: raw, mode: info.Mode()}
		}
	} else if os.IsNotExist(err) {
		tx.RecordCreatedFile(path)
	} else {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.Remove(tmpPath)
		}
	}()
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(body); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	cleanup = false
	return nil
}

func (tx *initTxn) WriteFileExclusive(path string, body []byte, perm os.FileMode) error {
	if err := tx.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	tx.RecordCreatedFile(path)
	if _, err := file.Write(body); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func (tx *initTxn) RecordCreatedFile(path string) {
	for _, existing := range tx.createdFiles {
		if existing == path {
			return
		}
	}
	tx.createdFiles = append(tx.createdFiles, path)
}

func (tx *initTxn) Rollback() error {
	var rollbackErrs []string
	for path, backup := range tx.modifiedFiles {
		if err := os.WriteFile(path, backup.contents, backup.mode); err != nil {
			rollbackErrs = append(rollbackErrs, fmt.Sprintf("restore %s: %v", path, err))
		}
	}
	for i := len(tx.createdFiles) - 1; i >= 0; i-- {
		if err := os.Remove(tx.createdFiles[i]); err != nil && !os.IsNotExist(err) {
			rollbackErrs = append(rollbackErrs, fmt.Sprintf("remove %s: %v", tx.createdFiles[i], err))
		}
	}
	for i := len(tx.createdDirs) - 1; i >= 0; i-- {
		if err := os.Remove(tx.createdDirs[i]); err != nil && !os.IsNotExist(err) && !strings.Contains(err.Error(), "directory not empty") {
			rollbackErrs = append(rollbackErrs, fmt.Sprintf("remove dir %s: %v", tx.createdDirs[i], err))
		}
	}
	if len(rollbackErrs) > 0 {
		return fmt.Errorf("rollback init: %s", strings.Join(rollbackErrs, "; "))
	}
	return nil
}
