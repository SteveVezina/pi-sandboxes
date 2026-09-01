# PROP-006: Add Local Template Library and Lifecycle

## Status
✅ Applied to block spec (2026-08-31)

## Block Spec Reference
`SPEC.md` §6 Features, §7 Acceptance Criteria, §8 Security Model, §18 Templates, §19 CLI requirements, §20 Daemon API, §25 Snapshot and rollback, §34 Configuration file, §36 Documentation requirements, §39 First coding-agent task list

## Revision Note (2026-08-31)

This PROP was authored 2026-07-15, before PROP-007, PROP-008, and PROP-009 were applied. Reconciled with the current block spec:

- **AC number:** the proposed criterion is now **AC-35** (AC-31–34 were taken by PROP-009: Agent Run, Egress Proxy, Single Output Channel, Host FS Decoupling). Cascade updated.
- **PROP-007 (image resolution):** the template `base` field resolves to an OCI image name per PROP-007; the `base: { image, digest }` block below is the resolved form, not a shorthand.
- **PROP-008 / ADR-005 (runtime driver contract):** template `compatibility.runtimes` is a *declared* compatibility hint. The authoritative runtime capability comes from each driver's `Probe` → `CapabilityReport`. The daemon reconciles the two and reports mismatches at validate time.
- **PROP-009 (sandbox rename + host decoupling):** "sandbox" terminology throughout; template snapshot/import/export must not reintroduce host-disk coupling — snapshots are daemon-managed assets under the Pi Box home, captured from container/rootfs state, never host bind mounts.
- **ADR-006 (egress enforcement):** template `network.domains` only *seeds* a new sandbox's `network.allow`; per-sandbox `network.mode` + the daemon egress proxy remain authoritative.

## Problem

The current template model is intentionally local and static. `SPEC.md` defines seven built-in templates and basic commands for list, inspect, build, update, and prune. The GUI currently reflects that shape: template cards show only a name, a short description, and a default marker.

That is enough for the first local baseline, but it leaves several product and operational gaps:

- Templates have too little metadata for users to understand what they contain, when they were last changed, which runtime modes they support, or what security/network defaults they request.
- There is no specified way to fork a built-in template into a local editable template.
- There is no specified way to snapshot a configured sandbox or template build state into a reusable local template.
- There is no specified template history, version, diff, rollback, or promotion flow.
- There is no specified import/export format for moving templates between local Pi nodes without introducing a central registry.
- `pi-box template update` is underspecified because local template lifecycle states are not defined.
- GUI template management cannot evolve beyond static cards without inventing API shapes outside the block spec.

Because template authoring, snapshots, import/export, metadata, and trust boundaries are public behavior across CLI, daemon API, GUI, and future remote contexts, this should be a block-spec amendment rather than a code-only enhancement.

## Requested By

2026-07-15: User observed that the GUI Templates view is too static and rigid, lacks detail, and needs local management, fork, and snapshot workflows. The user clarified that this should be local-node template management, not a central Docker-style registry.

## Proposed Amendment

Add a later feature for a daemon-managed local template library and richer template lifecycle. The existing F5 Template System remains the local static baseline. The new feature extends it with local authoring, fork, snapshot, history, import/export, and GUI management workflows.

### Feature addition

Add the following feature after F27.

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F28 | Local Template Library and Lifecycle | Manage templates as local daemon-owned assets: inspect, fork, edit metadata, validate, snapshot from sandboxes, version locally, diff, rollback, import, export, and promote templates while preserving local-first trust boundaries | M8 |

The feature spec (`docs/features/F28-*.md`) should phase the work: (1) richer metadata schema + `inspect` + `fork` + `validate`; (2) `snapshot` + `history` + `diff` + `rollback` + `promote`; (3) `import` / `export` bundles; (4) GUI management surface. AC-35 items map onto these phases.

### Local library model

`pi-sandboxd` owns the local template library under the Pi Box home. This is not a central package registry and does not require a hosted service.

Required local template sources:

- `builtin`: templates shipped with the product and refreshed by product updates.
- `local`: user-created or forked templates stored on the local node.
- `snapshot`: templates captured from a sandbox or built template state.
- `imported`: templates imported from a portable bundle created by another Pi node.

The daemon is the authority for mutating the local library. CLI, GUI, and SDK clients should use daemon/API operations for normal template lifecycle changes.

### Template metadata model

Extend the template schema to support richer inspectable metadata. The schema should remain YAML on disk and continue to support the existing minimal fields.

Example:

