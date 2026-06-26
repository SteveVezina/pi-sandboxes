# F04: Session Lifecycle

> Source: `SPEC.md` §6 Features F8
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F8 | Session Lifecycle | Create, list, inspect, destroy, TTL expiration, warm session reuse | M1 |

## Expanded Specification

The session lifecycle manager is the core state machine for sandbox sessions. It handles the full lifecycle from creation to destruction, with warm session reuse as the default pattern.

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

Each sandbox session has:
- **ID** — unique identifier (UUID v4)
- **Name** — human-readable name (from CLI)
- **Template** — template used for this session
- **Mode** — runtime mode (fast, compat)
- **State** — current lifecycle state
- **CreatedAt** — creation timestamp
- **UpdatedAt** — last state change timestamp
- **TTL** — time-to-live in seconds (default: 7200 = 2 hours)
- **LastUsedAt** — last exec timestamp (for TTL calculation)
- **Workspace** — workspace directory path
- **Artifacts** — artifacts directory path
- **Snapshots** — list of snapshot names

Metadata is stored under `~/.pi/sandboxes/<id>/meta.json`.

The session manager runs a background goroutine that:
1. Checks for TTL-expired sessions every 60 seconds
2. Calls destroy on expired sessions
3. Cleans up orphaned sessions on daemon restart

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-8.1: Sandbox created once and kept warm
- [ ] AC-8.2: Multiple exec calls reuse the same session
- [ ] AC-8.3: TTL expiration triggers cleanup
- [ ] AC-8.4: `pi box destroy --all` cleans all sandboxes
- [ ] AC-8.5: Session state persisted in `~/.pi/sandboxes/<id>/meta.json`
- [ ] AC-8.6: Orphaned sessions cleaned up on daemon restart

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/session/` | Session lifecycle manager |
| `~/.pi/sandboxes/<id>/meta.json` | Per-session metadata |
| `~/.pi/sandboxes/` | Session directory |
| F2: Daemon API | Session manager is the backend for API endpoints |
| F3: Fast Backend | Session manager dispatches to backend |

## Security Considerations

- Session IDs are UUIDs — not guessable
- Metadata stored under `~/.pi/` with restricted permissions
- TTL expiration uses daemon clock (not sandbox clock) to prevent time manipulation
- Destroy operation is idempotent (safe to call multiple times)
- No session data leaves the sandbox filesystem boundary

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| None | — | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go session manager in `pkg/session/` |
| **Configuration** | TTL defaults from `~/.pi/config.yaml` |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T4.1: Session metadata store

**Description:** Implement session metadata store under `~/.pi/sandboxes/<id>/meta.json`. CRUD operations for session state.

**Acceptance criteria:**
- [ ] `Create(name, template, mode)` creates `~/.pi/sandboxes/<id>/meta.json`
- [ ] `Get(id)` reads and parses metadata
- [ ] `Update(id, state)` updates state and timestamp
- [ ] `List()` returns all session IDs with status
- [ ] `Delete(id)` removes metadata directory
- [ ] Metadata directory permissions: `0755` (owner read/write/execute, group/others read/execute)

**Verification:**
- [ ] `go build ./pkg/session/...`
- [ ] Unit tests for CRUD operations

**Files:** `pkg/session/store.go`, `pkg/session/meta.go`
**Size:** S
**Status:** ✅
**Depends on:** None

### T4.2: Session state machine

**Description:** Implement session state machine with lifecycle transitions. State validation prevents invalid transitions.

**Acceptance criteria:**
- [ ] Valid transitions: CREATING→WARM, WARM→EXECUTING, EXECUTING→WARM, WARM→DESTROYING, EXECUTING→DESTROYING
- [ ] Invalid transitions are rejected with error
- [ ] State changes are persisted to meta.json
- [ ] State is thread-safe (concurrent access from multiple goroutines)

**Verification:**
- [ ] `go build ./pkg/session/...`
- [ ] Unit tests for state transitions

**Files:** `pkg/session/state.go`
**Size:** S
**Status:** ✅
**Depends on:** T4.1 (metadata store)

### T4.3: TTL expiration

**Description:** Implement background TTL checker. Runs every 60 seconds, destroys sessions past their TTL.

**Acceptance criteria:**
- [ ] Background goroutine checks TTL every 60 seconds
- [ ] Sessions past TTL are marked DESTROYING
- [ ] TTL calculated from LastUsedAt + TTL seconds
- [ ] TTL configurable per session (default: 7200s)
- [ ] TTL disabled when set to 0 (infinite)

**Verification:**
- [ ] `go build ./pkg/session/...`
- [ ] Integration test: session destroyed after TTL expires

**Files:** `pkg/session/ttl.go`
**Size:** S
**Status:** ✅
**Depends on:** T4.2 (state machine)

### T4.4: Orphan cleanup on restart

**Description:** On daemon start, scan for sessions in non-terminal states (CREATING, EXECUTING) that have no running backend process. Mark them as DESTROYED.

**Acceptance criteria:**
- [ ] On startup, scan all sessions in ~/.pi/sandboxes/
- [ ] Sessions with no running process are marked DESTROYED
- [ ] Orphan cleanup is logged
- [ ] Orphan cleanup does not delete workspace/artifacts (metadata only)

**Verification:**
- [ ] `go build ./pkg/session/...`
- [ ] Integration test: orphan session cleaned up on restart

**Files:** `pkg/session/orphans.go`
**Size:** S
**Status:** ✅
**Depends on:** T4.1 (metadata store), T4.2 (state machine)

## Verification Plan

- [ ] `go build ./pkg/session/...` succeeds
- [ ] Unit tests for metadata CRUD
- [ ] Unit tests for state machine transitions
- [ ] Integration test: TTL expiration triggers destroy
- [ ] Integration test: orphan cleanup on restart
- [ ] `pi box destroy --all` cleans all sandboxes

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| TTL expiration frequency not specified | §8 Session Lifecycle | Add: "Background checker runs every 60s" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Session migration between backends (e.g., fast → compat)
- Session sharing between users
- Remote session access (Milestone 6)
- Session pre-warming/pooling (future optimization)
