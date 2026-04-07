package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/changes/internal/config"
)

func TestDoctorResolvedScopeExplainsWinningLayout(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write xdg config: %v", err)
	}

	result, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeRepo,
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	if result.RequestedScope != string(DoctorScopeRepo) {
		t.Fatalf("requested scope = %q", result.RequestedScope)
	}
	if result.Repo == nil {
		t.Fatalf("repo scope result is nil")
	}
	if result.Repo.Status != DoctorStatusAuthoritative {
		t.Fatalf("repo status = %q", result.Repo.Status)
	}
	if result.Repo.SelectedStyle != string(config.StyleXDG) {
		t.Fatalf("selected style = %q", result.Repo.SelectedStyle)
	}
	if result.Repo.AuthoritativeStyle != string(config.StyleXDG) {
		t.Fatalf("authoritative style = %q", result.Repo.AuthoritativeStyle)
	}
	if result.Repo.PreferredStyle != result.Repo.AuthoritativeStyle {
		t.Fatalf("preferred style = %q, authoritative style = %q", result.Repo.PreferredStyle, result.Repo.AuthoritativeStyle)
	}

	candidate := doctorCandidateByStyle(t, result.Repo.Candidates, config.StyleXDG)
	if !candidate.IsAuthoritative {
		t.Fatalf("xdg candidate should be authoritative: %#v", candidate)
	}
	if !candidate.IsPreferred {
		t.Fatalf("xdg candidate should be preferred: %#v", candidate)
	}
	if candidate.Manifest == nil || candidate.Manifest.SchemaVersion != 1 {
		t.Fatalf("candidate manifest = %#v", candidate.Manifest)
	}
	if !doctorEvidenceContains(candidate.Evidence, "layout.toml", true) {
		t.Fatalf("candidate evidence missing layout.toml existence: %#v", candidate.Evidence)
	}
}

func TestDoctorAmbiguousScopePreservesCandidateConflictDetails(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}
	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleHome, cfg); err != nil {
		t.Fatalf("write home repo config: %v", err)
	}

	homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome := configureGlobalLayoutEnv(t)
	if err := writeManagedGlobalRepoInitDefaults(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, config.StyleXDG, "[repo.init]\nstyle = \"xdg\"\n"); err != nil {
		t.Fatalf("write managed global defaults: %v", err)
	}
	if err := writeLegacyGlobalRepoInitDefaults(t, homeDir, changesHome, xdgConfigHome, xdgDataHome, xdgStateHome, config.StyleHome, "[repo.init]\nstyle = \"home\"\nhome = \".changes-home\"\n"); err != nil {
		t.Fatalf("write legacy global defaults: %v", err)
	}

	result, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeAll,
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	if result.Repo == nil {
		t.Fatalf("repo scope result is nil")
	}
	if result.Repo.Status != DoctorStatusAmbiguous {
		t.Fatalf("repo status = %q", result.Repo.Status)
	}
	if result.Repo.AuthoritativeStyle != "" {
		t.Fatalf("repo authoritative style = %q, want empty", result.Repo.AuthoritativeStyle)
	}
	if len(result.Repo.Candidates) != 2 {
		t.Fatalf("repo candidates = %d, want 2", len(result.Repo.Candidates))
	}
	for _, style := range []config.Style{config.StyleXDG, config.StyleHome} {
		candidate := doctorCandidateByStyle(t, result.Repo.Candidates, style)
		if candidate.Status != string(config.StatusResolved) {
			t.Fatalf("%s candidate status = %q", style, candidate.Status)
		}
		if candidate.IsAuthoritative {
			t.Fatalf("%s candidate should not be authoritative during ambiguity", style)
		}
		if candidate.Manifest == nil {
			t.Fatalf("%s candidate missing manifest", style)
		}
	}

	if result.Global == nil {
		t.Fatalf("global scope result is nil")
	}
	if result.Global.Status != DoctorStatusAuthoritative {
		t.Fatalf("global status = %q", result.Global.Status)
	}
	if len(result.Global.Warnings) != 1 {
		t.Fatalf("global warnings = %#v, want 1 warning", result.Global.Warnings)
	}
	if result.Global.Warnings[0].Status != string(config.StatusLegacyOnly) {
		t.Fatalf("global warning status = %q", result.Global.Warnings[0].Status)
	}
}

