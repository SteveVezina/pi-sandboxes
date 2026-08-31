# F30: Egress Proxy

> Source: `SPEC.md` §6 Features F30
> Status: 🔵 In progress — T30.1 (per-sandbox egress policy) + T30.2 (daemon proxy listener) done; T30.3–T30.8 open (ADR-006 Accepted 2026-08-31)
> Category: Service-layer / Security

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F30 | Egress Proxy | Daemon-owned egress proxy enforces allowlists and injects scoped credentials into approved outbound requests without exposing tokens inside the sandbox | M8 |

## Expanded Specification

Outbound traffic from a sandbox routes through a daemon-owned egress proxy in restricted mode. The proxy enforces the network policy at runtime and injects credentials into approved Git or registry requests. The agent never sees the token in environment variables, files, process arguments, or logs.

The egress proxy is the secret delivery mechanism for Git tokens, registry auth, and future scoped outbound credentials. Host-disk plaintext secret storage under `~/.pi-box` is not allowed.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-32.1: Outbound requests from a sandbox route through the daemon-owned egress proxy in restricted mode
- [ ] AC-32.2: Allowlisted outbound Git or registry requests succeed with scoped injected credentials
- [ ] AC-32.3: Injected credentials are not readable from inside the sandbox
- [ ] AC-32.4: Non-allowlisted egress is denied and recorded in logs/history
- [ ] AC-34.3: Secrets are represented as egress-proxy injection rules and are not stored as plaintext files under `~/.pi-box`

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/network/` | Proxy routing and allowlist enforcement |
| `pkg/secrets/` | Credential injection rules, no plaintext host-disk secrets |
| `pkg/policy/` | Policy validation for egress and credential exposure |
| F11: Secrets & Network Model | Replaces secret broker shape |
| F17: Policy Enforcement | Enforces no plaintext secrets and no readable sandbox credentials |

## Security Considerations

- Deny by default for non-allowlisted egress.
- Credentials are injected only into approved outbound requests.
- Credentials must not appear inside sandbox environment, filesystem, process args, command output, or logs.
- Proxy decisions are recorded for audit/debugging without logging secret values.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F11: Secrets & Network Model | Internal feature | Needs re-verify |
| F17: Policy Enforcement | Internal feature | Needs re-verify |
| F9: Output Delivery | Internal feature | Needs re-verify |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Daemon-owned egress proxy and decision logs |
| **Security** | Credential injection rules and redaction |
| **Runtime integration** | Route sandbox outbound traffic through proxy in restricted mode |

**ADR references:** ADR-006 (Egress Enforcement and Credential Delivery) — Accepted 2026-08-31. Governs proxy shape, `NetworkSpec` driver input, credential resolution, and the no-TLS-MITM baseline.
**ADR gaps:** None — resolved by ADR-006.

## Tasks

> Re-scoped 2026-08-31 per ADR-006 (Accepted). Old T30.1–T30.3 (M/L/L) split
> into XS–M tasks. All 🔴 Not started.

### T30.1: Per-sandbox egress policy assembly ✅ *(2026-08-31)*

**Description:** Build a per-sandbox `network.Policy` at create time from `DefaultAllowlist` plus any per-sandbox allowlist additions, with `DefaultDeny` always applied, and persist the mode + allowlist on the sandbox record so the policy is rebuildable by sandbox ID. *(Implemented with the existing `network.Policy` mode/allowlist decision type via new `network.PolicyFor`; `EgressPolicy.Evaluate` (credential-injection evaluator) is wired later in T30.7–T30.8.)*

**Acceptance criteria:**
- [x] `CreateRequest` accepts `network.mode` (`none`|`restricted`|`open`, default `restricted`) and optional `network.allow` extra hosts — `pkg/api/sandbox_create.go`
- [x] Per-sandbox policy built and retrievable by sandbox ID — `sandboxNetworkPolicy(store, id)` in `pkg/api/network.go`; mode + allowlist persisted on `sandbox.Meta` (`network_mode`, `network_allow`)
- [x] `DefaultDeny` always subtracted, even if a user allow entry matches and even in `open` mode — `network.PolicyFor` sets `DenyList: DefaultDeny` for every mode
- [x] `open` requires explicit request; never chosen by default — `NewMeta` and the create handler default to `restricted`; legacy sandboxes with no persisted mode fall back to `restricted`
- [x] Invalid mode rejected with `400`

**Verification:**
- [x] Unit: `tests/network/scope_test.go` (default, extras, deny-wins, none, open, invalid)
- [x] Unit: `pkg/api/network_internal_test.go` (persisted mode retrievable by ID, none blocks all, legacy → restricted)
- [x] `go test ./...` — 457 pass

**Files:** `pkg/network/scope.go` (new), `pkg/api/sandbox_create.go`, `pkg/api/network.go` (new), `pkg/sandbox/meta.go`, `pkg/sandbox/store.go`
**Size:** S
**Depends on:** None

### T30.2: Daemon egress proxy listener ✅ *(2026-08-31)*

**Description:** Start one forward-proxy (`network.ProxyServer`) HTTP/CONNECT listener per daemon on `127.0.0.1:<port>`. Resolve the per-sandbox policy on each request from the `Proxy-Authorization` basic-auth username (= sandbox ID; T30.5 injects it). Refuse non-allowlisted hosts with `403`.

**Acceptance criteria:**
- [x] Proxy listener starts with the daemon and is addressable as `ProxyAddr` — `Daemon.SetEgressProxyPort` / `Daemon.ProxyAddr`, `pi-sandboxd --egress-proxy-port`
- [x] `CONNECT` (HTTPS tunnel) and plaintext HTTP forwarding both work — `ProxyServer.tunnel` / `ProxyServer.forward`
- [x] Each request is evaluated against the originating sandbox's policy — sandbox ID from `Proxy-Authorization`, `Daemon.egressPolicyResolver` → `network.PolicyFor`
- [x] Non-allowlisted host → `403`; missing sandbox identity → `407`; unknown sandbox → `403`

**Deviations:** default port is `0` (disabled) until T30.3–T30.5 complete the enforcement chain (nothing routes sandboxes through it yet). Credential injection deferred to T30.8 — `Proxy-Authorization` is used only as a sandbox-identity carrier here.

**Verification:**
- [x] Unit: `tests/network/proxy_server_test.go` (allow forwards, deny 403, unknown-sandbox 403, missing-auth 407, CONNECT tunnel, CONNECT deny 403)
- [x] Integration: `tests/daemon/daemon_test.go::TestDaemon_EgressProxy_EnforcesSandboxPolicy` (restricted allows github.com, denies evil.example.com; none-mode denies all)

**Files:** `pkg/network/proxy_server.go` (new), `pkg/daemon/daemon.go`, `cmd/pi-sandboxd/main.go`
**Size:** M
**Depends on:** T30.1

### T30.3: `NetworkSpec` driver contract + compat single-endpoint egress 🔴

**Description:** Add `NetworkSpec{Mode, ProxyAddr}` to `Driver.Create` (per ADR-006). Compat/secure (OCI) driver: attach the container to a daemon-managed network whose only reachable non-container endpoint is the proxy (or `--network none` + host proxy + egress firewall); keep `--cap-drop=ALL`, no `--network host`.

**Acceptance criteria:**
- [ ] `NetworkSpec` on `Driver.Create`; all drivers compile against it
- [ ] `restricted`: container can reach only `ProxyAddr`; `169.254.169.254`, host gateway, host localhost unreachable
- [ ] `none`: no outbound route
- [ ] `open`: unrestricted (opt-in only)

**Verification:**
- [ ] Integration (compat): `curl 169.254.169.254` from sandbox fails in `restricted`
- [ ] Integration (compat): direct `curl https://example.com` fails; via proxy env succeeds for allowlisted host

