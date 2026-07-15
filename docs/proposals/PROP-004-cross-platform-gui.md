# PROP-004: Add Cross-Platform GUI Workbench

## Status
✅ Applied to block spec (2026-06-26)

## Block Spec Reference
`SPEC.md` §3 Non-goals, §6 Features, §7 Acceptance Criteria, §11 Out of Scope, §12 High-level architecture, §13 User-facing runtime modes, §16 Workspace model, §34 Configuration file

## Problem

The block spec currently positions PI Agent Sandbox Runtime as local-first developer tooling with a daemon, CLI, API, and SDKs. It explicitly excludes GUI/browser desktop agents and desktop mode from the first release, while leaving `desktop` as a future runtime profile.

That is correct for the MVP, but the project now needs a specified path for a cross-platform GUI workbench similar in interaction model to the provided OpenWork screenshots, adapted to PI's sandbox use case:

- A desktop onboarding flow for choosing how to connect to a local or remote daemon.
- A workspace authorization flow that makes host-folder access explicit.
- A dashboard for creating, inspecting, reusing, and stopping sandbox sessions.
- Settings for daemon connection, runtime defaults, templates, policies, and diagnostics.
- A sessions view that exposes command history, logs, diffs, artifacts, snapshots, and lifecycle state.

Without a block-spec amendment, implementation would invent product surface, configuration keys, trust boundaries, and API expectations outside the spec-first workflow.

## Proposed Amendment

Add a later milestone for a cross-platform GUI workbench. The GUI is not part of the first release and does not replace the CLI or daemon API. It is a thin local-first client over `pi-sandboxd` and remote daemon contexts.

### Feature additions

Add the following features after F23.

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F24 | Cross-Platform GUI Workbench | Desktop application for macOS, Windows, and Linux that connects to local or remote PI daemons, creates and manages sandbox sessions, and exposes common sandbox workflows without replacing the CLI | M7 |
| F25 | GUI Workspace Authorization | Explicit project-folder selection, allowed folder management, and safe bind/copy workspace setup for GUI-launched sessions | M7 |
| F26 | GUI Session Operations | Dashboard and session views for create/list/inspect/exec/logs/diff/patch/artifacts/snapshots/destroy using existing daemon API operations | M7 |
| F27 | GUI Settings and Diagnostics | GUI controls for daemon connection, active context, default template/runtime mode/network policy, engine health, doctor output, and support bundle export | M7 |

### Product shape

The GUI should use a restrained desktop-tool interface:

- Left navigation: Dashboard, Sessions, Templates, Contexts, Policies, Settings.
- Status area: active daemon/context, connection state, runtime availability, and current version.
- Primary action: create a sandbox session from a project folder, repository URL, or template.
- Dashboard: recent sessions, active sandboxes, quick-start templates, daemon health, and storage usage.
- Session detail: command runner, streaming logs, command history, file diff, patch export, artifacts, snapshots, rollback, and destroy.
- Onboarding: choose local daemon, remote context, or connect later.
- Workspace authorization: choose a project folder, review allowed folders, choose workspace mode (`copy`, `bind`, or `overlay` where available), then start or connect.
- Settings: default template, runtime mode, network mode, resource defaults, daemon source, contexts, allowed folders, and diagnostics.

The visual direction may borrow the screenshots' calm dark desktop shell, centered onboarding, explicit authorization step, sidebar dashboard, and connection/settings panels, but PI-specific text and workflows must be used.

### GUI architecture

The GUI must be implemented as a client of the existing daemon and context contracts:

- The GUI uses `pi-sandboxd` over the local Unix socket or localhost API for local operation.
- The GUI uses F22/F23 remote contexts for remote daemon operation.
- The GUI must not run sandbox workloads in the renderer/UI process.
- The GUI must not introduce a second sandbox lifecycle implementation.
- The GUI may shell out to the `pi-box` CLI only for diagnostics or compatibility gaps; normal operations should use the daemon API or SDK.
- The GUI should share TypeScript SDK types where practical to avoid drift.
- The GUI stores user preferences separately from sandbox workspaces.

Recommended app stack for the first implementation:

- Tauri or another small cross-platform desktop shell.
- TypeScript frontend.
- Rust or native host bridge only for OS integration such as file picking, tray/menu integration, and local process supervision.

### Workspace authorization contract

GUI-launched sessions must preserve the existing filesystem isolation model.

- The GUI must require explicit user selection before using any host project folder.
- The GUI must show the selected project folder before session creation.
- The GUI must default to `workspace.mode: copy`.
- `bind` mode must be an explicit per-session or per-folder opt-in.
- Allowed folders are stored as GUI preferences and may be removed by the user.
- The GUI must not mount `$HOME` by default.
- The GUI must not mount SSH keys, cloud config, Kubernetes config, or the Docker socket by default.
- Folder picker grants are advisory UI state only; the daemon remains responsible for enforcement.

