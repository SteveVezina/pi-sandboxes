# F9: Output Delivery

> Source: `SPEC.md` §6 Features F9
> Status: 🟡 Partially implemented (list/pull/pack + `pi.artifact.delivered` emission done; archive size validation open — no spec default). *(2026-08-31: AC-9.4 closed — ADR-007 lifecycle event transport; 2026-08-28: CLI is `artifacts` not `output`.)*
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F9 | Output Delivery | Single deliverable channel for artifacts and patches from sandboxes | M1 |

## Expanded Specification

Output delivery is the only deliverable export path from a sandbox. Build artifacts, test reports, and coding-agent patches leave through `POST /v1/sandboxes/{id}/output`; `pi.artifact.delivered` fires only after this channel succeeds.

Default output sources inside sandbox:
- `/artifacts` — primary artifact directory
- `/workspace/dist` — build outputs
- `/workspace/build` — build outputs
- `/workspace/coverage` — test coverage reports
- `/workspace/test-results` — test result files
- `/workspace/target/release` — Rust release binaries
- workspace patch view — coding-agent deliverable produced from the workspace diff

Operations:
1. **List** — List available deliverables from known output sources.
2. **Pull** — Deliver selected artifacts or patch to a host destination through the output endpoint.
3. **Pack** — Create compressed archive (tar.zst) of selected output sources through the output endpoint.

