# RPT-0002 Real-World Changelog Model Fit For `changes`

## Research note

Date: 2026-04-03

This note summarizes what the recent real-world changelog ingestion experiments imply about the current `changes` model.

During this exploration, a large set of ignored-state collection, extraction, reconstruction, and comparison artifacts were generated for many products and libraries. The main temporary artifact types included:

- collection snapshots
- extracted fragment workspaces
- reconstructed release manifests
- rendered comparison outputs
- reconstruction reports

Examples of temporary artifact names included:

- `catalog-check.json`
- `reconstruction-report.json`
- per-product `changes/fragments/`
- per-source `changes/releases/<source-id>/`
- per-source `changes/rendered/<source-id>/repository_markdown.md`

Those artifacts remain ignored-state working material and are not being brought into the repository. This report records the architectural conclusions and a small number of illustrative values, not the raw collected data itself.

## Question

Is the current `changes` model sufficient to serve as the source of truth for:

- repository changelogs
- release notes
- user-facing "what's new" pages
- richer product-update surfaces

Or is it too narrowly defined?

## Executive conclusion

The current model is sufficient for conventional changelog capture, but it is too narrow for faithful release notes and richer user-facing update surfaces without additional metadata available at render time.

That is not a rejection of the current design. The core architecture still appears sound:

- fragments are good canonical change atoms
- release manifests are good lineage and inclusion records
- rendered outputs should remain views

The gap is expressiveness, not foundation.

In practice:

- the model works well for engineering-facing changelogs and package-style release histories
- the model starts to strain when a source organizes information by product area, audience, platform, channel, or release-display concepts rather than by simple change-type groupings

## Main finding

The experiments suggest that the current system captures **change facts**, but not enough **render-time context**.

That distinction matters.

For a basic changelog, it is often enough to know:

- title
- body
- change type
- bump
- whether the change is breaking
- which release included it

For a release note or "what's new" page, additional information often matters just as much:

- the display title of the release
- the month or named milestone of the release
- the release train or edition, such as stable vs insiders
- the semantic section the entry belongs in
- the ordering of sections and entries
- the product area or app surface involved
- the intended audience or platform

Without that context, the renderer is forced either to:

- infer structure from prose repeatedly, or
- flatten the content into generic buckets that do not match how the original release communication was actually organized

## Evidence from the experiments

### 1. Conventional changelogs fit the current model well

Package and library changelogs with clear version headings and grouped entries fit the current model naturally.

These sources generally already look like:

- release heading
- list of items
- possibly grouped into simple categories like added or fixed

That is close to the shape that `changes` already models well.

### 2. Aggregated release hubs and marketing-heavy pages are weak inputs

Some sources performed poorly not because the model was wrong, but because the source was not really a changelog in canonical form.

These sources often mix:

- release content
- navigation
- marketing language
- upgrade guidance
- documentation index material

That kind of input is a parsing problem first, and only secondarily a modeling problem.

### 3. Structured product release notes reveal the real limitation

The strongest evidence came from documentation-backed release-note sources such as Visual Studio Code.

Those release-note files are not primarily grouped by generic change types like:

- added
- fixed
- changed

Instead, they are grouped by semantic product sections such as:

- agent controls
- terminal
- accessibility
- languages

That is a different organizational model.

The current system can store those sections as fragment titles and bodies, but it cannot yet represent them as first-class render structure. As a result, reconstruction works much better once the parser understands the source, but the renderer still lacks an explicit way to say:

- this is the release heading
- these are the top-level release sections
- this section order matters

Illustrative values from the ignored-state trial support that conclusion.

For the Visual Studio Code repo-backed trial:

- the earlier repo-file path averaged about `0.6563` recall and `0.8271` precision
- the later H2-splitting path averaged about `0.6852` recall and `0.9148` precision
- one representative release improved from about `0.0136 / 0.5385` to about `0.8911 / 0.9892`

Those numbers are useful as examples, but the more important finding is structural: the quality improved when the system treated the file as one release and treated each `##` section as a first-class unit.

## What this says about sufficiency

### Sufficient today

The current model is sufficient for:

- canonical change capture
- append-only release lineage
- repository changelogs
- conventional package and library release histories
- deterministic release selection

### Not sufficient today

The current model is not quite sufficient for:

- high-fidelity release notes
- polished "what's new" pages
- multi-platform product updates
- channel-specific presentation from a single source of truth
- product-area-oriented documentation release notes

The reason is not that fragments and manifests are the wrong primitives.
The reason is that those primitives do not yet carry enough structured metadata for richer rendering.

## Design implication

The system likely needs a hybrid metadata model:

- strict core fields for durable semantics
- typed optional metadata for rendering and editorial structure
- a small escape hatch for source-specific extensions

This is preferable to either extreme:

- an overly rigid schema that cannot represent real release-note structures
- a fully loose metadata blob that weakens tooling and validation

## What should remain strict

The current core should stay strict and boring.

At the fragment level, the strict core still seems right:

- `title`
- `body`
- `type`
- `bump`
- `breaking`
- authorship and identity

At the release-manifest level, the strict core also still seems right:

- release identity
- version and lineage
- publication time
- inclusion of fragment IDs

Those are durable semantics and should not be diluted.

## What likely needs to be added

The experiments point toward a need for first-class metadata that is available at render time.

### Fragment-level metadata

The most useful additions appear to be:

- `section_key`
- `section_title`
- `area`
- `platforms`
- `audience`
- `display_order`

These fields would let a renderer reproduce the semantic structure of release notes without re-parsing body text.

### Release-level metadata

The most useful additions appear to be:

- `display_title`
- `month_label`
- `channel`
- `edition`
- `summary`
- `source_url`

These fields would let the same release be rendered differently for:

- a repository changelog
- a formal release page
- a user-facing in-product "what's new" page

## Why render-time metadata matters

The central lesson is that richer release communication is not just a different template over the same flat data.

It often depends on structural decisions such as:

- what the release is called
- what order the sections appear in
- which product areas are prominent
- which entries are user-facing and which are appendix material

Those are not all fragment-body concerns.
They are part of the release representation.

If the system does not model them explicitly, the renderer has to guess them from prose, which makes outputs brittle and source-specific.

## Recommended direction

The best next step is not to replace the current model.

It is to extend it carefully:

1. Keep fragments and manifests as the canonical core.
2. Add first-class release metadata for display and channel semantics.
3. Add first-class fragment metadata for section identity and ordering.
4. Preserve a small extension space for source-specific extras.

That would let `changes` stay simple for ordinary engineering changelogs while becoming expressive enough to support richer release-note systems.

## Bottom line

The experiments do not show that `changes` is fundamentally too narrow for changelogs.

They do show that it is currently too narrow for a broader ambition:

- one source of truth
- many renderings
- including polished release notes and "what's new" pages

So the answer is:

- for changelogs alone, the model is largely sufficient
- for richer release communication, the model needs additional structured metadata at render time

That is an evolution of the current design, not a repudiation of it.
