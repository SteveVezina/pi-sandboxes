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
| [F1](F01-cli-entry-point.md) | CLI Entry Point | 🟢 Reviewed | ✅ Implemented | M1 | — |
| [F2](F02-daemon-api.md) | Daemon API | 🟢 Reviewed | ✅ Implemented | M1 | F1 |
| [F3](F03-fast-backend.md) | Fast Backend | 🟢 Reviewed | ✅ Implemented | M1 | F8 |
| [F4](F15-compat-backend.md) | Compat Backend | 🟢 Reviewed | ✅ Implemented | M1 | F8, F5 |
| [F5](F07-template-system.md) | Template System | 🟢 Reviewed | ✅ Implemented | M1 | F3, F4 |
| [F6](F06-workspace-file-ops.md) | Workspace & File Operations | 🟢 Reviewed | ✅ Implemented | M1 | F8, F9 |
| [F7](F05-command-execution.md) | Command Execution | 🟢 Reviewed | ✅ Implemented | M1 | F3, F8 |
| [F8](F04-sandbox-lifecycle.md) | Sandbox Lifecycle | 🟢 Reviewed | ✅ Implemented | M1 | — |
| [F9](F08-output-delivery.md) | Output Delivery | 🟢 Reviewed | 🟡 list/pull/pack done (event emission, size validation open) | M1 | F6 |
| [F10](F09-logs-history.md) | Logs & Command History | 🟢 Reviewed | ✅ Implemented | M1 | F7 |
| [F11](F12-secrets-network.md) | Secrets & Network Model | 🟢 Reviewed (ADR-006 Accepted) | 🔴 Not started — enforcement tracked as F30 T30.1–T30.8 | M2 | F17, F30 |
| [F12](F13-cache-model.md) | Cache Model | 🟡 Re-verified 2026-08-31 | 🟡 mounts host-bind-free; per-sandbox-ID scoping (no reuse), `pkg/cache` unwired, no shared-layer/overlay | M2 | F5, F16 |
| [F13](F14-snapshot-rollback.md) | Snapshot & Rollback | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M2 | F8 |
| [F14](F11-benchmarks.md) | Benchmarks | 🟢 Reviewed | ✅ Implemented (re-verified 2026-08-31; cached-install threshold benchmarks depend on F13 cache-scoping fix) | M1 | F3, F4, F13 |
| [F15](F16-sdk.md) | SDKs | 🟢 Reviewed | ✅ Implemented (re-verified 2026-08-31) | M3 | F2 |
| [F16](F10-system-commands.md) | System Commands | 🟢 Reviewed | ✅ Implemented (re-verified 2026-08-31) | M1 | F8 |
| [F17](F17-policy-enforcement.md) | Policy Enforcement | 🟡 Re-verified 2026-08-31 | 🟡 host-mount + limits enforced; AC-17.5/AC-34.3 delivered by F30 T30.7–T30.8 (ADR-006) | M2 | F3, F4, F11, F30 |
| [F18](F18-secure-backend.md) | Secure Backend | 🟢 Reviewed | ✅ Implemented | M4 | F4, F17, F19 |
| [F19](F19-runtime-selection-fallback.md) | Runtime Selection & Fallback | 🟢 Reviewed | ✅ Implemented | M4 | F3, F4, F18 |
| [F20](F20-microvm-backend.md) | MicroVM Backend | 🟢 Reviewed | ✅ Implemented | M5 | F19, F21 |
| [F21](F21-microvm-guest-control-plane.md) | MicroVM Guest Control Plane | 🟢 Reviewed | ✅ Implemented | M5 | F20 |
| [F22](F22-remote-daemon-contexts.md) | Remote Daemon Contexts | 🟢 Reviewed | ✅ Implemented (re-verified 2026-08-31) | M6 | F2, F23 |
| [F23](F23-remote-daemon-transport-auth.md) | Remote Daemon Transport & Auth | 🟢 Reviewed | ✅ Implemented | M6 | F2, F15, F22 |
| [F24](F24-cross-platform-gui-workbench.md) | Cross-Platform GUI Workbench | 🟢 Reviewed | ✅ Implemented | M7 | F2, F15, F22, F23 |
| [F25](F25-gui-workspace-authorization.md) | GUI Workspace Authorization | 🟢 Reviewed | ✅ Implemented | M7 | F17, F24 |
| [F26](F26-gui-sandbox-operations.md) | GUI Sandbox Operations | ⚠️ Needs re-verify | ⚠️ Needs re-verify | M7 | F7, F6, F9, F10, F13, F24 |
| [F27](F27-gui-settings-diagnostics.md) | GUI Settings and Diagnostics | 🟢 Reviewed | ✅ Implemented | M7 | F16, F17, F19, F22, F24, F25 |
| [F29](F29-agent-run.md) | Agent Run | 🟡 Spec written | 🔴 Not started | M8 | F8, F9, F30 |
| [F30](F30-egress-proxy.md) | Egress Proxy | 🔵 In progress (ADR-006 Accepted 2026-08-31) | 🟡 2/8 tasks — T30.1 egress policy + T30.2 daemon proxy listener done | M8 | F11, F17 |

## Milestone Summary

| Milestone | Features | Status |
|-----------|----------|--------|
| M1: Local Linux MVP | F1, F2, F3, F4, F5, F6, F7, F8, F9, F10, F14, F16 | 🟡 Only F9 open (output-delivery event emission + size validation) |
| M2: Hardening & Cache | F11, F12, F13, F17 | ⚠️ F11 enforcement pending (F30 tasks), F13 cache-scoping gap, F17 partial |
| M3: Agent Integrations | F15 | ✅ Implemented |
| M4: Secure Backend | F18, F19 | ✅ Implemented |
| M5: MicroVM Backend | F20, F21 | ✅ Implemented |
| M6: Remote Daemon Mode | F22, F23 | ✅ Implemented |
| M7: Cross-Platform GUI Workbench | F24, F25, F26, F27 | ✅ Implemented |
| M8: Agent Loop and Egress | F29, F30 | 🟡 F30 Reviewed + tasks authored (ADR-006); F29 spec written |

## Summary

All 29 active feature specs tracked (F28 is reserved by pending PROP-006). PROP-005 moved the default Pi Box home to `~/.pi-box`; affected implemented features are marked for re-verification. PROP-008 (2026-07-14) introduced the runtime driver contract, capability reports, shared OCI engine, and no-downgrade selection engine (ADR-005); F3, F4, F18, and F19 tasks were reset where acceptance criteria changed. PROP-009 (2026-07-15) renamed lifecycle "sessions" to sandboxes, added F29/F30, consolidated deliverables into one output channel, and reset affected F6/F8/F9/F11/F12/F13/F17/F26 specs for re-verification. ADR-006 (Accepted 2026-08-31) chose the backend-neutral egress-enforcement mechanism (`NetworkSpec` on `Driver.Create`, daemon forward proxy, in-memory credential store): F11 moved from ⏸️ Blocked to 🟢 Reviewed, F30 to 🟢 Reviewed with tasks T30.1–T30.8 authored; enforcement itself is not yet implemented.

> Note: some legacy filenames predate PROP-002, but the feature labels, titles, source IDs, and index rows now use the canonical `SPEC.md` §6 feature IDs.
