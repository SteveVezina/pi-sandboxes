# F11: Secrets & Network Model

> Source: `SPEC.md` §6 Features F11
> Status: 🟢 Reviewed — ready for planning (ADR-006 Accepted 2026-08-31). The network/secrets *model* (modes, default-deny, allowlist, `exposeTo`, no-plaintext-secrets) is specified and its decision logic (`pkg/network`, `pkg/secrets`) is built and unit-tested. Runtime *enforcement* is delivered by F30 Egress Proxy tasks T30.1–T30.8 (ADR-006). F11's own tasks below now trace to those F30 tasks rather than duplicating them. The AC boxes stay unchecked until the F30 tasks land and the integration/security tests pass.
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F11 | Secrets & Network Model | Configurable network modes (none/restricted/open), domain allowlist, and credential handling through the egress proxy | M2 |

## Expanded Specification

The network and secrets model provides configurable network isolation and secure credential handling for sandboxes.

### Network Modes

| Mode | Description |
|------|-------------|
| `none` | No outbound network. |
| `restricted` | Domain allowlist through the daemon-owned egress proxy. Default. |
| `open` | Full outbound access. User must opt in. |

Default deny targets:
- `169.254.169.254` (cloud metadata)
- Host gateway IP
- Host localhost
- Private LANs (unless allowed)
- Local Kubernetes ranges (when applicable)

Domain-aware egress is preferred over IP-based filtering because package registries and Git hosts use dynamic IPs.

### Secrets Model

- Environment variables: deny-by-default
- SSH agent: opt-in through the egress proxy only
- Git credentials: injected by the egress proxy into approved outbound requests (not dumped into environment)
- Per-secret `exposeTo` policy (e.g., github-token → git only, never to shell)
- No plaintext secrets are stored on host disk under `~/.pi-box`