CLI commands *(2026-08-28: corrected — shipped code and SPEC.md's CLI table both use `artifacts`, not `output`; `output` only names the HTTP endpoint)*:
```bash
pi-box box artifacts list <name>
pi-box box artifacts pull <name> <dest>
pi-box box artifacts pack <name> --output artifacts.tar.gz
```

`diff` and `patch` remain read-only workspace views. File read/pull remains a debug and inspection API, not a deliverable export path. Snapshots are warm-start and rollback inputs only.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-9.1: `pi-box box artifacts list <id>` lists available deliverables *(2026-07-15: AC updated per PROP-009; 2026-08-28: CLI name corrected from "output" to "artifacts")*
- [x] AC-9.2: `pi-box box artifacts pull <id> <dest>` delivers artifacts or patches to host through `POST /v1/sandboxes/{id}/output` *(2026-07-15: AC updated per PROP-009)*
- [x] AC-9.3: `pi-box box artifacts pack <id> --output <file>` creates archive (tar.gz, not tar.zst — see Spec Gaps) *(2026-07-15: AC updated per PROP-009)*
- [x] AC-9.4: `pi.artifact.delivered` is emitted only after successful output-channel delivery *(2026-08-31: ADR-007 — `pkg/events` emitter; `handleOutputPull`/`handleOutputPack` call `events.Emit` after the copy/archive succeeds, before the response. Verified: `tests/events/events_test.go` + `pkg/api/events_internal_test.go`.)*
- [x] AC-33.1: Artifacts and patches leave the sandbox only through `POST /v1/sandboxes/{id}/output`
- [x] AC-33.2: `diff` and `patch` endpoints are read-only workspace views; export uses the output channel
- [x] AC-33.3: Snapshot export is absent

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/api/sandbox_output.go` | Output manifest, pack, and delivery |
| F6: Workspace & File Ops | Provides diff/patch views and known output source reads |
| F2: Daemon API | Adds `POST /v1/sandboxes/{id}/output`; old artifact export route is superseded |
| Lifecycle events | Emits `pi.artifact.delivered` only after output-channel success |

## Security Considerations

- Only known output sources are deliverable by default.
- Archive format uses zstd compression.
- No symbolic link following during output collection.
- File sizes validated before archiving to prevent DoS.
- Output channel is the only policy-reviewed deliverable path.

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F6: Workspace & File Ops | Internal feature | ✅ Implemented |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go handler in `pkg/api/sandbox_output.go`, routed at `POST /v1/sandboxes/{id}/output` |
| **Configuration** | Default output sources from spec |

**ADR references:** ADR-007 (Lifecycle Event Transport) — Proposed 2026-08-31; provides the `pkg/events` emitter used for AC-9.4.
**ADR gaps:** None (lifecycle-event transport resolved by ADR-007).

## Tasks

### T8.1: Output list ✅ *(2026-08-28: re-verified with a real-container integration test)*

**Description:** Implement output listing. Scans known output sources and returns a deliverable manifest. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [x] Lists files in `/artifacts`, `/workspace/dist`, `/workspace/build`, `/workspace/coverage`, `/workspace/test-results`, `/workspace/target/release`
- [x] Includes patch deliverable metadata when workspace changes exist
- [x] Returns file paths, sizes, and modification times
- [x] Empty directories excluded from listing (via `find -type f`, which only matches files present under a source dir)
- [ ] Symbolic links reported but not followed *(actual behavior: `find -type f` silently excludes symlinks rather than reporting them — safer than following, but doesn't satisfy "reported"; see Spec Gaps)*

**Verification:**
- [x] `go build ./pkg/api/...`
- [x] Integration test (real Docker container): `TestOutputChannel_E2E`, `tests/integration/output_test.go` — asserts seeded artifact and patch appear in the list

**Files:** `pkg/api/sandbox_output.go`
**Size:** S
**Depends on:** F6 (Workspace & File Ops — file read and patch view)

### T8.2: Output pull ✅ *(2026-08-28: re-verified with a real-container integration test; event emission split out as an open gap)*

**Description:** Implement output pull through `POST /v1/sandboxes/{id}/output`. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [x] `pi-box box artifacts pull demo ./artifacts` pulls selected deliverables to host
- [x] Patch delivery uses the output endpoint, not the read-only patch route
- [x] Directory structure preserved on host
- [ ] Progress reported for large transfers *(not implemented — `docker cp` gives no progress hook; same gap as F6/T6.4, see Spec Gaps)*
- [x] Only known output sources are copied by default
- [x] `pi.artifact.delivered` fires only after successful delivery *(2026-08-31: `events.Emit(events.ArtifactDelivered, …)` after the copy in `handleOutputPull`, per ADR-007)*

**Verification:**
- [x] `go build ./pkg/api/...`
- [x] Integration test (real Docker container): `TestOutputChannel_E2E` — pulls a seeded artifact and workspace patch, verifies host content
- [x] Test: output pull emits `pi.artifact.delivered` after a successful copy — `pkg/api/events_internal_test.go` covers the emitter path end to end; a real-container variant belongs in `tests/integration/output_test.go` when next touched

**Files:** `pkg/api/sandbox_output.go`
**Size:** M
**Depends on:** T8.1 (output list)

### T8.3: Output pack ✅ *(2026-08-28: re-verified with a real-container integration test; format corrected to match code)*

**Description:** Implement output packing through the output endpoint. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [x] `pi-box box artifacts pack demo --output artifacts.tar.gz` creates compressed archive
- [x] Archive contains selected deliverables from known output sources
- [x] Archive is valid tar.gz format *(SPEC.md's CLI example says `tar.zst`; shipped code uses Go's stdlib `compress/gzip`, no zstd dependency exists in go.mod — see Spec Gaps)*
- [ ] Archive size validated before creation *(still open — no size cap in `tarGzDir`/`handleOutputPack`; needs a block-spec default to validate against, see Spec Gaps. The `pi.artifact.delivered` event now reports the resulting byte count.)*

**Verification:**
- [x] `go build ./pkg/api/...`
- [x] Integration test (real Docker container): `TestOutputChannel_E2E` — packs a seeded artifact, verifies non-empty archive on host

**Files:** `pkg/api/sandbox_output.go`
**Size:** S
**Depends on:** T8.2 (output pull)

## Verification Plan

- [x] `go build ./pkg/api/...` succeeds
- [x] Output list returns correct files and patch metadata
- [x] Output pull delivers artifacts and patches through `POST /v1/sandboxes/{id}/output`
- [x] Output pack creates valid tar.gz archive
- [ ] Benchmark: artifact_export_20mb < 500ms local (SPEC.md §28) *(`ArtifactExport20MB` in `pkg/bench/benchmarks.go` is a stub — it writes+deletes a 20MB temp file and never calls the pack/output code path; owned by F14 Benchmarks, still ⚠️ Needs re-verify)*

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| CLI table says `pi-box box output list\|pull\|pack`; shipped code and SPEC.md's own §12.5 CLI table use `artifacts` | §12.5, §6 F9 | Reconcile — drop the `output` subcommand name wherever it still appears; `output` should only name the HTTP endpoint |
| Archive format specified as tar.zst; no zstd dependency exists and the shipped implementation uses tar.gz (`compress/gzip`) | §12.5 | Either amend to tar.gz, or accept a new zstd dependency and switch the implementation |
| No default artifact/archive size limit specified, so "validated before creation to prevent DoS" has nothing to validate against | §8 Security Model | Add a default max archive/output size (e.g. mirror `maxOutput: 8MiB` exec default, or a larger artifact-specific value) |
| `files pull`/output pull progress reporting for large transfers not achievable with current `docker cp`-based copy | §12.6 | Either drop the requirement or switch to a streaming copy implementation that exposes progress (same gap as F6/T6.4) |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| ~~Lifecycle event emission has no implementation~~ | F9, F4, F29 | **ADR-007 (Proposed 2026-08-31):** `pkg/events` emitter → always-on slog sink + opt-in `--events-webhook`. `pi.sandbox.created`/`destroyed` and `pi.artifact.delivered` wired; `pi.run.*` ready for F29. |

## Out of Scope

- Output versioning (future)
- Output checksums/hashes (future)
- Remote output storage (future)
- Output deduplication (future)
