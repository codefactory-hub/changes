+++
type = "fixed"
behavior = "fix"
release_notes_priority = 0
display_order = 0
+++

Narrow the repo-local xdg .gitignore rule to the authoritative state directory.

Repo-local xdg init and repair now ignore /.local/state/changes/ instead of the broader /.local/state/ parent directory, so unrelated repo-local state paths are not hidden.
