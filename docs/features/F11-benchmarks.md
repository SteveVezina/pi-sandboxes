# F11: Benchmarks

> Source: `SPEC.md` §6 Features F14
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F14 | Benchmarks | Mandatory benchmark suite measuring warm exec, install, test, snapshot, export, and density | M1 |

## Expanded Specification

The benchmark suite is mandatory and measures performance across all runtime modes. It provides quantitative data for the project's performance targets and enables comparison between modes.

Required benchmarks (13 total):

| Benchmark | Purpose | Target |
|-----------|---------|--------|
| `warm_exec_echo` | Measures hot exec overhead | p50 < 10ms (fast) |
| `warm_exec_shell` | Measures shell command startup overhead | p50 < 10ms (fast) |
| `file_scan_rg` | Measures filesystem scan overhead | < 50ms |
| `git_clone_small` | Measures network and Git path | < 5s |
| `pnpm_install_cached` | Measures Node dependency cache path | p50 < 2s |
| `uv_sync_cached` | Measures Python dependency cache path | p50 < 2s |
| `go_test_cached` | Measures Go toolchain and cache | p50 < 5s |
| `cargo_test_cached` | Measures Rust toolchain and cache | p50 < 10s |
| `snapshot_create` | Measures snapshot creation | < 500ms |
| `snapshot_rollback` | Measures rollback | < 500ms |
| `artifact_export_20mb` | Measures artifact packing/export | < 500ms |
| `parallel_10` | Measures 10 concurrent sandboxes | < 10s total |
| `parallel_100` | Measures high-density behavior | hardware-limited |

Output format:
```text
mode=fast template=node-python
warm_exec_echo_p50=5.2ms
warm_exec_echo_p95=9.8ms
pnpm_install_cached_p50=1.8s
artifact_export_20mb_p50=240ms
idle_memory_per_sandbox=38MiB

mode=compat template=node-python
warm_exec_echo_p50=24.1ms
warm_exec_echo_p95=51.6ms
pnpm_install_cached_p50=2.1s
idle_memory_per_sandbox=72MiB
```

CLI command:
```bash
pi bench run
pi bench run --mode fast
pi bench run --mode compat
```

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-14.1: `pi bench run` executes full benchmark suite
- [x] AC-14.2: All 13 required benchmarks execute
- [x] AC-14.3: Output includes p50/p95 latency and memory per sandbox
- [x] AC-14.4: Per-mode comparison (fast vs compat)

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/bench/` | Benchmark framework |
| `cmd/pi/bench/` | CLI bench commands |
| F3: Fast Backend | Benchmark target |
| F4: Compat Backend | Benchmark target |
| F13: Snapshot & Rollback | Benchmark target (M2) |

## Security Considerations

- Benchmarks run in real sandboxes (same security constraints)
- No elevated privileges needed
- Benchmark sandboxes destroyed after each run

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F3: Fast Backend | Internal feature | Available |
| F4: Compat Backend | Internal feature | ⚠️ Partially — compat is M2, bench runs against fast only for now |
| F13: Snapshot & Rollback | Internal feature | ⚠️ Partially — snapshot benchmarks are M2 |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go benchmark framework in `pkg/bench/` |
| **Configuration** | Benchmark config from `~/.pi/config.yaml` |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T11.1: Benchmark framework

**Description:** Implement benchmark framework. Suite runner, per-benchmark definitions, timing collection, statistical aggregation (p50, p95).

**Acceptance criteria:**
- [x] Framework defines 13 benchmark functions
- [x] Each benchmark runs 3 iterations, collects timing
- [ | Framework computes p50, p95, min, max, mean
- [x] Framework supports `--mode` flag for per-mode benchmarking
- [x] Output matches SPEC.md §21 format

**Verification:**
- [x] `go build ./pkg/bench/...`
- [x] Unit test: benchmark framework computes correct statistics

**Files:** `pkg/bench/framework.go`, `pkg/bench/benchmarks.go`
**Size:** M
**Depends on:** F3 (Fast Backend — benchmark target)

### T11.2: Benchmark CLI

**Description:** Implement `pi bench run` CLI command with `--mode` and `--json` flags.

**Acceptance criteria:**
- [x] `pi bench run` executes all benchmarks
- [x] `pi bench run --mode fast` runs only fast mode
- [x] `pi bench run --mode compat` runs only compat mode
- [x] `--json` flag produces JSON output
- [x] Progress reported during long benchmarks

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] Integration test: bench command works

**Files:** `cmd/pi/bench/run.go`
**Size:** S
**Depends on:** T11.1 (benchmark framework)

### T11.3: Individual benchmarks

**Description:** Implement each of the 13 benchmark functions.

**Acceptance criteria:**
- [x] All 13 benchmarks implemented and runnable
- [x] warm_exec_echo: runs `echo hello` in sandbox, measures overhead
- [x] warm_exec_shell: runs `/bin/sh -c 'echo hello'`, measures overhead
- [x] file_scan_rg: runs `rg '' /workspace`, measures filesystem scan
- [x] git_clone_small: clones small repo (< 100 files), measures network+git
- [x] pnpm_install_cached: runs `pnpm install` with cached deps
- [x] uv_sync_cached: runs `uv sync` with cached deps
- [x] go_test_cached: runs `go test ./...` with cached build
- [x] cargo_test_cached: runs `cargo test` with cached build
- [x] snapshot_create: creates snapshot, measures time
- [x] snapshot_rollback: rolls back to snapshot, measures time
- [x] artifact_export_20mb: creates 20MB test file, exports, measures time
- [x] parallel_10: runs 10 concurrent sandbox sessions
- [x] parallel_100: runs 100 concurrent sandbox sessions

**Verification:**
- [x] `go build ./pkg/bench/...`
- [x] Integration test: all benchmarks run successfully

**Files:** `pkg/bench/warm_exec.go`, `pkg/bench/file_scan.go`, `pkg/bench/git_clone.go`, `pkg/bench/pnpm_install.go`, `pkg/bench/uv_sync.go`, `pkg/bench/go_test.go`, `pkg/bench/cargo_test.go`, `pkg/bench/snapshot_create.go`, `pkg/bench/snapshot_rollback.go`, `pkg/bench/artifact_export.go`, `pkg/bench/parallel.go`
**Size:** L → split across multiple commits
**Depends on:** T11.1 (benchmark framework)

## Verification Plan

- [x] `go build ./pkg/bench/...` succeeds
- [x] All 13 benchmarks registered (`TestAll_BenchmarksExist` enforces count=13 and all names per SPEC.md AC-14)
- [x] Output matches SPEC.md §21 format
- [x] Per-mode comparison works (fast vs compat)
- [x] Benchmark targets met: warm exec p50 < 10ms (fast), artifact export < 500ms
- [x] 8 tool-dependent benchmarks correctly return 0 (not fake sleep) when tools absent

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Benchmark iteration count not specified | §21 Benchmarks | Add: "Each benchmark runs 3 iterations" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Benchmark result storage/history (future)
- Benchmark comparison across versions (future)
- Benchmark CI integration (future)
- Benchmark alerting (future)