```yaml
name: node-python
version: 1.2.0
description: Node.js and Python toolchain for full-stack agent work
summary: Node.js 22, pnpm, Python 3.13, uv, pip, git, ripgrep, jq
source:
  type: builtin            # builtin | local | snapshot | imported
  parent: base
  forkedFrom: ""
  snapshotOf: ""           # sandbox ID when type == snapshot
compatibility:             # DECLARED hint; driver Probe/CapabilityReport is authoritative (ADR-005)
  piBox: ">=0.1.0"
  runtimes:
    fast: supported
    compat: supported
    secure: supported
    microvm: planned
base:                      # resolved OCI image per PROP-007
  image: docker.io/library/debian:bookworm-slim
  digest: sha256:...
tools:
  - name: node
    version: "22"
  - name: pnpm
    version: "10"
  - name: python
    version: "3.13"
  - name: uv
    version: "pinned"
mounts:
  workspace: /workspace
  artifacts: /artifacts
caches:
  npm: /cache/npm
  pnpm: /cache/pnpm
  uv: /cache/uv
  pip: /cache/pip
network:                   # seeds a new sandbox's network.mode / network.allow; daemon egress proxy stays authoritative (ADR-006)
  default: restricted
  domains:
    - registry.npmjs.org
    - pypi.org
    - files.pythonhosted.org
resources:
  cpu: 2
  memory: 4Gi
  disk: 20Gi
lineage:
  generation: 1
  parentDigest: sha256:...
  contentDigest: sha256:...
metadata:
  license: internal
  maintainers:
    - name: local
  tags:
    - node
    - python
    - fullstack
  createdAt: "2026-07-15T00:00:00Z"
  updatedAt: "2026-07-15T00:00:00Z"
```

Required additions:

- Stable `name` and optional semantic `version`.
- Human-readable `summary`, `description`, `tags`, and maintainers.
- Runtime compatibility by mode.
- Source metadata (`builtin`, `local`, `snapshot`, or `imported`).
- Lineage metadata for fork/snapshot relationships and content digests.
- Tool names and pinned versions, while retaining compatibility with the current string list form.
- Network domains requested by the template, still subject to daemon policy.
- Resource defaults requested by the template, still subject to daemon policy.
- Timestamps and local revision history for user-visible changes.

### Local lifecycle operations

The local library must support these lifecycle actions:

- **Inspect:** show full template metadata, tools, caches, network domains, resources, source, lineage, and runtime compatibility.
- **Fork:** create an editable local template from a built-in, imported, snapshot, or local template.
- **Edit metadata:** update description, tags, resource defaults, network requests, and compatibility notes through validated daemon operations.
- **Validate:** check schema, pinned versions, cache mounts, network requests, runtime compatibility, and policy-relevant fields.
- **Snapshot:** capture a sandbox or built template state as a new local template.
- **Diff:** compare two local templates or two revisions of the same template.
- **History:** show local revisions and lineage.
- **Rollback:** restore a local template to a previous local revision.
- **Promote:** mark a local or snapshot template as the default or as a named stable template.
- **Export:** write a portable template bundle containing metadata, definition, optional build artifacts, content digest, and provenance.
- **Import:** load a portable template bundle into the local library after validation and policy checks.
- **Prune:** remove unused local revisions, snapshots, and build artifacts without deleting built-ins.

Portable bundles are for local/team transfer and backup. They are not a central marketplace or Docker-style registry.

### Template snapshot behavior

Snapshotting a template from a sandbox must be explicit. The operation must:

- Record the source sandbox ID, template name, runtime mode, and creation time.
- Capture only template/build state needed for future sandboxes, not arbitrary workspace source files by default.
- Exclude secrets, SSH keys, cloud credentials, bearer tokens, daemon config, logs containing secrets, and host-only paths.
- Produce a local template that can be inspected, validated, forked, exported, or used for new sandbox creation.
- Respect runtime-specific limitations. Fast mode may only snapshot metadata and cache/toolchain assumptions; compat and microVM modes may support stronger image/rootfs snapshots.

If a runtime cannot safely snapshot executable template state, the daemon must report that limitation and offer metadata-only fork/export behavior.

### CLI additions

Extend `pi-box template` with local lifecycle commands:

```bash
pi-box template inspect <name> [--json]
pi-box template fork <source> <new-name> [--description <text>]
pi-box template snapshot <sandbox-id> <new-name> [--metadata-only] [--include-artifacts=false]
pi-box template validate <path-or-name> [--strict] [--json]
pi-box template diff <left> <right> [--json]
pi-box template history <name> [--json]
pi-box template rollback <name> <revision>
pi-box template promote <name> [--default]
pi-box template export <name> --output <path>
pi-box template import <path> [--name <new-name>]
pi-box template prune [--snapshots] [--revisions] [--artifacts]
```

