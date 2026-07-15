# F21: MicroVM Guest Control Plane

> Source: `SPEC.md` §6 Features F21
> Status: 🟢 Implemented
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

- [x] AC-24.1: `pi-init` starts inside the guest and reports readiness
- [x] AC-24.2: `pi-agentd` communicates with the host over virtio-vsock
- [x] AC-24.3: Exec requests stream stdout/stderr over the guest control channel
- [x] AC-24.4: Guest lifecycle events map back to sandbox state
- [x] AC-24.5: File and artifact transfer work without direct host filesystem mounting
- [x] AC-24.6: Host and guest exchange newline-delimited JSON frames over virtio-vsock
- [x] AC-24.7: Exec stdout/stderr stream frames carry base64 payloads
- [x] AC-24.8: Final exec response includes exit code, duration, timeout, and truncation metadata
- [x] AC-24.9: Host marks sandbox warm only after the guest sends `ready`

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-init/` | Guest init binary |
| `cmd/pi-agentd/` | Guest agent binary |
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

### T21.1: Guest init and readiness ✅

**Description:** Build `pi-init` and start `pi-agentd` inside the guest.

**Acceptance criteria:**
- [x] `pi-init` starts inside the guest
- [x] `pi-agentd` starts and reports readiness
- [x] Readiness maps to sandbox state
- [x] Host marks sandbox warm only after `ready`

**Verification:**
- [x] Guest binary smoke test: `pi-init` starts `pi-agentd` and receives `ready`
- [x] Unit test: readiness maps to sandbox state only after `ready`
- [x] Integration test: readiness reported to host

**Files:** `cmd/pi-init/main.go`, `cmd/pi-agentd/main.go`, `pkg/runtime/microvm/readiness.go`, `tests/runtime/microvm/readiness_test.go`
**Size:** M
**Depends on:** F20

### T21.2: Vsock frame codec ✅

**Description:** Implement newline-delimited JSON frame encoding/decoding for host and guest.

**Acceptance criteria:**
- [x] Frames include type, id, session_id, method, payload, and optional error
- [x] Codec handles request, response, event, and stream frames
- [x] Invalid frames return actionable errors

**Verification:**
- [x] Unit tests for valid frame round-trip
- [x] Unit tests for invalid frame errors

**Files:** `pkg/runtime/microvm/protocol.go`, `tests/runtime/microvm/protocol_test.go`
**Size:** M
**Depends on:** None

### T21.3: Exec streaming protocol ✅

**Acceptance criteria:**
- [x] Exec streams stdout/stderr
- [x] Stream frames carry base64 payloads
- [x] Final exec response includes exit code, duration, timeout, and truncation metadata

**Verification:**
- [x] Unit test: exec request frame encoding
- [x] Unit test: stream frame encoding
- [x] Unit test: final exec response metadata

**Files:** `pkg/runtime/microvm/protocol.go`, `tests/runtime/microvm/protocol_test.go`
**Size:** M
**Depends on:** T21.2

### T21.4: File and artifact transfer protocol ✅

**Acceptance criteria:**
- [x] File transfer works
- [x] Artifact transfer works
- [x] No direct host filesystem mount is required

**Verification:**
- [x] Unit test: file read/write transfer frames
- [x] Unit test: artifact list/pull transfer frames
- [x] Integration test: file/artifact transfer over guest control plane transport

**Files:** `pkg/runtime/microvm/protocol.go`, `tests/runtime/microvm/transfer_test.go`, `pkg/runtime/microvm/files.go`, `pkg/runtime/microvm/artifacts.go`
**Size:** M
**Depends on:** T21.2

## Verification Plan

- [x] Guest binaries build
- [x] Guest reports readiness
- [x] Exec streams over vsock
- [x] File/artifact transfer works without host mounts
- [x] Protocol codec handles all frame types

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