**Files:** `pkg/runtime/driver.go`, `pkg/runtime/compat/`, `pkg/runtime/oci/cli.go`, `pkg/api/sandbox_create.go`
**Size:** M
**Depends on:** T30.2

### T30.4: Fast backend single-endpoint egress 🔴

**Description:** Fast driver: sandbox network namespace with veth to a daemon-owned bridge; `nftables` default-drop egress with a single accept rule for `ProxyAddr`. `HTTP_PROXY`/`HTTPS_PROXY` env still set.

**Acceptance criteria:**
- [ ] `restricted`: namespace can reach only `ProxyAddr`
- [ ] `none`: namespace has no external route
- [ ] Metadata IP and host gateway unreachable in `restricted`

**Verification:**
- [ ] Integration (fast, Linux): same egress negative tests as T30.3

**Files:** `pkg/runtime/fast/`, `pkg/runtime/fast/mounts.go`
**Size:** M
**Depends on:** T30.2

### T30.5: Proxy env injection into exec 🔴

**Description:** Inject `HTTP_PROXY`, `HTTPS_PROXY`, `NO_PROXY`, and `GIT_*` proxy env into every exec in a `restricted` sandbox so proxy-aware tooling works without TLS interception.

**Acceptance criteria:**
- [ ] Restricted-mode exec environment contains proxy vars pointing at `ProxyAddr`
- [ ] `none`/`open` modes do not inject proxy vars
- [ ] User-supplied env cannot unset the proxy vars in `restricted`

