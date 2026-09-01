# ADR-009: Dependency Cache Scoping

## Status

Accepted (2026-08-31, human goal: "finish all possible tasks").

Unblocks: F12 Cache Model (AC-12.2 / AC-12.5), and the `*_cached`
threshold benchmarks in F14 which cannot pass without cross-sandbox cache
reuse. No block-spec change — `SPEC.md` §17 already mandates
"caches scoped by template/runtime/user" and a shared read-only layer plus
per-sandbox writable overlay. This ADR chooses the concrete Docker
mechanism and defers the strict overlay model.

## Context

Today `createCompatContainer` builds cache volumes as
`managedVolumeName("cache", sandboxID, name)` — one Docker named volume
**per sandbox ID**. Every new sandbox therefore starts with an empty
cache; `pnpm_install_cached`, `uv_sync_cached`, `go_test_cached`,
`cargo_test_cached` measure cold installs. `pkg/cache` implements the
correct `template/runtime/user` scope key (`Scope.String()`) and a
`~/.pi-box/runtime/caches/<scope>/` layout but has no callers outside its
own test.

## Decision

### 1. Scope key = template / runtime / user

Cache volume names are derived from
`cache.Scope{Template, Runtime, User}` (`pkg/cache/scope.go`), not the
sandbox ID. Sibling sandboxes of the same template + runtime + user share
one cache volume per cache type. `User` is `"default"` for now (single
local user); the field exists for later multi-user daemons.

Volume name: `pi-sandbox-cache-<template>-<runtime>-<user>-<type>`
(each segment sanitized to `[a-zA-Z0-9._-]`).

### 2. Baseline mount: one shared read-write named volume per scope

The scope volume is mounted **read-write** at `/cache/<type>`. Package
managers (npm, pnpm, uv, pip, cargo, go) use content-addressed stores and
their own lock files and tolerate concurrent readers with occasional
writers — this is how CI shared caches work. This is the M8 baseline and
what makes the `*_cached` benchmarks meaningful.

### 3. Strict model (shared RO lower + per-sandbox writable upper)

`SPEC.md` §17's "shared read-only plus per-sandbox writable overlay" is a
**Linux overlayfs** construction (`lowerdir` = the scope cache,
`upperdir` = a per-sandbox dir, `workdir` = a per-sandbox dir). It gives
write isolation between sandboxes at the cost of a mount-namespace setup
per sandbox. Deferred to a Linux-host follow-up (tracked in F12). The
baseline (§2) is a correct, weaker form: reuse without isolation.

### 4. `pkg/cache` is wired, not deleted

`cache.Scope` becomes the single source of the scope key. The
daemon-managed-volume path in `pkg/api` uses it. `cache.Manager`'s
host-directory layout (`~/.pi-box/runtime/caches/<scope>/`) is retained
for the fast backend and for `pi-box system prune`/`disk-usage`
accounting.

### 5. Pruning

`pi-box system prune` removes scope cache volumes with no live sandbox
referencing their scope. A scope volume outliving its sandboxes is
expected (that is the point) — prune is the reclaim path.

## Consequences

- `createCompatContainer` and the exec container path change the cache
  source from per-ID to per-scope. Existing per-ID cache volumes become
  orphans reclaimed by prune.
- The four `*_cached` benchmarks can now show warm numbers on a second
  run with the same template.
- Two sandboxes of the same template racing a first `pnpm install` may
  both populate the cache; pnpm's store is content-addressed so this is
  safe, just briefly redundant.
- Write isolation between same-template sandboxes is not provided until
  the overlay follow-up. A sandbox can corrupt a shared cache for its
  siblings (same risk as a shared CI cache). Acceptable for local
  developer tooling; revisit for untrusted workloads (which should use
  `network: none` / `secure` mode and may opt out of cache sharing).

## References

- `SPEC.md` §17 Cache model; §19 benchmarks
- `docs/features/F13-cache-model.md` (F12), `docs/features/F11-benchmarks.md` (F14)
- `pkg/cache/scope.go`, `pkg/api/sandbox_create.go`
- ADR-005 (runtime driver contract — cache mounts are a driver input)
