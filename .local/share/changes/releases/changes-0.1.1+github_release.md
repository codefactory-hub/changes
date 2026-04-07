# Release 0.1.1

## Fixed

- Narrow the repo-local xdg .gitignore rule to the authoritative state directory. Repo-local xdg init and repair now ignore /.local/state/changes/ instead of the broader /.local/state/ parent directory, so unrelated repo-local state paths are not hidden.
