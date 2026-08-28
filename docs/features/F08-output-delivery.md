# F9: Output Delivery

> Source: `SPEC.md` §6 Features F9
> Status: ⚠️ Needs re-verify
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

CLI commands:
```bash
pi-box box output list <name>
pi-box box output pull <name> <dest>
pi-box box output pack <name> --output artifacts.tar.zst
```

`diff` and `patch` remain read-only workspace views. File read/pull remains a debug and inspection API, not a deliverable export path. Snapshots are warm-start and rollback inputs only.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-9.1: `pi-box box output list <id>` lists available deliverables *(2026-07-15: AC updated per PROP-009)*
- [ ] AC-9.2: `pi-box box output pull <id> <dest>` delivers artifacts or patches to host through `POST /v1/sandboxes/{id}/output` *(2026-07-15: AC updated per PROP-009)*
- [ ] AC-9.3: `pi-box box output pack <id> --output <file>` creates archive *(2026-07-15: AC updated per PROP-009)*
- [ ] AC-9.4: `pi.artifact.delivered` is emitted only after successful output-channel delivery *(2026-07-15: added per PROP-009)*
- [ ] AC-33.1: Artifacts and patches leave the sandbox only through `POST /v1/sandboxes/{id}/output`
- [ ] AC-33.2: `diff` and `patch` endpoints are read-only workspace views; export uses the output channel
- [ ] AC-33.3: Snapshot export is absent

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/artifacts/` or successor output package | Output manifest, pack, and delivery |
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
| **Service-layer** | Go output/artifacts package and daemon endpoint |
| **Configuration** | Default output sources from spec |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T8.1: Output list ⚠️

**Description:** Implement output listing. Scans known output sources and returns a deliverable manifest. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [ ] Lists files in `/artifacts`, `/workspace/dist`, `/workspace/build`, `/workspace/coverage`, `/workspace/test-results`, `/workspace/target/release`
- [ ] Includes patch deliverable metadata when workspace changes exist
- [ ] Returns file paths, sizes, and modification times
- [ ] Empty directories excluded from listing
- [ ] Symbolic links reported but not followed

**Verification:**
- [ ] `go build ./pkg/artifacts/...`
- [ ] Unit test: output list returns correct files and patch metadata

**Files:** `pkg/artifacts/list.go`
**Size:** S
**Depends on:** F6 (Workspace & File Ops — file read and patch view)

### T8.2: Output pull ⚠️

**Description:** Implement output pull through `POST /v1/sandboxes/{id}/output`. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [ ] `pi-box box output pull demo ./artifacts` pulls selected deliverables to host
- [ ] Patch delivery uses the output endpoint, not the read-only patch route
- [ ] Directory structure preserved on host
- [ ] Progress reported for large transfers
- [ ] Only known output sources are copied by default
- [ ] `pi.artifact.delivered` fires only after successful delivery

**Verification:**
- [ ] `go build ./pkg/artifacts/...`
- [ ] Integration test: output pull delivers artifacts from sandbox
- [ ] Integration test: patch delivery emits `pi.artifact.delivered`

**Files:** `pkg/artifacts/pull.go`, `pkg/api/sandbox_output.go`
**Size:** M
**Depends on:** T8.1 (output list)

### T8.3: Output pack ⚠️

**Description:** Implement output packing through the output endpoint. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [ ] `pi-box box output pack demo --output artifacts.tar.zst` creates compressed archive
- [ ] Archive contains selected deliverables from known output sources
- [ ] Archive is valid tar.zst format
- [ ] Archive size validated before creation

**Verification:**
- [ ] `go build ./pkg/artifacts/...`
- [ ] Integration test: pack and unpack archive

**Files:** `pkg/artifacts/pack.go`, `pkg/api/sandbox_output.go`
**Size:** S
**Depends on:** T8.2 (output pull)

## Verification Plan

- [ ] `go build ./pkg/artifacts/...` succeeds
- [ ] Output list returns correct files and patch metadata
- [ ] Output pull delivers artifacts and patches through `POST /v1/sandboxes/{id}/output`
- [ ] Output pack creates valid tar.zst archive
- [ ] Benchmark: artifact_export_20mb < 500ms local (SPEC.md §28)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| — | — | — |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Output versioning (future)
- Output checksums/hashes (future)
- Remote output storage (future)
- Output deduplication (future)
