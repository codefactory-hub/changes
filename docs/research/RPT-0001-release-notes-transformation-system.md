
# RPT-0001 Release Notes Transformation System For `changes`

## Research note

This revision now incorporates the user-supplied raw text and HTML of Anna Pickard’s original Slack article directly. That removes the earlier uncertainty: the Slack-specific principles in this memo are grounded in the article itself, not just later retrospectives or second-hand summaries.

The most important additions from the article are architectural, not merely stylistic:

- release notes are optional and disposable, so if someone reads them they should get something real and tangible out of them
- release notes do several jobs at once: announce what is new, show that bug reports are heard and acted on, educate people about the product, say sorry sincerely when needed, celebrate engineering work, and still carry some charm
- the workflow is editorial: keep a running list, edit it down, divide it into **new** and **fixed**, order the new items by excitement, and keep fixes focused on things users actually experienced, reported, or would find meaningfully odd or interesting
- nothing goes out without clarification from an editor who is willing to approach the notes with the mindset of a regular user and ask basic questions
- humor is seasoning, not substance; if something genuinely hurt users, do not cover it with jokes
- content must remain primary; voice matters only insofar as it helps people read and understand the content

## Directly extracted design principles from Anna Pickard’s article

These are the principles I would now treat as first-class design inputs for `changes`, because they are explicit in the article rather than inferred from later commentary:

1. **Optional surfaces must still justify themselves.** Release notes are “disposable” and not required reading, so the system should optimize for giving a reader a concrete takeaway quickly rather than assuming a captive audience.
2. **Release notes are multi-purpose by design.** They are simultaneously about announcement, responsiveness, education, apology, celebration, and tone. That means one flat rendering is often not enough; layered derived artifacts are justified.
3. **The editorial split is simple and useful: `new` and `fixed`.** That split should be easy to produce from canonical data, but it should stay an editorial/output construct rather than becoming manifest semantics.
4. **“New” items should be ordered by excitement.** That implies derived bundles should allow editorial ordering fields that are independent from fragment creation order or manifest order.
5. **“Fixed” items should be externally meaningful.** Pickard’s standard was not “every internal repair,” but things users experienced, reported, or would recognize as meaningfully odd. That argues for explicit fragment metadata such as `customer_visible`, `externally_felt`, and `release_notes_priority`.
6. **A less-technical editor is part of the quality system.** The article repeatedly describes clarification, back-and-forth, examples, and “stupid questions” from a regular-user perspective. That strongly supports a review state and editor role for derived narrative bundles.
7. **Severe pain needs plain language.** If a bug genuinely hurt people, the system should be able to force a plain apology / acknowledgement path and suppress playful rendering in the relevant message unit.
8. **Humor needs pacing and limits.** Pickard’s ideas about rhythm, not putting too many funny lines in a row, and treating humor like salt all point to a “voice budget,” which is best enforced in narrative review and channel rendering, not in fragment authoring.
9. **Content outranks character.** The system should default toward clarity and traceability, with charm as an optional enhancer. In schema terms, facts need provenance; tone does not get to rewrite provenance.
10. **Fast-release teams need a low-friction path.** The article explicitly mentions mobile teams preferring a plain list so they can move to the next iteration. That validates a workflow where teams can stop at factual fragments + manifests, and an editor or AI-assisted narrative pass can happen later without blocking engineering throughput.

## Executive recommendation

Keep the current architecture exactly where it is strongest:

- **Fragments** remain the canonical factual atoms.
- **Release manifests** remain canonical lineage and selection records.
- **Narrative artifacts** become a new, explicitly derived layer.
- **Channel outputs** remain disposable views over canonical + derived data.

The right next step is **not** “make manifests richer” and **not** “render everything directly from raw fragments forever.”

Instead, add one new derived layer: a **reviewable narrative bundle** per release line and version. That bundle should sit between selection and channel rendering. It should be:

- traceable to fragment IDs and manifest lineage
- allowed to contain interpretation and audience-specific phrasing
- reviewable by humans
- safe for AI assistance
- clearly marked as **derived, not canonical**

Conceptually, the system becomes:

1. **Canonical facts**
   - fragments
   - release manifests
2. **Computed release slice**
   - “what changed on this line since parent_version?”
3. **Derived narrative bundle**
   - clustered themes
   - audience-specific message units
   - tone-safe summaries
   - provenance map back to fragments
4. **Channel renders**
   - GitHub/GitLab release body
   - Slack announcement
   - App Store notes
   - email/blog/update page copy

That model matches the strongest lessons from Slack-style release communication: release notes are not just a ledger, they are a user touchpoint; but the touchpoint works because the underlying product truth is solid, the voice is deliberate, and the message is adapted to where the user encounters it. The modern in-product examples in Candu reinforce a multi-layer, multi-channel strategy, while the Slack GitHub Action example shows Slack announcements as a downstream publication step, not the source of truth. Released’s “train from examples, then generate and review” workflow is useful as an AI pattern, but only as a derived layer with review. GitHub and GitLab’s release descriptions are distribution surfaces, not canonical storage; GitHub’s generated release notes are explicitly ephemeral. Apple’s App Store field is localized and capped at 4000 characters, which is exactly why a separate compressed end-user layer is needed.  

## 1. Conceptual model

### Unified model

Treat release communication as a **compilation pipeline** built on top of durable source records.

The layers should be:

#### Layer A — Canonical factual layer
This is your current architecture.

- **Fragments**: durable factual records of individual changes
- **Release manifests**: append-only lineage records that say which fragment IDs became part of a given release step on a given line

This layer answers:
- what changed?
- when did it become part of this release line?
- which release step introduced it?
- what raw authorial record supports it?

This layer must stay boring, explicit, and auditable.

#### Layer B — Computed release slice
This is a deterministic view, not a new source of truth.

