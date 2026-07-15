# F8: Sandbox Lifecycle

> Source: `SPEC.md` §6 Features F8
> Status: ⚠️ Needs re-verify
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F8 | Sandbox Lifecycle | Create, list, inspect, destroy, TTL expiration, warm sandbox reuse | M1 |

## Expanded Specification

The sandbox lifecycle manager is the core state machine for sandboxes. It handles the full lifecycle from creation to destruction, with warm sandbox reuse as the default pattern.

Lifecycle states:
```
CREATING → WARM → EXECUTING → WARM → ... → DESTROYING → DESTROYED
                ↑                                    |
                └──────── TTL EXPIRED ───────────────┘
```

State transitions:
1. **CREATING** — Sandbox metadata created, backend initialization started
2. **WARM** — Sandbox ready, no active command running
3. **EXECUTING** — Command running inside sandbox
4. **DESTROYING** — Cleanup initiated (backend teardown, metadata removal)
5. **DESTROYED** — Sandbox fully cleaned up

Each sandbox has:
- **ID** — unique identifier (UUID v4)
- **Name** — human-readable name (from CLI)
- **Template** — template used for this sandbox
- **Mode** — runtime mode (fast, compat)
- **State** — current lifecycle state
- **CreatedAt** — creation timestamp
- **UpdatedAt** — last state change timestamp
- **TTL** — time-to-live in seconds (default: 7200 = 2 hours)
- **LastUsedAt** — last exec timestamp (for TTL calculation)
- **Workspace** — workspace directory path
- **Artifacts** — artifacts directory path
- **Snapshots** — list of snapshot names

Metadata is stored under `~/.pi-box/sandboxes/<id>/meta.json`.

