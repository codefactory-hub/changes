---
created: 2026-04-21
status: accepted
---

# ADR-0008: Separate human release auth from automation and agent auth

## Context

`changes` has two distinct authenticated operating modes around release work:

- a maintainer running local release-helper scripts on their own machine
- GitHub Actions or another unattended runner publishing release artifacts and updating the Homebrew tap

Those paths do not share the same trust boundary, UX expectations, or failure modes. The repository also may be used by agents in the future, but the tool should not treat agent execution as a reason to normalize provider-specific secret lookup or generic authenticated shell access in its public CLI surface.

## Decision

- Human local release verification may rely on externally injected environment variables, file-backed secret delivery, or a local launcher or wrapper outside the tool core.
- Unattended automation such as GitHub Actions must use its own runner-side secret injection and must not depend on a human desktop-auth session.
- The `changes` CLI and repository helper scripts do not expose a generic authenticated shell model for agents and do not make provider-specific secret-manager commands part of their public contract.
- Any future agent-facing integration must use a narrower broker, tool-host, or automation-specific boundary instead of inheriting a broadly authenticated operator terminal.
- Repo docs must describe human-local release guidance separately from unattended automation guidance so maintainers can see which path they are using.

## Consequences

Human release rehearsal can stay pragmatic without forcing the same assumptions onto CI or future agent integrations. Automation remains headless and explicit, while the repository avoids turning local provider workflows into the public `changes` command surface.