It resolves a release manifest against its parent lineage and produces:
- release line
- version
- parent version
- newly added fragments at this step
- optionally, cumulative fragments reachable on the line
- scopes/types/bump summaries
- any deterministic grouping metadata

This layer answers:
- what is “the delta” for this release step?
- what is reachable on this line?
- what is preview-only versus stable?

This is best treated like a compiler IR: useful, structured, reproducible, but not usually hand-authored.

#### Layer C — Derived narrative bundle
This should be the new interpretive layer.

It contains:
- grouped themes
- message units
- audience-specific summaries
- optional titles/intros/CTAs
- explicit provenance back to fragment IDs
- generation metadata and review state

This layer answers:
- how should humans understand this release?
- what matters to maintainers vs support vs end users?
- how much can we compress without lying?
- which statements are purely factual summaries and which are interpretive abstractions?

This is where Slack-style release communication belongs.

#### Layer D — Channel-specific outputs
These are final renders for specific surfaces.

Examples:
- repository changelog page
- GitHub/GitLab release body
- Slack announcement
- App Store “What’s New”
- email digest
- blog/update post
- in-product “What’s New” payload

This layer answers:
- how should the message look here?
- what should be omitted because of surface limits?
- what CTA or link belongs in this channel?

This layer is disposable and regenerable.

### How the current `changes` philosophy maps onto the sources

The strongest synthesis is:

- **Slack/Anna Pickard** argues that release notes are a rare, low-friction place to be useful, human, honest, and memorable. The later Slack retrospective makes that even clearer: the notes should show progress, show responsiveness to bug reports, educate users about hidden features, celebrate engineering work, acknowledge mistakes, and be fun enough that people want to read and share them.
- **Released** demonstrates a practical AI workflow: give examples, generate issue descriptions, then generate intro/title, then review. That is a good model for a derived layer, not for canonical storage.
- **Candu** frames release notes as a multi-channel communication system: archive, in-product visibility, contextual nudges, targeted messages, and CTAs. That maps cleanly onto “narrative bundle -> multiple renders.”
- **Slack’s GitHub Action docs** show release announcements as a downstream publishing step that accepts a release body string and posts it into Slack via a workflow. That reinforces that Slack is an output target, not where selection logic should live.
- **GitHub/GitLab/Apple docs** reinforce that release pages and store notes are channel surfaces with their own constraints. They are important outputs, but they should not own your source semantics.

### “Manifests as Git commits” is a strong foundation

Yes, this is a strong design foundation, with one important caveat.

It is strong because release manifests, like commits:

- are append-only
- are lineage-aware
- are meaningful because of parent linkage
- identify a specific incremental step
- support reachability and diff reasoning
- keep selection explicit instead of inferred

That is exactly the right mental model for release selection.

The caveat: manifests are analogous to **commits**, but they should **not** absorb the role of commit messages, changelog prose, or marketing copy. In Git, the commit object identifies a graph step; many downstream tools summarize it differently. `changes` should preserve that separation.

So the analogy is:

- fragment = durable change atom
- release manifest = release-selection commit
- computed release slice = resolved diff
- narrative bundle = curated release explanation
- channel output = rendered presentation for a surface

That is a healthy architecture.

## 2. Transformation pipeline design

The recommended pipeline is below.

### Stage 0 — Fragment authoring
**Input:** engineering/product/support knowledge at change time  
**Output:** fragment files  
**Canonical:** yes  
**Persisted:** yes  

Transformations:
- none beyond schema validation and normalization

Rules:
- factual, local, specific
- no audience compression required
- may include terse technical body and structured metadata
- should be durable even if never used in a public-facing note

No interpretation should be added here beyond normal author judgment.

### Stage 1 — Manifest creation
**Input:** selected fragment IDs for a release step  
**Output:** release manifest  
**Canonical:** yes  
**Persisted:** yes  

Transformations:
- explicit selection
- explicit parent linkage
- line assignment (stable/preview)
- version assignment

No narrative belongs here.

### Stage 2 — Release slice resolution
**Input:** release manifest + parent lineage + referenced fragments  
**Output:** resolved release slice object  
**Canonical:** no  
**Persisted:** usually no; cache optional  

Transformations:
- lineage traversal
- delta resolution
- validation that all fragment IDs exist
- deterministic grouping by type, scope, bump, component, platform, locale, audience hints
- deterministic “must include” extraction (breaking changes, migrations, security, operator action items)

This stage is deterministic and should be reproducible bit-for-bit.

### Stage 3 — Theme clustering / message planning
**Input:** release slice  
**Output:** theme graph or message plan  
**Canonical:** no  
**Persisted:** optional in v1, yes if AI is used  

Transformations:
- group related fragments into themes
- identify duplicates/overlaps
- choose likely user-facing versus internal-only themes
- identify candidate headlines
- identify “must never omit” items
- identify items that should remain technical appendix only

This is the first interpretive stage. It may be:
- deterministic/rule-based
- AI-assisted
- or hybrid

This is where information begins to compress:
- several raw fragments can become one theme
- low-level implementation detail may move to appendix-only status
- cross-cutting fixes can be merged into a single statement

Traceability requirement:
- every theme must keep a list of source fragment IDs
- theme-level claims should be marked as either “composed from” or “directly stated by” fragments

### Stage 4 — Narrative bundle generation
**Input:** release slice + theme graph  
**Output:** narrative bundle  
**Canonical:** no  
**Persisted:** yes, when review matters  

Transformations:
- write message units for audiences
- produce titles, intros, summaries, callouts
- define “engineer”, “support/product”, “end-user”, and “broadcast” variants
- mark confidence and review status
- record provenance per message unit

This is the highest-value new layer.

A good narrative bundle should contain:
- release metadata
- source release reference
- theme list
- message units
- audience sections
- required warnings / upgrade notes
- excluded items and rationale
- provenance map

