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

## Rules

- **NEVER** modify `SPEC.md` directly — always use a PROP.
- **NEVER** create a new PROP while unapproved proposals exist (unless BLOCKING).
- **NEVER** skip numbering — next PROP = highest existing + 1.
- After acceptance: apply via `pi-apply-prop` skill, which cascades into ADRs, feature specs, and INDEX files.
