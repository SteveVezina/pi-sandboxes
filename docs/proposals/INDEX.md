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

## Rules

- **NEVER** modify `SPEC.md` directly — always use a PROP.
- **NEVER** create a new PROP while unapproved proposals exist (unless BLOCKING).
- **NEVER** skip numbering — next PROP = highest existing + 1.
- After acceptance: apply via `pi-apply-prop` skill, which cascades into ADRs, feature specs, and INDEX files.
