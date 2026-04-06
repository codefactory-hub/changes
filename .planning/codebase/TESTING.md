# Testing Patterns

**Analysis Date:** 2026-04-06

## Test Framework

**Runner:**
- Go's built-in `testing` package.
- Config: no dedicated test config file is present; test execution is driven by `go test ./...` in `README.md` and `.github/workflows/ci.yml`.

**Assertion Library:**
- Standard library only. No `stretchr/testify`, `go-cmp`, or assertion helpers are imported.

**Run Commands:**
```bash
go test ./...                    # Run all repository tests
go test ./internal/cli           # Run a single package, useful for CLI integration tests
go test ./... -cover             # Ad hoc coverage run; not wired into CI
```

## Test File Organization

**Location:**
- Tests are co-located with the package under test, such as `internal/app/app_test.go`, `internal/config/config_test.go`, and `internal/render/render_test.go`.
- The main CLI behavior suite lives in `internal/cli/app_integration_test.go`.

**Naming:**
- Use `*_test.go` files and `TestXxx` functions with long behavior-oriented names, such as `TestCommitReleaseRejectsPlanWhenPendingFragmentsChange` in `internal/app/app_test.go`.

**Structure:**
```text
internal/<package>/<package>_test.go
internal/cli/app_integration_test.go
internal/changelog/testdata/rebuild.golden
```

## Test Structure

**Suite Organization:**
```go
func TestLoadRejectsInvalidPublicAPI(t *testing.T) {
	repoRoot := t.TempDir()
	path := RepoConfigPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}

	_, err := Load(repoRoot)
	if err == nil || !strings.Contains(err.Error(), "versioning.public_api must be one of stable, unstable") {
		t.Fatalf("unexpected error: %v", err)
	}
}
```

**Patterns:**
- Follow arrange/act/assert inside a single top-level `TestXxx` function.
- Prefer one behavior per test function; subtests with `t.Run` are not used anywhere under `internal/` or `cmd/`.
- Tests do not use `t.Parallel()`. The suite favors deterministic filesystem state and explicit sequencing.
- Use `t.TempDir()` heavily to isolate repo fixtures, as in `internal/app/app_test.go`, `internal/cli/app_integration_test.go`, and `internal/render/render_test.go`.
- Freeze timestamps with `time.Date(...)` and deterministic randomness with `bytes.NewReader(...)` for stable file names and outputs, especially in `internal/app/app_test.go` and `internal/cli/app_integration_test.go`.

## Mocking

**Framework:** None

**Patterns:**
```go
app := NewApp(&stdout, &stderr)
app.Now = func() time.Time { return fixedTime }
app.Random = bytes.NewReader([]byte{1, 2, 3})
app.IsTTY = func() bool { return true }
app.Stdin = strings.NewReader("minor\n")
app.EditFile = func(path string) error { ... }
```

```go
_, err := initializeWithDeps(ctx, req, initializeDeps{
	createAdoptionBootstrap:  createAdoptionBootstrap,
	writeHistoryImportPrompt: writeHistoryImportPrompt,
	stageHook: func(stage string) error {
		if stage == "after_bootstrap" {
			return errBoom
		}
		return nil
	},
})
```

**What to Mock:**
- Time, randomness, TTY detection, stdin, and editor behavior through injected `App` fields in `internal/cli/app.go`, exercised in `internal/cli/app_integration_test.go`.
- Stage failures and side-effect seams through `initializeDeps` in `internal/app/init.go`, exercised in `internal/app/app_test.go`.

**What NOT to Mock:**
- Core domain packages are usually exercised with real temp directories and real file IO instead of fake repositories. `internal/app/app_test.go`, `internal/render/render_test.go`, and `internal/releases/releases_test.go` use the real `config`, `fragments`, `releases`, and `render` packages together.
- Git repository detection is tested with a real `git init` subprocess in `internal/cli/app_integration_test.go` rather than stubbing repo discovery.

## Fixtures and Factories

