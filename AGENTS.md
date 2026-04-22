# AGENTS.md

Use the repo-local governance records as the primary source of truth for document routing and implementation etiquette in this repository.

Current governance records:

- [ADR-0001: Use repo-local XDG layout](docs/03-decisions/ADR-0001-repo-local-xdg-layout.md)
- [ADR-0002: Keep fragment artifacts durable](docs/03-decisions/ADR-0002-durable-fragment-artifacts.md)
- [ADR-0003: Represent releases as parent-linked records](docs/03-decisions/ADR-0003-parent-linked-release-manifests.md)
- [ADR-0004: Treat rendered outputs as views](docs/03-decisions/ADR-0004-rendered-outputs-are-views.md)
- [ADR-0005: Establish the initial stable version baseline](docs/03-decisions/ADR-0005-initial-stable-version-baseline.md)
- [ADR-0006: Use Homebrew cask distribution for internal releases](docs/03-decisions/ADR-0006-homebrew-cask-distribution.md)
- [ADR-0007: Adopt lightweight repository governance lifecycle](docs/03-decisions/ADR-0007-adopt-lightweight-repository-governance-lifecycle.md)
- [ADR-0008: Separate human release auth from automation and agent auth](docs/03-decisions/ADR-0008-separate-human-release-auth-from-automation-and-agent-auth.md)
- [ADR-0009: Adopt provider-neutral release secret ingestion](docs/03-decisions/ADR-0009-adopt-provider-neutral-release-secret-ingestion.md)
- [PLN-2026-04-21-01: Modernize governance and adopt relevant toolsmith patterns](docs/04-plans/PLN-2026-04-21-01-modernize-governance-and-adopt-relevant-toolsmith-patterns.md)

## Documentation Governance

- Keep durable repo process policy in `README.md`, `AGENTS.md`, ADRs, templates, and folder structure. During `repository-governance` audit and remediation work, the skill's canonical governance semantics control ambiguity resolution unless the repository has an explicit accepted compatible local exception.
- Use the lifecycle folders `docs/01-ideas/`, `docs/02-research/`, `docs/03-decisions/`, `docs/04-plans/`, `docs/05-insights/`, and `docs/99-archive/`, with matching prefixed subfolders inside `docs/99-archive/`.
- Route documents with this sharp rule: ADR = what we decided. Plan = what we are doing right now.
- Create ideas for early thoughts, research for uncertainty reduction, decisions for durable choices, plans for implementation starting now, and insights for execution or operational learning.
- Archived docs never return to live folders. When archiving, set `status: archived`, move the file under the matching archive folder, and do not modify it afterward.
- All governed docs include `created: YYYY-MM-DD`. ADRs also include `status: proposed | accepted | superseded`.
- Governed docs are Markdown documents and may include YAML frontmatter wherever the governance contract requires metadata.
- In user-facing governed prose, reference other repo-authored Markdown docs with inline Markdown links that use relative URLs and human-readable link text. Prefer the destination H1 when it reads naturally, and otherwise use a concise prose variant that stays clear in sentence context.
- Prefer prose-embedded links over raw backticked paths or path-only code blocks when pointing readers to another governed doc.
- Keep raw code formatting for literal filenames, path patterns, shell commands, inventories, and non-user-facing internal reference material.
- Plans may optionally include `governance_audit: remediation-plan` when they establish the repository's post-audit governance baseline. The newest such plan is the audit cutoff for historical-format drift in older plans, insights, and similar closed execution records, but not for live governing docs.
- Use these filename patterns: `idea-{slug}.md`, `RPT-YYYY-MM-DD-NN-{slug}.md`, `ADR-NNNN-{slug}.md`, `PLN-YYYY-MM-DD-NN-{slug}.md`, and `INS-YYYY-MM-DD-NN-{slug}.md`.
- For research, plans, and insights, `NN` is the zero-padded same-day sequence for that document kind.
- When upgrading an existing repository to this rule, derive same-day sequence from the best available creation signal in this order: explicit timestamp metadata, filesystem creation time, filesystem modification time, then stable lexical filename order.
- Use one shared slug recipe: strip diacritics, lowercase, replace `&` with `and`, remove apostrophes, remove the standard stop-word set, replace remaining non-alphanumerics with `-`, collapse repeated `-`, and trim leading and trailing `-`.
- Preserve pre-baseline historical governed docs conservatively. Records created before [PLN-2026-04-21-01: Modernize governance and adopt relevant toolsmith patterns](docs/04-plans/PLN-2026-04-21-01-modernize-governance-and-adopt-relevant-toolsmith-patterns.md) may retain legacy filenames or lighter metadata after migration into canonical folders.
- Keep operational guides that do not fit the governed lifecycle under `docs/guides/`.

