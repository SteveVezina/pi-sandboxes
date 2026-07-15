# F20: MicroVM Backend

> Source: `SPEC.md` §6 Features F20
> Status: 🟢 Implemented
> Category: Service-layer / Infrastructure

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F20 | MicroVM Backend | Firecracker or Cloud Hypervisor backend with `pi-vmm-manager`, tiny guest rootfs, workspace disk, template snapshot restore, artifact export, and reseed-on-restore behavior | M5 |

## Expanded Specification

MicroVM backend adds VM-grade sandbox isolation. It is snapshot-first: a tiny guest rootfs boots with a template snapshot, attaches a workspace disk, runs a reseed-on-restore hook, and exposes command/file/artifact operations through the guest control plane.

Per ADR-001, the first backend targets Firecracker. Cloud Hypervisor remains a later backend behind the same runtime interface. MicroVM mode requires Linux with `/dev/kvm`; unavailable host support is reported as unavailable instead of silently falling back.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-23.1: `pi-vmm-manager` can start and stop a microVM sandbox
- [x] AC-23.2: Firecracker or Cloud Hypervisor backend boots a tiny guest rootfs
- [x] AC-23.3: Template snapshot restore creates a ready workspace quickly
- [x] AC-23.4: Workspace disk persists sandbox changes for the session
- [x] AC-23.5: Artifact export works from microVM sandboxes
- [x] AC-23.6: Reseed-on-restore hook runs after snapshot restore
- [x] AC-23.7: Benchmarks include microVM mode comparison
- [x] AC-23.8: MicroVM backend reports unavailable when `/dev/kvm` or Firecracker is unavailable
- [x] AC-23.9: Guest rootfs is read-only
- [x] AC-23.10: Workspace disk is writable ext4
- [x] AC-23.11: Artifact export uses the guest control plane

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-vmm-manager/` | MicroVM manager binary |
| `pkg/runtime/microvm/` | MicroVM backend |
| `pkg/bench/` | MicroVM benchmark comparison |
| `docs/features/F21-microvm-guest-control-plane.md` | Guest control dependency |
| `docs/decisions/ADR-001-microvm-backend-architecture.md` | Backend architecture decision |

## Security Considerations

- MicroVM mode must avoid direct host filesystem mounts.
- Workspace and artifact transfer go through explicit disks/control channels.
- Guest images and snapshots must be treated as trusted build artifacts.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F19: Runtime Selection & Fallback | Internal feature | Selects microVM backend |
| F21: MicroVM Guest Control Plane | Internal feature | Required guest agent |
| F17: Policy Enforcement | Internal feature | Security defaults |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | MicroVM runtime package and manager binary |
| **Infrastructure** | Firecracker/Cloud Hypervisor configuration and guest rootfs |

**ADR references:** ADR-001 (MicroVM Backend Architecture), ADR-002 (MicroVM Guest Control Protocol).
**ADR gaps:** None.

## Tasks

### T20.1: Host capability and runtime availability ✅

**Description:** Detect whether microVM mode can run on the host, including Linux, `/dev/kvm`, and Firecracker availability.

**Acceptance criteria:**
- [x] Reports unavailable when `/dev/kvm` is missing
- [x] Reports unavailable when Firecracker is missing
- [x] Does not silently fall back from explicit microVM mode

**Verification:**
- [x] `go build ./cmd/pi-vmm-manager/...`
- [x] Unit test: unavailable host capability is reported

**Files:** `cmd/pi-vmm-manager/main.go`, `pkg/runtime/microvm/runtime.go`, `tests/runtime/microvm/capability_test.go`
**Size:** M
**Depends on:** F19

### T20.2: Firecracker lifecycle and guest rootfs ✅

**Description:** Start and stop a Firecracker-backed microVM sandbox using a read-only guest rootfs.

**Acceptance criteria:**
- [x] `pi-vmm-manager` starts a microVM sandbox
- [x] `pi-vmm-manager` stops a microVM sandbox
- [x] Guest rootfs is read-only

**Verification:**
- [x] Integration test: microVM start/stop (`tests/runtime/microvm/lifecycle_test.go`)
- [x] Integration test: guest rootfs is read-only (same file)

**Files:** `pkg/runtime/microvm/sandbox.go`, `tests/runtime/microvm/lifecycle_test.go`
**Size:** M
**Depends on:** T20.1

### T20.3: Workspace disk, template restore, and reseed ✅

**Description:** Attach a writable ext4 workspace disk, restore from a read-only template snapshot, and run reseed before readiness.

**Acceptance criteria:**
- [x] Workspace disk is writable ext4
- [x] Template snapshot restore works
- [x] Reseed hook runs after workspace attachment and before guest ready

**Verification:**
- [x] Integration test: workspace disk persists session changes (`tests/runtime/microvm/snapshot_test.go`)
- [x] Integration test: reseed hook ordering (same file)

**Files:** `pkg/runtime/microvm/snapshot.go`, `tests/runtime/microvm/snapshot_test.go`
**Size:** M
**Depends on:** T20.2, F21

### T20.4: Artifact export and microVM benchmarks ✅

**Description:** Export artifacts through the guest control plane and include microVM mode in benchmark output.

**Acceptance criteria:**
- [x] Artifact export uses guest control plane
- [x] Artifact export works from microVM sandboxes
- [x] Benchmarks include microVM mode comparison

**Verification:**
- [x] Integration test: artifact export through guest control plane (`tests/runtime/microvm/artifacts_test.go`)
- [x] `pi bench run --mode microvm` recognized (`tests/bench/microvm_mode_test.go`)

**Files:** `pkg/runtime/microvm/export.go`, `pkg/bench/modes.go`, `cmd/pi/bench/commands.go`, `tests/runtime/microvm/artifacts_test.go`, `tests/bench/microvm_mode_test.go`
**Size:** M
**Depends on:** T20.3, F21

## Verification Plan

- [x] MicroVM manager builds (`go build ./cmd/pi-vmm-manager/...`)
- [x] MicroVM sandbox boots (fake-VMM driven unit test, real VMM gated on Linux+KVM host)
- [x] Workspace disk persists session changes (unit test)
- [x] Artifact export and benchmarks work (`go test ./tests/runtime/microvm/...`, `go test ./tests/bench/...`)
- [x] Host capability failures are actionable

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| — | — | — |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Remote daemon transport
- Kubernetes-backed execution
