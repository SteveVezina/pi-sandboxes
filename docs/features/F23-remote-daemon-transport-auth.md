# F23: Remote Daemon Transport & Auth

> Source: `SPEC.md` §6 Features F23
> Status: 🟢 Implemented
> Category: Service-layer / Integration

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F23 | Remote Daemon Transport & Auth | SSH/Tailscale/WireGuard-friendly remote daemon access with secure local-to-remote API authentication and remote workstation support | M6 |

## Expanded Specification

Remote transport and auth allow the local CLI/SDK to talk to a remote `pi-sandboxd` without changing the sandbox API. The transport must be compatible with SSH-friendly forwarding and private networks such as Tailscale or WireGuard.

Per ADR-003, remote daemon access keeps the existing daemon HTTP API unchanged. Supported transports are `unix`, `http`, and `ssh`; `http` requires bearer-token auth and `ssh` uses SSH agent transport authentication.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-26.1: Remote daemon API calls are authenticated
- [x] AC-26.2: Remote access works over SSH-friendly transport
- [x] AC-26.3: Tailscale/WireGuard network paths are supported without API redesign
- [x] AC-26.4: Credentials are not persisted inside sandbox workspaces
- [x] AC-26.5: Remote workstation use case works end-to-end
- [x] AC-26.6: `http` remote contexts require bearer-token auth
- [x] AC-26.7: `ssh` remote contexts use SSH agent transport authentication
- [x] AC-26.8: Remote auth failures never fall back to unauthenticated access

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/remote/` | Remote daemon transport/auth (new — to be created) |
| `cmd/pi/context/` | Contexts reference remote transport settings |
| `sdk/typescript/` | Remote daemon connection support |
| `sdk/python/` | Remote daemon connection support |
| `docs/decisions/ADR-003-remote-context-and-auth-model.md` | Remote context/auth model decision |

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

**ADR references:** ADR-003 (Remote Context and Auth Model).
**ADR gaps:** None.

## Tasks

### T23.1: Remote client transport abstraction ✅

**Description:** Implement daemon client transport selection for `unix`, `http`, and `ssh` contexts without changing daemon routes.

**Acceptance criteria:**
- [x] `unix` transport keeps local socket behavior
- [x] `http` transport targets direct HTTP endpoints
- [x] `ssh` transport supports SSH-forwarded daemon access
- [x] Tailscale/WireGuard paths do not require API redesign

**Verification:**
- [x] Unit tests for transport selection (`tests/remote/client_test.go`)
- [x] Integration test: remote daemon request over configured transport (`tests/integration/remote_daemon_test.go`)

**Files:** `pkg/remote/client.go`, `tests/remote/client_test.go`
**Size:** M
**Depends on:** F2, F22

### T23.2: Remote authentication enforcement ✅

**Description:** Enforce remote auth rules for HTTP bearer tokens and SSH agent transport authentication.

**Acceptance criteria:**
- [x] Remote API calls authenticate successfully
- [x] `http` contexts require bearer-token auth
- [x] `ssh` contexts use SSH agent transport authentication
- [x] Remote auth failures never fall back to unauthenticated access
- [x] Credentials are not persisted in sandbox workspaces

**Verification:**
- [x] Unit tests for auth handling (`tests/remote/client_test.go`)
- [x] Unit test: auth failure does not fall back (`TestClient_RemoteAuthFailureDoesNotFallback`)

**Files:** `pkg/remote/client.go`, `pkg/remote/auth.go`
**Size:** M
**Depends on:** T23.1

### T23.3: SDK remote connection support ✅

**Description:** Add remote context connection support to TypeScript and Python SDKs.

**Acceptance criteria:**
- [x] TypeScript SDK can use remote context connection options
- [x] Python SDK can use remote context connection options
- [x] SDK auth failures are actionable

**Verification:**
- [x] TypeScript SDK tests pass (`tests/sdk/remote_auth_test.go`)
- [x] Python SDK tests pass (same file)

**Files:** `sdk/typescript/src/client.ts`, `sdk/python/src/pi_sandbox/__init__.py`, `tests/sdk/remote_auth_test.go`
**Size:** M
**Depends on:** T23.2

### T23.4: Remote workstation end-to-end flow ✅

**Acceptance criteria:**
- [x] Remote context can create a sandbox
- [x] Remote sandbox can clone/exec/diff/export
- [x] Credentials are not persisted in the workspace

**Verification:**
- [x] End-to-end test: remote workstation flow (`tests/integration/remote_daemon_test.go`)

**Files:** `tests/integration/remote_daemon_test.go`
**Size:** M
**Depends on:** T23.3

## Verification Plan

- [x] Remote auth works
- [x] SSH-friendly transport works
- [x] Remote workstation flow works end-to-end
- [x] Credentials are not persisted in sandbox workspaces
- [x] Auth failures do not fall back to unauthenticated access

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

- Hosted control plane
- Kubernetes-backed daemon mode