The sandbox manager runs a background goroutine that:
1. Checks for TTL-expired sandboxes every 60 seconds
2. Calls destroy on expired sandboxes
3. Cleans up orphaned sandboxes on daemon restart

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-8.1: Sandbox created once and kept warm
- [x] AC-8.2: Multiple exec calls reuse the same sandbox
- [x] AC-8.3: TTL expiration triggers cleanup
- [x] AC-8.4: `pi-box box destroy --all` cleans all sandboxes
- [x] AC-8.5: Sandbox state persisted in `~/.pi-box/sandboxes/<id>/meta.json`
- [x] AC-8.6: Orphaned sandboxes cleaned up on daemon restart

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/sandbox/` | Sandbox lifecycle manager |
| `~/.pi-box/sandboxes/<id>/meta.json` | Per-sandbox metadata |
| `~/.pi-box/sandboxes/` | Sandbox directory |
| F2: Daemon API | Sandbox manager is the backend for API endpoints |
| F3: Fast Backend | Sandbox manager dispatches to backend |

## Security Considerations

- Sandbox IDs are UUIDs — not guessable
- Metadata stored under `~/.pi-box/` with restricted permissions
- TTL expiration uses daemon clock (not sandbox clock) to prevent time manipulation
- Destroy operation is idempotent (safe to call multiple times)
- No sandbox data leaves the sandbox filesystem boundary

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| None | — | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go sandbox manager in `pkg/sandbox/` |
| **Configuration** | TTL defaults from `~/.pi-box/config.yaml` |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T4.1: Sandbox metadata store ⚠️

**Description:** Implement sandbox metadata store under `~/.pi-box/sandboxes/<id>/meta.json`. CRUD operations for sandbox state. *(2026-07-15: terminology and package path updated per PROP-009; re-verify against `pkg/sandbox/`.)*

**Acceptance criteria:**
- [x] `Create(name, template, mode)` creates `~/.pi-box/sandboxes/<id>/meta.json`
- [x] `Get(id)` reads and parses metadata
- [x] `Update(id, state)` updates state and timestamp
- [x] `List()` returns all sandbox IDs with status
- [x] `Delete(id)` removes metadata directory
- [x] Metadata directory permissions: `0755` (owner read/write/execute, group/others read/execute)

**Verification:**
- [x] `go build ./pkg/sandbox/...`
- [x] Unit tests for CRUD operations

**Files:** `pkg/sandbox/store.go`, `pkg/sandbox/meta.go`
**Size:** S
**Status:** ⚠️ Needs re-verify
**Depends on:** None

### T4.2: Sandbox state machine ⚠️

**Description:** Implement sandbox state machine with lifecycle transitions. State validation prevents invalid transitions. *(2026-07-15: lifecycle terminology updated per PROP-009.)*

**Acceptance criteria:**
- [x] Valid transitions: CREATING→WARM, WARM→EXECUTING, EXECUTING→WARM, WARM→DESTROYING, EXECUTING→DESTROYING
- [x] Invalid transitions are rejected with error
- [x] State changes are persisted to meta.json
- [x] State is thread-safe (concurrent access from multiple goroutines)

**Verification:**
- [x] `go build ./pkg/sandbox/...`
- [x] Unit tests for state transitions

**Files:** `pkg/sandbox/state.go`
**Size:** S
**Status:** ⚠️ Needs re-verify
**Depends on:** T4.1 (metadata store)

### T4.3: TTL expiration ⚠️

**Description:** Implement background TTL checker. Runs every 60 seconds, destroys sandboxes past their TTL. *(2026-07-15: lifecycle terminology updated per PROP-009.)*

**Acceptance criteria:**
- [x] Background goroutine checks TTL every 60 seconds
- [x] Sandboxes past TTL are marked DESTROYING
- [x] TTL calculated from LastUsedAt + TTL seconds
- [x] TTL configurable per sandbox (default: 7200s)
- [x] TTL disabled when set to 0 (infinite)

**Verification:**
- [x] `go build ./pkg/sandbox/...`
- [x] Integration test: sandbox destroyed after TTL expires

**Files:** `pkg/sandbox/ttl.go`
**Size:** S
**Status:** ⚠️ Needs re-verify
**Depends on:** T4.2 (state machine)

### T4.4: Orphan cleanup on restart ⚠️

**Description:** On daemon start, scan for sandboxes in non-terminal states (CREATING, EXECUTING) that have no running backend process. Mark them as DESTROYED. *(2026-07-15: lifecycle terminology updated per PROP-009.)*

**Acceptance criteria:**
- [x] On startup, scan all sandboxes in ~/.pi-box/sandboxes/
- [x] Sandboxes with no running process are marked DESTROYED
- [x] Orphan cleanup is logged
- [x] Orphan cleanup does not delete workspace/artifacts (metadata only)

**Verification:**
- [x] `go build ./pkg/sandbox/...`
- [x] Integration test: orphan sandbox cleaned up on restart

**Files:** `pkg/sandbox/orphans.go`
**Size:** S
**Status:** ⚠️ Needs re-verify
**Depends on:** T4.1 (metadata store), T4.2 (state machine)

## Verification Plan

- [x] `go build ./pkg/sandbox/...` succeeds
- [x] Unit tests for metadata CRUD
- [x] Unit tests for state machine transitions
- [x] Integration test: TTL expiration triggers destroy
- [x] Integration test: orphan cleanup on restart
- [x] `pi-box box destroy --all` cleans all sandboxes
- [x] AC-8.4: `pi-box box destroy --all` via cobra `--all` flag (`tests/box/box_ac_test.go::TestBoxDestroyAll_CleansAllSandboxes`)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|

### Resolved gaps

| Gap | Block Spec Section | Resolution |
|-----|-------------------|------------|
| Sandbox marked WARM before runtime resource exists | §7 Sandbox state machine | PROP-007 — added state verification: container must be running before transitioning to WARM state |
| TTL expiration frequency not specified | §8 Sandbox Lifecycle | Add: "Background checker runs every 60s" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Sandbox migration between backends (e.g., fast → compat)
- Sandbox sharing between users
- Remote sandbox access (Milestone 6)
- Sandbox pre-warming/pooling (future optimization)
