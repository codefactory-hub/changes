+++
type = "added"
public_api = "add"
behavior = "new"
+++

Add adoption bootstrap support for repositories that start using `changes` after they already have released versions.

`changes init --current-version <semver|unreleased>` can now establish a release-history baseline, create a standard adoption release when needed, and generate a repo-specific LLM prompt for reconstructing older release history.
