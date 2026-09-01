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

Bearer-token auth is enforced **server-side** by the daemon, not only wired
client-side. When a token is configured (`PI_DAEMON_TOKEN` env var), every
route except `GET /health` and CORS pre-flight (`OPTIONS`) requires
`Authorization: Bearer <token>`; a missing or wrong token returns `401` and
never falls through to handler execution. When the daemon's HTTP listener is
bound to a non-loopback address (`--http-addr` other than `127.0.0.1`/`::1`)
the token is **mandatory** — the daemon refuses to start without it
(fail-closed). The Unix socket and a loopback-only HTTP listener keep the
"local user trust" model and need no token.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-26.1: Remote daemon API calls are authenticated *(2026-09-01: server-side bearer enforcement added — see T23.5; previously only the client attached the header)*
- [x] AC-26.2: Remote access works over SSH-friendly transport
- [x] AC-26.3: Tailscale/WireGuard network paths are supported without API redesign
- [x] AC-26.4: Credentials are not persisted inside sandbox workspaces
- [x] AC-26.5: Remote workstation use case works end-to-end
- [x] AC-26.6: `http` remote contexts require bearer-token auth
- [x] AC-26.7: `ssh` remote contexts use SSH agent transport authentication
- [x] AC-26.8: Remote auth failures never fall back to unauthenticated access *(2026-09-01: enforced server-side — 401 short-circuits the middleware chain; non-loopback bind without a token fails startup)*

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/remote/` | Remote daemon transport/auth |
| `cmd/pi-box/context/` | Contexts reference remote transport settings |
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

### T23.5: Server-side bearer-token enforcement ✅

**Description:** The daemon HTTP API must reject unauthenticated requests when a
token is configured, and must fail-closed when its HTTP listener is exposed on
a non-loopback address. Closes the gap where AC-26.1/AC-26.8 were only wired
client-side.

**Acceptance criteria:**
- [x] `PI_DAEMON_TOKEN` env var configures the expected bearer token
- [x] With a token set, every route except `GET /health` and `OPTIONS` requires
      `Authorization: Bearer <token>`; missing/wrong → `401`, handler not called
- [x] Token comparison is constant-time
- [x] `--http-addr` selects the HTTP bind host (default `127.0.0.1`); `PORT` /
      `PI_HTTP_ADDR` env fallbacks for PaaS
- [x] Non-loopback `--http-addr` without `PI_DAEMON_TOKEN` → daemon refuses to
      start with an actionable error
- [x] Unix socket and loopback HTTP are unaffected when no token is set

**Verification:**
- [x] `pkg/daemon` unit tests: 401 without token, 200 with token, `/health` open,
      fail-closed on public bind (`tests/daemon` / `pkg/daemon/auth_internal_test.go`)
- [x] `go build ./... && go test ./...`

**Files:** `pkg/daemon/auth.go`, `pkg/daemon/daemon.go`, `cmd/pi-sandboxd/main.go`
**Size:** M
**Depends on:** T23.2

## Verification Plan

- [x] Remote auth works
- [x] SSH-friendly transport works
- [x] Remote workstation flow works end-to-end
- [x] Credentials are not persisted in sandbox workspaces
- [x] Auth failures do not fall back to unauthenticated access
- [x] Server-side: 401 without a valid token when `PI_DAEMON_TOKEN` set
- [x] Server-side: daemon refuses to start on a non-loopback bind without a token

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
