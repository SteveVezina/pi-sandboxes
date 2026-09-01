# F30: Egress Proxy

> Source: `SPEC.md` §6 Features F30
> Status: 🔵 In progress — T30.1/T30.2/T30.5/T30.6 done; T30.3 partial (contract + none-mode + proxy-env; L3 isolation → T30.4); T30.4 open (needs Linux host); T30.7/T30.8 open (gated on checkpoint review — credential handling). (ADR-006 Accepted 2026-08-31)
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

### T30.3: `NetworkSpec` driver contract + compat proxy wiring 🟡 *(2026-08-31 — contract + none-mode + proxy-env done; compat L3 isolation folded into T30.4)*

**Description:** Add `NetworkSpec{Mode, ProxyAddr}` to the driver contract (ADR-006). Map egress mode onto the OCI create args.

**Acceptance criteria:**
- [x] `NetworkSpec` on `SandboxSpec` / `oci.ContainerSpec` / `compat.ContainerSpec`; darwin-buildable drivers compile against it *(gvisor/runsc was already non-compiling against the current `oci.Engine` before this task — see F18 note in plan.md; its one `NetworkMode`→`Network` line was updated but the file's other breakage is out of scope)*
- [x] `none`: `--network none` — no outbound route (`oci.egressArgs`)
- [x] `open` / unset: `--network bridge`, no proxy env
- [x] `restricted` + `ProxyAddr`: `--network bridge` + `HTTP(S)_PROXY=http://<sandboxID>:x@<ProxyAddr>` + `NO_PROXY` injected into the container env; sandbox ID threads `create` → `meta` → `sandboxEgressNetwork` → `compat.ContainerSpec` → `oci.ContainerSpec`
- [x] `restricted` without a running daemon proxy (`--egress-proxy-port` unset): `--network bridge`, no proxy env — no behaviour change by default
- [ ] **`restricted` L3 single-endpoint isolation** (drop everything except `ProxyAddr` so non-proxy-aware clients and raw sockets also fail closed) — **moved to T30.4** for both compat and fast

**Verification:**
- [x] Unit: `tests/runtime/oci/engine_test.go` — none → `--network none` + no proxy env; restricted+proxy → sandbox-scoped `HTTP_PROXY`; restricted without proxy → bridge only
- [ ] Integration (compat, Linux+Docker): `curl 169.254.169.254` fails in `restricted` — needs T30.4

**Files:** `pkg/runtime/driver.go`, `pkg/runtime/oci/{engine,cli}.go`, `pkg/runtime/compat/create.go`, `pkg/api/{network,sandbox_create,sandbox_exec}.go`, `pkg/daemon/daemon.go`
**Size:** M
**Depends on:** T30.2

### T30.4: L3 single-endpoint egress isolation (fast + compat) 🔴 — **Linux host required**

**Description:** Make `ProxyAddr` the *only* reachable outbound endpoint in `restricted` mode, at the network layer, so raw sockets and non-proxy-aware clients also fail closed (ADR-006). Two backends:
- **fast**: sandbox network namespace with veth to a daemon-owned bridge; `nftables` default-drop egress + single accept rule for `ProxyAddr`.
- **compat/secure**: daemon-managed Docker network containing only the proxy, or `--network none` + host proxy reachable via a single firewall rule; drop `169.254.169.254`, host gateway, host localhost.

**Acceptance criteria:**
- [ ] `restricted`: sandbox can reach only `ProxyAddr` (fast + compat)
- [ ] `none`: no external route (already true for compat via `--network none`; add for fast)
- [ ] `169.254.169.254`, host gateway, host localhost unreachable in `restricted`
- [ ] `HTTP_PROXY`/`HTTPS_PROXY` env still set (from T30.3) so proxy-aware tools work

**Verification:**
- [ ] Integration (fast, Linux): `curl 169.254.169.254` and direct `curl https://github.com` both fail; `curl` via `$HTTP_PROXY` to an allowlisted host succeeds
- [ ] Integration (compat, Linux+Docker): same

**Files:** `pkg/runtime/fast/`, `pkg/runtime/compat/`, `pkg/runtime/oci/`, daemon bridge/network setup
**Size:** L → split when picked up on a Linux host
**Depends on:** T30.3

### T30.5: Proxy env injection into exec ✅ *(2026-08-31 — delivered by T30.3)*

**Description:** Ensure `HTTP_PROXY`/`HTTPS_PROXY`/`NO_PROXY` are present in `restricted`-mode sandbox processes so proxy-aware tooling works without TLS interception.

**Acceptance criteria:**
- [x] Restricted-mode container env contains proxy vars pointing at `ProxyAddr` — injected via `docker run -e` in `oci.egressArgs` (T30.3); every `docker exec` inherits the container env
- [x] `none`/`open` modes do not inject proxy vars
- [x] Caller cannot unset the proxy vars — `ExecRequest` carries no structured env (commands are shell strings); the container-level env is authoritative. An in-shell `unset HTTP_PROXY` bypass is exactly what T30.4's network-layer enforcement closes.
- [x] git honours `HTTP_PROXY`/`HTTPS_PROXY` directly — no separate `GIT_*` needed

**Verification:**
- [x] Unit: `tests/runtime/oci/engine_test.go` proves the create-time env
- [ ] Integration: `npm`/`pip` install through proxy for an allowlisted registry (Linux+Docker) — with T30.4

**Files:** `pkg/runtime/oci/cli.go` (T30.3)
**Size:** — (folded into T30.3)
**Depends on:** T30.3

### T30.6: Egress denial logging ✅ *(2026-08-31)*

**Description:** Record each proxy denial in the originating sandbox's egress log (F10), host + reason only, no credential material.

**Acceptance criteria:**
- [x] Denied egress is recorded per-sandbox — `logs.Manager.RecordEgress` appends to `~/.pi-box/sandboxes/<id>/logs/egress.jsonl`; daemon `egressDecisionSink` writes on every `!Allowed` decision
- [x] Exposed via API (`GET /v1/sandboxes/<id>/logs?action=egress`) and CLI (`pi-box egress <name>`)
- [x] No credential material in the log line — only `{timestamp, host, allowed, reason}`; `reason` is a fixed internal label
- [x] Allow decisions recorded only at daemon debug level, never in the sandbox egress log

**Verification:**
- [x] Unit: `tests/logs/logs_test.go` — `RecordEgress`/`EgressEvents` round-trip, newest-first, empty-when-absent
- [x] Integration: `tests/daemon/daemon_test.go::TestDaemon_EgressProxy_EnforcesSandboxPolicy` asserts the `evil.example.com` denial lands in the restricted sandbox's egress log and no allow entries appear

**Files:** `pkg/logs/egress.go` (new), `pkg/daemon/daemon.go`, `pkg/api/sandbox_logs.go`, `cmd/pi-box/box/box.go`
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

- [x] All tests pass (469); build clean; `gofmt`/`go vet` clean
- [ ] `restricted` sandbox provably cannot reach `169.254.169.254` in both fast and compat — **blocked: T30.4 needs a Linux+Docker host; not verifiable on the current darwin dev machine.** none-mode (`--network none`) and the proxy allowlist path are verified; the L3 drop-all-but-proxy rule is not yet implemented.
- [x] Contract compliance: `NetworkSpec{Mode, ProxyAddr}` on `SandboxSpec`/`oci.ContainerSpec`/`compat.ContainerSpec` matches ADR-006
- [ ] **Human review before credential work (T30.7–T30.8)** — pending. Credential registration + injection touch real secret handling; do not start until reviewed.

**Checkpoint verdict (2026-08-31): PARTIAL PASS.** Policy assembly, daemon proxy, mode→arg mapping, and denial logging are done and tested. Two items remain before F30 can close: T30.4 (Linux host) and the T30.7/T30.8 credential work (human review gate).

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
