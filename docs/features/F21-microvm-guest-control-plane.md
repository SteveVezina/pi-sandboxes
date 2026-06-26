# F21: MicroVM Guest Control Plane

> Source: `SPEC.md` §6 Features F21
> Status: 🟡 Partially implemented (T21.1, T21.3, T21.4 remaining)
> Category: Service-layer / Infrastructure

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F21 | MicroVM Guest Control Plane | Guest-side `pi-init` and `pi-agentd` over virtio-vsock for command execution, lifecycle coordination, file/artifact transfer, and sandbox readiness reporting | M5 |

## Expanded Specification

The guest control plane runs inside microVM sandboxes. `pi-init` prepares the guest environment and starts `pi-agentd`; `pi-agentd` communicates with the host over virtio-vsock to handle exec, file, artifact, and lifecycle operations.

Per ADR-002, host and guest exchange newline-delimited JSON frames over virtio-vsock. Exec output is streamed using `stream` frames with base64 payloads, and the host marks a sandbox warm only after receiving the guest `ready` event.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-24.1: `pi-init` starts inside the guest and reports readiness
- [ ] AC-24.2: `pi-agentd` communicates with the host over virtio-vsock
- [ ] AC-24.3: Exec requests stream stdout/stderr over the guest control channel
- [ ] AC-24.4: Guest lifecycle events map back to sandbox state
- [ ] AC-24.5: File and artifact transfer work without direct host filesystem mounting
- [ ] AC-24.6: Host and guest exchange newline-delimited JSON frames over virtio-vsock
- [ ] AC-24.7: Exec stdout/stderr stream frames carry base64 payloads
- [ ] AC-24.8: Final exec response includes exit code, duration, timeout, and truncation metadata
- [ ] AC-24.9: Host marks sandbox warm only after the guest sends `ready`

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-init/` | Guest init binary (new — to be created) |
| `cmd/pi-agentd/` | Guest agent binary (new — to be created) |
| `pkg/runtime/microvm/` | Host-side vsock client |
| F20: MicroVM Backend | Consumes guest readiness/control |
| `docs/decisions/ADR-002-microvm-guest-control-protocol.md` | Guest protocol decision |

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

**ADR references:** ADR-001 (MicroVM Backend Architecture), ADR-002 (MicroVM Guest Control Protocol).
**ADR gaps:** None.

## Tasks

### T21.1: Guest init and readiness

**Description:** Build `pi-init` and start `pi-agentd` inside the guest.

**Acceptance criteria:**
- [ ] `pi-init` starts inside the guest
- [ ] `pi-agentd` starts and reports readiness
- [ ] Readiness maps to sandbox state
- [ ] Host marks sandbox warm only after `ready`

**Verification:**
- [ ] Guest image smoke test
- [ ] Integration test: readiness reported to host

**Files:** `cmd/pi-init/main.go (new — to be created)`, `cmd/pi-agentd/main.go (new — to be created)`
**Size:** M
**Depends on:** F20

### T21.2: Vsock frame codec ✅

**Description:** Implement newline-delimited JSON frame encoding/decoding for host and guest.

**Acceptance criteria:**
- [ ] Frames include type, id, session_id, method, payload, and optional error
- [ ] Codec handles request, response, event, and stream frames
- [ ] Invalid frames return actionable errors

**Verification:**
- [x] Unit tests for valid frame round-trip
- [x] Unit tests for invalid frame errors

**Files:** `pkg/runtime/microvm/protocol.go`, `tests/runtime/microvm/protocol_test.go`
**Size:** M
**Depends on:** None

### T21.3: Exec streaming protocol

**Acceptance criteria:**
- [ ] Exec streams stdout/stderr
- [ ] Stream frames carry base64 payloads
- [ ] Final exec response includes exit code, duration, timeout, and truncation metadata

**Verification:**
- [ ] Integration test: microVM exec streaming
- [ ] Unit test: stream frame encoding

**Files:** `pkg/runtime/microvm/vsock.go (new — to be created)`, `pkg/runtime/microvm/exec.go (new — to be created)`, `tests/runtime/microvm/exec_test.go (new — to be created)`
**Size:** M
**Depends on:** T21.2

### T21.4: File and artifact transfer protocol

**Acceptance criteria:**
- [ ] File transfer works
- [ ] Artifact transfer works
- [ ] No direct host filesystem mount is required

**Verification:**
- [ ] Integration test: microVM file/artifact transfer

**Files:** `pkg/runtime/microvm/files.go (new — to be created)`, `pkg/runtime/microvm/artifacts.go (new — to be created)`, `tests/runtime/microvm/transfer_test.go (new — to be created)`
**Size:** M
**Depends on:** T21.2

## Verification Plan

- [ ] Guest binaries build
- [ ] Guest reports readiness
- [ ] Exec streams over vsock
- [ ] File/artifact transfer works without host mounts
- [ ] Protocol codec handles all frame types

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

- Non-microVM backend control protocols
