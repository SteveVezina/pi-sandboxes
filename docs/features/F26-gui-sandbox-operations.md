# F26: GUI Sandbox Operations

> Source: `SPEC.md` §6 Features F26
> Status: 🟢 Reviewed — re-verified 2026-08-31. `apps/gui/src/api.ts` covers every F26 operation (list/get/create/destroy/clone/exec/execStream/logs/logsHistory/diff/patch/output list+pull+pack/snapshot list+create+rollback+delete) and every route matches the current daemon router (`/output`, `/logs/history`, `/snapshot/*`) — no stale `session` terminology, no removed `/artifacts/*` or `/exec/ws` routes. `network` is passed through on exec (AC-29.8). GUI builds clean (`pnpm --filter @pi-sandbox/gui build` — tsc + vite) via the pnpm/turbo workspace.
> Category: Client / Integration

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F26 | GUI Sandbox Operations | Dashboard and sandbox views for create/list/inspect/exec/logs/diff/patch/output deliverables/snapshots/destroy using existing daemon API operations | M7 |

## Expanded Specification

The GUI provides operational views for active and recent sandboxes. Users can inspect sandbox state, run commands with streaming output, review command history and logs, inspect diffs, export patches, list and pull output deliverables, create snapshots, roll back snapshots, and destroy sandboxes.

These operations use existing daemon API operations. The GUI displays feature availability when the daemon reports that a capability is unavailable.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-29.1: Dashboard lists recent and active sandboxes
- [x] AC-29.2: GUI can create, inspect, and destroy sandboxes
- [x] AC-29.3: GUI can run commands with streaming stdout/stderr
- [x] AC-29.4: GUI displays command history, logs, exit code, duration, timeout status, and truncation status
- [x] AC-29.5: GUI can display workspace diff and export patch
- [x] AC-29.6: GUI can list and pull output deliverables
- [x] AC-29.7: GUI can create and rollback snapshots when the daemon reports snapshot support
- [x] AC-29.8: GUI command execution sends the selected network mode through the daemon exec API

## Interface Impact

| Component | Impact |
|-----------|--------|
| `apps/gui/` | Dashboard, sandbox detail, command runner, logs, diff, output deliverables, snapshots |
| `pi-sandboxd` API | Source of sandbox operations and streaming exec |
| `sdk/typescript/` | Shared client/types for sandbox operations |
| Sandbox lifecycle state | GUI action availability and daemon conflict responses |

## Security Considerations

- Command execution still occurs in sandboxes through the daemon.
- Output pulls must use daemon output-channel behavior and not bypass workspace policy.
- Snapshot rollback must clearly identify the target sandbox and snapshot before execution.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F5: Command Execution | Internal feature | Implemented |
| F6: Workspace & File Operations | Internal feature | Implemented |
| F8: Output Delivery | Internal feature | Needs re-verify |
| F9: Logs & Command History | Internal feature | Implemented |
| F14: Snapshot & Rollback | Internal feature | Implemented |
| F24: Cross-Platform GUI Workbench | Internal feature | Implemented |
| ADR-004 | Architecture decision | Accepted |

## Implementation Approach

Build GUI sandbox state around daemon API responses and streaming channels. The dashboard should favor scannable operational state. Sandbox detail should group command execution, history/logs, file changes, output deliverables, and snapshots without duplicating daemon logic.

**ADR references:** ADR-004 (GUI Workbench Architecture and Trust Boundaries).
**ADR gaps:** None.

## Tasks

### T26.1: Dashboard sandbox list ⚠️

**Description:** Show recent and active sandboxes with lifecycle state, template, mode, workspace source, created time, TTL, and last command summary where available. *(2026-07-15: terminology updated per PROP-009; GUI labels and data model need re-verification.)*

**Acceptance criteria:**
- [x] Dashboard lists active sandboxes
- [x] Dashboard lists recent sandboxes when available
- [x] Sandbox rows show lifecycle state and key metadata

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [x] Live daemon smoke verifies list/inspect rendering

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`, `apps/gui/src/styles.css`
**Size:** M
**Depends on:** F24

### T26.2: Command runner and streaming logs ⚠️

**Description:** Add sandbox command execution with streaming stdout/stderr and command result metadata. *(2026-07-15: lifecycle terminology updated per PROP-009.)*

**Acceptance criteria:**
- [x] GUI can run commands in a selected sandbox
- [x] stdout/stderr stream while command runs
- [x] Exit code, duration, timeout, and truncation state are visible
- [x] Selected network mode is included in streaming exec requests

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [x] Live daemon NDJSON smoke verifies stdout and final done event
- [x] API tests verify exec accepts valid network modes and rejects invalid network modes

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`
**Size:** M
**Depends on:** T26.1, F5

### T26.3: Diff, patch, output deliverables, and snapshots ⚠️

**Description:** Add operational panels for workspace diff, patch export, output delivery, snapshot creation, and rollback. *(2026-07-15: output channel replaces artifact pull/export per PROP-009.)*

**Acceptance criteria:**
- [x] GUI displays workspace diff
- [x] GUI can request patch export
- [x] GUI lists output deliverables
- [x] GUI lists snapshots when supported
- [x] GUI pulls output deliverables
- [x] GUI creates and rolls back snapshots when supported

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [x] Live daemon smoke verifies output list/pull and snapshot create/list/rollback routes

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`
**Size:** M
**Depends on:** T26.2, F6, F8, F14

### T26.4: State-gated sandbox controls ⚠️

**Description:** Gate GUI actions by daemon lifecycle state so users cannot start commands or destructive workspace operations while a sandbox is creating, executing, destroying, or destroyed. *(2026-07-15: terminology updated per PROP-009.)*

**Acceptance criteria:**
- [x] Exec and clone controls are enabled only when the selected sandbox is `WARM`
- [x] Output pull/pack and snapshot create/rollback/delete controls are enabled only when the selected sandbox is `WARM`
- [x] Destroy is disabled for `DESTROYING` and `DESTROYED` sandboxes
- [x] Sandbox detail explains why controls are disabled for non-ready states
- [x] Daemon exec and shell routes reject non-`WARM` sandboxes with HTTP 409 before starting execution

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [x] API tests verify exec rejects non-ready sandbox state

**Files:** `apps/gui/src/main.tsx`, `apps/gui/src/styles.css`, `pkg/api/sandbox_exec.go`, `pkg/api/sandbox_shell.go`, `tests/api/exec_test.go`
**Size:** S
**Depends on:** T26.2, F4

## Verification Plan

- [x] Live daemon smoke covers sandbox list, inspect, exec stream, logs, diff, patch, output deliverables, snapshots, and destroy-capable routes
- [x] Manual smoke verifies local sandbox rendering against the daemon

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

- Implementing daemon sandbox operations in the GUI
- Browser/computer-use agents
