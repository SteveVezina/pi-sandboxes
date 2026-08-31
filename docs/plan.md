# Implementation Plan

> **Navigation only.** Active cursor + cross-feature dependency graph.
> Task lists and task status live in `docs/features/F{N}-*.md`.

## Active Cursor

**Current phase:** PROP-008/009 re-verify sweep + M8 unblock — F1/F2/F3/F4/F5/F6/F8/F10/F16 complete; F9 partial (2 open gaps); F13 partial (cache scoping gap); F17 re-verified; ADR-006 (egress enforcement) Accepted + cascaded → F11/F30 unblocked, F30 tasks authored
**Next work (2026-08-31):** re-verify backlog cleared; ADR-006 accepted + cascaded; F30 T30.1 + T30.2 done. NEXT: F30 T30.3 (`NetworkSpec` on `Driver.Create` + compat single-endpoint egress). — Historical below.

**Prior (2026-08-28):** F1/F2/F8/F4/F3/F5/F6/F9/F10/F11 re-verified. F10 gap: `history` CLI command didn't exist, `logs` only printed a one-line summary instead of full stdout/stderr — both implemented. Also fixed a pre-existing test bug (`tests/logs/logs_test.go`) that was leaking log directories into the real ~/.pi-box on every test run; ~2MB of accumulated leftovers cleaned up manually. Note: ~44 stray UUID-named dirs also sit under ~/.pi-box/sandboxes/ from tests/api/* runs (Manager always resolves via the real $HOME, not test-isolated) — not fixed, out of scope, flagging for awareness.
F11 was significantly overstated: network mode (none/restricted/open) and default-deny targets are validated as strings and unit-tested as pure decision logic in `pkg/network/`, but nothing in the daemon ever calls that logic — sandbox containers always get Docker's default `bridge` network, `exec.Request.NetworkMode` is stored but never read, and `EgressProxy`/`Broker`/SSH/token helpers in `pkg/network` and `pkg/secrets` have zero callers outside their own tests. Downgraded F11 to 🔴 Not enforced / ⏸️ Blocked rather than implementing enforcement under time pressure — this is genuinely the same architecture gap as F30 Egress Proxy (still 🔴 Not started) and needs an ADR first (how sandboxes attach to a mode-appropriate network; how real secret values get resolved instead of the current `"[credential-injected]"` placeholder; whether network mode is per-sandbox or per-exec). Did not touch F30 or attempt enforcement — flagged as a blocker instead.
F17 re-verified 2026-08-31: partly the same story as F11 but not entirely. The host-mount and process-limit ACs are genuinely enforced — compat create/exec always derive workspace/artifacts/cache sources from daemon-managed named volumes (`managedVolumeName` in `pkg/api/sandbox_create.go`); `CreateRequest` exposes no arbitrary-mount field; host paths only via explicit `WorkspaceMode: "bind"` opt-in. Added regression guard `pkg/api/mounts_internal_test.go`. Closed AC-34.1 + the two "writable host cache bind mount rejected" sub-items. Still blocked: AC-17.5 (git creds via egress proxy) and AC-34.3 (secrets as egress-proxy injection rules, not plaintext under ~/.pi-box) — same egress-enforcement ADR blocker as F11/F30. F17 now 🟡 partial rather than ⚠️.
2026-08-31 continued: (a) wrote **ADR-006** (egress enforcement + credential delivery, status Proposed) — resolves the ADR gap blocking F11/F17-remainder/F30; needs human acceptance + cascade. (b) **F16 System Commands re-verified → ✅ Implemented** (all 4 commands wired, 21 tests pass, no stale "session" terms). (c) **F13 Cache Model re-verified → 🟡 partial**: cache mounts are host-bind-free (Docker named volumes, AC-12.4 ✅) BUT scoped per sandbox ID → zero cross-sandbox cache reuse (AC-12.2/12.5 reset); `pkg/cache` (proper template/runtime/user scoping) is dead code imported only by its test — same unwired-library pattern as pre-ADR-006 `pkg/network`. Cached-install benchmarks can't pass until fixed. See F13 § Implementation gaps.
2026-08-31 (ADR-006 accepted + cascaded): ADR-005 amended with `NetworkSpec` on `Driver.Create`; F30 tasks re-authored T30.1–T30.8 (S/M sizes, checkpoint after T30.4); F11 T12.1/T12.2 now trace to F30 tasks instead of duplicating; F17 AC-17.5/AC-34.3 point at F30 T30.7–T30.8. F11 ⏸️→🟢, F30 🟡→🟢.
2026-08-31 re-verify sweep DONE: F14 Benchmarks, F15 SDKs, F18 Secure Backend, F22 Remote Contexts all re-verified → ✅ Implemented (185 targeted tests pass; 2 skips are repo-root env guards, not AC masks; fixed a malformed checkbox in F14 T11.1). Milestones M3/M4/M6 → ✅ Implemented; M1 now only F9 open; M2 gated on F30 tasks + F13 cache scoping.
**No ⚠️ Needs re-verify features remain.** Open work: (1) F30 T30.1–T30.8 (M8 egress — reviewed, ready to execute); (2) F9 output-delivery event emission + size validation (M1 close-out); (3) F13 cache-scoping fix (needs small PROP/ADR); (4) F29 Agent Run spec review.
**Blockers:** Lifecycle event transport ADR needed before AC-9.4 (and F4/F29's related lifecycle-event ACs) can close. Sandbox egress enforcement: **ADR-006 Accepted 2026-08-31 and cascaded** — F11 → 🟢 Reviewed, F30 → 🟢 Reviewed with tasks T30.1–T30.8 authored, `Driver.Create` contract note added to ADR-005. F30 implementation is now unblocked; recommend `/skill:pi-review-plan` on the F30 task list before executing.

## Cross-Feature Dependency Graph

```
F01 (CLI) ──→ F02 (Daemon API) ──→ F04 (Sandbox Lifecycle) ──→ F03 (Fast Backend)
                                                          └──→ F15 (Compat Backend)
F07 (Templates) ──→ F03/F15 ──→ F05 (Exec) ──→ F09 (Logs)
       │              │         │
       └──────────────┘         └──→ F17 (Policy) ──→ F12 (Network)
                              │                    └──→ F13 (Cache)
                              │
F06 (Files) ──→ F04 ──→ F08 (Artifacts)
                              │
F14 (Snapshots) ──────────────┘
                              │
F11 (Benchmarks) ─────────────┘ (depends on F03, F15, F14)
                              │
F10 (System) ─────────────────┘ (depends on F04)
                              │
F16 (SDKs) ───────────────────┘ (depends on F02)

F18 (Secure Backend) ──→ F19 (Runtime Selection & Fallback)
                               │
                               └──→ F20 (MicroVM Backend) ──→ F21 (MicroVM Guest Control Plane)

F23 (Remote Transport/Auth) ──→ F22 (Remote Daemon Contexts)

F02/F16/F22/F23 ──→ F24 (GUI Workbench) ──→ F25 (Workspace Authorization)
                                │             ├──→ F26 (Sandbox Operations)
                                │             └──→ F27 (Settings & Diagnostics)
F05/F06/F08/F09/F14 ────────────┘
F10/F17/F19 ───────────────────────────────────────┘
```

## Execution Order (All Complete)

### ✅ Phase 1 — Foundation (M1 core)
1. **F04** Sandbox Lifecycle ✅
2. **F01** CLI Entry Point ✅
3. **F02** Daemon API ✅
4. **F03** Fast Backend ✅
5. **F17** Policy Enforcement ✅
6. **F05** Command Execution ✅

### ✅ Phase 2 — M1 completion
7. **F06** Workspace & File Ops ✅
8. **F07** Template System ✅
9. **F08** Artifact Export ✅
10. **F09** Logs & History ✅
11. **F10** System Commands ✅

### ✅ Phase 3 — M1 benchmark
12. **F11** Benchmarks ✅

### ✅ Phase 4 — M2 hardening
13. **F12** Secrets & Network ✅
14. **F13** Cache Model ✅
15. **F14** Snapshot & Rollback ✅
16. **F15** Compat Backend ✅

### ✅ Phase 5 — M3 agent integrations
17. **F16** SDKs ✅

### ✅ Phase 6 — M4 secure backend
20. **F19** Runtime Selection & Fallback ✅
21. **F18** Secure Backend ✅
22. **Compatibility documentation** ✅

### ✅ Phase 7 — M5 microVM backend
23. **F21** MicroVM Guest Control Plane ✅
24. **F20** MicroVM Backend ✅

### ✅ Phase 8 — M6 remote daemon mode
25. **F23** Remote Daemon Transport & Auth ✅
26. **F22** Remote Daemon Contexts ✅

### ✅ Code quality closes (CORE.md watch-outs)
27. **Benchmarks** Replace `time.Sleep()` fallbacks with proper tool-absent returns ✅
28. **Doctor** Add config.yaml creation, system command validation, daemon binary check ✅
29. **Shell** Implement WebSocket PTY shell (`pkg/api/sandbox_shell.go`, `pkg/terminal/`) ✅

### ✅ Phase 9 — M7 cross-platform GUI workbench
30. **F24** Cross-Platform GUI Workbench ✅
31. **F25** GUI Workspace Authorization ✅
32. **F26** GUI Sandbox Operations ✅
33. **F27** GUI Settings and Diagnostics ✅

### 🔵 Phase 10 — M8 agent loop and egress
34. **F30** Egress Proxy — 🔵 In progress (2/8). T30.1 per-sandbox egress policy; T30.2 daemon forward proxy (`network.ProxyServer`, CONNECT + HTTP, sandbox ID via `Proxy-Authorization`, `--egress-proxy-port`, default disabled). 464 tests green. NEXT: T30.3 `NetworkSpec` on `Driver.Create` + compat single-endpoint egress.
35. **F29** Agent Run — 🟡 Spec written. Depends on F8, F9, F30.
36. **F11** Secrets & Network Model — enforcement lands with F30 tasks.

## Risk Tracking

| Risk | Status | Mitigation |
|------|--------|------------|
| Linux-only backends (F03, F15) | New | Document macOS/Windows workarounds; focus on Linux MVP |
| seccomp profile complexity | New | Start with minimal profile, expand iteratively |
| Landlock availability | New | Fallback to mount namespace only |
| OCI runtime detection | New | Try multiple runtimes in priority order |
| GUI API metadata gaps | New | Specify missing read-only daemon endpoints before coding GUI operations |
| Desktop security drift | New | Keep GUI as thin client; daemon policy remains authoritative |
| Egress proxy baseline does not intercept TLS — allowlisted TLS conn can carry arbitrary bytes | Active | Accepted per ADR-006 (domain-level allowlist is the stated F11 model); transparent redirect + TLS MITM is a documented follow-up |
| `restricted`-mode single-endpoint networking differs per backend (nftables / OCI network / guest firewall) | Active | ADR-006 isolates it to each driver's `Create`; checkpoint after F30 T30.4 verifies metadata-IP unreachable in both fast and compat |
