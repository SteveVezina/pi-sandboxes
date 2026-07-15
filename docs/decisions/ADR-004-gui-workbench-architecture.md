# ADR-004: GUI Workbench Architecture and Trust Boundaries

## Status
Accepted

## Date
2026-06-26

## Scope
- **Topology:** local and remote daemon contexts
- **Question type:** framework and trust boundary
- **Does NOT answer:** browser/computer-use agent runtime design, hosted control plane design, or enterprise identity
- **Depends on / extends:** ADR-003 (Remote Context and Auth Model)

## Context

PROP-004 adds an M7 cross-platform GUI workbench for PI Agent Sandbox Runtime. The GUI needs to present onboarding, workspace authorization, sandbox operations, settings, diagnostics, and support bundle export while preserving the block's local-first daemon architecture.

The primary risk is accidentally creating a second sandbox lifecycle in the desktop app or weakening the daemon's filesystem and secret isolation model through UI convenience features.

## Options Considered

### Option A: Thin GUI client over `pi-sandboxd`
- **Pros:** Reuses daemon API, SDK contracts, remote contexts, policy enforcement, and existing sandbox lifecycle.
- **Cons:** Requires daemon metadata endpoints to be complete enough for a good GUI.

### Option B: Desktop app embeds its own sandbox runtime
- **Pros:** Could simplify first-run local demos.
- **Cons:** Duplicates lifecycle behavior, risks security drift, and violates the daemon as source-of-truth architecture.

### Option C: CLI-wrapper GUI
- **Pros:** Fast to prototype around existing commands.
- **Cons:** Harder to stream structured state, weaker error handling, and likely to drift from SDK/API contracts.

## Decision

Use Option A.

The GUI workbench is a thin client over `pi-sandboxd` and the existing F22/F23 context model. It may use the TypeScript SDK or daemon API directly for normal operations. It may shell out to `pi-box` only for diagnostics or compatibility gaps.

The preferred first implementation stack is Tauri or another small cross-platform desktop shell with a TypeScript frontend. Rust or native host code is limited to OS integration such as file picking, tray/menu behavior, local process supervision, and support bundle collection.

The GUI must not:

- run sandbox workloads in the UI process
- implement a separate sandbox lifecycle
- mount host folders directly without daemon policy enforcement
- inject host secrets into sandbox workspaces
- treat folder picker grants as authoritative security policy

Daemon policy remains authoritative. GUI preferences are client hints and defaults only.

## Consequences

- F24-F27 can be implemented without changing the runtime backend model.
- Missing read-only daemon metadata endpoints must be specified before code if the GUI needs them.
- Workspace authorization remains explicit and user-visible, while enforcement stays in the daemon.
- The GUI can support local and remote daemons through the same context/auth model used by the CLI and SDKs.

## References

- `SPEC.md` §6 Features F24-F27
- `SPEC.md` §7 AC-27 through AC-30
- `SPEC.md` §31 Milestone 7
- `docs/features/F24-cross-platform-gui-workbench.md`
- `docs/features/F25-gui-workspace-authorization.md`
- `docs/features/F26-gui-sandbox-operations.md`
- `docs/features/F27-gui-settings-diagnostics.md`
- PROP-004