### Configuration additions

Add GUI preferences under the user-level PI configuration, separate from daemon policy:

```yaml
gui:
  rememberLastConnection: true
  activeContext: local
  allowedFolders:
    - path: /Users/example/project
      defaultWorkspaceMode: copy
  defaults:
    template: node-python
    mode: auto
    network: restricted
```

Daemon policy remains authoritative. If GUI preferences conflict with daemon policy, daemon policy wins and the GUI displays the policy error.

### API requirements

The GUI should consume the existing API wherever possible. If these endpoints are not already present when F24-F27 begin, they must be specified before implementation:

- List templates and inspect template metadata.
- Report runtime/backend availability.
- Report daemon version and health.
- Return session list with lifecycle state, template, mode, workspace source, created time, TTL, and last command summary.
- Stream exec output for a selected session.
- Return command history and logs.
- Return diff, patch, artifact, snapshot, and storage usage metadata.

### Non-goals

The GUI workbench does not add these capabilities:

- Browser/computer-use agents.
- Full remote SaaS control plane.
- Multi-user enterprise identity.
- Kubernetes controller or CRDs.
- A separate sandbox runtime embedded in the desktop app.
- Direct access to host secrets outside existing daemon policy.

## Acceptance Criteria Additions

Add acceptance criteria after AC-26.

### AC-27: Cross-Platform GUI Workbench Works (F24)
- [ ] GUI app starts on macOS, Windows, and Linux development hosts
- [ ] GUI can connect to a local daemon
- [ ] GUI can connect to a configured remote context
- [ ] GUI shows connected/disconnected state and daemon version
- [ ] GUI can create a sandbox session without shelling out for normal lifecycle operations
- [ ] GUI does not implement a separate sandbox lifecycle outside `pi-sandboxd`

### AC-28: GUI Workspace Authorization Works (F25)
- [ ] User must explicitly select a project folder before GUI-launched local workspace access
- [ ] Selected project folder is displayed before session creation
- [ ] Default workspace mode is `copy`
- [ ] `bind` mode requires explicit opt-in
- [ ] Allowed folders can be listed and removed from GUI settings
- [ ] Host home directory, SSH keys, cloud config, Kubernetes config, and Docker socket are not mounted by default

### AC-29: GUI Session Operations Work (F26)
- [ ] Dashboard lists recent and active sandbox sessions
- [ ] GUI can create, inspect, and destroy sessions
- [ ] GUI can run commands with streaming stdout/stderr
- [ ] GUI displays command history, logs, exit code, duration, timeout status, and truncation status
- [ ] GUI can display workspace diff and export patch
- [ ] GUI can list and pull artifacts
- [ ] GUI can create and rollback snapshots when the daemon reports snapshot support

### AC-30: GUI Settings and Diagnostics Work (F27)
- [ ] GUI can view and change active context
- [ ] GUI can set default template, runtime mode, and network mode preferences
- [ ] GUI displays runtime/backend availability from daemon diagnostics
- [ ] GUI exposes `pi-box system doctor` equivalent results
- [ ] GUI can export a support bundle containing daemon diagnostics, GUI logs, version metadata, and redacted configuration
- [ ] Daemon policy overrides conflicting GUI preferences

## Cascade Required on Acceptance

When this PROP is accepted:

1. Update `SPEC.md` to add F24-F27, AC-27 through AC-30, and M7.
2. Move GUI workbench and desktop mode from future/out-of-scope language into later-milestone scope while keeping GUI/browser desktop agents out of M1.
3. Add feature specs for F24-F27.
4. Add an ADR for GUI architecture and trust boundaries.
5. Update `docs/features/INDEX.md` and `docs/plan.md`.
6. Review daemon/API feature specs for missing read-only metadata endpoints needed by the GUI and write follow-up PROPs if public API additions are required.
7. Mark this proposal applied in `docs/proposals/INDEX.md`.

## Implementation Blocked?

Resolved. GUI implementation may proceed through the F24-F27 feature specs.

CLI, daemon, backend, SDK, and M1-M6 work may continue unchanged because this proposal only scopes a later GUI milestone.

## Cascade completed

Completed on 2026-06-26:

- Updated `SPEC.md` with F24-F27, AC-27 through AC-30, and M7 GUI workbench scope.
- Moved GUI workbench and desktop-mode language into later-milestone scope while keeping browser/computer-use agents out of the first release.
- Added ADR-004 (GUI Workbench Architecture and Trust Boundaries).
- Added F24-F27 feature specs with acceptance criteria, tasks, dependencies, and verification plans.
- Updated `docs/features/INDEX.md` and `docs/plan.md` to start M7 planning.
- Updated `docs/proposals/INDEX.md` to mark PROP-004 applied.
