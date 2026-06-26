# Implementation Plan

> **Navigation only.** Active cursor + cross-feature dependency graph.
> Task lists and task status live in `docs/features/F{N}-*.md`.

## Active Cursor

**Current phase:** Phase 1 — Foundation (M1 core)
**Next feature to implement:** F04 (Session Lifecycle)
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
```

## Execution Order

### Phase 1 — Foundation (M1 core)
1. **F04** Session Lifecycle (foundation for all sessions)
2. **F01** CLI Entry Point (thin client, delegates to daemon)
3. **F02** Daemon API (14 endpoints, delegates to backends)
4. **F03** Fast Backend (Linux isolation)
5. **F17** Policy Enforcement (security foundation)
6. **F05** Command Execution (hot path)

### Phase 2 — M1 completion
7. **F06** Workspace & File Ops
8. **F07** Template System
9. **F08** Artifact Export
10. **F09** Logs & History
11. **F10** System Commands

### Phase 3 — M1 benchmark
12. **F11** Benchmarks (runs against F03 for now; F15/F14 added in Phase 4)

### Phase 4 — M2 hardening
13. **F12** Secrets & Network
14. **F13** Cache Model
15. **F14** Snapshot & Rollback
16. **F15** Compat Backend

### Phase 5 — M3 agent integrations
17. **F16** SDKs

## Risk Tracking

| Risk | Status | Mitigation |
|------|--------|------------|
| Linux-only backends (F03, F15) | New | Document macOS/Windows workarounds; focus on Linux MVP |
| seccomp profile complexity | New | Start with minimal profile, expand iteratively |
| Landlock availability | New | Fallback to mount namespace only |
| OCI runtime detection | New | Try multiple runtimes in priority order |
