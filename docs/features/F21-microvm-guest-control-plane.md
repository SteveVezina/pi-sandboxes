# F21: MicroVM Guest Control Plane

> Source: `SPEC.md` §6 Features F21
> Status: 🟡 Spec written
> Category: Service-layer / Infrastructure

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F21 | MicroVM Guest Control Plane | Guest-side `pi-init` and `pi-agentd` over virtio-vsock for command execution, lifecycle coordination, file/artifact transfer, and sandbox readiness reporting | M5 |

## Expanded Specification

The guest control plane runs inside microVM sandboxes. `pi-init` prepares the guest environment and starts `pi-agentd`; `pi-agentd` communicates with the host over virtio-vsock to handle exec, file, artifact, and lifecycle operations.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-24.1: `pi-init` starts inside the guest and reports readiness
- [ ] AC-24.2: `pi-agentd` communicates with the host over virtio-vsock
- [ ] AC-24.3: Exec requests stream stdout/stderr over the guest control channel
- [ ] AC-24.4: Guest lifecycle events map back to sandbox state
- [ ] AC-24.5: File and artifact transfer work without direct host filesystem mounting

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-init/` | Guest init binary (new — to be created) |
| `cmd/pi-agentd/` | Guest agent binary (new — to be created) |
| `pkg/runtime/microvm/` | Host-side vsock client |
| F20: MicroVM Backend | Consumes guest readiness/control |

## Security Considerations

- Guest control channel must be scoped to the owning microVM.
- File/artifact transfer must not expose host paths directly.
- Lifecycle state must be authenticated or bound to the VM instance.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F20: MicroVM Backend | Internal feature | Host-side backend |
| F17: Policy Enforcement | Internal feature | Sandbox policy |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Guest binaries and host vsock client |
| **Infrastructure** | Guest image packaging |

## Tasks

### T21.1: Guest init and readiness

**Acceptance criteria:**
- [ ] `pi-init` starts inside the guest
- [ ] `pi-agentd` starts and reports readiness
- [ ] Readiness maps to sandbox state

**Verification:**
- [ ] Guest image smoke test
- [ ] Integration test: readiness reported to host

**Files:** `cmd/pi-init/main.go (new — to be created)`, `cmd/pi-agentd/main.go (new — to be created)`
**Size:** L
**Depends on:** F20

### T21.2: Vsock exec/file/artifact protocol

**Acceptance criteria:**
- [ ] Exec streams stdout/stderr
- [ ] File transfer works
- [ ] Artifact transfer works
- [ ] No direct host filesystem mount is required

**Verification:**
- [ ] Integration test: microVM exec streaming
- [ ] Integration test: microVM file/artifact transfer

**Files:** `pkg/runtime/microvm/vsock.go (new — to be created)`, `pkg/runtime/microvm/protocol.go (new — to be created)`
**Size:** L
**Depends on:** T21.1

## Verification Plan

- [ ] Guest binaries build
- [ ] Guest reports readiness
- [ ] Exec streams over vsock
- [ ] File/artifact transfer works without host mounts

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Guest control protocol shape is not specified | §15 MicroVM mode, §31 M5 | Add vsock message contract |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Vsock protocol and guest lifecycle state machine | F20, F21 | ADR for microVM guest protocol |

## Out of Scope

- Non-microVM backend control protocols

