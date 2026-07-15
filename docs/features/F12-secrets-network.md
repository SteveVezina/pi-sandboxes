# F11: Secrets & Network Model

> Source: `SPEC.md` §6 Features F11
> Status: ⚠️ Needs re-verify
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

- [x] AC-11.1: `none` mode blocks all outbound network
- [x] AC-11.2: `restricted` mode enforces domain allowlist
- [x] AC-11.3: `open` mode allows full outbound access
- [x] AC-11.4: Default deny: metadata endpoint (169.254.169.254), host localhost, private LANs
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

### T12.1: Network policy engine

**Description:** Implement network policy engine. Domain allowlist, default deny, network mode enforcement.

**Acceptance criteria:**
- [x] `none` mode: no network access (iptables DROP all or namespace without network)
- [x] `restricted` mode: domain allowlist enforced
- [x] `open` mode: no restrictions
- [x] Default deny: 169.254.169.254, host localhost, private LANs
- [x] Domain-aware egress (not IP-based) for dynamic registries

**Verification:**
- [x] `go build ./pkg/network/...`
- [x] Integration test: `none` mode blocks all network
- [x] Integration test: `restricted` mode allows only allowed domains
- [x] Integration test: `open` mode allows all network

**Files:** `pkg/network/policy.go`, `pkg/network/modes.go`
**Size:** M
**Depends on:** F17 (Policy Enforcement — policy foundation)

### T12.2: Egress-proxy credential injection ⚠️

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
- [x] Integration test: Git clone works with brokered credentials

**Files:** `pkg/secrets/broker.go`, `pkg/secrets/ssh.go`, `pkg/secrets/token.go`
**Size:** M
**Depends on:** F17 (Policy Enforcement)

## Verification Plan

- [x] `go build ./pkg/network/...` and `go build ./pkg/secrets/...` succeed
- [x] Network modes work correctly (none/restricted/open)
- [x] Default deny targets blocked
- [ ] Git credentials injected by egress proxy and not readable in sandbox
- [x] Git operations work with brokered credentials

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Domain allowlist enforcement mechanism not specified | §17 Network model | Add: "DNS-based egress filtering with domain allowlist" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How to enforce domain allowlist in fast mode (no proxy)? | F12 | ADR-NNN: Domain-aware egress in namespace sandbox |

## Out of Scope

- Per-exec network override (future — currently per-sandbox)
- VPN support (explicitly out of scope per SPEC.md §25)
- Packet capture (explicitly out of scope per SPEC.md §25)
- Advanced debuggers/profilers (explicitly out of scope per SPEC.md §25)
