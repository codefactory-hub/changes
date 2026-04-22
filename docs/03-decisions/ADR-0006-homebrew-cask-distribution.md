---
created: 2026-04-02
status: accepted
---

# ADR-0006: Use Homebrew cask distribution for internal releases

## Status

Accepted

## Context

The tool must be distributed as a compiled binary through GoReleaser and published into an internal/private Homebrew tap.

CLI tooling is often discussed in terms of Homebrew formulae, but the delivery requirement here is binary-first internal distribution with explicit artifact URLs and credentialed access.

## Decision

Use a Homebrew cask publication path in the release automation for the bootstrap release workflow.

Keep the GoReleaser configuration placeholder-driven so repository owners can supply organization-specific owners, URLs, branches, and tokens without hardcoding them into the repository.

## Consequences

- the bootstrap release path matches the internal binary-delivery requirement
- tap ownership and credentials remain configurable
- future changes to signing, notarization, or distribution policy can evolve without changing the core `changes` data model