**Test Data:**
```go
record := releases.ReleaseRecord{
	Product:          "changes",
	Version:          "0.1.0",
	CreatedAt:        time.Date(2026, 4, 2, 18, 0, 0, 0, time.UTC),
	AddedFragmentIDs: []string{"f1"},
}
```

**Location:**
- Most fixture data is inline through struct literals and raw file bodies in test functions.
- Golden-file verification is used for changelog rendering in `internal/changelog/changelog_test.go` with `internal/changelog/testdata/rebuild.golden`.
- Template fixture setup is helper-driven rather than checked in wholesale. `ensureBuiltinTemplates` in `internal/render/render_test.go` materializes built-in templates into a temp repo before rendering assertions.
- Local helper functions stay at the bottom of the test file and call `t.Helper()`, such as `gitInit` in `internal/cli/app_integration_test.go` and `assertDirEmptyOrMissing` in `internal/app/app_test.go`.

## Coverage

**Requirements:** None enforced
- CI runs `go test ./...` in `.github/workflows/ci.yml`.
- No coverage threshold, `-coverprofile`, or report upload is configured.

**View Coverage:**
```bash
go test ./... -cover
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Test Types

**Unit Tests:**
- Pure or mostly isolated package behavior is covered in files such as `internal/versioning/versioning_test.go`, `internal/semverpolicy/policy_test.go`, `internal/reporoot/reporoot_test.go`, and `internal/config/config_test.go`.
- These tests rely on in-memory values or temp files and assert exact outputs or error text.

**Integration Tests:**
- `internal/cli/app_integration_test.go` is the main integration suite. It creates a temp Git repo, changes into it with `t.Chdir`, runs CLI commands through `App.Run`, and verifies filesystem side effects and stdout/stderr behavior.
- `internal/app/app_test.go` also behaves like service-level integration testing by composing `config`, `fragments`, `releases`, and `render` around temp repositories.
- Rendering and release lineage packages mix unit and integration-style verification by assembling real records/fragments and rendering actual template output in `internal/render/render_test.go`, `internal/changelog/changelog_test.go`, and `internal/releases/*_test.go`.

**E2E Tests:**
- Not used as a separate layer. There are no black-box binary invocation suites, browser tests, or external service tests.
- The closest end-to-end coverage is `TestAppEndToEnd` in `internal/cli/app_integration_test.go`, which exercises `init`, `create`, `status`, `release`, and `render` in one repository lifecycle.

## Common Patterns

**Async Testing:**
```go
ctx, cancel := context.WithCancel(context.Background())
cancel()

if _, err := Initialize(ctx, InitializeRequest{RepoRoot: repoRoot}); !errors.Is(err, context.Canceled) {
	t.Fatalf("Initialize error = %v, want %v", err, context.Canceled)
}
```
- Asynchronous behavior is limited to context cancellation. There are no goroutine coordination or eventual-consistency test patterns in the current suite.

**Error Testing:**
```go
if err == nil || !strings.Contains(err.Error(), "unsupported mode") {
	t.Fatalf("unexpected error: %v", err)
}
```

```go
if _, err := Detect(start); !errors.Is(err, ErrNotGitRepo) {
	t.Fatalf("Detect error = %v, want %v", err, ErrNotGitRepo)
}
```
- Prefer `errors.Is` for sentinel/cancellation cases and `strings.Contains` for user-facing message fragments.
- Fail fast with `t.Fatalf`; softer `t.Error` accumulation is not part of the current style.

## Coverage Posture

- Package coverage breadth is good for the current size of the codebase: every `internal/` package has tests, and `go test ./...` passes for all packages.
- Depth is strongest around filesystem state transitions, rendering behavior, release lineage, and CLI flag validation.
- There are no benchmarks, fuzz tests, or race-test conventions in the repo today.
- Because there is no enforced coverage gate, maintain confidence by adding behavior tests in the package being changed and, for CLI-visible behavior, extending `internal/cli/app_integration_test.go`.

---

*Testing analysis: 2026-04-06*