This is also where AI can safely help the most, as long as:
- the bundle remains reviewable
- every claim is anchored to fragment IDs
- the system distinguishes exact fact vs abstraction

### Stage 5 — Channel render
**Input:** release slice and/or reviewed narrative bundle  
**Output:** channel-specific files or payloads  
**Canonical:** no  
**Persisted:** usually no, unless intentionally exported  

Transformations:
- length compression
- tone adaptation
- markdown formatting
- CTA insertion
- platform-specific field constraints
- localization variants
- message ordering

This stage is usually deterministic once the narrative bundle exists.

Examples:
- GitHub/GitLab release body: markdown sections
- Slack: concise lead + 2–4 bullets + CTA link
- App Store: ultra-short user-facing bullets, max 4000 chars, localizable
- email/blog: fuller story, screenshots, links, rollout context

### Where information is lost

Information loss is good only if it is controlled.

Loss should happen in the following places:

1. **Theme clustering**
   - multiple fragments collapse into one theme
2. **Audience adaptation**
   - internal implementation detail is omitted for user-facing layers
3. **Channel render**
   - summaries get shorter
   - CTAs replace detail
   - technical appendix may be removed entirely

Loss should **not** happen at:
- fragment storage
- manifest storage
- provenance links
- required warnings / breaking changes / operator actions

### Where interpretation is added

Interpretation is added at:
- theme naming
- “what this means for users” phrasing
- tone choice
- title/intro writing
- deciding which details belong in appendix vs main message
- identifying likely relevance for audiences

Interpretation must be explicit and reviewable.

### How traceability is preserved

Traceability should be preserved with three stable identifiers:

- `fragment_id`
- `theme_id`
- `message_unit_id`

Flow:
- fragments feed themes
- themes feed message units
- message units feed renders

Every final sentence should be traceable either:
- directly to fragments, or
- indirectly through a message unit that lists source fragments

A CLI trace command should be able to answer:
- “which fragments support this Slack bullet?”
- “which message unit did this App Store sentence come from?”
- “what got dropped between engineer view and end-user view?”

## 3. Data model and schema

## What remains canonical

### Fragment metadata that should remain canonical

Keep canonical:
- `fragment_id`
- `created_at`
- `authored_by`
- `title`
- `body`
- `type`
- `bump` / impact
- `scopes`
- `platforms`
- `components`
- `tickets` / links to work items
- `audience_hints` (optional but valuable)
- `visibility` (internal, external, restricted)
- `requires_action` (bool)
- `breaking_change` (bool)
- `security_relevant` (bool)
- `locales` or localization notes if applicable

Useful additions:
- `user_impact_summary` (short factual phrase, still canonical if carefully constrained)
- `related_fragment_ids`
- `release_notes_priority` (e.g. required, normal, appendix_only)
- `customer_visible` (bool)
- `support_relevance` (bool)

Be careful: anything that starts to sound like polished prose should remain derived, not canonical.

### Release manifest metadata that should remain canonical

Keep canonical:
- `version`
- `line`
- `parent_version`
- `released_at`
- `fragment_ids_added`
- `status` (draft/final/published if you track it)
- `release_metadata` that affects selection semantics only

Do **not** add:
- narrative titles
- end-user summaries
- Slack copy
- App Store copy
- big prose blobs

## Derived narrative artifacts

### Recommendation

Yes: add derived narrative artifacts as their own files.

But do **not** create many disconnected artifact types at first.

For v1, create **one persisted bundle per release version + line + revision** that can carry multiple audiences inside it.

Suggested path:

```text
.local/share/changes/narratives/<line>/<version>/bundle.v1.yaml
```

Optional localized variants later:

```text
.local/share/changes/narratives/<line>/<version>/bundle.en-US.v1.yaml
.local/share/changes/narratives/<line>/<version>/bundle.fr-FR.v1.yaml
```

Final channel renders should usually go to state, not share:

```text
.local/state/changes/rendered/<line>/<version>/github_release.md
.local/state/changes/rendered/<line>/<version>/slack_announcement.md
.local/state/changes/rendered/<line>/<version>/app_store.en-US.txt
```

That keeps:
- reviewable derived artifacts in durable storage
- channel outputs disposable by default

### Suggested narrative bundle schema

```yaml
schema_version: 1
artifact_kind: narrative_bundle
artifact_id: narr-stable-2.4.0-v1
status: draft   # draft | reviewed | approved | published
source_release:
  line: stable
  version: 2.4.0
  parent_version: 2.3.5
  manifest_ref: stable/2.4.0
source_fragments:
  - frg_01HZX...
  - frg_01HZY...
generation:
  strategy: hybrid
  model: gpt-5.4
  created_at: 2026-04-02T17:30:00Z
  prompt_fingerprint: sha256:...
review:
  reviewed_by: null
  reviewed_at: null
themes:
  - theme_id: theme_search
    label: Search and findability
    source_fragment_ids:
      - frg_search_latency
      - frg_search_typo_ranking
    interpretation_level: summary
    must_include_for:
      - end_user
      - support
  - theme_id: theme_admin
    label: Admin visibility
    source_fragment_ids:
      - frg_audit_export
    interpretation_level: direct
message_units:
  - unit_id: mu_001
    audience: end_user
    theme_id: theme_search
    kind: highlight
    text: Search is now much faster and more forgiving of small typos.
    source_fragment_ids:
      - frg_search_latency
      - frg_search_typo_ranking
    interpretation_level: summary
    review_required: true
  - unit_id: mu_002
    audience: support
    theme_id: theme_admin
    kind: capability
    text: Workspace admins can now export audit logs from Settings > Security.
    source_fragment_ids:
      - frg_audit_export
    interpretation_level: direct
    review_required: true
audience_sections:
  engineers:
    include_units: [mu_010, mu_011, mu_012]
  support_product:
    include_units: [mu_001, mu_002, mu_003]
  end_users:
    include_units: [mu_001, mu_004, mu_005]
channel_guidance:
  github_release:
    include_audience: support_product
  slack:
    include_audience: broadcast
    max_bullets: 3
  app_store:
    include_audience: end_users
    max_chars: 4000
```

