# F11: Secrets & Network Model

> Source: `SPEC.md` §6 Features F11
> Status: 🔴 Not enforced — policy decision logic exists as an unwired library (2026-08-28: re-verified). Network mode (none/restricted/open) and default-deny targets are accepted and syntax-validated by the exec API but never applied to real sandbox network access: sandbox containers are always created with Docker's default `bridge` network regardless of declared mode, `exec.Request.NetworkMode` is stored but never read by the exec engine, and `Policy.IsAllowed`/`EgressProxy` (the actual enforcement/decision code in `pkg/network/`) are called from nowhere outside their own package and tests. This is the same architectural gap as F30 Egress Proxy (🔴 Not started) — real enforcement needs that proxy to exist. See Spec Gaps and ADR gaps below; previously-checked ACs in this doc overstated what the code does and have been reset.
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

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T12.1: Network policy engine ⚠️ *(2026-08-28: reset — decision logic exists and is unit-tested; runtime enforcement does not exist. "Integration test" claims below were unit tests of the decision function, not tests against a real sandbox's network access — corrected.)*

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

**Files:** `pkg/network/policy.go`, `pkg/network/modes.go`, `pkg/network/egress_policy.go`, `pkg/network/egress_proxy.go` (all unwired — no caller outside `pkg/network/` and its tests)
**Size:** M
**Depends on:** F17 (Policy Enforcement — policy foundation), F30 (Egress Proxy — 🔴 Not started; real enforcement needs this to exist)

### T12.2: Egress-proxy credential injection ⚠️ *(2026-08-28: re-verified — building blocks exist as unwired, unit-tested libraries; nothing calls them during a real clone)*

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

**Files:** `pkg/secrets/broker.go`, `pkg/secrets/ssh.go`, `pkg/secrets/token.go`, `pkg/secrets/inject.go`, `pkg/secrets/rules.go` (all unwired — no caller outside `pkg/secrets/` and its tests)
**Size:** M
**Depends on:** F17 (Policy Enforcement), F30 (Egress Proxy — 🔴 Not started)

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
| How to enforce domain allowlist in fast mode (no proxy)? | F11 | ADR-NNN: Domain-aware egress in namespace sandbox |
| No enforcement point exists at all yet — `pkg/network`'s `Policy`/`EgressProxy` and `pkg/secrets`' `Broker`/SSH/token helpers are complete, unit-tested libraries with zero callers outside their own packages. Need: (1) how sandbox containers/namespaces actually get attached to a mode-appropriate network (per-sandbox `--network none`? all traffic forced through a daemon-side proxy via `HTTP_PROXY`/iptables redirect?), (2) how the egress proxy resolves and injects a real secret value instead of the current `"[credential-injected]"` placeholder, (3) whether "network mode" is a per-sandbox or per-exec property, given `ExecRequest.Network` implies per-exec but containers are created once with a fixed network. | F11, F30 | ADR-NNN: sandbox egress enforcement architecture (blocks F11 AC-11.1-11.5, F30 in full) |

## Out of Scope

- Per-exec network override (future — currently per-sandbox)
- VPN support (explicitly out of scope per SPEC.md §25)
- Packet capture (explicitly out of scope per SPEC.md §25)
- Advanced debuggers/profilers (explicitly out of scope per SPEC.md §25)
