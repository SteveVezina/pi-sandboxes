# F20: MicroVM Backend

> Source: `SPEC.md` §6 Features F20
> Status: 🟡 Spec written
> Category: Service-layer / Infrastructure

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F20 | MicroVM Backend | Firecracker or Cloud Hypervisor backend with `pi-vmm-manager`, tiny guest rootfs, workspace disk, template snapshot restore, artifact export, and reseed-on-restore behavior | M5 |

## Expanded Specification

MicroVM backend adds VM-grade sandbox isolation. It is snapshot-first: a tiny guest rootfs boots with a template snapshot, attaches a workspace disk, runs a reseed-on-restore hook, and exposes command/file/artifact operations through the guest control plane.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-23.1: `pi-vmm-manager` can start and stop a microVM sandbox
- [ ] AC-23.2: Firecracker or Cloud Hypervisor backend boots a tiny guest rootfs
- [ ] AC-23.3: Template snapshot restore creates a ready workspace quickly
- [ ] AC-23.4: Workspace disk persists sandbox changes for the session
- [ ] AC-23.5: Artifact export works from microVM sandboxes
- [ ] AC-23.6: Reseed-on-restore hook runs after snapshot restore
- [ ] AC-23.7: Benchmarks include microVM mode comparison

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-vmm-manager/` | MicroVM manager binary (new — to be created) |
| `pkg/runtime/microvm/` | MicroVM backend (new — to be created) |
| `pkg/bench/` | MicroVM benchmark comparison |
| `docs/features/F21-microvm-guest-control-plane.md` | Guest control dependency |

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

## Tasks

### T20.1: VMM manager and backend lifecycle

**Acceptance criteria:**
- [ ] Start/stop microVM sandbox
- [ ] Boot tiny guest rootfs
- [ ] Attach workspace disk

**Verification:**
- [ ] `go build ./cmd/pi-vmm-manager/...`
- [ ] Integration test: microVM start/stop

**Files:** `cmd/pi-vmm-manager/main.go (new — to be created)`, `pkg/runtime/microvm/runtime.go (new — to be created)`
**Size:** L
**Depends on:** F19

### T20.2: Snapshot restore, reseed, artifacts, and benchmarks

**Acceptance criteria:**
- [ ] Template snapshot restore works
- [ ] Reseed hook runs after restore
- [ ] Artifact export works
- [ ] Benchmarks include microVM mode

**Verification:**
- [ ] Integration test: snapshot restore and artifact export
- [ ] `pi bench run --mode microvm --json`

**Files:** `pkg/runtime/microvm/snapshot.go (new — to be created)`, `pkg/runtime/microvm/artifacts.go (new — to be created)`
**Size:** L
**Depends on:** T20.1, F21

## Verification Plan

- [ ] MicroVM manager builds
- [ ] MicroVM sandbox boots
- [ ] Workspace disk persists session changes
- [ ] Artifact export and benchmarks work

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Firecracker vs Cloud Hypervisor selection is not decided | §15 MicroVM mode, §31 M5 | Add backend choice and host requirements |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| MicroVM implementation choice and disk/snapshot format | F20, F21 | ADR for microVM architecture |

## Out of Scope

- Remote daemon transport
- Kubernetes-backed execution

