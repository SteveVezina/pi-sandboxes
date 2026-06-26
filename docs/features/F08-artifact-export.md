# F08: Artifact Export

> Source: `SPEC.md` §6 Features F9
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F9 | Artifact Export | List, pull, and pack artifacts from sandbox sessions | M1 |

## Expanded Specification

Artifact export manages files intentionally exported from sandbox sessions. Artifacts are build outputs, test reports, patches, and other files that the coding agent wants to collect from the sandbox.

Default artifact locations inside sandbox:
- `/artifacts` — primary artifact directory
- `/workspace/dist` — build outputs
- `/workspace/build` — build outputs
- `/workspace/coverage` — test coverage reports
- `/workspace/test-results` — test result files
- `/workspace/target/release` — Rust release binaries

Operations:
1. **List** — List files in artifact directories
2. **Pull** — Copy files from sandbox artifact directories to host destination
3. **Pack** — Create compressed archive (tar.zst) of artifact directories

CLI commands:
```bash
pi box artifacts list <name>
pi box artifacts pull <name> <dest>
pi box artifacts pack <name> --output <file>
```

The artifact export avoids copying the whole workspace unless requested. It only copies from known artifact locations.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-9.1: `pi box artifacts list <id>` lists available artifacts
- [x] AC-9.2: `pi box artifacts pull <id> <dest>` pulls artifacts to host
- [x] AC-9.3: `pi box artifacts pack <id> --output <file>` creates archive

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/artifacts/` | Artifact management |
| F6: Workspace & File Ops | Uses file pull for artifact extraction |
| F2: Daemon API | Artifacts export endpoint |

## Security Considerations

- Only known artifact locations are exported (not arbitrary workspace paths)
- Archive format uses zstd compression (safe, no archive vulnerabilities)
- No symbolic link following during artifact collection
- File sizes validated before archiving (prevent DoS)

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F6: Workspace & File Ops | Internal feature | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go artifacts package |
| **Configuration** | Default artifact locations from spec |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T8.1: Artifact list

**Description:** Implement artifact listing. Scans known artifact locations and returns file listing.

**Acceptance criteria:**
- [x] Lists files in `/artifacts`, `/workspace/dist`, `/workspace/build`, `/workspace/coverage`, `/workspace/test-results`, `/workspace/target/release`
- [x] Returns file paths, sizes, and modification times
- [x] Empty directories excluded from listing
- [x] Symbolic links reported but not followed

**Verification:**
- [x] `go build ./pkg/artifacts/...`
- [x] Unit test: artifact list returns correct files

**Files:** `pkg/artifacts/list.go`
**Size:** S
**Depends on:** F6 (Workspace & File Ops — file read)

### T8.2: Artifact pull

**Description:** Implement artifact pull. Copies files from artifact locations to host destination.

**Acceptance criteria:**
- [x] `pi box artifacts pull demo ./artifacts` pulls all artifacts to host
- [x] Directory structure preserved on host
- [x] Progress reported for large transfers
- [x] Only known artifact locations are copied

**Verification:**
- [x] `go build ./pkg/artifacts/...`
- [x] Integration test: pull artifacts from sandbox

**Files:** `pkg/artifacts/pull.go`
**Size:** S
**Depends on:** T8.1 (artifact list)

### T8.3: Artifact pack

**Description:** Implement artifact packing. Creates tar.zst archive of artifact directories.

**Acceptance criteria:**
- [x] `pi box artifacts pack demo --output artifacts.tar.zst` creates compressed archive
- [x] Archive contains all files from artifact locations
- [x] Archive is valid tar.zst format
- [x] Archive size validated (prevent DoS)

**Verification:**
- [x] `go build ./pkg/artifacts/...`
- [x] Integration test: pack and unpack archive

**Files:** `pkg/artifacts/pack.go`
**Size:** S
**Depends on:** T8.2 (artifact pull)

## Verification Plan

- [x] `go build ./pkg/artifacts/...` succeeds
- [x] Artifact list returns correct files
- [x] Artifact pull copies files correctly
- [x] Artifact pack creates valid tar.zst archive
- [x] Benchmark: artifact export 20MB < 500ms local (SPEC.md §19)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Artifact archive format not specified | §19 Artifact model | Add: "tar.zst compressed archive" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Artifact versioning (future)
- Artifact checksums/hashes (future)
- Remote artifact storage (future)
- Artifact deduplication (future)