This is not source of truth. It is a derived, traceable communication bundle.

## Example files

### Example low-level fragment

```markdown
---
schema_version: 1
fragment_id: frg_search_latency
created_at: 2026-04-01T13:24:11Z
authors:
  - tim
type: improvement
bump: patch
scopes: [search, ios]
platforms: [ios]
components: [search-service, mobile-client]
tickets: [IOS-1482]
visibility: external
customer_visible: true
support_relevance: true
requires_action: false
breaking_change: false
security_relevant: false
release_notes_priority: normal
title: Reduce search latency for recent conversations
---

Reduced p95 search latency for recent conversations from ~680ms to ~340ms by
caching tokenized query results and precomputing top workspace hits.
```

### Example release manifest

```yaml
schema_version: 1
kind: release_manifest
line: stable
version: 2.4.0
parent_version: 2.3.5
released_at: 2026-04-02T18:00:00Z
fragment_ids_added:
  - frg_search_latency
  - frg_search_typo_ranking
  - frg_passkey_signin
  - frg_audit_export
  - frg_invite_link_crash
  - frg_sync_battery
```

### Example derived narrative bundle

```yaml
schema_version: 1
artifact_kind: narrative_bundle
artifact_id: narr-stable-2.4.0-v1
status: reviewed
source_release:
  line: stable
  version: 2.4.0
  parent_version: 2.3.5
  manifest_ref: stable/2.4.0
source_fragments:
  - frg_search_latency
  - frg_search_typo_ranking
  - frg_passkey_signin
  - frg_audit_export
  - frg_invite_link_crash
  - frg_sync_battery
themes:
  - theme_id: theme_find_stuff
    label: Search and findability
    source_fragment_ids:
      - frg_search_latency
      - frg_search_typo_ranking
  - theme_id: theme_signin
    label: Faster, safer sign-in
    source_fragment_ids:
      - frg_passkey_signin
      - frg_invite_link_crash
  - theme_id: theme_admin
    label: Admin controls
    source_fragment_ids:
      - frg_audit_export
message_units:
  - unit_id: mu_001
    audience: end_users
    theme_id: theme_find_stuff
    text: Search is faster and now handles small typos more gracefully.
    source_fragment_ids:
      - frg_search_latency
      - frg_search_typo_ranking
    interpretation_level: summary
  - unit_id: mu_002
    audience: end_users
    theme_id: theme_signin
    text: Signing in is smoother with passkey support, and we fixed a crash when opening some shared invite links.
    source_fragment_ids:
      - frg_passkey_signin
      - frg_invite_link_crash
    interpretation_level: mixed
  - unit_id: mu_003
    audience: support_product
    theme_id: theme_admin
    text: Workspace admins can export audit logs from Settings > Security for compliance reviews.
    source_fragment_ids:
      - frg_audit_export
    interpretation_level: direct
```

### Example final output rendering — repository changelog

```markdown
# 2.4.0

## Features
- Added passkey sign-in support on iOS.
- Workspace admins can export audit logs from Settings > Security.

## Improvements
- Reduced p95 latency for recent-conversation search.
- Improved typo tolerance in search ranking.
- Reduced background sync battery use on low-power devices.

## Fixes
- Fixed a crash when opening some shared invite links.
```

This can be rendered directly from canonical release data or from the engineer section of the narrative bundle.

### Example final output rendering — GitHub/GitLab release body

```markdown
# 2.4.0

## Highlights
- Search is faster and more forgiving of small typos.
- Passkey sign-in is now available on iOS.
- Workspace admins can export audit logs for compliance reviews.

## Fixes and improvements
- Fixed a crash when opening some shared invite links.
- Reduced background sync battery usage on low-power devices.

## Notes for admins
- Audit log exports are available in **Settings > Security**.
```

### Example final output rendering — Slack announcement

```markdown
:rocket: **2.4.0 is live**

Highlights:
• Search is faster and better with small typos.
• Passkey sign-in is now available on iOS.
• Admins can export audit logs from Settings > Security.

Read the full release notes: https://example.com/releases/2.4.0
```

### Example final output rendering — App Store

```text
What’s New in 2.4.0

• Faster search with better typo handling
• Passkey sign-in support on iPhone
• Fix for a crash when opening some shared invite links
• Lower battery use during background sync
```

## 4. AI integration strategy

## Principle

AI should help with **interpretation and expression**, never with **selection truth**.

### Deterministic, non-AI stages

These should remain deterministic:

- fragment schema validation
- manifest creation
- lineage traversal
- release slice resolution
- basic grouping by metadata
- required-item extraction
- channel length measurement
- markdown/text rendering
- platform field validation
- trace lookup

These are about truth, not taste.

### AI-appropriate stages

AI can help safely at:

1. **Theme clustering**
   - identify which fragments belong together
   - suggest theme labels
2. **Audience translation**
   - rewrite technical fragment language into user language
3. **Layered summary generation**
   - produce engineer/product/user/broadcast variants
4. **Lead/title generation**
   - write short intros and headers
5. **Compression**
   - shorten reviewed narrative units to fit App Store, Slack, email subject constraints

### Safety rules for AI use

1. **AI never selects the release**
   - manifests already did that
2. **AI never invents new facts**
   - it may only restate, merge, or prioritize facts anchored to source fragments
3. **Every AI statement must carry source fragment IDs**
4. **Interpretive level must be explicit**
   - `direct`
   - `summary`
   - `abstraction`
   - `cta`
5. **Human review is required**
   - for external-facing prose
   - for anything marked `abstraction`
   - for anything involving security, billing, legal, compliance, or breaking-change messaging
