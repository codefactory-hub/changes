# Phase 1 Implementation Gate

## Phase 2 entry gate

- [ ] Supported styles: xdg and home only.
- [ ] Scopes resolved independently: global and repo.
- [ ] Global bootstrap precedence: flags > CHANGES_HOME > XDG env vars > built-in default locations.
- [ ] Repo init precedence: flags > [repo.init] defaults > CHANGES_HOME signal > XDG env signal > built-in default locations.
- [ ] Operational validity requires parseable layout.toml with matching scope and style.
- [ ] Legacy-only detection is doctor-visible but invalid for ordinary commands.
- [ ] Multiple supported candidates = ambiguity error.
- [ ] Repair or manifest stamping is explicit only.
- [ ] Doctor tiers: default concise, --explain rich, --json structured.
- [ ] Migration prompt is an advisory Markdown brief with required verification and explicit no-dual-write instructions.
- [ ] Global config bootstrap keys are limited to [repo.init].
- [ ] Repo state ignore rules are /.local/state/ and /.changes/state/.

Open questions: none.

Phase 2 may begin only when every checklist item is Pass.