Initial Git support:
1. Public HTTPS clone
2. User-approved SSH agent use through the egress proxy for Git only
3. User-approved token injection scoped to Git operations through the egress proxy

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-11.1: `none` mode blocks all outbound network *(2026-08-28: reset — no code path sets container/namespace network to "none"; `exec.Request.NetworkMode` is accepted and stored but never read by the exec engine)*
- [ ] AC-11.2: `restricted` mode enforces domain allowlist *(2026-08-28: reset — `Policy.IsAllowed` implements the decision correctly in isolation, per `tests/network/network_test.go`, but nothing calls it at request time; no proxy or DNS filter sits in the sandbox's egress path)*
- [ ] AC-11.3: `open` mode allows full outbound access *(2026-08-28: reset — true today only because nothing restricts anything; not a verified "open works as declared" state, since "restricted" and "none" don't actually differ from it)*
- [ ] AC-11.4: Default deny: metadata endpoint (169.254.169.254), host localhost, private LANs *(2026-08-28: reset — `DefaultDeny` is real data in `pkg/network/modes.go` but, like `IsAllowed`, is never consulted at runtime; 169.254.169.254 is currently reachable from every sandbox)*
- [ ] AC-11.5: Git credentials brokered through the egress proxy (not dumped into environment) *(2026-07-15: AC updated per PROP-009)*
- [ ] AC-32.1: Outbound requests from a sandbox route through the daemon-owned egress proxy in restricted mode *(2026-07-15: added per PROP-009)*
- [ ] AC-32.2: Injected credentials are not readable from inside the sandbox *(2026-07-15: added per PROP-009)*

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/network/` | Network policy management |
| `pkg/secrets/` | Egress-proxy credential rules |
| F3: Fast Backend | Network namespace policy |
| F4: Compat Backend | Container network policy |
| F17: Policy Enforcement | Network/secrets are part of policy |

## Security Considerations

- Domain allowlist enforced at network layer (not application layer)
- Default deny all outbound (except allowed domains)
- SSH and token credentials never appear in sandbox environment
- Secret exposure scoped per-secret with `exposeTo` policy
- No plaintext secrets under `~/.pi-box`
- Metadata endpoint (169.254.169.254) explicitly blocked

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F17: Policy Enforcement | Internal feature | Depends on policy foundation |
| F3: Fast Backend | Internal feature | Network policy applied by backend |
| F4: Compat Backend | Internal feature | Network policy applied by backend |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go network/secrets packages |
| **Infrastructure** | Network namespace policy, container network policy |

**ADR references:** ADR-006 (Egress Enforcement and Credential Delivery) — Accepted 2026-08-31. Answers all three open questions below: (1) `restricted` mode makes the daemon proxy the only routable outbound endpoint, set per-driver via `NetworkSpec` on `Driver.Create`; (2) proxy resolves `credential id → real value` from an in-memory `CredentialStore` fed by OS keychain / daemon env / off-`~/.pi-box` file; (3) network mode is per-sandbox (fixed at create), not per-exec. Enforcement work is tracked as F30 T30.1–T30.8.
**ADR gaps:** None — resolved by ADR-006.

## Tasks

### T12.1: Network policy engine 🔴 *(2026-08-31: decision logic done + unit-tested; runtime enforcement is F30 T30.1–T30.5 per ADR-006. This task closes when those land and the "real sandbox" integration tests below pass.)*

**Description:** Implement network policy engine. Domain allowlist, default deny, network mode enforcement.

**Acceptance criteria:**
- [x] `Policy.IsAllowed(host)` correctly decides allow/deny for none/restricted/open modes plus default-deny targets, as a pure function
- [ ] `none` mode: no network access (iptables DROP all or namespace without network) — **not wired**: sandbox containers always get Docker's default `bridge` network; no fast-mode namespace network isolation exists either
- [ ] `restricted` mode: domain allowlist enforced — **not wired**: no proxy or DNS filter intercepts sandbox egress; `EgressProxy` in `pkg/network/egress_proxy.go` is a complete, tested `http.RoundTripper` but is never instantiated by the daemon
- [ ] `open` mode: no restrictions — trivially true today (nothing restricts anything), not a verified distinct state
- [ ] Default deny: 169.254.169.254, host localhost, private LANs — **not wired**: `DefaultDeny` data exists but is never consulted at runtime
- [x] Domain-aware egress (not IP-based) for dynamic registries — true of the decision function's design; moot without an enforcement point calling it

**Verification:**
- [x] `go build ./pkg/network/...`
- [x] Unit test: `Policy.IsAllowed` returns correct decisions for none/restricted/open and default-deny targets (`tests/network/network_test.go`)
- [ ] Integration test: `none` mode blocks all network from a real sandbox — **does not exist**
- [ ] Integration test: `restricted` mode allows only allowed domains from a real sandbox — **does not exist**
- [ ] Integration test: `open` mode allows all network from a real sandbox — **does not exist**

**Files:** `pkg/network/policy.go`, `pkg/network/modes.go`, `pkg/network/egress_policy.go`, `pkg/network/egress_proxy.go`
**Size:** M
**Depends on:** F17 (Policy Enforcement), F30 T30.1–T30.5 (Egress Proxy — 🟢 Reviewed, ADR-006)
**Implemented by:** F30 T30.1 (policy assembly), T30.2 (proxy listener), T30.3/T30.4 (driver single-endpoint egress), T30.5 (proxy env), T30.6 (denial logging)

### T12.2: Egress-proxy credential injection 🟡 *(2026-08-31: F30 T30.7 done — credential registration API + in-memory store + daemon value sources (no `~/.pi-box` plaintext). F30 T30.8 (injection into approved requests) still open.)*

**Description:** Implement egress-proxy credential injection for Git credentials. SSH agent and token use are scoped to approved outbound requests, and credentials are never readable from inside the sandbox. *(2026-07-15: AC updated per PROP-009.)*

**Acceptance criteria:**
- [ ] SSH credentials are usable only through approved Git egress
- [ ] Token injection is scoped to approved Git operations
- [ ] Secrets never appear in sandbox environment variables or filesystem
- [ ] Secret exposure scoped per-secret with `exposeTo` policy
- [ ] Plaintext secrets are not stored under `~/.pi-box`

**Verification:**
- [x] `go build ./pkg/secrets/...`
- [ ] Integration test: injected credentials not visible in sandbox environment or filesystem
- [ ] Integration test: Git clone works with brokered credentials *(2026-08-28: reset — no such test exists; `pkg/api/sandbox_clone.go` calls plain `git clone` inside the container with no credential broker involved at all, and `EgressProxy.injectCredentials` doesn't even resolve a real secret value — it writes the literal string `"[credential-injected]"`)*

**Files:** `pkg/secrets/broker.go`, `pkg/secrets/ssh.go`, `pkg/secrets/token.go`, `pkg/secrets/inject.go`, `pkg/secrets/rules.go`
**Size:** M
**Depends on:** F17 (Policy Enforcement), F30 T30.7–T30.8 (Egress Proxy — 🟢 Reviewed, ADR-006)
**Implemented by:** F30 T30.7, T30.8

## Verification Plan

- [x] `go build ./pkg/network/...` and `go build ./pkg/secrets/...` succeed
- [ ] Network modes work correctly (none/restricted/open) *(decision logic only; not enforced at runtime)*
- [ ] Default deny targets blocked *(data only; not enforced at runtime)*
- [ ] Git credentials injected by egress proxy and not readable in sandbox
- [ ] Git operations work with brokered credentials *(clone works, but with zero broker involvement — plain unauthenticated `git clone`)*

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Domain allowlist enforcement mechanism not specified | §17 Network model | Add: "DNS-based egress filtering with domain allowlist" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| ~~How to enforce domain allowlist in fast mode (no proxy)?~~ | F11 | **ADR-006:** fast mode also routes through the daemon proxy — network namespace `nftables` default-drop with a single accept rule for `ProxyAddr`. No separate namespace-local filter. |
| ~~No enforcement point exists at all yet~~ (three sub-questions on network attachment, real secret resolution, per-sandbox vs per-exec) | F11, F30 | **ADR-006 (Accepted 2026-08-31):** sandbox egress enforcement architecture. Blocks F11 AC-11.1-11.5 and F30 in full until Accepted + cascaded. |

## Out of Scope

- Per-exec network override (future — currently per-sandbox)
- VPN support (explicitly out of scope per SPEC.md §25)
- Packet capture (explicitly out of scope per SPEC.md §25)
- Advanced debuggers/profilers (explicitly out of scope per SPEC.md §25)
