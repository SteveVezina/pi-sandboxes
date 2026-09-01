# F28: Local Template Library and Lifecycle

> Source: `SPEC.md` §6 Features F28, §18.1
> Status: 🔵 In progress — T28.1 (metadata + inspect/fork/validate) + T28.2 (history/diff/rollback) done; T28.2b (snapshot), T28.2c (promote), T28.3 (bundles), T28.4 (GUI) open
> Category: Service-layer / CLI / GUI

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F28 | Local Template Library and Lifecycle | Manage templates as local daemon-owned assets: inspect, fork, edit metadata, validate, snapshot from sandboxes, version locally, diff, rollback, import, export, and promote templates while preserving local-first trust boundaries | M8 |

## Expanded Specification

See `SPEC.md` §18.1. The daemon owns a local template library under the Pi
Box home. F5 Template System remains the static local baseline; F28 adds
authoring, fork, snapshot, history, import/export, and GUI management on
top of it.

Sources: `builtin`, `local`, `snapshot`, `imported`. The daemon is the
sole authority for mutating the library. Templates stay YAML on disk and
keep their existing minimal fields; the schema is extended (version,
summary, tags, source, lineage, `compatibility.runtimes`, resolved `base`
image + digest, pinned tool versions, `network.domains`, `resources`,
timestamps, revision history).

Reconciliation with earlier decisions:
- `base` is a resolved OCI image name + digest (PROP-007).
- `compatibility.runtimes` is a declared hint; the driver `Probe` /
  `CapabilityReport` is authoritative (ADR-005). Validate reports
  mismatches.
- `network.domains` only seeds a new sandbox's `network.allow`; the
  daemon egress proxy stays authoritative (ADR-006).
- Snapshots are daemon-managed assets captured from container/rootfs
  state — never host bind mounts (PROP-009 host decoupling).

## Acceptance Criteria

Mapped from `SPEC.md` § AC-35:

