# F18: Secure Backend

> Source: `SPEC.md` §6 Features F18
> Status: 🟢 Implemented
> Category: Service-layer / Infrastructure

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F18 | Secure Backend | gVisor-backed runtime mode for unknown or untrusted repositories, including runsc integration, secure-mode lifecycle, compatibility notes, and benchmark comparison against fast/compat | M4 |

## Expanded Specification

Secure backend adds a `secure` runtime mode backed by gVisor/runsc. It must preserve the daemon API and CLI surface used by fast and compat modes while adding stronger isolation for unknown or untrusted repositories.

The secure backend must provide actionable compatibility errors. If gVisor is unavailable or cannot run a workload, the runtime selection layer decides whether fallback is allowed.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-21.1: `pi-box box create --mode secure <template>` creates a sandbox using gVisor/runsc when available
- [x] AC-21.2: Secure sandboxes execute commands through the same daemon API as fast/compat sandboxes
- [x] AC-21.3: Secure mode does not mount the host home directory or Docker socket by default
- [x] AC-21.4: Secure mode exposes compatibility errors with actionable guidance
- [x] AC-21.5: Benchmarks compare fast vs compat vs secure modes

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/runtime/gvisor/` | Secure backend implementation |
| `pkg/runtime/` | Runtime registry and lifecycle dispatch |
| `cmd/pi-box/box` | `--mode secure` support |
| `cmd/pi-box/system` | Doctor/runtime availability reporting |
| `pkg/bench/` | Secure-mode benchmark comparison |

## Security Considerations

- Secure mode is intended for unknown or untrusted repositories.
- Host home and Docker socket are never mounted by default.
- Compatibility failures must not silently relax policy.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F15: Compat Backend | Internal feature | Fallback target |
| F17: Policy Enforcement | Internal feature | Required security defaults |
| F19: Runtime Selection & Fallback | Internal feature | Selects/falls back from secure mode |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | gVisor backend package and runtime dispatch |
| **Infrastructure** | runsc availability and host compatibility checks |

## Tasks

### T18.1: gVisor backend lifecycle

**Acceptance criteria:**
- [x] Create/destroy/exec work through gVisor/runsc
- [x] Secure backend returns actionable errors when unavailable
- [x] Secure backend enforces default policy

**Verification:**
- [x] `go build ./pkg/runtime/gvisor/...`
- [x] Integration test: secure sandbox create/exec/destroy

**Files:** `pkg/runtime/gvisor/runtime.go`
**Size:** L
**Depends on:** F17, F19

### T18.2: Secure-mode benchmark comparison

**Acceptance criteria:**
- [x] Benchmark suite can run secure mode
- [x] Output compares fast, compat, and secure

**Verification:**
- [x] `pi-box bench run --mode secure --json`

**Files:** `pkg/bench/benchmarks.go`, `cmd/pi-box/bench/commands.go`
**Size:** M
**Depends on:** T18.1

## Verification Plan

- [x] Secure backend builds
- [x] Secure create/exec/destroy works when runsc is available
- [x] Security defaults remain enforced
- [x] Benchmarks include secure mode

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Exact runsc invocation/config is not specified | §14 Secure mode, §31 M4 | Add secure backend runtime contract if implementation needs more detail |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Runtime registry/fallback policy | F18, F19, F20 | ADR for runtime selection |

## Out of Scope

- MicroVM isolation
- Remote daemon transport

