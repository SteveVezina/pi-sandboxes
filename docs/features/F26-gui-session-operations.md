# F26: GUI Session Operations

> Source: `SPEC.md` §6 Features F26
> Status: 🟡 Spec written
> Category: Client / Integration

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F26 | GUI Session Operations | Dashboard and session views for create/list/inspect/exec/logs/diff/patch/artifacts/snapshots/destroy using existing daemon API operations | M7 |

## Expanded Specification

The GUI provides operational views for active and recent sandbox sessions. Users can inspect session state, run commands with streaming output, review command history and logs, inspect diffs, export patches, list and pull artifacts, create snapshots, roll back snapshots, and destroy sessions.

These operations use existing daemon API operations. The GUI displays feature availability when the daemon reports that a capability is unavailable.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-29.1: Dashboard lists recent and active sandbox sessions
- [ ] AC-29.2: GUI can create, inspect, and destroy sessions
- [ ] AC-29.3: GUI can run commands with streaming stdout/stderr
- [ ] AC-29.4: GUI displays command history, logs, exit code, duration, timeout status, and truncation status
- [ ] AC-29.5: GUI can display workspace diff and export patch
- [ ] AC-29.6: GUI can list and pull artifacts
- [ ] AC-29.7: GUI can create and rollback snapshots when the daemon reports snapshot support

## Interface Impact

| Component | Impact |
|-----------|--------|
| `apps/gui/` | Dashboard, session detail, command runner, logs, diff, artifacts, snapshots (new — to be created) |
| `pi-sandboxd` API | Source of session operations and streaming exec |
| `sdk/typescript/` | Shared client/types for session operations |

## Security Considerations

- Command execution still occurs in sandbox sessions through the daemon.
- Artifact pulls must use daemon export/pull behavior and not bypass workspace policy.
- Snapshot rollback must clearly identify the target session and snapshot before execution.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F5: Command Execution | Internal feature | Implemented |
| F6: Workspace & File Operations | Internal feature | Implemented |
| F8: Artifact Export | Internal feature | Implemented |
| F9: Logs & Command History | Internal feature | Implemented |
| F14: Snapshot & Rollback | Internal feature | Implemented |
| F24: Cross-Platform GUI Workbench | Internal feature | Spec written |
| ADR-004 | Architecture decision | Accepted |

## Implementation Approach

Build GUI session state around daemon API responses and streaming channels. The dashboard should favor scannable operational state. Session detail should group command execution, history/logs, file changes, artifacts, and snapshots without duplicating daemon logic.

**ADR references:** ADR-004 (GUI Workbench Architecture and Trust Boundaries).
**ADR gaps:** None.

## Tasks

### T26.1: Dashboard session list ⚠️

**Description:** Show recent and active sessions with lifecycle state, template, mode, workspace source, created time, TTL, and last command summary where available.

**Acceptance criteria:**
- [x] Dashboard lists active sessions
- [x] Dashboard lists recent sessions when available
- [x] Session rows show lifecycle state and key metadata

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [x] Live daemon smoke verifies list/inspect rendering

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`, `apps/gui/src/styles.css`
**Size:** M
**Depends on:** F24

### T26.2: Command runner and streaming logs ⚠️

**Description:** Add session command execution with streaming stdout/stderr and command result metadata.

**Acceptance criteria:**
- [x] GUI can run commands in a selected session
- [ ] stdout/stderr stream while command runs
- [x] Exit code, duration, timeout, and truncation state are visible

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [ ] Streaming exec integration test against mock daemon

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`
**Size:** M
**Depends on:** T26.1, F5

### T26.3: Diff, patch, artifacts, and snapshots ⚠️

**Description:** Add operational panels for workspace diff, patch export, artifact pull, snapshot creation, and rollback.

**Acceptance criteria:**
- [x] GUI displays workspace diff
- [x] GUI can request patch export
- [x] GUI lists artifacts
- [x] GUI lists snapshots when supported
- [ ] GUI pulls artifacts
- [ ] GUI creates and rolls back snapshots when supported

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [ ] Mock daemon integration tests for diff/patch/artifact/snapshot actions

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`
**Size:** M
**Depends on:** T26.2, F6, F8, F14

## Verification Plan

- [ ] Mock daemon tests cover session list, inspect, exec stream, logs, diff, patch, artifacts, snapshots, and destroy
- [ ] Manual smoke verifies a local session end-to-end from create through destroy

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

- Implementing daemon session operations in the GUI
- Browser/computer-use agents
