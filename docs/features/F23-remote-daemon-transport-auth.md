# F23: Remote Daemon Transport & Auth

> Source: `SPEC.md` §6 Features F23
> Status: 🟡 Spec written
> Category: Service-layer / Integration

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F23 | Remote Daemon Transport & Auth | SSH/Tailscale/WireGuard-friendly remote daemon access with secure local-to-remote API authentication and remote workstation support | M6 |

## Expanded Specification

Remote transport and auth allow the local CLI/SDK to talk to a remote `pi-sandboxd` without changing the sandbox API. The transport must be compatible with SSH-friendly forwarding and private networks such as Tailscale or WireGuard.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-26.1: Remote daemon API calls are authenticated
- [ ] AC-26.2: Remote access works over SSH-friendly transport
- [ ] AC-26.3: Tailscale/WireGuard network paths are supported without API redesign
- [ ] AC-26.4: Credentials are not persisted inside sandbox workspaces
- [ ] AC-26.5: Remote workstation use case works end-to-end

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/remote/` | Remote daemon transport/auth (new — to be created) |
| `cmd/pi/context/` | Contexts reference remote transport settings |
| `sdk/typescript/` | Remote daemon connection support |
| `sdk/python/` | Remote daemon connection support |

## Security Considerations

- Remote API calls must be authenticated.
- Credentials must not be written into sandbox workspaces.
- Transport setup must preserve the existing API contract.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F2: Daemon API | Internal feature | Remote API target |
| F22: Remote Daemon Contexts | Internal feature | User-facing context selection |
| F16: SDKs | Internal feature | SDK remote support |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Remote client transport and auth |
| **Integration** | SSH/private-network friendly access patterns |

## Tasks

### T23.1: Remote transport and authentication

**Acceptance criteria:**
- [ ] Remote API calls authenticate successfully
- [ ] SSH-friendly transport works
- [ ] Tailscale/WireGuard paths do not require API redesign

**Verification:**
- [ ] Unit tests for auth handling
- [ ] Integration test: remote daemon request over configured transport

**Files:** `pkg/remote/client.go (new — to be created)`, `pkg/remote/auth.go (new — to be created)`
**Size:** L
**Depends on:** F2, F22

### T23.2: Remote workstation end-to-end flow

**Acceptance criteria:**
- [ ] Remote context can create a sandbox
- [ ] Remote sandbox can clone/exec/diff/export
- [ ] Credentials are not persisted in the workspace

**Verification:**
- [ ] End-to-end test: remote workstation flow

**Files:** `tests/integration/remote_daemon_test.go (new — to be created)`
**Size:** L
**Depends on:** T23.1

## Verification Plan

- [ ] Remote auth works
- [ ] SSH-friendly transport works
- [ ] Remote workstation flow works end-to-end
- [ ] Credentials are not persisted in sandbox workspaces

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Remote authentication scheme is not specified | §31 M6 | PROP-003 |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Remote auth and transport design | F22, F23 | ADR for remote daemon access |

## Out of Scope

- Hosted control plane
- Kubernetes-backed daemon mode
