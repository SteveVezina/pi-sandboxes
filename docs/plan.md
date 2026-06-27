# Implementation Plan

> **Navigation only.** Active cursor + cross-feature dependency graph.
> Task lists and task status live in `docs/features/F{N}-*.md`.

## Active Cursor

**Current phase:** Phase 9 — M7 cross-platform GUI workbench
**Next work:** Complete F24 remote context bridge/auth failure handling, then finish F25 workspace authorization hardening
**Blockers:** None

## Cross-Feature Dependency Graph

```
F01 (CLI) ──→ F02 (Daemon API) ──→ F04 (Session Lifecycle) ──→ F03 (Fast Backend)
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
                                │             ├──→ F26 (Session Operations)
                                │             └──→ F27 (Settings & Diagnostics)
F05/F06/F08/F09/F14 ────────────┘
F10/F17/F19 ───────────────────────────────────────┘
```

## Execution Order (All Complete)

### ✅ Phase 1 — Foundation (M1 core)
1. **F04** Session Lifecycle ✅
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

### 🟡 Phase 9 — M7 cross-platform GUI workbench
30. **F24** Cross-Platform GUI Workbench 🔵
31. **F25** GUI Workspace Authorization 🔵
32. **F26** GUI Session Operations ✅
33. **F27** GUI Settings and Diagnostics ✅

## Risk Tracking

| Risk | Status | Mitigation |
|------|--------|------------|
| Linux-only backends (F03, F15) | New | Document macOS/Windows workarounds; focus on Linux MVP |
| seccomp profile complexity | New | Start with minimal profile, expand iteratively |
| Landlock availability | New | Fallback to mount namespace only |
| OCI runtime detection | New | Try multiple runtimes in priority order |
| GUI API metadata gaps | New | Specify missing read-only daemon endpoints before coding GUI operations |
| Desktop security drift | New | Keep GUI as thin client; daemon policy remains authoritative |