6. **Prompt inputs should exclude irrelevant repo text**
   - only pass the release slice, not the entire repo
7. **Outputs should be structured first**
   - JSON/YAML message units before freeform markdown when possible

### Recommended AI workflow

#### AI stage A — cluster fragments into themes
Prompt goal:
- propose themes
- assign fragment IDs to themes
- identify must-include items
- identify appendix-only items

Expected output:
- strict JSON
- no prose publishing yet

Example prompt:

```text
You are helping build a release communication plan.

Input:
- A release slice containing fragment IDs, titles, metadata, and bodies.

Task:
1. Group fragments into 3-7 themes.
2. For each theme, provide:
   - theme_id
   - short label
   - source_fragment_ids
   - why these belong together
   - whether the theme matters to:
     engineers, support_product, end_users, broadcast
3. Mark any fragment that must never be omitted because it is:
   - breaking
   - requires action
   - security relevant
   - support critical

Rules:
- Do not invent facts.
- Use only information explicitly present in the fragments.
- If uncertain, keep fragments separate.
- Output JSON only.
```

#### AI stage B — generate audience message units
Prompt goal:
- create audience-specific summaries with provenance

Example prompt:

```text
You are generating derived release-note message units.

For each requested audience, write short statements that:
- are faithful to the source fragments
- explain why the change matters
- avoid internal jargon unless the audience is engineers

For every statement output:
- unit_id
- audience
- text
- source_fragment_ids
- interpretation_level: direct | summary | abstraction
- review_required: true | false

Rules:
- Do not mention functionality not present in the source fragments.
- If multiple fragments are merged into one statement, include all source IDs.
- Prefer understatement to overclaiming.
- Output JSON only.
```

#### AI stage C — compress for channel constraints
Prompt goal:
- compress already reviewed message units

Example prompt:

```text
You are compressing approved message units for a specific channel.

Channel: App Store
Constraints:
- maximum 4000 characters
- concise, user-facing
- no internal ticket references
- no promises about future behavior
- keep required fixes/actions if present

Input:
- reviewed message units only

Task:
- produce 3-6 bullets
- preserve factual accuracy
- drop detail before dropping required warnings
```

#### AI stage D — optional title/intro generation
Prompt goal:
- write the lead, not the facts

Example prompt:

```text
Write 5 candidate titles and 3 candidate intros for this reviewed narrative bundle.

Audience: support_product
Tone: clear, concise, human
Avoid:
- hype
- jokes that obscure meaning
- claims unsupported by message units
```

### Preserving factual correctness

Require the system to verify:

- every message unit references existing fragment IDs
- every channel render only uses approved units
- no render introduces claims absent from units
- length compression does not remove required warnings
- “must include” flags survive all downstream transforms

A simple but powerful pattern:
- AI proposes
- deterministic validators check structure
- human reviews wording
- renderers format

## 5. Audience layering model

The canonical release data should not be presented the same way to all audiences.

### A. Engineers / maintainers

Optimization target:
- accuracy
- completeness
- operational detail
- traceability

Preserve verbatim:
- technical nouns
- component names
- migration details
- platform qualifiers
- scope and type grouping

Can be abstracted:
- title polish
- ordering for readability

Never drop:
- breaking changes
- operational actions
- platform limitations
- security-relevant detail appropriate for the audience

Best source:
- canonical release slice, optionally lightly assisted by an engineer narrative layer

Persistence:
- not always necessary as a separate artifact in v1

### B. Product and support teams

Optimization target:
- “what changed”
- “why it matters”
- “how to explain it”
- “what customers may ask”

Preserve verbatim:
- user-visible capability names
- admin paths/settings locations
- known caveats
- rollout limits if relevant

Can be abstracted:
- implementation detail
- lower-level performance mechanics

Never drop:
- customer-facing behavior changes
- known limitations
- support-critical bug fixes
- required user/admin actions

Best source:
- reviewed narrative bundle

Persistence:
- yes, this should usually be reviewable

### C. End users

Optimization target:
- benefit
- clarity
- confidence
- brevity

Preserve verbatim:
- feature names users can see
- concrete actions users can take
- fixes users may have felt

Can be abstracted:
- architecture
- internal terminology
- exact technical mechanism

Never drop:
- anything that changes what the user must do
- anything that avoids confusion after upgrade
- major bug fixes users are likely to recognize

Best source:
- reviewed end-user section of the narrative bundle

Persistence:
- often yes if you support review/localization

### D. Broadcast channels (Slack, email teaser, social-ish team channels)

Optimization target:
- scannability
- urgency
- shareability
- click-through

Preserve verbatim:
- release version
- 2–4 key highlights
- CTA destination

Can be abstracted:
- almost everything else

Never drop:
- critical warnings if the channel is the primary alert surface
- link to fuller release notes

Best source:
- reviewed broadcast section derived from the narrative bundle

Persistence:
- optional; render on demand is often enough once message units are approved

### Recommended layering rule

Use this hierarchy:

- **Engineer layer**: closest to canonical data
- **Support/Product layer**: explanatory bridge
- **End-user layer**: benefits and behavioral impact
- **Broadcast layer**: teaser + pointer

This is the practical way to reconcile Slack-style personality with traceability. Slack’s public-facing narrative is not a replacement for precise change data; it is a human layer on top of it.

## 6. Channel-specific output design

## Slack announcement format

The Slack GitHub Action example is instructive because it treats the release body as upstream content, then passes it through a workflow that lets someone choose the destination channel and posts the announcement. That implies Slack is best treated as a short-form downstream publication channel, not the place where you compose the master release story.

Recommended Slack shape:

- one-line headline with version
- 2–4 bullets
- one CTA link to fuller notes
- optional emoji, but only if it does not obscure meaning

Good for:
- internal announcements
- customer community channels
- support-team awareness
- launch-room coordination