- [x] AC-35.1: Template metadata includes version, summary, description, tags, source, lineage, compatibility, tool versions, cache mounts, network domains, resource defaults, digest, and timestamps *(2026-08-31: `Template` extended with all-optional fields in `pkg/template/template.go`; `ContentDigest()` in `digest.go`)*
- [x] AC-35.2: Existing minimal built-in templates remain valid template definitions *(2026-08-31: `TestValidate_BuiltinsAllPass` — all 7 pass; new fields are `omitempty`)*
- [x] AC-35.3: `pi-box template inspect <name>` shows detailed metadata, lineage, compatibility, and policy-relevant settings *(also `GET /v1/templates/{name}` → template + resolved image + digest + validation problems)*
- [x] AC-35.4: `pi-box template fork <source> <new-name>` creates an editable local template *(2026-08-31: `Store.Fork` in `fork.go` — records `source.forkedFrom` + `lineage` generation + parent digest; `POST /v1/templates/fork`)*
- [ ] AC-35.5: `pi-box template snapshot <sandbox-id> <new-name>` creates a local template when the runtime supports safe snapshotting *(T28.2)*
- [x] AC-35.6: `pi-box template validate <path-or-name>` validates schema, pinned versions, cache mounts, network requests, runtime compatibility, and policy-relevant fields *(2026-08-31: `Template.Validate()` in `validate.go`; `POST /v1/templates/validate` (installed name or inline definition))*
- [x] AC-35.7: `pi-box template diff` and `history` show local template changes and lineage *(2026-08-31: T28.2 — revision store + `diff`/`history` CLI + API)*
- [x] AC-35.8: `pi-box template rollback` restores a local template to a prior revision *(2026-08-31: T28.2 — `Store.Rollback`, recorded as a new revision, bumps lineage)*
- [ ] AC-35.9: `pi-box template export` creates a portable bundle with metadata, definition, digest, and provenance
- [ ] AC-35.10: `pi-box template import` validates and installs a portable local bundle
- [ ] AC-35.11: Snapshot, import, fork, rollback, promote, and export operations are audited and policy-checked
- [ ] AC-35.12: GUI Templates view can show details, source/lineage, validation errors, history/diff, and fork/snapshot/import/export/promote actions
- [ ] AC-35.13: Template snapshots and exports exclude secrets and workspace source files by default

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/template/` | Schema expansion, validation, provenance, local revision store |
| `pkg/api/` | `/v1/templates*` fork/snapshot/validate/history/diff/rollback/promote/import/export |
| `cmd/pi-box/` | `pi-box template` local lifecycle subcommands |
| GUI | Templates view becomes a management surface |
| F5 Template System | Static baseline; local lifecycle delegated here |
| F13 Snapshot & Rollback | Template snapshot overlaps sandbox snapshot primitives |
| F17 Policy Enforcement | Template-requested domains/mounts/resources + import/snapshot/export are policy-checked |

## Security Considerations

- Built-in templates are trusted; imported bundles untrusted until validated + explicitly installed.
- Snapshots/exports exclude secrets, SSH keys, cloud credentials, tokens, daemon config, host-only paths, and workspace source by default.
- Bundles carry content digests for audit.
- No lifecycle op persists credentials in template YAML, workspaces, bundles, or artifacts.
- Template-requested domains/resources/mounts/runtimes never override daemon policy.
- Snapshot fails closed when a runtime cannot safely capture executable state (offers metadata-only).

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F5 Template System | Internal feature | ✅ Implemented (static baseline) |
| F13 Snapshot & Rollback | Internal feature | 🟡 Partial |
| F17 Policy Enforcement | Internal feature | 🟡 Partial |
| PROP-007 (image resolution) | Applied | `base` resolved-image form |
| ADR-005 (driver contract) | Accepted | capability reports |
| ADR-006 (egress) | Accepted | `network.allow` seeding |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go template library + validation + revision store in `pkg/template/` |
| **Configuration** | Daemon-owned library under `~/.pi-box/templates/` with per-template revision history |

**ADR references:** None yet.
**ADR gaps:**
- Local template snapshot boundaries and runtime-specific behavior.
- Template schema versioning and compatibility policy.
- Import/export bundle format and secret-exclusion guarantees.

## Tasks

> Phased per PROP-006. Author detailed ACs/verification per task before executing.

### T28.1: Metadata schema + inspect + fork + validate ✅ *(2026-08-31)*

**Description:** Extended the template YAML schema (version, summary, description, tags, source, lineage, compatibility, network domains, resources, timestamps) — all optional, built-ins unchanged. `Template.Validate()`, `Template.ContentDigest()`, `Store.Fork()`. API: `GET /v1/templates`, `GET /v1/templates/{name}`, `POST /v1/templates/fork`, `POST /v1/templates/validate`. CLI: `template fork`, `template validate`; `template inspect` extended.

**Acceptance criteria:** AC-35.1, AC-35.2, AC-35.3, AC-35.4, AC-35.6 — all ✅
**Verification:** `tests/template/f28_test.go` (validate, digest stability, fork lineage + collision), `pkg/api/templates_internal_test.go` (list/inspect/fork-409/validate). Suite 502 pass. `website/docs/{cli,api}/templates.md` updated.
**Files:** `pkg/template/{template,validate,digest,fork}.go`, `pkg/api/templates.go`, `pkg/daemon/router.go`, `cmd/pi-box/template/{commands,template}.go`
**Size:** M
**Depends on:** F5

### T28.2: Local revision store — history + diff + rollback ✅ *(2026-08-31)*

**Description:** Every write to a local template records a revision under `~/.pi-box/templates/<name>/revisions/` (`N.yaml` + `index.json`). `Store.History`, `Store.GetRevision`, `Store.Rollback` (recorded as a new revision, bumps lineage), `Store.ResolveRef` (`name` or `name@N`), `template.Diff` (line-oriented, no diff-lib dependency). API: `GET /v1/templates/{name}/history`, `POST /v1/templates/diff`, `POST /v1/templates/{name}/rollback`. CLI: `template history|diff|rollback`.

**Acceptance criteria:** AC-35.7, AC-35.8 — ✅
**Verification:** `tests/template/f28_test.go::TestRevisions_HistoryRollbackDiff`, `pkg/api/templates_internal_test.go::TestTemplateHistoryDiffRollback`. Suite 504 pass. Site docs updated.
**Files:** `pkg/template/revision.go` (new), `pkg/template/template.go` (Create records a revision), `pkg/api/templates.go`, `pkg/daemon/router.go`, `cmd/pi-box/template/commands.go`
**Size:** M
**Depends on:** T28.1

### T28.2b: Snapshot a sandbox into a local template ⏳

**Description:** `pi-box template snapshot <sandbox-id> <new-name>` — records source sandbox ID/mode/time; excludes secrets + workspace source; runtime-aware (fast = metadata + cache/toolchain assumptions; compat/microvm = image/rootfs). Fails closed when a runtime cannot safely capture executable state (offers metadata-only).

**Acceptance criteria:** AC-35.5, AC-35.11
**Files:** `pkg/template/`, `pkg/api/templates.go`, runtime snapshot hooks
**Size:** L → split on a Linux host
**Depends on:** T28.1, F13

### T28.2c: Promote (set default) ⏳

**Description:** `pi-box template promote <name> [--default]` — mark a local/snapshot template as the default for `pi-box box create` with no template arg. Needs a `default_template` key in `~/.pi-box/config.yaml` (no config-default mechanism exists yet).

**Acceptance criteria:** AC-35.11 (promote is audited/policy-checked)
**Files:** `pkg/template/`, config plumbing, `pkg/api/sandbox_create.go`
**Size:** S
**Depends on:** T28.1

### T28.3: Import / export bundles ⏳

**Description:** Portable bundle format (metadata + definition + optional build artifacts + content digest + provenance). `export` writes it; `import` validates + policy-checks + installs as `imported` source. Untrusted until explicitly installed.

**Acceptance criteria:** AC-35.9, AC-35.10, AC-35.11, AC-35.13
**Files:** `pkg/template/bundle.go` (new), `pkg/api/templates.go`
**Size:** M
**Depends on:** T28.1

### T28.4: GUI template management surface ⏳

**Description:** Extend the Workbench Templates view from static cards to a management surface: detail view, fork, snapshot, diff/history, import/export, promote, validation/policy warnings. GUI stays a daemon client — no mutation logic outside the API.

**Acceptance criteria:** AC-35.12
**Files:** GUI templates view
**Size:** M
**Depends on:** T28.1, T28.2, T28.3, F24

## Verification Plan

- [ ] `go test ./pkg/template/... ./pkg/api/...`
- [ ] Round-trip: fork → edit → validate → snapshot → diff → rollback
- [ ] Bundle: export → import on a fresh library → digests match
- [ ] Security: snapshot/export of a sandbox with a seeded secret file → secret absent from the bundle
- [ ] Built-in templates still validate under the extended schema

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Bundle wire format (tar layout, manifest schema) not specified | §18.1 | Specify once T28.3 design settles (likely a follow-up PROP or ADR) |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Snapshot boundaries per runtime (what fast/compat/secure/microvm each capture) | F28, F13, F20 | ADR: local template snapshot boundaries |
| Template schema versioning + forward/backward compatibility policy | F28, F5 | ADR: template schema versioning |
| Import/export bundle format + secret-exclusion guarantees | F28, F17 | ADR: template bundle format |

## Out of Scope

- Central hosted registry / marketplace
- Docker Hub-style search / pull / push
- Automatic remote template installation at sandbox create
- Implicit template execution hooks
- Registry credentials / remote publishing auth