Existing commands keep their meaning:

- `template list` lists installed local templates and may show source/version/default columns.
- `template inspect <name>` shows full local metadata, lineage, digest, compatibility, network domains, caches, and resource defaults.
- `template build <name>` continues to build local artifacts for compatible runtimes.
- `template update <name>` is local-only unless a future proposal specifies remote update sources. For F28 it may refresh built-ins from product-managed definitions or update derived metadata.
- `template prune` removes unused local template artifacts, revisions, and snapshots, not built-in definitions.

### Daemon API additions

Add daemon endpoints or equivalent SDK operations before GUI implementation depends on them:

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/v1/templates` | List local templates with summary metadata |
| `GET` | `/v1/templates/{name}` | Inspect local template details |
| `POST` | `/v1/templates/fork` | Fork an existing local template into a new local template |
| `POST` | `/v1/templates/snapshot` | Snapshot a sandbox or built template state into a new local template |
| `POST` | `/v1/templates/validate` | Validate a local template definition or import bundle |
| `GET` | `/v1/templates/{name}/history` | Show local revision history and lineage |
| `POST` | `/v1/templates/diff` | Compare templates or revisions |
| `POST` | `/v1/templates/{name}/rollback` | Roll back a local template to a previous revision |
| `POST` | `/v1/templates/{name}/promote` | Promote a local template or mark it as default |
| `POST` | `/v1/templates/export` | Export a local template bundle |
| `POST` | `/v1/templates/import` | Import a template bundle after validation |

All mutating operations must be policy-checked, auditable, and explicit. Importing, snapshotting, or promoting a template must not bypass existing network, filesystem, runtime, or secret constraints.

### GUI additions

Extend the Templates view from static cards into a local management surface:

- Installed templates list with name, version, source, default marker, runtime compatibility, tags, and last updated time.
- Template detail view showing tools, versions, caches, mounts, network domains, resource defaults, source, lineage, content digest, and supported runtimes.
- Fork action for built-in, imported, snapshot, and local templates.
- Snapshot action from an existing sandbox when the runtime supports it.
- Diff and history views for local templates.
- Import/export actions for portable template bundles.
- Promote/set-default action.
- Validation results and policy warnings when a template requests broader network, resources, or runtime support than the daemon allows.

The GUI remains a daemon client. It must not implement its own template mutation logic outside daemon/API contracts.

### Security and trust requirements

- Built-in templates remain trusted local definitions shipped with the product.
- Imported template bundles are untrusted until validated and explicitly installed.
- Snapshot templates must exclude secrets and workspace source files by default.
- Template import/export must include content digests so users can compare and audit bundles.
- Template mutations must be explicit and auditable.
- Templates may request network domains, resource defaults, mounts, and runtime compatibility, but daemon policy remains authoritative.
- No template lifecycle operation may persist credentials inside template YAML, sandbox workspaces, exported bundles, or artifacts.
- Template snapshot operations must respect runtime limits and fail closed when a runtime cannot safely capture template state.

### Non-goals

The local template library feature does not add:

- A central hosted registry or marketplace.
- Docker Hub-style search, pull, or push semantics.
- Automatic remote template installation during sandbox creation.
- Template execution hooks that run implicitly at sandbox create time.
- Registry credentials or remote publishing auth.
- Bypassing daemon policy for template-requested domains, mounts, resources, or runtime modes.

## Acceptance Criteria Additions

Add acceptance criteria after AC-34.

### AC-35: Local Template Library and Lifecycle Works (F28)
- [ ] Template metadata includes version, summary, description, source, lineage, compatibility, tool versions, cache mounts, network domains, resource defaults, digest, and timestamps
- [ ] Existing minimal built-in templates remain valid template definitions
- [ ] `pi-box template inspect <name>` shows detailed metadata, lineage, compatibility, and policy-relevant settings
- [ ] `pi-box template fork <source> <new-name>` creates an editable local template derived from an existing template
- [ ] `pi-box template snapshot <sandbox-id> <new-name>` creates a local template from sandbox/template state when the runtime supports safe snapshotting
- [ ] `pi-box template validate <path-or-name>` validates schema, pinned versions, cache mounts, network requests, runtime compatibility, and policy-relevant fields
- [ ] `pi-box template diff` and `history` show local template changes and lineage
- [ ] `pi-box template rollback` restores a local template to a prior revision
- [ ] `pi-box template export` creates a portable bundle with metadata, definition, digest, and provenance
- [ ] `pi-box template import` validates and installs a portable local bundle
- [ ] Snapshot, import, fork, rollback, promote, and export operations are audited and policy-checked
- [ ] GUI Templates view can show installed template details, source/lineage, validation errors, history/diff, fork, snapshot, import/export, and promote actions
- [ ] Template snapshots and exports exclude secrets and workspace source files by default

## Rationale

Templates are the main user-facing abstraction for repeatable agent environments. Keeping them static makes the system easy to bootstrap, but it prevents users from evolving known-good local environments into reusable templates.

A local template library gives users the lifecycle they need without turning Pi Sandbox into a package registry product. The daemon remains the local authority, the GUI gets useful management affordances, and templates can still move between nodes through explicit import/export bundles.

## Impact

Features affected (canonical `SPEC.md` §6 IDs):

- F1 CLI Entry Point: new `template` subcommands and JSON output shapes.
- F2 Daemon API: new local template fork/snapshot/validate/history/diff/rollback/promote/import/export endpoints.
- F5 Template System: schema expansion, validation, provenance, local revision tracking, and local lifecycle operations.
- F11 Secrets & Network Model: snapshot/import/export must exclude secrets and respect network policy.
- F12 Cache Model: template metadata describes cache mounts and cache compatibility.
- F13 Snapshot & Rollback: template snapshot and rollback behavior overlaps with sandbox snapshot primitives.
- F15 Compat Backend: local template snapshots may reference built artifacts or images.
- F17 Policy Enforcement: template-requested domains, mounts, resources, snapshots, imports, and exports require policy checks.
- F20 MicroVM Backend: future template snapshots need digest/version provenance.
- F24 Cross-Platform GUI Workbench: Templates view becomes a richer management surface.
- F27 GUI Settings and Diagnostics: default template and diagnostics should account for local source, lineage, and validation state.

ADRs likely required:

- Local template snapshot boundaries and runtime-specific behavior.
- Template schema versioning and compatibility policy.
- Import/export bundle format and secret-exclusion guarantees.

Implementation blocked:

- Yes for local template fork/snapshot/import/export/history/diff/rollback/promote behavior and richer GUI template management, because the block spec currently only specifies static local templates.
- No for maintaining the existing seven built-in templates and current local list/inspect/build/update/prune behavior.

## Assumption While Awaiting Acceptance

Until accepted, implementation should keep templates local-first and static. The GUI may improve visual presentation of existing local template fields, but it should not invent template fork, snapshot, history, import/export, or promotion API contracts.

## Cascade Required on Acceptance

When this PROP is accepted:

1. Update `SPEC.md` to add F28 and AC-35 (after AC-34).
2. Update `SPEC.md` §18 Templates with the richer metadata schema, local library model, lifecycle operations, snapshot behavior, security constraints, and GUI expectations.
3. Update `SPEC.md` §19 CLI requirements with local lifecycle `pi-box template` commands.
4. Update `SPEC.md` §20 Daemon API with template fork/snapshot/validate/history/diff/rollback/promote/import/export endpoints or equivalent SDK operations.
5. Create `docs/features/F28-local-template-library.md`.
6. Update F5 Template System to mark local lifecycle operations as delegated to F28 rather than generic future work.
7. Update F13, F24, and F27 feature specs with template snapshot and GUI template-management behavior.
8. Add ADRs for local template snapshot boundaries, schema versioning, and import/export bundle format.
9. Update `docs/contracts/pi-sandboxd.md` with the new API shapes once specified.
10. Update `docs/features/INDEX.md` and `docs/proposals/INDEX.md`.

## Implementation Blocked?

No longer blocked — accepted and cascaded 2026-08-31.

## Cascade completed

**Accepted by human 2026-08-31.**

- `SPEC.md`: §6 F28 row; §7 AC-35 block; §18.1 new "Local template library (F28)" subsection; §19 CLI local-lifecycle commands (`fork|snapshot|validate|diff|history|rollback|promote|export|import`) + §9 CLI summary line; §9 Daemon API `/v1/templates*` endpoint rows.
- Feature specs created: `docs/features/F28-local-template-library.md` (Status 🟡 Spec written; T28.1–T28.4 phased tasks).
- Feature specs cascaded: F5 (Out of Scope → local lifecycle delegated to F28), F13 (Out of Scope → template-from-sandbox snapshot is F28), F24 (Out of Scope → rich template management is F28), F27 (Out of Scope → template-picker lineage is F28).
- INDEX: `docs/features/INDEX.md` F28 row + M8 milestone + summary; `docs/proposals/INDEX.md` status → ✅ Applied.
- `docs/plan.md`: Active Cursor updated — F28 is now open M8 work.
- No task resets: F28 is new; F5/F13/F24/F27 cascade was scope-note only (no AC changes).
