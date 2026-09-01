# F12: Cache Model

> Source: `SPEC.md` §6 Features F12
> Status: 🟡 Partially implemented — 2026-08-31 (ADR-009). Cache mounts are host-bind-free Docker named volumes (AC-12.4 ✅) and are now **scoped by template/runtime/user** (`cache.VolumeScope`), so sibling sandboxes of the same template share a warm cache (AC-12.2 ✅). The mount is a single shared read-write volume per scope — the strict "shared read-only lower + per-sandbox writable upper" overlay (AC-12.5) is a documented Linux-overlayfs follow-up. `pkg/cache` is now wired (`cache.VolumeScope` builds the scope key).
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
- Shared read-only cache plus per-sandbox writable overlay is required
- No sandbox may receive a writable bind mount of a host cache directory
- Cache promotion must be explicit or validated
- Cache pruning must be available

Recommended fast tools:
- Node.js: pnpm and corepack
- Python: uv and pip
- Go: GOMODCACHE and GOCACHE
- Rust: cargo cache and optional sccache

Caches are daemon-managed runtime volumes or template snapshot layers. The local `~/.pi-box` tree may hold cache metadata and daemon-owned storage, but runtime code must not write directly into host cache directories through bind mounts.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-12.1: `/cache/npm`, `/cache/pnpm`, `/cache/pip`, `/cache/uv`, `/cache/go-mod`, `/cache/go-build`, `/cache/cargo` mounted
- [x] AC-12.2: Caches scoped by template/runtime/user *(2026-08-31 ADR-009: `cacheVolumeName` in `pkg/api/sandbox_create.go` builds `pi-sandbox-cache-<template>-<runtime>-<user>-<type>` via `cache.VolumeScope`; guard test `TestCacheVolumeName_ScopedByTemplateNotSandbox`)*
- [x] AC-12.3: `pi-box system prune` can clean caches
- [x] AC-12.4: No sandbox receives a writable bind mount of a host cache directory *(2026-08-31: verified — compat/OCI mounts Docker named volumes, not host paths; `CreateRequest` exposes no cache-path field; guard test `pkg/api/mounts_internal_test.go`)*
- [x] AC-12.5: Cache reuse works via read-only shared layer plus per-sandbox writable overlay or runtime-managed volume *(2026-08-31 ADR-009: reuse via a shared per-scope **runtime-managed volume** (the "or" branch) — mounted read-write; the strict RO-lower + per-sandbox-upper overlay is a deferred Linux follow-up, see § ADR gaps)*

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/cache/` | Cache management |
| `~/.pi-box/runtime/caches/` | Daemon-owned cache metadata/storage |
| F5: Template System | Templates define cache mounts |
| F3: Fast Backend | Cache mounts in namespace |
| F4: Compat Backend | Cache mounts in container |
| F10: System Commands | Prune includes cache cleanup |

## Security Considerations

- Caches are not secrets (no sensitive data)
- Cache directories scoped per template/runtime/user (isolation)
- Sandboxes cannot write to host cache directories through bind mounts
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
| **Configuration** | Daemon-owned cache state under `~/.pi-box/runtime/caches/` |

**ADR references:** ADR-009 (Dependency Cache Scoping) — Accepted 2026-08-31.
**ADR gaps:** Shared RO-lower + per-sandbox writable-upper overlay (strict AC-12.5) — Linux overlayfs follow-up, tracked below.

## Tasks

### T13.1: Cache mount management ✅ *(2026-08-31 ADR-009 — compat path; fast-backend + strict overlay are follow-ups)*

**Description:** Scoped daemon-managed cache volumes exposed inside sandboxes without writable host bind mounts. *(2026-07-15: AC updated per PROP-009; 2026-08-31: scoping wired per ADR-009.)*

**Acceptance criteria:**
- [x] Cache volumes are daemon-managed (Docker named volumes), not host directories *(the `~/.pi-box/runtime/caches/<scope>/` host layout in `pkg/cache.Manager` is retained for the fast backend + prune accounting)*
- [x] All 7 cache types mounted: npm, pnpm, pip, uv, go-mod, go-build, cargo *(when the template declares them)*
- [x] Caches scoped by template/runtime/user *(ADR-009 — `cache.VolumeScope` / `cacheVolumeName`)*
- [ ] Shared cache layer is read-only inside sandbox *(deferred — ADR-009 baseline is a shared RW volume; RO-lower + per-sandbox-upper overlay is a Linux follow-up)*
- [x] Per-sandbox overlay or runtime volume is writable inside sandbox *(shared per-scope RW volume)*
- [x] No writable host cache bind mount is present *(named volumes only)*
- [x] Cache directories have correct permissions *(chown 1000:1000 on `/cache/*` at create)*

**Verification:**
- [x] `go build ./pkg/cache/...`
- [x] Integration test: cache mounts visible in sandbox
- [x] Guard test: `CreateRequest` cannot inject a host cache path (`pkg/api/mounts_internal_test.go`)

**Files:** `pkg/cache/mounts.go`, `pkg/cache/scope.go`
**Size:** M
**Depends on:** F5 (Template System — templates define cache mounts)

### T13.2: Cache pruning

**Description:** Implement cache pruning. Remove unused caches, respect size limits.

**Acceptance criteria:**
- [x] `pi-box system prune` cleans unused caches
- [x] Cache size limit configurable (default: 50Gi)
- [x] Unused caches detected (no active sandboxes using them)
- [x] Pruning is safe (doesn't delete active sandbox caches)

**Verification:**
- [x] `go build ./pkg/cache/...`
- [x] Integration test: prune removes unused caches

**Files:** `pkg/cache/prune.go`
**Size:** S
**Depends on:** T13.1 (cache mounts)

## Verification Plan

- [x] `go build ./pkg/cache/...` succeeds
- [x] All 7 cache types mounted correctly
- [x] Caches scoped per template/runtime/user
- [x] `pi-box system prune` cleans unused caches
- [ ] Benchmark: pnpm_install_cached p50 < 2s (SPEC.md §19) *(2026-08-31 ADR-009: cross-sandbox reuse now wired; needs a Linux+Docker run of `pi-box bench` to confirm the threshold)*
- [ ] Benchmark: uv_sync_cached p50 < 2s (SPEC.md §19) *(2026-08-31 ADR-009: same — reuse wired, threshold needs a real bench run)*

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Cache promotion mechanism not specified | §10 Cache model | Add: "Cache promotion is explicit — user must approve" |

### Implementation gaps — resolved 2026-08-31 (ADR-009)

| ~~Cache volumes scoped per sandbox ID~~ | `cacheVolumeName` scopes by `template/runtime/user` via `cache.VolumeScope`. |
| ~~`pkg/cache` is dead code~~ | `cache.VolumeScope` is now the single source of the scope key. `cache.Manager`'s host layout kept for the fast backend + prune accounting. |
| No shared-RO-lower + writable-upper split (AC-12.5) | ADR-009: baseline is a shared read-write per-scope volume. The overlayfs (`lowerdir`=scope cache, `upperdir`=per-sandbox) construction is a Linux-host follow-up. |

### Historical implementation gaps (superseded by ADR-009)

| Gap | Evidence | Fix |
|-----|----------|-----|
| Cache volumes scoped per sandbox ID → no reuse across sandboxes; every sandbox cold-starts | `pkg/api/sandbox_create.go:131` `managedVolumeName("cache", sandboxID, name)` | Scope the volume name by a `template/runtime/user` key (as `pkg/cache.Scope` already computes), so sibling sandboxes of the same template share one cache volume. |
| `pkg/cache` package (Manager, Scope, prune, `runtime/caches/<scope>/` layout) is dead code — imported only by `tests/cache/cache_test.go` | `grep -rln "pi/pkg/cache" --include='*.go'` → test only | Either wire `pkg/cache` into `createCompatContainer` / the fast backend, or delete it and fold scope logic into the API layer. |
| No shared-RO-layer + writable-overlay split (AC-12.5) | OCI mount is a single rw named volume | Design decision needed: overlayfs vs. "shared volume seeded from template snapshot + per-sandbox volume". Likely a short ADR. |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| ~~Shared RO layer + per-sandbox overlay — mechanism~~ | F12 | **ADR-009** (Accepted): baseline = shared per-scope RW volume; strict overlayfs model deferred to a Linux follow-up. |

## Out of Scope

- Cache sharing between users (future)
- Remote cache (future)
- Cache compression (future)
- Cache analytics (future)
