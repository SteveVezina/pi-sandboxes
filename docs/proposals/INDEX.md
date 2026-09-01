# Proposals Index

Track all PROP (spec amendment proposals) for this project.

## Legend

| Status | Meaning |
|--------|---------|
| 🟡 Proposed | Written, awaiting human review |
| ✅ Accepted | Human approved, ready to apply |
| ✅ Applied to block spec | Cascade complete |
| ❌ Withdrawn | Dropped by author |
| 🔴 Rejected | Human rejected |

## Active Proposals

| PROP | Title | Status | Impact | Created |
|------|-------|--------|--------|---------|
| [PROP-001](PROP-001-spec-format.md) | Add Structured Block Spec Schema to SPEC.md | ✅ Applied to block spec | All features (enables spec-driven workflow) | 2026-06-26 |
| [PROP-002](PROP-002-m1-m6-feature-table.md) | Extract M1-M6 Milestones Into the Features Table | ✅ Applied to block spec | M4-M6 feature extraction and feature-doc ID cascade | 2026-06-26 |
| [PROP-003](PROP-003-f20-f23-contracts.md) | Specify F20-F23 MicroVM and Remote Contracts | ✅ Applied to block spec | Unblocks F20-F23 implementation | 2026-06-26 |
| [PROP-004](PROP-004-cross-platform-gui.md) | Add Cross-Platform GUI Workbench | ✅ Applied to block spec | Adds later M7 GUI features F24-F27 and AC-27 through AC-30 | 2026-06-26 |
| [PROP-005](PROP-005-pi-box-home.md) | Move Pi Box Home Out of `~/.pi` | ✅ Applied to block spec | Moves sandbox runtime default state root to `~/.pi-box` to avoid Pi coding agent collisions | 2026-07-15 |
| [PROP-006](PROP-006-local-template-library.md) | Add Local Template Library and Lifecycle | 🟡 Proposed (revised 2026-08-31) | Adds later F28/**AC-35** for rich template metadata, local fork/snapshot/import/export/history/diff/rollback/promote workflows, and GUI template management. Reconciled with applied PROP-007/008/009 | 2026-07-15 |
| [PROP-007](PROP-007-image-resolution.md) | Resolve Template `base` Field to OCI Image Name | ✅ Applied to block spec (2026-07-15) | Fixes compat mode sandbox creation — template `base: debian-slim` must resolve to `docker.io/library/debian:bookworm-slim` before container creation | 2026-07-15 |
| [PROP-008](PROP-008-runtime-driver-contract.md) | Formal Runtime Driver Contract and Shared OCI Engine | ✅ Complete (2026-07-15) | All P0-P3 tasks done: driver contract, capability reports, shared OCI engine, selection engine, compat hardening, secure backend rebuild, lifecycle recovery | 2026-07-14 |
| [PROP-009](PROP-009-agent-loop-first.md) | Agent-Loop-First Re-Aim — Rename Sessions to Sandboxes, One Output Channel, Host Decoupling | ✅ Applied to block spec (2026-07-15) | Renames "session" → "sandbox" (spec/docs, stream frame field), adds F29 Agent Run and F30 Egress Proxy, consolidates artifact/patch export into one output channel, removes host cache bind mounts and host-disk secrets | 2026-07-14 |

## Rules

- **NEVER** modify `SPEC.md` directly — always use a PROP.
- **NEVER** create a new PROP while unapproved proposals exist (unless BLOCKING).
- **NEVER** skip numbering — next PROP = highest existing + 1.
- After acceptance: apply via `pi-apply-prop` skill, which cascades into ADRs, feature specs, and INDEX files.
