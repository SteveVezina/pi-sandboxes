# F28: Local Template Library and Lifecycle

> Source: `SPEC.md` §6 Features F28, §18.1
> Status: 🔵 In progress — T28.1/T28.2/T28.2c/T28.3/T28.4 done; only T28.2b (snapshot from a sandbox — Linux/runtime hooks) remains
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
- [x] AC-35.9: `pi-box template export` creates a portable bundle with metadata, definition, digest, and provenance *(2026-08-31: T28.3 — `template.ExportBundle` → OCI image layout tar per ADR-008; `POST /v1/templates/export`)*
- [x] AC-35.10: `pi-box template import` validates and installs a portable local bundle *(2026-08-31: T28.3 — `Store.Import` verifies blob digests + `Validate` + collision check, installs `source.type: imported`; `POST /v1/templates/import`)*
- [x] AC-35.11: Snapshot, import, fork, rollback, promote, and export operations are audited and policy-checked *(2026-08-31: `pkg/template/audit.go` — structured `slog` line per fork/rollback/import/promote; `Validate` runs on fork/rollback/import/export. Snapshot audit lands with T28.2b.)*
- [x] AC-35.12: GUI Templates view can show details, source/lineage, validation errors, history/diff, and fork/snapshot/import/export/promote actions *(2026-08-31 T28.4: details, source/lineage, digest, compatibility, validation, revision history + rollback, diff (template or `name@N`), fork, validate, set-default, export/import. The `snapshot` action alone waits on T28.2b.)*
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
| F13 Snapshot & Rollback | Internal feature | ✅ Implemented (2026-08-31) |
| F17 Policy Enforcement | Internal feature | 🟡 Partial |
| PROP-007 (image resolution) | Applied | `base` resolved-image form |
| ADR-005 (driver contract) | Accepted | capability reports |
| ADR-006 (egress) | Accepted | `network.allow` seeding |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go template library + validation + revision store in `pkg/template/` |
| **Configuration** | Daemon-owned library under `~/.pi-box/templates/` with per-template revision history |

**ADR references:** ADR-008 (Template Bundle Format) — Accepted 2026-08-31; governs T28.3 (OCI image layout, file-or-optional-registry transport, secret exclusion).
**ADR gaps:**
- Local template snapshot boundaries and runtime-specific behavior.
- Template schema versioning and compatibility policy.
- ~~Import/export bundle format~~ — ADR-008 (Accepted 2026-08-31).

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

### T28.2c: Promote (set default) ✅ *(2026-08-31)*

**Description:** `pi-box template promote <name> --default` writes a
`.default` marker file in the templates dir (`Store.SetDefault` /
`Store.Default`, validates the name exists). `CreateSandbox` resolves an
empty `req.Template` from `Store.Default()` before falling back to `base`;
a deleted default resolves to `""`. `POST /v1/templates/{name}/promote`
`{default:true}`. Audited via `pkg/template/audit.go`.

**Acceptance criteria:** AC-35.11 (promote audited) — ✅
**Verification:** `tests/template/f28_test.go::TestPromote_SetsAndResolvesDefault`
(set, resolve, missing-template rejection, dangling-default → `""`).
Suite 509 pass.
**Files:** `pkg/template/promote.go` (new), `pkg/template/audit.go` (new),
`pkg/api/{templates,sandbox_create}.go`, `pkg/daemon/router.go`,
`cmd/pi-box/template/{commands,template}.go`
**Size:** S
**Depends on:** T28.1

### T28.3: Import / export bundles ✅ *(2026-08-31 — ADR-008)*

**Description:** `template.ExportBundle` writes an OCI image layout tar
(`oci-layout` + `index.json` + content-addressed blobs): config blob =
provenance (`contentDigest`, `lineage`, `source`, exporter version,
export time); one definition layer = template YAML;
`vnd.pi-sandbox.template.*` media types + `artifactType`. `ReadBundle`
verifies every blob digest, checks `artifactType`, re-runs `Validate`.
`Store.Import` installs with `source.type: imported`, rejects name
collisions. `POST /v1/templates/export` (octet-stream bundle) /
`POST /v1/templates/import` (raw tar body, `?name=`, 32 MiB cap). CLI:
`template export -o <file>` / `template import <file> [--name]`.
`ContentDigest` now also excludes `source` so a fork/import of the same
definition has the same digest.

**Deviations from ADR-008:** artifact layers (`--include-artifacts`) and
the `oci://<ref>` registry transport are deferred to a follow-up — the
file bundle is the baseline. AC-35.13's secret-file exclusion applies
there; the definition itself carries no secret fields.

**Acceptance criteria:** AC-35.9, AC-35.10 — ✅. AC-35.11 partial (Validate
runs; audit-log line open). AC-35.13 N/A until artifact layers.
**Verification:** `tests/template/f28_test.go` (round-trip + digest match,
tampered-blob rejection, non-template archive rejection),
`pkg/api/templates_internal_test.go` (export → import rename, collision
409). Suite 508 pass. Site docs updated.
**Files:** `pkg/template/bundle.go` (new), `pkg/template/digest.go`,
`pkg/api/templates.go`, `pkg/daemon/router.go`, `cmd/pi-box/template/{commands,template}.go`
**Size:** M
**Depends on:** T28.1, ADR-008

### T28.4: GUI template management surface 🟢 *(2026-08-31 — snapshot UI waits on T28.2b)*

**Description:** The Workbench Templates view is now a live management surface (`apps/gui/src/main.tsx` `TemplatesView` + `apps/gui/src/api.ts` template methods): fetches `GET /v1/templates`, per-template detail (source/lineage, content digest + generation, resolved image, network, declared runtime compatibility), validation problems, revision history with per-revision rollback, and Fork / Validate / Set-default actions. GUI stays a pure daemon client — every mutation is a `/v1/templates*` call.

Diff (template or `name@N` revision), export (bundle download), and import (bundle file upload) are wired. Only the snapshot-from-sandbox action is deferred — it needs T28.2b.

**Acceptance criteria:** AC-35.12 — done except the snapshot action (blocked on T28.2b)
**Verification:** `pnpm --filter @pi-sandbox/gui build` (tsc + vite) clean.
**Files:** `apps/gui/src/{main.tsx,api.ts,styles.css}`
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
| Bundle wire format chosen by ADR-008 (OCI image layout). §18.1 says "not a Docker-Hub-style pull/push"; ADR-008 allows an **optional** OCI-ref transport for `import`/`export` (never mandatory, no Pi-operated registry). | §18.1 | Clarify: "no *mandatory* registry and no hosted marketplace; an OCI registry reference is an optional transport for explicit import/export using the caller's own registry auth." Small wording change — recommended alongside ADR-008 acceptance, not a blocker. |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Snapshot boundaries per runtime (what fast/compat/secure/microvm each capture) | F28, F13, F20 | ADR: local template snapshot boundaries |
| Template schema versioning + forward/backward compatibility policy | F28, F5 | ADR: template schema versioning |
| ~~Import/export bundle format + secret-exclusion guarantees~~ | F28, F17 | **ADR-008** (Accepted 2026-08-31) — OCI image layout, optional registry transport |

## Out of Scope

- Central hosted registry / marketplace
- Docker Hub-style search / pull / push
- Automatic remote template installation at sandbox create
- Implicit template execution hooks
- Registry credentials / remote publishing auth
