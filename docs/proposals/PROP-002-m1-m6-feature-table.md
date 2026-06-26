# PROP-002: Extract M1-M6 Milestones Into the Features Table

## Status
✅ Applied to block spec (2026-06-26)

## Block Spec Reference
`SPEC.md` §6 Features, §7 Acceptance Criteria, §31 Milestones

## Problem

`SPEC.md` §31 defines six product milestones, but `SPEC.md` §6 only extracts feature rows for M1-M3 work. M4-M6 deliverables are present in the milestone prose but are not represented in the structured Features table that drives `docs/features/F{N}-*.md`.

This means the spec-driven workflow cannot create feature specs for:

- M4 secure/gVisor backend work
- M5 microVM backend work
- M6 remote daemon mode work

There is also feature-ID drift between the current `SPEC.md` §6 table and `docs/features/INDEX.md`. For example, `SPEC.md` uses F4 for Compat Backend, while the feature docs currently use F15 for Compat Backend. Because `SPEC.md` is the master source of truth, accepting this proposal must include a cascade that realigns generated feature docs and indexes with the final feature IDs.

## Proposed Amendment

Update `SPEC.md` §6 Features so it covers all work from M1 through M6.

Keep F1-F17 as the existing M1-M3/M2 hardening surface, and add:

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F18 | Secure Backend | gVisor-backed runtime mode for unknown or untrusted repositories, including runsc integration, secure-mode lifecycle, compatibility notes, and benchmark comparison against fast/compat | M4 |
| F19 | Runtime Selection & Fallback | Runtime detection and backend selection across fast, compat, secure, and future microVM modes, including fallback to compat mode when secure mode is unavailable or incompatible | M4 |
| F20 | MicroVM Backend | Firecracker or Cloud Hypervisor backend with `pi-vmm-manager`, tiny guest rootfs, workspace disk, template snapshot restore, artifact export, and reseed-on-restore behavior | M5 |
| F21 | MicroVM Guest Control Plane | Guest-side `pi-init` and `pi-agentd` over virtio-vsock for command execution, lifecycle coordination, file/artifact transfer, and sandbox readiness reporting | M5 |
| F22 | Remote Daemon Contexts | CLI context management for local and remote daemons, including `pi context create/use/list/inspect/delete` and context-aware `pi box` commands | M6 |
| F23 | Remote Daemon Transport & Auth | SSH/Tailscale/WireGuard-friendly remote daemon access with secure local-to-remote API authentication and remote workstation support | M6 |

## Acceptance Criteria Additions

Add structured acceptance criteria for the newly extracted features.

### AC-21: Secure Backend Works (F18)
- [ ] `pi box create --mode secure <template>` creates a sandbox using gVisor/runsc when available
- [ ] Secure sandboxes execute commands through the same daemon API as fast/compat sandboxes
- [ ] Secure mode does not mount the host home directory or Docker socket by default
- [ ] Secure mode exposes compatibility errors with actionable guidance
- [ ] Benchmarks compare fast vs compat vs secure modes

### AC-22: Runtime Selection and Fallback Works (F19)
- [ ] `pi system doctor` reports available runtime backends
- [ ] Backend selection honors explicit `--mode` requests
- [ ] Auto-selection prefers an available compatible backend based on trust/config
- [ ] Secure-mode startup failure can fall back to compat mode when policy permits
- [ ] Fallback decisions are visible in logs/history

### AC-23: MicroVM Backend Works (F20)
- [ ] `pi-vmm-manager` can start and stop a microVM sandbox
- [ ] Firecracker or Cloud Hypervisor backend boots a tiny guest rootfs
- [ ] Template snapshot restore creates a ready workspace quickly
- [ ] Workspace disk persists sandbox changes for the session
- [ ] Artifact export works from microVM sandboxes
- [ ] Reseed-on-restore hook runs after snapshot restore
- [ ] Benchmarks include microVM mode comparison

### AC-24: MicroVM Guest Control Plane Works (F21)
- [ ] `pi-init` starts inside the guest and reports readiness
- [ ] `pi-agentd` communicates with the host over virtio-vsock
- [ ] Exec requests stream stdout/stderr over the guest control channel
- [ ] Guest lifecycle events map back to sandbox state
- [ ] File and artifact transfer work without direct host filesystem mounting

### AC-25: Remote Daemon Contexts Work (F22)
- [ ] `pi context create workstation ssh://gpu-box.local` creates a remote context
- [ ] `pi context use workstation` switches the active context
- [ ] `pi context list` shows local and remote contexts
- [ ] `pi box create` uses the active context
- [ ] Commands can override the active context explicitly

### AC-26: Remote Transport and Auth Work (F23)
- [ ] Remote daemon API calls are authenticated
- [ ] Remote access works over SSH-friendly transport
- [ ] Tailscale/WireGuard network paths are supported without API redesign
- [ ] Credentials are not persisted inside sandbox workspaces
- [ ] Remote workstation use case works end-to-end

## Cascade Required on Acceptance

When this PROP is accepted, apply it with the normal proposal workflow:

1. Update `SPEC.md` §6 with F18-F23.
2. Add AC-21 through AC-26 to `SPEC.md` §7.
3. Reconcile existing feature-doc IDs with `SPEC.md` §6. `SPEC.md` feature IDs win.
4. Add feature specs for F18-F23 under `docs/features/`.
5. Update `docs/features/INDEX.md` so M1-M6 are represented.
6. Update `docs/plan.md` with M4-M6 execution order and dependencies.
7. Update this proposal and `docs/proposals/INDEX.md` statuses after applying.

## Compatibility Notes

This proposal does not require implementing M4-M6 immediately. It only fixes the structured feature extraction so future work can be planned and reviewed through the same spec-driven workflow as M1-M3.

## Cascade completed

Completed on 2026-06-26:

- Updated `SPEC.md` §6 with F18-F23.
- Added AC-21 through AC-26 to `SPEC.md` §7.
- Added feature specs for F18-F23 under `docs/features/`.
- Updated `docs/features/INDEX.md` with M4-M6 rows and milestone summary.
- Updated `docs/plan.md` with M4-M6 execution phases.
- Legacy F01-F17 feature-doc ID drift was documented in `docs/features/INDEX.md`; full renaming/reconciliation is left as a follow-up cascade because it affects many existing files without changing the newly accepted PROP-002 requirements.
