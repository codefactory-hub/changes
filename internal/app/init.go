package app

import (
	"bytes"
	"context"
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
	"github.com/example/changes/internal/templates"
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
	ensureDefaultFiles       func(string, config.Config) (templates.FileSet, error)
	createAdoptionBootstrap  func(string, config.Config, versioning.Version, []releases.ReleaseRecord, time.Time, io.Reader) (fragments.Fragment, releases.ReleaseRecord, string, error)
	writeHistoryImportPrompt func(string, config.Config, historyImportPromptData) (string, error)
	stageHook                func(string) error
}

func Initialize(ctx context.Context, req InitializeRequest) (InitializeResult, error) {
	_ = ctx
	deps := initializeDeps{
		ensureDefaultFiles:       templates.EnsureDefaultFiles,
		createAdoptionBootstrap:  createAdoptionBootstrap,
		writeHistoryImportPrompt: writeHistoryImportPrompt,
	}
	return initializeWithDeps(req, deps)
}

func initializeWithDeps(req InitializeRequest, deps initializeDeps) (result InitializeResult, err error) {
	cfg, err := loadExistingOrDefaultConfig(req.RepoRoot)
	if err != nil {
		return InitializeResult{}, err
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

	tx := newInitTxn()
	defer func() {
		if err != nil {
			if rollbackErr := tx.Rollback(); rollbackErr != nil {
				err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
			}
		}
	}()

	if err := tx.MkdirAll(filepath.Dir(config.RepoConfigPath(req.RepoRoot)), 0o755); err != nil {
		return InitializeResult{}, err
	}
	for _, dir := range []string{
		config.FragmentsDir(req.RepoRoot, cfg),
		config.ReleasesDir(req.RepoRoot, cfg),
		config.PromptsDir(req.RepoRoot, cfg),
		config.TemplatesDir(req.RepoRoot, cfg),
		config.StateDir(req.RepoRoot, cfg),
	} {
		if err := tx.MkdirAll(dir, 0o755); err != nil {
			return InitializeResult{}, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	if _, statErr := os.Stat(config.RepoConfigPath(req.RepoRoot)); os.IsNotExist(statErr) {
		raw, encodeErr := encodeTOML(cfg)
		if encodeErr != nil {
			return InitializeResult{}, encodeErr
		}
		if err := tx.WriteFile(config.RepoConfigPath(req.RepoRoot), raw, 0o644); err != nil {
			return InitializeResult{}, fmt.Errorf("write config: %w", err)
		}
	} else if statErr != nil {
		return InitializeResult{}, fmt.Errorf("stat config: %w", statErr)
	}

	fileSet, err := deps.ensureDefaultFiles(req.RepoRoot, cfg)
	if err != nil {
		return InitializeResult{}, err
	}
	for _, path := range fileSet.CreatedPaths {
		tx.RecordCreatedFile(path)
	}

	changelogPath := config.ChangelogPath(req.RepoRoot, cfg)
	if _, err := tx.WriteFileIfMissing(changelogPath, []byte("# Changelog\n"), 0o644); err != nil {
		return InitializeResult{}, fmt.Errorf("write starter changelog: %w", err)
	}

	if err := ensureGitignoreWithTx(tx, req.RepoRoot); err != nil {
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

	result = InitializeResult{RepoRoot: req.RepoRoot}
	if currentVersion != nil && releases.LatestFinalHeadForProduct(records, cfg.Project.Name) == nil {
		fragment, record, recordPath, err := deps.createAdoptionBootstrap(req.RepoRoot, cfg, *currentVersion, records, req.Now, req.Random)
		if err != nil {
			return InitializeResult{}, err
		}
		tx.RecordCreatedFile(fragment.Path)
		tx.RecordCreatedFile(recordPath)
		result.AdoptionFragment = &fragment
		result.AdoptionRecord = &record

		if deps.stageHook != nil {
			if err := deps.stageHook("after_bootstrap"); err != nil {
				return InitializeResult{}, err
			}
		}

		promptPath, err := deps.writeHistoryImportPrompt(req.RepoRoot, cfg, historyImportPromptData{
			Product:             cfg.Project.Name,
			CurrentVersionLabel: currentVersionLabel,
			ChangelogPath:       config.ChangelogPath(req.RepoRoot, cfg),
			ConfigPath:          config.RepoConfigPath(req.RepoRoot),
			FragmentsDir:        config.FragmentsDir(req.RepoRoot, cfg),
			ReleasesDir:         config.ReleasesDir(req.RepoRoot, cfg),
			TemplatesDir:        config.TemplatesDir(req.RepoRoot, cfg),
			PromptPath:          config.HistoryImportPromptPath(req.RepoRoot, cfg),
			AdoptionRecordPath:  recordPath,
			AdoptionFragment:    result.AdoptionFragment,
			AdoptionRecord:      result.AdoptionRecord,
		})
		if err != nil {
			return InitializeResult{}, err
		}
		tx.RecordCreatedFile(promptPath)
		result.PromptPath = promptPath

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

func writeHistoryImportPrompt(repoRoot string, cfg config.Config, data historyImportPromptData) (string, error) {
	path := config.HistoryImportPromptPath(repoRoot, cfg)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create prompt directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(renderHistoryImportPrompt(repoRoot, data)), 0o644); err != nil {
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

func ensureGitignoreWithTx(tx *initTxn, repoRoot string) error {
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
	if err := tx.WriteFile(path, []byte(body), 0o644); err != nil {
		return fmt.Errorf("write .gitignore: %w", err)
	}
	return nil
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
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := tx.WriteFile(path, body, perm); err != nil {
		return false, err
	}
	return true, nil
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
	return os.WriteFile(path, body, perm)
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