Bad for:
- exhaustive change detail
- long technical appendix
- anything needing screenshots unless you add richer Slack blocks later

Generate from:
- **broadcast** message units, or
- a shortened support/product section

Do not generate Slack directly from raw fragments except for engineering-only channels.

## GitHub/GitLab release body

GitHub releases are tied to tags and can have release notes and assets. GitHub also offers generated release notes, but those generated notes are explicitly not saved anywhere until used. GitLab requires a release description, recommends including a changelog, and supports Markdown. That makes both platforms strong publication surfaces but weak canonical stores.

Recommended shape:

- title / version
- highlights
- fixes and improvements
- upgrade/admin notes
- assets / links
- full technical appendix only if the audience expects it

Generate from:
- support/product layer for public repos
- engineer or hybrid layer for developer tools

Include:
- markdown headings
- bullets
- links to docs/issues when helpful

Avoid:
- raw fragment dump unless the repo is explicitly developer-first
- channel-specific humor that ages badly

## App Store release notes

Apple’s “What’s New in this Version” field is localized, required after the first version, and limited to 4000 characters. That means you need a deliberately compressed, end-user-focused output, not a direct render of your repository release body.

Recommended shape:

- 3–6 bullets
- benefits first
- user-recognizable language
- no internal project names
- no ticket numbers
- no speculative future promises

Generate from:
- reviewed **end-user** message units
- optionally locale-specific bundles later

Prioritize:
1. visible features
2. user-felt fixes
3. trust-building quality improvements

Drop first:
- internal architecture detail
- admin-only features for consumer apps
- implementation mechanics

## Email / blog-style release notes

This surface has the most room.

Recommended shape:
- headline
- short intro / why this release matters
- grouped sections by theme
- screenshots/GIFs if useful
- CTA links
- optional customer-facing explanations or examples

This is where Candu’s “multi-channel” lesson matters most: the fuller archive/blog/help-center page should be the durable detailed narrative surface, while Slack/email/in-product nudges point into it.

Generate from:
- reviewed support/product or end-user narrative bundle
- optionally merge with docs links and media

## In-product “What’s New” / help center / proactive messages

Candu’s article is especially valuable here. The core lesson is not “pick one release notes location.” It is “use layered distribution”:

- persistent archive / history
- contextual prompts/tooltips
- proactive notifications (email/pop-up)
- segmentation where appropriate
- CTAs that let users try the feature now

That means your architecture should support:
- one fuller canonical-to-narrative page
- many thin announcement surfaces derived from it

## 7. Tool architecture proposal

## Design principle

Add a **narrative subsystem**, not a new selection system.

### Recommended command surface

#### Existing conceptual areas
- fragment authoring
- release selection / manifest creation
- rendering

#### New area
- narrative planning and review

### Proposed commands

```text
changes release resolve <version> [--line stable|preview]
```
Deterministically resolve the release slice.

```text
changes narrative plan <version> [--line stable|preview] [--out ...]
```
Create or update a narrative bundle from the resolved release slice.

Options:
- `--strategy deterministic|ai|hybrid`
- `--audiences engineers,support_product,end_users,broadcast`
- `--persist`
- `--revision v2`

```text
changes narrative review <artifact-ref>
```
Open/print a reviewable narrative bundle and allow state changes:
- draft -> reviewed
- reviewed -> approved

```text
changes narrative trace <artifact-ref> [--unit mu_001]
```
Show provenance for themes and message units.

```text
changes render <version-or-artifact> --pack github_release
changes render <version-or-artifact> --pack slack_announcement
changes render <version-or-artifact> --pack app_store --locale en-US
```

Render from:
- canonical release slice for technical packs
- narrative bundle for audience/channel packs

```text
changes export <version-or-artifact> --target github
changes export <version-or-artifact> --target gitlab
changes export <version-or-artifact> --target slack
changes export <version-or-artifact> --target appstore
```

These should be adapters, not canonical storage.

### File layout proposal

```text
.config/changes/config.toml
.local/share/changes/fragments/
.local/share/changes/releases/
.local/share/changes/templates/
.local/share/changes/narratives/
  stable/
    2.4.0/
      bundle.v1.yaml
      bundle.v2.yaml
  preview/
    2.4.0-rc.2/
      bundle.v1.yaml
.local/state/changes/
  resolved/
    stable-2.4.0.json
  rendered/
    stable/
      2.4.0/
        repository_markdown.md
        github_release.md
        slack_announcement.md
        app_store.en-US.txt
```

### Keep selection separate from rendering

The boundary should be:

- `release` commands own canonical selection
- `narrative` commands own derived interpretive artifacts
- `render` commands own final presentation
- `export` commands own integration with external systems

That keeps your core philosophy intact.

### Go library shape

The Go library should expose packages roughly like:

```text
changes/fragment
changes/release
changes/lineage
changes/resolve
changes/narrative
changes/render
changes/export
changes/trace
```

Where:
- `resolve` is deterministic
- `narrative` can host AI hooks behind interfaces
- `render` is pure transformation
- `export` is platform-specific IO

Suggested interfaces:

```go
type ReleaseResolver interface {
    Resolve(ref ReleaseRef) (ReleaseSlice, error)
}

type NarrativePlanner interface {
    Plan(ctx context.Context, slice ReleaseSlice, opts NarrativePlanOptions) (NarrativeBundle, error)
}

type Renderer interface {
    Render(input RenderInput, pack string, opts RenderOptions) ([]byte, error)
}

type Tracer interface {
    TraceUnit(bundle NarrativeBundle, unitID string) (TraceResult, error)
}
```

This keeps the system usable as:
- a CLI
- a library
- a CI/CD step
- a future service if you ever want one

## 8. Tradeoffs and failure modes

### Risk: over-automation turns nuanced changes into mush
Problem:
- too much compression
- vague statements like “performance improvements”
- loss of specificity

