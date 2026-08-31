# ADR-006: Egress Enforcement and Credential Delivery

## Status

Proposed (2026-08-31) — awaiting human acceptance.

Blocks: F11 (Secrets & Network Model), F30 (Egress Proxy), and the secrets /
egress-proxy acceptance criteria of F17 (Policy Enforcement). No block-spec
amendment is required — `SPEC.md` §6 (F11, F30) and §8 already mandate the
behaviour; this ADR only chooses the backend-neutral mechanism, which is the
open ADR gap recorded in F11, F17, and F30.

## Context

The network and secrets decision logic exists but is wired to nothing:

- `pkg/network` (`EgressPolicy.Evaluate`, `EgressProxy.RoundTrip`, `Mode.IsAllowed`,
  `DefaultDeny`, `DefaultAllowlist`) is correct in isolation and unit-tested, but
  has no callers outside its own tests. No proxy process is ever started and no
  listener is bound.
- Sandbox containers are always created on the runtime's default network
  (Docker `bridge` for compat/secure). `exec.Request.NetworkMode` is accepted,
  syntax-validated, and stored, then never read by the exec engine.
- `169.254.169.254` and the host gateway are reachable from every sandbox today.
- `pkg/secrets` (`CredentialStore`, `Broker`, `InjectGitToken`) stores no
  plaintext on disk (good) but credential *values* are placeholders — the proxy
  injects the literal string `"[credential-injected]"`. There is no path from a
  real token to an outbound request.

ADR-005 established that everything above the `Driver` interface is
runtime-neutral. Egress enforcement is the first security control that needs a
small, explicit addition to the driver contract rather than per-backend daemon
special cases.

## Decision

### 1. Network mode is a driver input, enforced by the driver

Add a `NetworkSpec` to `Driver.Create`:

```go
type NetworkSpec struct {
    Mode      NetworkMode // "none" | "restricted" | "open"
    ProxyAddr string      // host:port of the daemon egress proxy; set iff Mode == restricted
}
```

| Mode | Driver obligation |
|------|-------------------|
| `none` | Sandbox has no outbound route. No network namespace connectivity / `--network none`. |
| `restricted` (default) | The **only** reachable outbound endpoint is `ProxyAddr`. Everything else — including `169.254.169.254`, the host gateway, and host localhost — is unreachable at the network layer, not just at the application layer. |
| `open` (opt-in) | Unrestricted outbound. Requires explicit user opt-in; never selected by `auto`. |

Per-backend implementation of `restricted` (the one backend-specific slice):

- **fast**: sandbox network namespace with a veth pair to a daemon-owned bridge;
  `nftables` default-drop egress with a single accept rule for `ProxyAddr`.
- **compat / secure (OCI)**: attach the container to a daemon-managed user
  network whose only other member is the proxy, or run the proxy on the host
  and apply an egress firewall; `--cap-drop=NET_RAW` remains, no `--network host`.
- **microvm**: the guest control plane (ADR-002) configures the guest firewall
  to the same single-endpoint rule.

Rationale for network-layer enforcement over `HTTP_PROXY` env alone: env-var
proxy configuration is advisory and a process doing raw sockets bypasses it. By
making `ProxyAddr` the only routable destination, a bypass attempt fails closed.
`HTTP_PROXY` / `HTTPS_PROXY` / `GIT_PROXY` env is *also* set so that
proxy-aware tooling (git, npm, pip, cargo, go) works without TLS interception.

### 2. The egress proxy is a daemon-owned forward proxy

- The daemon starts one `EgressProxy` listener per daemon (not per sandbox),
  bound to a loopback/bridge address handed to sandboxes as `ProxyAddr`.
- It terminates `CONNECT` for HTTPS and forwards plaintext HTTP, evaluating each
  request's target host against the sandbox's `EgressPolicy`
  (`DefaultAllowlist` + per-sandbox additions, minus `DefaultDeny`).
- Requests to non-allowlisted hosts are refused with a `403` and recorded as an
  egress denial in that sandbox's logs/history (F10), with host but no
  credential material.
- No TLS man-in-the-middle in the baseline. Credential injection (§3) therefore
  applies to the `CONNECT`/HTTP metadata and to git-over-HTTPS via the git
  credential helper protocol, not by rewriting an established TLS stream.

### 3. Credential delivery without sandbox exposure

- Credentials are registered with the daemon out of band
  (`POST /v1/credentials` → `{id, type, hosts, injectAs}` + value). The value is
  held in the daemon's in-memory `CredentialStore` only. **Nothing is written
  under `~/.pi-box`** (AC-34.3).
- Daemon credential *sources* (for resolving real values) are, in priority
  order: OS keychain entry, environment variable present in the daemon's own
  environment at start, or a file path **outside** `~/.pi-box` named in daemon
  config. The sandbox never sees any of these.
- Sandbox create/exec references credentials by `id` only. The proxy resolves
  `id → value` at request time and injects:
  - **git-token**: served to the in-sandbox git credential helper over the proxy
    control channel, scoped to the approved host; never placed in env, argv,
    files, or command output.
  - **registry-auth**: injected as the `Authorization` header on the forwarded
    request to the approved registry host.
- All proxy decision logs pass through `secrets.Redact` before emission.

### 4. Defaults

- New sandboxes default to `Mode: restricted`.
- `DefaultDeny` is always subtracted, even from a user allowlist entry that would
  otherwise match (overrides cannot relax default deny — consistent with F17).
- `defaultTimeout`, `maxOutput`, `maxProcesses` are unchanged by this ADR.

## Consequences

- The `Driver` interface gains `NetworkSpec` on `Create`; F3, F4, F18, F20
  drivers each implement the single-endpoint egress rule. F19 selection is
  unaffected (mode is orthogonal to isolation tier).
- F11 AC-11.1..11.5, F17 AC-17.5 / AC-34.3, and F30 AC-32.1..32.4 / AC-34.3
  become implementable. Their tasks stay 🔴 until this ADR is Accepted and
  cascaded.
- `pkg/network` and `pkg/secrets` keep their current shapes; they gain callers
  (daemon proxy bootstrap, driver network setup, `/v1/credentials` handler).
- Baseline does not intercept TLS, so a sandbox can still open an allowlisted
  TLS connection and send arbitrary bytes over it. Domain-level allowlisting is
  the stated F11 model ("enforced at network layer, not application layer");
  payload inspection is explicitly out of scope.
- Transparent redirect + optional TLS interception (for non-proxy-aware
  clients and tighter control) is a documented follow-up, not required for M8.

## References

- `SPEC.md` §6 Features F11, F30; §8 Security Model
- `docs/features/F12-secrets-network.md` (F11), `docs/features/F17-policy-enforcement.md`,
  `docs/features/F30-egress-proxy.md`
- ADR-005 (runtime driver contract — this ADR extends `Driver.Create`)
- ADR-002 (microVM guest control protocol — guest firewall path)
- `pkg/network/egress_proxy.go`, `pkg/network/egress_policy.go`,
  `pkg/network/modes.go`, `pkg/secrets/rules.go`