## Planning And Implementation

- Before starting implementation, check whether the working directory is clean.
- If the working directory is not clean, pause and tell the user.
- Do not begin implementation until the user explicitly chooses one of these three paths:
  - checkpoint the current changes
  - treat the current changes as part of the approved work
  - proceed intentionally with a dirty working directory
- If the active runtime provides a structured human-response or multiple-choice UI, prefer that UI for the same three dirty-worktree choices. Otherwise, ask the same three dirty-worktree choices in plain text.
- If the active runtime supports a planning stage with structured human approval, perform the dirty-worktree decision during that planning stage when practical.
- If the user chooses to checkpoint the current changes, use the `git-workflow` skill to create cohesive commits until the working directory is clean.
- Host planning and written repository plans are not the same thing.
- A written repository plan is required before implementation for non-trivial changes.
- A written repository plan is not required for trivial changes.
- Treat a change as non-trivial when it materially affects user-visible behavior or UI, introduces or materially changes a public interface or command surface, spans multiple files or subsystems in a way that benefits from sequencing, introduces migration or rollback concerns, or otherwise requires documented implementation intent.
- Treat a change as trivial when it is narrowly scoped and does not materially change user-visible behavior, public interfaces, or implementation scope.
- If the threshold is unclear, ask whether the work should be treated as trivial or non-trivial rather than defaulting to a written repository plan.
- Planning in the host does not by itself require saving a written repository plan.
- When the agent determines that work is trivial under this policy, it should say so and proceed without creating a durable plan record.
- When a written repository plan is required, the first implementation step must be to save that plan as a Markdown document under `docs/04-plans/` using the naming rules for plans.
- After the plan file is saved, implementation work may begin.
- Every written repository plan must direct the agent to use the `git-workflow` skill as often as prudent to create semantically cohesive commits during implementation.
- A written repository plan may be completed in one commit or in multiple commits, depending on the natural shape of the work.
- A written repository plan is complete when its intended implementation and required validation are finished, the completed work has been committed, and the working directory is clean.
- Git actions must run in series and never in parallel because concurrent Git operations create avoidable lock conflicts and unnecessary user-facing failures.
- The last implementation step in every written repository plan must be to use the `git-workflow` skill to finish committing the work completed for that plan and leave the working directory clean.
- After a written repository plan is complete, additional trivial follow-up work may proceed without reopening the completed plan or creating a new plan.
- Create a new written repository plan for follow-up work only when that follow-up work is itself non-trivial.

## Go toolchain paths for agents

Prefer plain Go commands first in this repository:

```bash
go test ./...
```

Only add writable-path overrides when the environment actually requires them, such as a sandbox or permission error involving Go's cache or module workspace.

When overrides are needed, keep them minimal:

- Start with `GOCACHE` only.
- Use `/tmp` or a repo-local cache directory for writable cache data.
- Add `GOMODCACHE`, `GOPATH`, or `GOBIN` only if the specific command fails without them.
- Do not redirect Go paths preemptively when the plain command already works.

Example fallback:

```bash
env GOCACHE=/tmp/changes-go-build go test ./...
```

If a stricter sandbox still requires repo-local writable paths, use a repo-local fallback intentionally rather than by default.
