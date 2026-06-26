# Features Index

Dashboard of all features for this project.

## Legend

| Status | Meaning |
|--------|---------|
| 🔴 Not started | No spec yet |
| 🟡 Spec written | Spec exists, not reviewed |
| 🟢 Reviewed | Spec approved, ready for planning |
| 🔵 In progress | Tasks being executed |
| ⚠️ Needs re-verify | Spec changed, tasks need re-verification |
| ✅ Implemented | All tasks done |
| ⏸️ Blocked | Blocked on proposal or dependency |

## Features

| Feature | Name | Spec Status | Impl Status | Milestone | Depends On |
|---------|------|-------------|-------------|-----------|------------|
| [F01](F01-cli-entry-point.md) | CLI Entry Point | 🟡 Written | 🔴 Not started | M1 | — |
| [F02](F02-daemon-api.md) | Daemon API | 🟡 Written | 🔴 Not started | M1 | F01 |
| [F03.*🟡 Written.*✅ Implemented | M1 | F04 |
| [F04.*🟢 Implemented.*✅ Implemented | M1 | — |
| [F05](F05-command-execution.md) | Command Execution | 🟡 Written | 🔴 Not started | M1 | F03, F04 |
| [F06](F06-workspace-file-ops.md) | Workspace & File Ops | 🟡 Written | 🔴 Not started | M1 | F04 |
| [F07](F07-template-system.md) | Template System | 🟡 Written | 🔴 Not started | M1 | F03, F04 |
| [F08](F08-artifact-export.md) | Artifact Export | 🟡 Written | 🔴 Not started | M1 | F06 |
| [F09](F09-logs-history.md) | Logs & History | 🟡 Written | 🔴 Not started | M1 | F05 |
| [F10](F10-system-commands.md) | System Commands | 🟡 Written | 🔴 Not started | M1 | F04 |
| [F11](F11-benchmarks.md) | Benchmarks | 🟡 Written | 🔴 Not started | M1 | F03, F04, F13 |
| [F12](F12-secrets-network.md) | Secrets & Network | 🟡 Written | 🔴 Not started | M2 | F17 |
| [F13](F13-cache-model.md) | Cache Model | 🟡 Written | 🔴 Not started | M2 | F07, F10 |
| [F14](F14-snapshot-rollback.md) | Snapshot & Rollback | 🟡 Written | 🔴 Not started | M2 | F04 |
| [F15](F15-compat-backend.md) | Compat Backend | 🟡 Written | 🔴 Not started | M2 | F04, F07 |
| [F16](F16-sdk.md) | SDKs | 🟡 Written | 🔴 Not started | M3 | F02 |
| [F17](F17-policy-enforcement.md) | Policy Enforcement | 🟡 Written | 🔴 Not started | M2 | F03, F04, F12 |

## Milestone Summary

| Milestone | Features | Status |
|-----------|----------|--------|
| M1: Local Linux MVP | F01, F02, F03, F04, F05, F06, F07, F08, F09, F10, F11 | 🔴 Not started |
| M2: Hardening & Cache | F12, F13, F14, F15, F17 | 🔴 Not started |
| M3: Agent Integrations | F16 | 🔴 Not started |
