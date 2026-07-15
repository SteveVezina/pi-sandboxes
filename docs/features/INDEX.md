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
| [F1](F01-cli-entry-point.md) | CLI Entry Point | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | — |
| [F2](F02-daemon-api.md) | Daemon API | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | F1 |
| [F3](F03-fast-backend.md) | Fast Backend | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | F8 |
| [F4](F15-compat-backend.md) | Compat Backend | 🟢 Reviewed | ✅ Implemented | M1 | F8, F5 |
| [F5](F07-template-system.md) | Template System | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | F3, F4 |
| [F6](F06-workspace-file-ops.md) | Workspace & File Operations | 🟢 Reviewed | ✅ Implemented | M1 | F8 |
| [F7](F05-command-execution.md) | Command Execution | 🟢 Reviewed | ✅ Implemented | M1 | F3, F8 |
| [F8](F04-session-lifecycle.md) | Session Lifecycle | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | — |
| [F9](F08-artifact-export.md) | Artifact Export | 🟢 Reviewed | ✅ Implemented | M1 | F6 |
| [F10](F09-logs-history.md) | Logs & Command History | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | F7 |
| [F11](F12-secrets-network.md) | Secrets & Network Model | 🟢 Reviewed | ✅ Implemented | M2 | F17 |
| [F12](F13-cache-model.md) | Cache Model | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M2 | F5, F16 |
| [F13](F14-snapshot-rollback.md) | Snapshot & Rollback | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M2 | F8 |
| [F14](F11-benchmarks.md) | Benchmarks | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | F3, F4, F13 |
| [F15](F16-sdk.md) | SDKs | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M3 | F2 |
| [F16](F10-system-commands.md) | System Commands | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M1 | F8 |
| [F17](F17-policy-enforcement.md) | Policy Enforcement | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M2 | F3, F4, F11 |
| [F18](F18-secure-backend.md) | Secure Backend | 🟢 Reviewed | ✅ Implemented | M4 | F4, F17, F19 |
| [F19](F19-runtime-selection-fallback.md) | Runtime Selection & Fallback | 🟢 Reviewed | ✅ Implemented | M4 | F3, F4, F18 |
| [F20](F20-microvm-backend.md) | MicroVM Backend | 🟢 Reviewed | ✅ Implemented | M5 | F19, F21 |
| [F21](F21-microvm-guest-control-plane.md) | MicroVM Guest Control Plane | 🟢 Reviewed | ✅ Implemented | M5 | F20 |
| [F22](F22-remote-daemon-contexts.md) | Remote Daemon Contexts | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M6 | F2, F23 |
| [F23](F23-remote-daemon-transport-auth.md) | Remote Daemon Transport & Auth | 🟢 Reviewed | ✅ Implemented | M6 | F2, F15, F22 |
| [F24](F24-cross-platform-gui-workbench.md) | Cross-Platform GUI Workbench | 🟢 Reviewed | ✅ Implemented | M7 | F2, F15, F22, F23 |
| [F25](F25-gui-workspace-authorization.md) | GUI Workspace Authorization | 🟢 Reviewed | ✅ Implemented | M7 | F17, F24 |
| [F26](F26-gui-session-operations.md) | GUI Session Operations | 🟢 Reviewed | ✅ Implemented | M7 | F7, F6, F9, F10, F13, F24 |
| [F27](F27-gui-settings-diagnostics.md) | GUI Settings and Diagnostics | 🟢 Reviewed | ✅ Implemented | M7 | F16, F17, F19, F22, F24, F25 |

## Milestone Summary

| Milestone | Features | Status |
|-----------|----------|--------|
| M1: Local Linux MVP | F1, F2, F3, F4, F5, F6, F7, F8, F9, F10, F14, F16 | ⚠️ Needs re-verify |
| M2: Hardening & Cache | F11, F12, F13, F17 | ⚠️ Needs re-verify |
| M3: Agent Integrations | F15 | ⚠️ Needs re-verify |
| M4: Secure Backend | F18, F19 | ✅ Implemented |
| M5: MicroVM Backend | F20, F21 | ✅ Implemented |
| M6: Remote Daemon Mode | F22, F23 | ⚠️ Needs re-verify |
| M7: Cross-Platform GUI Workbench | F24, F25, F26, F27 | ✅ Implemented |

## Summary

All 27 feature specs tracked. PROP-005 moved the default Pi Box home to `~/.pi-box`; affected implemented features are marked for re-verification.

> Note: some legacy filenames predate PROP-002, but the feature labels, titles, source IDs, and index rows now use the canonical `SPEC.md` §6 feature IDs.
