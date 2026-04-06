# ADR-0001 Repo-local XDG Layout

## Status

Accepted

## Context

`changes` operates inside arbitrary Git repositories. Its configuration, durable release records, and templates need to be reviewable alongside the repository content they affect.

Storing the primary source-of-truth data in user-home directories would make review harder, couple release state to one workstation, and complicate automation that runs against many repositories.

## Decision

Use repository-relative XDG-like paths as the primary layout:

- `.config/changes/config.toml`
- `.local/share/changes/fragments/`
- `.local/share/changes/releases/`
- `.local/share/changes/prompts/`
- `.local/share/changes/templates/`
- `.local/state/`

Treat `.config/changes/**` and `.local/share/changes/**` as committed repository data.

Treat `.local/state/**` as transient local state that should be ignored by Git.

User-level XDG locations may still be supported later for shared defaults, but they are not the primary source of truth for a managed repository.

## Consequences

- repository-local changelog state is reviewable and portable
- automation can work against a repository without depending on user-home state
- transient state remains clearly separated from durable release data
