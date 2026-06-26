# F24: Cross-Platform GUI Workbench

> Source: `SPEC.md` §6 Features F24
> Status: 🟡 Spec written
> Category: Client / Integration

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F24 | Cross-Platform GUI Workbench | Desktop application for macOS, Windows, and Linux that connects to local or remote PI daemons, creates and manages sandbox sessions, and exposes common sandbox workflows without replacing the CLI | M7 |

## Expanded Specification

The GUI workbench is a cross-platform desktop client for PI Agent Sandbox Runtime. It provides onboarding, connection status, top-level navigation, and the primary workflow for creating sandbox sessions from a project folder, repository URL, or template.

The GUI is a client surface over `pi-sandboxd`. It uses the local daemon API, SDKs, and F22/F23 remote context contracts rather than embedding a second runtime. The first screen should be the usable workbench or onboarding flow, not a marketing page.

The visual direction is a calm desktop-tool UI: left navigation, prominent connection state, recent/active sessions, quick-start templates, and a primary create-session action. Text and workflows must be PI-specific.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-27.1: GUI app starts on macOS, Windows, and Linux development hosts
- [ ] AC-27.2: GUI can connect to a local daemon
- [ ] AC-27.3: GUI can connect to a configured remote context
- [ ] AC-27.4: GUI shows connected/disconnected state and daemon version
- [ ] AC-27.5: GUI can create a sandbox session without shelling out for normal lifecycle operations
- [ ] AC-27.6: GUI does not implement a separate sandbox lifecycle outside `pi-sandboxd`

## Interface Impact

| Component | Impact |
|-----------|--------|
| `apps/gui/` | Cross-platform desktop GUI (new — to be created) |
| `sdk/typescript/` | Preferred shared client/types for GUI operations |
| `pi-sandboxd` API | Source of lifecycle, daemon health, and session state |
| F22/F23 contexts | Remote daemon selection and auth |

## Security Considerations

- GUI connection state must not imply that sandbox policy has been weakened.
- Remote credentials stay in the existing context/auth model and outside sandbox workspaces.
- The GUI must not run sandbox workloads in the renderer/UI process.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F2: Daemon API | Internal feature | Implemented |
| F16: SDKs | Internal feature | Implemented |
| F22: Remote Daemon Contexts | Internal feature | Implemented |
| F23: Remote Daemon Transport & Auth | Internal feature | Implemented |
| ADR-004 | Architecture decision | Accepted |

## Implementation Approach

Use a thin desktop shell around a TypeScript frontend. The preferred stack is Tauri or another small cross-platform shell. Normal operations use the daemon API or TypeScript SDK. Shelling out to `pi` is limited to diagnostics or compatibility gaps.

**ADR references:** ADR-003 (Remote Context and Auth Model), ADR-004 (GUI Workbench Architecture and Trust Boundaries).
**ADR gaps:** None.

## Tasks

### T24.1: GUI app scaffold and shell ✅

**Description:** Create the cross-platform desktop application scaffold with top-level navigation, app chrome, and build/dev commands.

**Acceptance criteria:**
- [x] App starts locally in development mode
- [x] App defines Dashboard, Sessions, Templates, Contexts, Policies, and Settings navigation
- [x] App shell renders without requiring a daemon connection

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [x] Browser smoke screenshots verify nonblank app shell at desktop and mobile widths

**Files:** `apps/gui/package.json`, `apps/gui/index.html`, `apps/gui/tsconfig.json`, `apps/gui/vite.config.ts`, `apps/gui/src/main.tsx`, `apps/gui/src/styles.css`
**Size:** M
**Depends on:** None

### T24.2: Daemon and context connection ⚠️

**Description:** Connect the GUI to local daemon and configured remote contexts through existing API/SDK contracts.

**Acceptance criteria:**
- [ ] Local daemon connection works
- [ ] Remote context connection works
- [ ] Connected/disconnected state and daemon version are visible
- [ ] Auth failures are actionable and do not fall back to unauthenticated access

**Verification:**
- [ ] Unit tests for connection state
- [ ] Integration smoke against local daemon API

**Files:** `apps/gui/ (new — to be created)`, `sdk/typescript/src/client.ts`
**Size:** M
**Depends on:** T24.1, F22, F23

### T24.3: Create-session entry flow ⚠️

**Description:** Add the primary create-session flow for project folder, repository URL, and template starts.

**Acceptance criteria:**
- [ ] User can start a sandbox session through daemon API/SDK calls
- [ ] Normal lifecycle creation does not shell out to `pi`
- [ ] Created session appears in dashboard state

**Verification:**
- [ ] GUI integration test creates a session against a mock daemon

**Files:** `apps/gui/ (new — to be created)`
**Size:** M
**Depends on:** T24.2, F25

## Verification Plan

- [ ] GUI app starts on all supported host platforms in CI or release smoke checks
- [ ] Local and remote daemon connection states are covered by tests
- [ ] Create-session flow is verified against mock and local daemon targets

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

- Browser/computer-use agents
- Hosted control plane
- A separate sandbox runtime embedded in the GUI
