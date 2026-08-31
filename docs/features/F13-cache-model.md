# F12: Cache Model

> Source: `SPEC.md` §6 Features F12
> Status: 🟡 Partially implemented — re-verified 2026-08-31. Cache mounts exist and are **host-bind-free** (compat path mounts Docker *named* volumes `pi-sandbox-cache-<sandboxID>-<type>`, never host paths — AC-12.4 ✅). But the wired path scopes cache volumes **per sandbox ID**, not by template/runtime/user, so there is **no cache reuse across sandboxes** — every new sandbox starts cold (AC-12.2 reset, AC-12.5 not met). `pkg/cache` (which implements proper `template/runtime/user` scoping via `Scope.String()` and a `runtime/caches/<scope>/` layout) is imported only by its own test — it is dead code, same pattern as `pkg/network` before ADR-006. See § Spec Gaps.
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
- [ ] AC-12.2: Caches scoped by template/runtime/user *(2026-08-31: reset — wired path scopes by sandbox ID only (`managedVolumeName("cache", sandboxID, type)`); `pkg/cache.Scope` implements template/runtime/user scoping but has no callers)*
- [x] AC-12.3: `pi-box system prune` can clean caches
- [x] AC-12.4: No sandbox receives a writable bind mount of a host cache directory *(2026-08-31: verified — compat/OCI mounts Docker named volumes, not host paths; `CreateRequest` exposes no cache-path field; guard test `pkg/api/mounts_internal_test.go`)*
- [ ] AC-12.5: Cache reuse works via read-only shared layer plus per-sandbox writable overlay or runtime-managed volume *(2026-08-31: not met — per-sandbox-ID volumes, no shared layer, no overlay, no reuse)*

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

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T13.1: Cache mount management ⚠️ (host-bind-free ✅; template/runtime/user scoping + shared-layer/overlay NOT wired)

**Description:** Implement cache mount management. Scoped daemon-managed cache layers or volumes are exposed inside sandboxes without writable host bind mounts. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [ ] Cache metadata/storage created under daemon-owned `~/.pi-box/runtime/caches/<scope>/` *(2026-08-31: `pkg/cache.Manager.volumePath` builds exactly this path, but `pkg/cache` is unwired; the live path uses Docker named volumes instead)*
- [x] All 7 cache types mounted: npm, pnpm, pip, uv, go-mod, go-build, cargo *(when the template declares them)*
- [ ] Caches scoped by template/runtime/user *(2026-08-31: reset — scoped by sandbox ID in the wired path)*
- [ ] Shared cache layer is read-only inside sandbox *(not implemented)*
- [ ] Per-sandbox overlay or runtime volume is writable inside sandbox *(only a plain per-sandbox rw volume; no shared+overlay split)*
- [x] No writable host cache bind mount is present *(2026-08-31: verified — named volumes only)*
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
- [ ] Benchmark: pnpm_install_cached p50 < 2s (SPEC.md §19) *(2026-08-31: cannot pass — no cross-sandbox cache reuse; blocked on cache scoping fix)*
- [ ] Benchmark: uv_sync_cached p50 < 2s (SPEC.md §19) *(2026-08-31: same blocker)*

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Cache promotion mechanism not specified | §10 Cache model | Add: "Cache promotion is explicit — user must approve" |

### Implementation gaps (2026-08-31 re-verify)

| Gap | Evidence | Fix |
|-----|----------|-----|
| Cache volumes scoped per sandbox ID → no reuse across sandboxes; every sandbox cold-starts | `pkg/api/sandbox_create.go:131` `managedVolumeName("cache", sandboxID, name)` | Scope the volume name by a `template/runtime/user` key (as `pkg/cache.Scope` already computes), so sibling sandboxes of the same template share one cache volume. |
| `pkg/cache` package (Manager, Scope, prune, `runtime/caches/<scope>/` layout) is dead code — imported only by `tests/cache/cache_test.go` | `grep -rln "pi/pkg/cache" --include='*.go'` → test only | Either wire `pkg/cache` into `createCompatContainer` / the fast backend, or delete it and fold scope logic into the API layer. |
| No shared-RO-layer + writable-overlay split (AC-12.5) | OCI mount is a single rw named volume | Design decision needed: overlayfs vs. "shared volume seeded from template snapshot + per-sandbox volume". Likely a short ADR. |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Shared read-only cache layer + per-sandbox writable overlay — mechanism (overlayfs? seeded volume?) | F12 | ADR-NNN: Cache layering model (small; can fold into a cache-scoping PROP) |

## Out of Scope

- Cache sharing between users (future)
- Remote cache (future)
- Cache compression (future)
- Cache analytics (future)