Mitigation:
- require theme/source traceability
- keep `must_include` and `appendix_only` flags
- compare rendered outputs against source fragment counts
- lint against banned generic phrases unless explicitly justified

### Risk: AI hallucinates or overclaims
Problem:
- invents feature scope
- implies impact not in evidence
- upgrades “fixed some paths” to “login is rock solid”

Mitigation:
- structured outputs first
- strict fragment-ID anchoring
- interpretation-level flags
- mandatory review for abstractions
- deterministic validators that reject orphan claims

### Risk: too many intermediate artifacts
Problem:
- operational complexity
- review fatigue
- repo clutter

Mitigation:
- persist exactly one derived artifact type in v1: `narrative_bundle`
- keep resolved slices and final renders in state unless explicitly exported
- version bundles only when materially revised

### Risk: too few reviewable layers
Problem:
- AI or template transforms become opaque
- no place for support/product review

Mitigation:
- make the narrative bundle the review checkpoint
- require approval before public channel renders in workflows that matter

### Risk: canonical and derived layers get confused
Problem:
- teams start editing narrative bundles as if they were truth
- fragments go stale because people only care about the derived copy

Mitigation:
- explicit metadata: `artifact_kind`, `status`, `source_release`
- CLI UI that clearly labels canonical vs derived
- trace command that always points back to fragments
- documentation/ADR language: “bundles explain releases; fragments and manifests define them”

### Risk: render targets become too channel-specific too early
Problem:
- you end up baking App Store rules into selection logic
- Slack copy decisions pollute fragment authoring

Mitigation:
- preserve channel logic in render packs and export adapters
- use audience sections as the bridge, not channel-specific fields in fragments

### Risk: preview/stable lines get narratively tangled
Problem:
- preview copy leaks into stable or vice versa
- repeated highlights across RCs become confusing

Mitigation:
- bind every narrative bundle to a `line + version + parent_version`
- allow preview narrative bundles to inherit reviewed message units but require explicit re-selection for stable publication
- store “first introduced in preview line” as derived metadata if helpful, not as canonical selection semantics

## 9. Example end-to-end flow

## Step 1 — Raw fragments

Assume these fragments exist:

1. `frg_search_latency`
   - reduced p95 recent-conversation search latency from ~680ms to ~340ms
2. `frg_search_typo_ranking`
   - improved ranking for one-character typos in conversation names
3. `frg_passkey_signin`
   - added passkey sign-in on iOS 18+
4. `frg_invite_link_crash`
   - fixed crash opening some shared invite links
5. `frg_audit_export`
   - admins can export audit logs from Settings > Security
6. `frg_sync_battery`
   - reduced battery use during background sync on low-power devices
7. `frg_copy_empty_state`
   - updated empty-state copy on search results screen
8. `frg_refactor_indexer`
   - refactored token indexer pipeline; no intended user-visible change

## Step 2 — Parent-linked manifests

Preview line:

```yaml
line: preview
version: 2.4.0-rc.1
parent_version: 2.3.5
fragment_ids_added:
  - frg_search_latency
  - frg_search_typo_ranking
  - frg_passkey_signin
  - frg_invite_link_crash
```

```yaml
line: preview
version: 2.4.0-rc.2
parent_version: 2.4.0-rc.1
fragment_ids_added:
  - frg_sync_battery
```

Stable line:

```yaml
line: stable
version: 2.4.0
parent_version: 2.3.5
fragment_ids_added:
  - frg_search_latency
  - frg_search_typo_ranking
  - frg_passkey_signin
  - frg_invite_link_crash
  - frg_audit_export
  - frg_sync_battery
  - frg_copy_empty_state
```

Note:
- preview and stable are separate lines
- the stable release explicitly selects what becomes stable
- no fragments are consumed or deleted

## Step 3 — Deterministic grouping/clustering seed

Rule-based seed groups:

- `search`
  - frg_search_latency
  - frg_search_typo_ranking
  - frg_copy_empty_state
- `signin`
  - frg_passkey_signin
  - frg_invite_link_crash
- `admin`
  - frg_audit_export
- `reliability`
  - frg_sync_battery
- `internal_only`
  - frg_refactor_indexer (not in stable manifest anyway)

## Step 4 — Derived narrative bundle

Reviewed bundle excerpt:

```yaml
themes:
  - theme_id: theme_search
    label: Search feels faster and easier to use
    source_fragment_ids:
      - frg_search_latency
      - frg_search_typo_ranking
      - frg_copy_empty_state
  - theme_id: theme_signin
    label: Sign-in and invite reliability
    source_fragment_ids:
      - frg_passkey_signin
      - frg_invite_link_crash
  - theme_id: theme_admin
    label: Admin visibility and compliance
    source_fragment_ids:
      - frg_audit_export
message_units:
  - unit_id: mu_user_search
    audience: end_users
    text: Search is now faster, more forgiving of small typos, and clearer when no results are found.
    source_fragment_ids:
      - frg_search_latency
      - frg_search_typo_ranking
      - frg_copy_empty_state
    interpretation_level: summary
  - unit_id: mu_user_signin
    audience: end_users
    text: You can now sign in with a passkey on supported iPhones, and we fixed a crash affecting some shared invite links.
    source_fragment_ids:
      - frg_passkey_signin
      - frg_invite_link_crash
    interpretation_level: mixed
  - unit_id: mu_support_admin
    audience: support_product
    text: Workspace admins can export audit logs from Settings > Security for compliance and review workflows.
    source_fragment_ids:
      - frg_audit_export
    interpretation_level: direct
  - unit_id: mu_user_reliability
    audience: end_users
    text: Background sync uses less battery on low-power devices.
    source_fragment_ids:
      - frg_sync_battery
    interpretation_level: summary
```

## Step 5 — Final outputs

### Repository changelog

