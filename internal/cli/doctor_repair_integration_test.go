package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/example/changes/internal/config"
)

func TestDoctorRepairHelpSurfaceIncludesRepoOnlyGrammar(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)

	if err := app.Run(context.Background(), []string{"help", "doctor"}); err != nil {
		t.Fatalf("help doctor returned error: %v", err)
	}

	body := stdout.String()
	for _, fragment := range []string{
		"changes doctor --scope repo --repair",
		"changes doctor --migration-prompt --scope global|repo --to xdg|home [--home PATH] [--output PATH]",
		"Repair one legacy repo-local layout by restoring its authoritative manifest without moving data.",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("doctor help missing %q:\n%s", fragment, body)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("help doctor should not write stderr:\n%s", stderr.String())
	}
}

func TestDoctorRepairRepairsLegacyRepoAndReportsAuthoritativeLayout(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write legacy xdg config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 7, 10, 0, 0, 0, time.UTC)
	}

	if err := app.Run(context.Background(), []string{"doctor", "--scope", "repo", "--repair"}); err != nil {
		t.Fatalf("doctor --repair returned error: %v\nstderr=%s", err, stderr.String())
	}

	body := stdout.String()
	for _, fragment := range []string{
		"repo: repaired xdg",
		".config/changes/layout.toml",
		"authoritative: xdg",
		"root: .",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("repair output missing %q:\n%s", fragment, body)
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("doctor --repair should not write stderr:\n%s", stderr.String())
	}
}

func TestDoctorRepairFailsLoudlyOnAmbiguousLegacyRepo(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleXDG, config.Default()); err != nil {
		t.Fatalf("write legacy xdg config: %v", err)
	}
	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write legacy home config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	err := app.Run(context.Background(), []string{"doctor", "--scope", "repo", "--repair"})
	if err == nil {
		t.Fatalf("doctor --repair should fail for ambiguous legacy repo")
	}
	body := stderr.String()
	for _, fragment := range []string{
		"error:",
		"ambiguous",
		"changes doctor --migration-prompt --scope repo --to xdg|home",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("stderr missing %q:\n%s", fragment, body)
		}
	}
	if stdout.Len() != 0 {
		t.Fatalf("doctor --repair should not write stdout on failure:\n%s", stdout.String())
	}
}

func TestDoctorRepairLeavesStatusOperational(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	if err := writeLegacyRepoConfigForStyle(t, repoRoot, config.StyleHome, config.Default()); err != nil {
		t.Fatalf("write legacy home config: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	app := NewApp(&stdout, &stderr)
	app.Now = func() time.Time {
		return time.Date(2026, 4, 7, 10, 5, 0, 0, time.UTC)
	}

	if err := app.Run(context.Background(), []string{"doctor", "--scope", "repo", "--repair"}); err != nil {
		t.Fatalf("doctor --repair returned error: %v\nstderr=%s", err, stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	if err := app.Run(context.Background(), []string{"status"}); err != nil {
		t.Fatalf("status returned error after repair: %v\nstderr=%s", err, stderr.String())
	}
	if !strings.Contains(stdout.String(), "Current version: unreleased") {
		t.Fatalf("status output missing current version after repair:\n%s", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("status should not write stderr after repair:\n%s", stderr.String())
	}
}

func TestDoctorRepairRejectsInvalidFlagCombinations(t *testing.T) {
	repoRoot := t.TempDir()
	gitInit(t, repoRoot)
	t.Chdir(repoRoot)

	var cases = []struct {
		name string
		args []string
		want string
	}{
		{name: "json", args: []string{"doctor", "--scope", "repo", "--repair", "--json"}, want: "--repair cannot be combined with --json"},
		{name: "explain", args: []string{"doctor", "--scope", "repo", "--repair", "--explain"}, want: "--repair cannot be combined with --explain"},
		{name: "migration", args: []string{"doctor", "--scope", "repo", "--repair", "--migration-prompt"}, want: "--repair cannot be combined with --migration-prompt"},
		{name: "to", args: []string{"doctor", "--scope", "repo", "--repair", "--to", "home"}, want: "--repair cannot be combined with --to"},
		{name: "home", args: []string{"doctor", "--scope", "repo", "--repair", "--home", ".changes-next"}, want: "--repair cannot be combined with --home"},
		{name: "output", args: []string{"doctor", "--scope", "repo", "--repair", "--output", "repair.txt"}, want: "--repair cannot be combined with --output"},
		{name: "scope", args: []string{"doctor", "--scope", "global", "--repair"}, want: "--repair requires --scope repo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer

			app := NewApp(&stdout, &stderr)
			err := app.Run(context.Background(), tc.args)
			if err == nil {
				t.Fatalf("Run returned nil error for %v", tc.args)
			}
			if !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("stderr missing %q:\n%s", tc.want, stderr.String())
			}
			if stdout.Len() != 0 {
				t.Fatalf("stdout should be empty on invalid flag combinations:\n%s", stdout.String())
			}
		})
	}
}
