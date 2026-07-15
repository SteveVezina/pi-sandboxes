# F30: Egress Proxy

> Source: `SPEC.md` §6 Features F30
> Status: 🟡 Spec written
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

**ADR references:** None yet.
**ADR gaps:** Domain-aware proxy enforcement shape may need an ADR if implementation differs by backend.

## Tasks

### T30.1: Proxy policy model

**Description:** Define allowlist and credential-injection policy evaluated by the daemon-owned egress proxy.

**Acceptance criteria:**
- [ ] Policy supports host/domain allowlist entries
- [ ] Policy supports per-credential `exposeTo` rules
- [ ] Policy rejects plaintext secret file storage under `~/.pi-box`
- [ ] Policy decisions redact credentials in logs

**Verification:**
- [ ] Unit tests for allow/deny decisions
- [ ] Unit tests for redaction

**Files:** `pkg/network/egress_policy.go`, `pkg/secrets/rules.go`
**Size:** M
**Depends on:** F17

### T30.2: Restricted-mode proxy routing

**Description:** Route restricted-mode sandbox outbound traffic through the daemon-owned proxy.

**Acceptance criteria:**
- [ ] Restricted-mode HTTP(S) egress reaches allowlisted domains through proxy
- [ ] Non-allowlisted egress is denied
- [ ] Denials are recorded in logs/history
- [ ] Open and none modes preserve their specified semantics

**Verification:**
- [ ] Integration test: allowlisted Git/registry request succeeds
- [ ] Integration test: non-allowlisted egress is denied and logged

**Files:** `pkg/network/egress_proxy.go`, backend network integration files
**Size:** L
**Depends on:** T30.1

### T30.3: Credential injection

**Description:** Inject scoped credentials into approved outbound requests without exposing them inside the sandbox.

**Acceptance criteria:**
- [ ] Git token injection works for approved Git hosts
- [ ] Registry auth injection works for approved registries
- [ ] Credential is absent from sandbox environment, files, process args, and command output
- [ ] Credential value is redacted from daemon logs

**Verification:**
- [ ] Security test: token not readable from inside sandbox
- [ ] Integration test: authenticated Git clone succeeds through proxy

**Files:** `pkg/secrets/inject.go`, `pkg/network/egress_proxy.go`
**Size:** L
**Depends on:** T30.2

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
| Which backend-neutral mechanism routes all restricted-mode traffic through the proxy? | F30, F3, F4, F18, F20 | ADR-NNN: Backend-neutral egress proxy routing |

## Out of Scope

- Enterprise identity management
- VPN support
- Packet capture UI