```markdown
# 2.4.0

## Features
- Added passkey sign-in on supported iPhones.
- Added audit log export for workspace admins.

## Improvements
- Reduced recent-conversation search latency.
- Improved typo handling in search results.
- Updated search empty-state copy.
- Reduced battery use during background sync on low-power devices.

## Fixes
- Fixed a crash affecting some shared invite links.
```

Traceability:
- mostly direct
- little interpretation
- almost one-to-one with fragments

### GitHub release body

```markdown
# 2.4.0

## Highlights
- Search is faster and handles small typos better.
- Passkey sign-in is now available on supported iPhones.
- Workspace admins can export audit logs from Settings > Security.

## Fixes and reliability
- Fixed a crash affecting some shared invite links.
- Reduced battery use during background sync on low-power devices.

## UI polish
- Improved the search empty state when no results are found.
```

Traceability:
- “Search is faster and handles small typos better.” <= `mu_user_search` <= fragments latency + typo
- “Passkey sign-in is now available...” <= `mu_user_signin` <= fragment passkey
- “Workspace admins can export audit logs...” <= `mu_support_admin` <= fragment audit export

Compression:
- combines 2–3 fragments into one highlight
- removes raw latency numbers from the main body

### Slack announcement

```markdown
:rocket: **2.4.0 is live**

Highlights:
• Faster search with better typo handling
• Passkey sign-in on supported iPhones
• Audit log export for workspace admins

Full notes: https://example.com/releases/2.4.0
```

Traceability:
- each bullet maps to one approved message unit
- crash fix and battery improvement were dropped here because of length
- that is acceptable because Slack is teaser-format, not canonical detail

### App Store summary

```text
What’s New in 2.4.0

• Faster search with better typo handling
• Passkey sign-in on supported iPhones
• Fix for a crash affecting some shared invite links
• Lower battery use during background sync
```

Traceability:
- derived from reviewed end-user units
- admin audit export omitted because this hypothetical app store audience is broad end users
- that omission is acceptable if the feature is not broadly relevant to consumer readers

## Sentence-level traceability example

| Final sentence | Source fragments | Compression | Interpretation |
|---|---|---|---|
| Search is faster and handles small typos better. | `frg_search_latency`, `frg_search_typo_ranking` | 2 fragments -> 1 statement | summary |
| Passkey sign-in is now available on supported iPhones. | `frg_passkey_signin` | none | direct |
| Fixed a crash affecting some shared invite links. | `frg_invite_link_crash` | none | direct |
| Lower battery use during background sync. | `frg_sync_battery` | technical details removed | summary |

This is the behavior you want:
- factual origin is preserved
- compression is visible
- interpretation is classified
- omission is deliberate

## What to adopt, reject, and defer

### Adopt now

- one new persisted artifact type: `narrative_bundle`
- deterministic `release resolve`
- traceable message units
- audience layers: engineer, support/product, end-user, broadcast
- render packs that can consume either release slices or narrative bundles
- AI only in interpretive stages, with structured outputs and review

### Reject

- putting polished prose in manifests
- making platform release bodies canonical
- deleting fragments after release
- scraping commit messages as truth
- AI-generated publication with no provenance or review
- channel-specific selection logic

### Future work

- localization bundles
- multi-repo aggregation
- screenshot/media attachment planning
- quality metrics (readability, click-through, support deflection, adoption)
- reviewer workflows integrated into CI/CD
- reusable “house voice” prompt packs per repo or org
- richer Slack Block Kit and in-product payload exports

## Recommended architecture

The recommended architecture is:

- **Canonical**
  - fragments
  - release manifests
- **Computed**
  - release slice
- **Derived**
  - narrative bundle with themes + message units + provenance
- **Rendered**
  - channel-specific outputs
- **Exported**
  - GitHub/GitLab/Slack/App Store publication adapters

The most important rule is:

> **Selection determines truth. Narrative explains truth. Rendering formats explanation. Export publishes rendering.**

Keep those responsibilities separate and `changes` can grow from a fragment-and-manifest tool into a real release communication system without sacrificing its core philosophy.

## Staged rollout plan

### Phase 1 — Deterministic foundation
Implement:
- `release resolve`
- release slice object
- trace command for manifest -> fragments
- richer fragment metadata (`customer_visible`, `support_relevance`, `requires_action`, `breaking_change`, `release_notes_priority`)

Goal:
- make the current system better without any AI dependency

### Phase 2 — Narrative bundle v1
Implement:
- `narrative plan`
- persisted `bundle.v1.yaml`
- deterministic theme seeding
- human-edited message units
- render packs for:
  - repository_markdown
  - github_release
  - slack_announcement
  - app_store

Goal:
- introduce the reviewable interpretive layer

### Phase 3 — AI-assisted planning
Implement:
- structured AI theme clustering
- audience-unit generation
- provenance validation
- review states and approval gating

Goal:
- speed up narrative creation while preserving correctness

### Phase 4 — Channel adapters and workflow integration
Implement:
- GitHub/GitLab export adapters
- Slack export adapter
- App Store text export
- CI jobs that fail if required warnings/actions are omitted
- optional reviewer approval checks before publication

Goal:
- operationalize publication without making platforms canonical

### Phase 5 — Advanced communication system
Implement:
- localization
- multi-repo rollups
- segmented in-product outputs
- metrics on engagement and note quality
- reusable org-level style packs

Goal:
- turn `changes` into a full release communication pipeline, not just a changelog renderer

## Source notes

Required source material used:
- Anna Pickard / Slack archived article (attempted fetch; partially reconstructed from later Slack retrospective and corroborating references)
- Released: “How to Create Release Notes Like Slack”
- Candu: “How to Write Release Notes: Best Practices, Examples & Templates”
- Slack Developer Docs: post release announcements workflow

Supplemental official docs used for channel constraints:
- GitHub releases docs
- GitLab release description docs
- Apple App Store Connect “What’s New in this Version” reference
