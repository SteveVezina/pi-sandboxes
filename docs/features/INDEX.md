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
| [F01](F01-cli-entry-point.md) | CLI Entry Point | 🟡 Written | ✅ Implemented | M1 | — |
| [F02](F02-daemon-api.md) | Daemon API | 🟡 Written | ✅ Implemented | M1 | F01 |
| [F03](F03-fast-backend.md) | Fast Backend | 🟡 Written | ✅ Implemented | M1 | F04 |
| [F04](F04-session-lifecycle.md) | Session Lifecycle | 🟢 Implemented | ✅ Implemented | M1 | — |
| [F05](F05-command-execution.md) | Command Execution | 🟡 Written | ✅ Implemented | M1 | F03, F04 |
| [F06](F06-workspace-file-ops.md) | Workspace & File Ops | 🟡 Written | ✅ Implemented | M1 | F04 |
| [F07](F07-template-system.md) | Template System | 🟡 Written | ✅ Implemented | M1 | F03, F04 |
| [F08](F08-artifact-export.md) | Artifact Export | 🟡 Written | ✅ Implemented | M1 | F06 |
| [F09](F09-logs-history.md) | Logs & History | 🟡 Written | ✅ Implemented | M1 | F05 |
| [F10](F10-system-commands.md) | System Commands | 🟡 Written | ✅ Implemented | M1 | F04 |
| [F11](F11-benchmarks.md) | Benchmarks | 🟡 Written | ✅ Implemented | M1 | F03, F04, F13 |
| [F12](F12-secrets-network.md) | Secrets & Network | 🟡 Written | ✅ Implemented | M2 | F17 |
| [F13](F13-cache-model.md) | Cache Model | 🟡 Written | ✅ Implemented | M2 | F07, F10 |
| [F14](F14-snapshot-rollback.md) | Snapshot & Rollback | 🟡 Written | ✅ Implemented | M2 | F04 |
| [F15](F15-compat-backend.md) | Compat Backend | 🟡 Written | ✅ Implemented | M2 | F04, F07 |
| [F16](F16-sdk.md) | SDKs | 🟡 Written | ✅ Implemented | M3 | F02 |
| [F17](F17-policy-enforcement.md) | Policy Enforcement | 🟡 Written | ✅ Implemented | M2 | F03, F04, F12 |
| [F18](F18-secure-backend.md) | Secure Backend | 🟡 Written | 🔴 Not started | M4 | F15, F17, F19 |
| [F19](F19-runtime-selection-fallback.md) | Runtime Selection & Fallback | 🟡 Written | 🔴 Not started | M4 | F03, F15, F18 |
| [F20](F20-microvm-backend.md) | MicroVM Backend | 🟡 Written | 🔴 Not started | M5 | F19, F21 |
| [F21](F21-microvm-guest-control-plane.md) | MicroVM Guest Control Plane | 🟡 Written | 🔴 Not started | M5 | F20 |
| [F22](F22-remote-daemon-contexts.md) | Remote Daemon Contexts | 🟡 Written | 🔴 Not started | M6 | F02, F23 |
| [F23](F23-remote-daemon-transport-auth.md) | Remote Daemon Transport & Auth | 🟡 Written | 🔴 Not started | M6 | F02, F16, F22 |

## Milestone Summary

| Milestone | Features | Status |
|-----------|----------|--------|
| M1: Local Linux MVP | F01, F02, F03, F04, F05, F06, F07, F08, F09, F10, F11 | ✅ Implemented |
| M2: Hardening & Cache | F12, F13, F14, F15, F17 | ✅ Implemented |
| M3: Agent Integrations | F16 | ✅ Implemented |
| M4: Secure Backend | F18, F19 | 🔴 Not started |
| M5: MicroVM Backend | F20, F21 | 🔴 Not started |
| M6: Remote Daemon Mode | F22, F23 | 🔴 Not started |

## Summary

23 feature specs tracked. F18-F23 were extracted from `SPEC.md` §31 by PROP-002 and are not started.

> Note: existing feature document numbering predates PROP-002 and has known drift from `SPEC.md` §6. `SPEC.md` remains the source of truth; a follow-up cascade should reconcile legacy F01-F17 filenames and IDs.
