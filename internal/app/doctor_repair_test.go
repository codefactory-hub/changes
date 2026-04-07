package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/example/changes/internal/config"
)

func TestDoctorRepairRepoStampsPreferredLegacyCandidate(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write legacy xdg repo config: %v", err)
	}

	result, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeRepo,
		Repair:   true,
	})
	if err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	if result.Repo == nil {
		t.Fatalf("repo scope result is nil")
	}
	if result.Repo.Status != DoctorStatusAuthoritative {
		t.Fatalf("repo status = %q", result.Repo.Status)
	}
	if result.Repo.AuthoritativeStyle != string(config.StyleXDG) {
		t.Fatalf("authoritative style = %q", result.Repo.AuthoritativeStyle)
	}
	if result.Repo.AuthoritativeRoot == "" {
		t.Fatalf("authoritative root should not be empty")
	}
	if result.Repo.Repair == nil {
		t.Fatalf("repair details are nil")
	}
	if !result.Repo.Repair.Changed {
		t.Fatalf("repair should report changed=true")
	}
	if result.Repo.Repair.ManifestPath != filepath.Join(repoRoot, ".config", "changes", "layout.toml") {
		t.Fatalf("manifest path = %q", result.Repo.Repair.ManifestPath)
	}
	if result.Repo.Repair.AuthoritativeStyle != string(config.StyleXDG) {
		t.Fatalf("repair authoritative style = %q", result.Repo.Repair.AuthoritativeStyle)
	}
}

func TestDoctorRepairRepoRejectsAmbiguousLegacyCandidates(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write legacy xdg repo config: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, cfg); err != nil {
		t.Fatalf("write legacy home repo config: %v", err)
	}

	_, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeRepo,
		Repair:   true,
	})
	if err == nil {
		t.Fatalf("Doctor should fail on ambiguous legacy repair")
	}
	if !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %v, want ambiguity guidance", err)
	}
}

func TestDoctorRepairRepoLeavesOrdinaryCommandsOperational(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, cfg); err != nil {
		t.Fatalf("write legacy home repo config: %v", err)
	}

	if _, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeRepo,
		Repair:   true,
	}); err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	if _, _, err := config.LoadWithAuthority(repoRoot); err != nil {
		t.Fatalf("LoadWithAuthority returned error after repair: %v", err)
	}
	if _, err := Status(context.Background(), StatusRequest{RepoRoot: repoRoot}); err != nil {
		t.Fatalf("Status returned error after repair: %v", err)
	}
}

func TestDoctorRepairRepoDoesNotMigrateData(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleXDG, cfg); err != nil {
		t.Fatalf("write legacy xdg repo config: %v", err)
	}
	originalFragment := filepath.Join(repoRoot, ".local", "share", "changes", "fragments", "legacy.md")
	if err := os.MkdirAll(filepath.Dir(originalFragment), 0o755); err != nil {
		t.Fatalf("mkdir legacy fragment dir: %v", err)
	}
	if err := os.WriteFile(originalFragment, []byte("legacy"), 0o644); err != nil {
		t.Fatalf("write legacy fragment: %v", err)
	}

	if _, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeRepo,
		Repair:   true,
	}); err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	if _, err := os.Stat(originalFragment); err != nil {
		t.Fatalf("original fragment missing after repair: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, ".changes", "data", "fragments", "legacy.md")); !os.IsNotExist(err) {
		t.Fatalf("repair should not create migrated home fragment, stat err=%v", err)
	}
}

func TestDoctorRepairRepoPreservesAuthoritativeStateIgnoreRule(t *testing.T) {
	repoRoot := t.TempDir()
	cfg := config.Default()

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, cfg); err != nil {
		t.Fatalf("write legacy home repo config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, ".gitignore"), []byte("vendor/\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	if _, err := Doctor(context.Background(), DoctorRequest{
		RepoRoot: repoRoot,
		Scope:    DoctorScopeRepo,
		Repair:   true,
	}); err != nil {
		t.Fatalf("Doctor returned error: %v", err)
	}

	gitignore, err := os.ReadFile(filepath.Join(repoRoot, ".gitignore"))
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}
	if !strings.Contains(string(gitignore), "/.changes/state/") {
		t.Fatalf(".gitignore missing home state ignore entry:\n%s", gitignore)
	}
}
