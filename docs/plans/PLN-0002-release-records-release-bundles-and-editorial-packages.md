# PLN-0002 Release Records, Release Bundles, And Editorial Packages

## Scope

Evolve `changes` around four layers with distinct responsibilities:

- `ReleaseRecord`: canonical stored release files
- `ReleaseBundle`: the assembled factual data for one release
- `EditorialPackage`: a derived editorial communication artifact for one release
- `RenderedRelease`: the final formatted output for a specific surface

The first milestone replaces the current manifest-centric model with product-aware release records, a library-backed SemVer model, companion-record support, and deterministic release-bundle assembly.

## Design constraints

- `Manifest` becomes `ReleaseRecord` across code, tests, docs, and CLI language.
- Release records are stored under `.local/share/changes/releases/`.
- Release record filenames follow `<product>-<version>.toml`.
- Every release record file includes both `product` and `version` in the file body.
- `version` is the only persisted version identity field.
- `target_version` is derived from `version` and is not stored.
- `channel` is not stored.
- No opaque release-record ID is introduced.
- `github.com/Masterminds/semver/v3` is the version authority.
- A base release record is required for every release identity.
- Base release records must not contain build metadata.
- Companion release records are optional and use build metadata to identify additional records for the exact same release.
- Build metadata groups companion records; it never affects release precedence or ordering.
- Release identity is `(product, version without build metadata)`.
- Rendering stays downstream of canonical release assembly.
- Editorial shaping happens after release-bundle assembly, not inside release records.

## Milestone 1

- Rename `Manifest` to `ReleaseRecord`.
- Replace the hand-rolled version parser with `github.com/Masterminds/semver/v3`.
- Define base-record and companion-record schemas.
- Make product first-class in release storage and lookup.
- Extend fragment metadata with section, audience, platform, and ordering hints.
- Add deterministic `ReleaseBundle` assembly:
  - start from the base release record
  - discover companion records for the same release identity
  - resolve lineage from the base release record
  - load referenced fragments
  - apply deterministic grouping, ordering, and validation
- Update rendering to consume `ReleaseBundle`.
- Add `changes resolve --product <product> --version <version>`.

## Record model

### Base `ReleaseRecord`

Base records carry:

- `product`
- `version`
- `parent_version`
- `created_at`
- `added_fragment_ids`
- `display_title`
- `summary`
- `edition`
- `source_url`
- ordered section definitions for the release

The base record anchors lineage and fragment selection.

### Companion `ReleaseRecord`

Companion records carry:

- `product`
- `version` with build metadata
- `companion_purpose`
- only the additional fields relevant to that companion purpose

Companion records refer to the exact same release as the base record and never replace it.

## Bundle model

`ReleaseBundle` is the assembled factual data for one release. It includes:

- the base release record
- associated companion records
- lineage context
- ordered sections
- ordered entries
- must-include fragment IDs
- provenance back to fragment IDs
- audience, platform, area, and ordering hints carried by fragments

Assembly always starts from the base record. Companion records are associated records, not overrides by default.

## Fragment metadata additions

Add optional fragment fields for:

- `section_key`
- `area`
- `platforms`
- `audiences`
- `customer_visible`
- `support_relevance`
- `requires_action`
- `release_notes_priority`
- `display_order`

Ownership split:

- fragments own per-entry semantics and hints
- base release records own release-wide framing, section titles, and section order
- companion records own companion-specific additions only

## Follow-up work

- persisted `EditorialPackage` artifacts derived from `ReleaseBundle`
- AI-assisted editorial drafting on top of `ReleaseBundle` and `EditorialPackage`
- channel-specific render/export adapters that consume `ReleaseBundle` or `EditorialPackage`
