# F13: Cache Model

> Source: `SPEC.md` §6 Features F12
> Status: 🟡 Spec written
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F12 | Cache Model | Scoped dependency cache mounts, cache pruning, cache promotion | M2 |

## Expanded Specification

Dependency caches are first-class citizens in the sandbox runtime. Coding-agent latency is often dominated by package installation and builds.

Scoped cache mounts:
```text
/cache/npm
/cache/pnpm
/cache/yarn
/cache/pip
/cache/uv
/cache/go-mod
/cache/go-build
/cache/cargo
/cache/sccache
```

Policy:
- Caches are not secrets
- Caches scoped by template/runtime/user
- Shared read-only cache plus per-session writable overlay is preferred
- Cache promotion must be explicit or validated
- Cache pruning must be available

Recommended fast tools:
- Node.js: pnpm and corepack
- Python: uv and pip
- Go: GOMODCACHE and GOCACHE
- Rust: cargo cache and optional sccache

Cache directories stored under `~/.pi/caches/<scope>/` where scope is `<template>/<runtime>/<user>`.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-12.1: `/cache/npm`, `/cache/pnpm`, `/cache/pip`, `/cache/uv`, `/cache/go-mod`, `/cache/go-build`, `/cache/cargo` mounted
- [ ] AC-12.2: Caches scoped by template/runtime/user
- [ ] AC-12.3: `pi system prune` can clean caches

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/cache/` | Cache management |
| `~/.pi/caches/` | Cache storage |
| F5: Template System | Templates define cache mounts |
| F3: Fast Backend | Cache mounts in namespace |
| F4: Compat Backend | Cache mounts in container |
| F10: System Commands | Prune includes cache cleanup |

## Security Considerations

- Caches are not secrets (no sensitive data)
- Cache directories scoped per template/runtime/user (isolation)
- No world-readable cache directories
- Cache pruning respects user ownership

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F5: Template System | Internal feature | Templates define cache mounts |
| F10: System Commands | Internal feature | Prune uses cache cleanup |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go cache package |
| **Configuration** | Cache directories under `~/.pi/caches/` |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T13.1: Cache mount management

**Description:** Implement cache mount management. Scoped cache directories, mount into sandbox sessions.

**Acceptance criteria:**
- [ ] Cache directories created under `~/.pi/caches/<scope>/`
- [ ] All 7 cache types mounted: npm, pnpm, pip, uv, go-mod, go-build, cargo
- [ ] Caches scoped by template/runtime/user
- [ ] Caches mounted as read-write in sandbox
- [ ] Cache directories have correct permissions

**Verification:**
- [ ] `go build ./pkg/cache/...`
- [ ] Integration test: cache mounts visible in sandbox

**Files:** `pkg/cache/mounts.go`, `pkg/cache/scope.go`
**Size:** M
**Depends on:** F5 (Template System — templates define cache mounts)

### T13.2: Cache pruning

**Description:** Implement cache pruning. Remove unused caches, respect size limits.

**Acceptance criteria:**
- [ ] `pi system prune` cleans unused caches
- [ ] Cache size limit configurable (default: 50Gi)
- [ ] Unused caches detected (no active sessions using them)
- [ ] Pruning is safe (doesn't delete active session caches)

**Verification:**
- [ ] `go build ./pkg/cache/...`
- [ ] Integration test: prune removes unused caches

**Files:** `pkg/cache/prune.go`
**Size:** S
**Depends on:** T13.1 (cache mounts)

## Verification Plan

- [ ] `go build ./pkg/cache/...` succeeds
- [ ] All 7 cache types mounted correctly
- [ ] Caches scoped per template/runtime/user
- [ ] `pi system prune` cleans unused caches
- [ ] Benchmark: pnpm_install_cached p50 < 2s (SPEC.md §19)
- [ ] Benchmark: uv_sync_cached p50 < 2s (SPEC.md §19)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Cache promotion mechanism not specified | §10 Cache model | Add: "Cache promotion is explicit — user must approve" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Cache sharing between users (future)
- Remote cache (future)
- Cache compression (future)
- Cache analytics (future)