func TestDoctorStructuredInspectionSupportsConciseExplainAndJSON(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, cfg); err != nil {
		t.Fatalf("write legacy home repo config: %v", err)
	}

	result, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeRepo,
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	if result.Repo == nil {
		t.Fatalf("repo scope result is nil")
	}
	if result.Repo.SelectedStyle != string(config.StyleXDG) {
		t.Fatalf("selected style = %q", result.Repo.SelectedStyle)
	}
	if result.Repo.SelectedRoot == "" {
		t.Fatalf("selected root should not be empty")
	}
	if len(result.Repo.PrecedenceInputs) == 0 {
		t.Fatalf("precedence inputs should not be empty")
	}
	if len(result.Repo.Candidates) != 2 {
		t.Fatalf("repo candidates = %d", len(result.Repo.Candidates))
	}
	if len(result.Repo.Candidates[0].Evidence) == 0 {
		t.Fatalf("candidate evidence should not be empty")
	}
	if len(result.Repo.Warnings) != 1 {
		t.Fatalf("repo warnings = %#v, want 1 warning", result.Repo.Warnings)
	}

	raw, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal doctor result: %v", err)
	}
	body := string(raw)
	for _, fragment := range []string{
		`"requested_scope":"repo"`,
		`"selected_style":"xdg"`,
		`"precedence_inputs":[`,
		`"candidates":[`,
		`"warnings":[`,
		`"summary":`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("json output missing %q:\n%s", fragment, body)
		}
	}
}

func TestDoctorMigrationPromptIncludesDeterministicMetadata(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}
	writeDoctorArtifact(t, filepath.Join(repoRoot, ".config", "changes", "config.extra.toml"))
	writeDoctorArtifact(t, filepath.Join(repoRoot, ".local", "share", "changes", "fragments", "fragment-one.md"))
	writeDoctorArtifact(t, filepath.Join(repoRoot, ".local", "state", "changes", "session.lock"))

	result, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot:              repoRoot,
		Scope:                 DoctorScopeRepo,
		GenerateMigrationPrompt: true,
		DestinationStyle:      config.StyleHome,
		DestinationHome:       ".changes-next",
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	prompt := result.MigrationPrompt
	for _, section := range []string{
		"## Requested Migration",
		"## Origin Layout",
		"## Destination Layout",
		"## Artifact Inventory",
		"## Ambiguity and Conflict Notes",
		"## Required Verification",
		"## Safety Rules",
	} {
		if !strings.Contains(prompt, section) {
			t.Fatalf("migration prompt missing section %q:\n%s", section, prompt)
		}
	}

	for _, fragment := range []string{
		"Requested scope: repo",
		"Requested destination style: home",
		filepath.Join(repoRoot, ".config", "changes"),
		filepath.Join(repoRoot, ".local", "share", "changes"),
		filepath.Join(repoRoot, ".local", "state", "changes"),
		filepath.Join(repoRoot, ".changes-next"),
		"Manifest schema_version: 1",
		"config inventory",
		"data inventory",
		"state inventory",
	} {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("migration prompt missing %q:\n%s", fragment, prompt)
		}
	}
}

func TestDoctorMigrationPromptIncludesConflictNotesAndVerification(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write xdg repo config: %v", err)
	}
	if err := writeManagedRepoConfigForStyle(t, repoRoot, config.StyleHome, cfg); err != nil {
		t.Fatalf("write home repo config: %v", err)
	}

	result, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot:                repoRoot,
		Scope:                   DoctorScopeRepo,
		GenerateMigrationPrompt: true,
		DestinationStyle:        config.StyleXDG,
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	prompt := result.MigrationPrompt
	for _, fragment := range []string{
		"Status: ambiguous",
		"No authoritative origin candidate exists.",
		"Competing candidates:",
		"Preserve exactly one authoritative destination.",
		"Do not dual-write or keep two live authoritative layouts.",
		"## Required Verification",
		"Confirm the final layout resolves through changes doctor",
	} {
		if !strings.Contains(prompt, fragment) {
			t.Fatalf("migration prompt missing %q:\n%s", fragment, prompt)
		}
	}
}

func doctorCandidateByStyle(t *testing.T, candidates []DoctorCandidate, style config.Style) DoctorCandidate {
	t.Helper()
	for _, candidate := range candidates {
		if candidate.Style == string(style) {
			return candidate
		}
	}
	t.Fatalf("missing candidate for style %s", style)
	return DoctorCandidate{}
}

func doctorEvidenceContains(items []DoctorEvidence, name string, exists bool) bool {
	for _, item := range items {
		if item.Name == name && item.Exists == exists {
			return true
		}
	}
	return false
}

func writeDoctorArtifact(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte("artifact"), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