**Verification:**
- [ ] Unit: env assembly per mode
- [ ] Integration: `npm`/`pip` install through proxy for allowlisted registry

**Files:** `pkg/api/sandbox_exec.go`, exec engine env assembly
**Size:** S
**Depends on:** T30.3

### T30.6: Egress denial logging 🔴

**Description:** Record each proxy denial (and optionally allow) as an entry in the originating sandbox's logs/history (F10), host only, credentials redacted via `secrets.Redact`.

**Acceptance criteria:**
- [ ] Denied egress appears in `pi-box logs`/`history` for that sandbox
- [ ] No credential material in the log line
- [ ] Allow decisions optionally recorded at debug level

**Verification:**
- [ ] Integration: denied request produces a redacted history entry

**Files:** `pkg/network/egress_proxy.go`, `pkg/logs/` (or F10 log sink)
**Size:** S
**Depends on:** T30.2

### T30.7: Credential registration API + daemon sources 🔴

**Description:** `POST /v1/credentials` → `{id, type, hosts, injectAs}` + value held in the in-memory `CredentialStore` only. Daemon resolves real values from OS keychain / daemon env / off-`~/.pi-box` file (config-named). Nothing written under `~/.pi-box`.

**Acceptance criteria:**
- [ ] `POST /v1/credentials` registers a credential rule; value never persisted to disk
- [ ] Daemon credential-source resolution order: keychain → daemon env → configured file path outside `~/.pi-box`
- [ ] `GET`/list returns rules with values redacted
- [ ] Sandbox create/exec references credentials by `id`

**Verification:**
- [ ] Unit: source resolution order, no-disk-write assertion
- [ ] Integration: register → referenced by sandbox

**Files:** `pkg/api/credentials.go` (new), `pkg/secrets/rules.go`, `pkg/secrets/broker.go`
**Size:** M
**Depends on:** None

### T30.8: Credential injection into approved requests 🔴

**Description:** Proxy resolves `id → value` at request time and injects: git-token via the in-sandbox git credential helper over the proxy control channel (scoped to approved host); registry-auth as the `Authorization` header on the forwarded request. Replace the `"[credential-injected]"` placeholder in `EgressProxy.injectCredentials`.

**Acceptance criteria:**
- [ ] Authenticated git clone of an approved private host succeeds through the proxy
- [ ] Registry auth injected for approved registry host
- [ ] Credential absent from sandbox env, files, process args, command output
- [ ] Credential value redacted from daemon logs

**Verification:**
- [ ] Security test: token not readable from inside sandbox (env dump, `/proc`, fs scan)
- [ ] Integration: authenticated git clone through proxy

**Files:** `pkg/network/egress_proxy.go`, `pkg/secrets/inject.go`
**Size:** M
**Depends on:** T30.7, T30.3

### Checkpoint: After T30.1–T30.4

- [ ] All tests pass; build clean
- [ ] `restricted` sandbox provably cannot reach `169.254.169.254` in both fast and compat
- [ ] Contract compliance: `NetworkSpec` matches ADR-006
- [ ] Human review before credential work (T30.7–T30.8)

## Verification Plan

- [ ] `go test ./pkg/network ./pkg/secrets ./pkg/policy`
- [ ] Integration test for allowed authenticated Git egress
- [ ] Integration test for denied non-allowlisted egress
- [ ] Security test proving injected credential is not readable inside sandbox

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| — | — | — |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| ~~Which backend-neutral mechanism routes all restricted-mode traffic through the proxy?~~ | F30, F3, F4, F18, F20 | **ADR-006 (Accepted 2026-08-31):** `NetworkSpec{Mode, ProxyAddr}` added to `Driver.Create`; each driver makes `ProxyAddr` the only routable outbound endpoint in `restricted` mode. |

## Out of Scope

- Enterprise identity management
- VPN support
- Packet capture UI
