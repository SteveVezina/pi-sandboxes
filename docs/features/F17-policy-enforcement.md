# F17: Policy Enforcement

> Source: `SPEC.md` §6 Features F17
> Status: 🟡 Partially implemented (T17.1/T17.2 secrets+egress-proxy items blocked on egress-enforcement ADR)
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F17 | Policy Enforcement | Default security policy: no host home mount, no Docker socket, process limits, output limits | M2 |

## Expanded Specification

Policy enforcement is the security foundation that applies default constraints to all sandboxes. It enforces the security model defined in SPEC.md §8 across all backends.

Default policy:
```yaml
defaults:
  filesystem:
    hostHomeMount: false
    workspace: read-write
    artifacts: read-write
    caches: scoped
    root: read-only-where-possible

  process:
    maxProcesses: 256
    defaultTimeout: 120s
    maxOutput: 8MiB

  network:
    mode: restricted
    deny:
      - 169.254.169.254
      - host-localhost
      - private-lans
      - cluster-local
    allow:
      - github.com
      - registry.npmjs.org
      - pypi.org
      - files.pythonhosted.org
      - proxy.golang.org
      - crates.io
      - static.crates.io

  secrets:
    env: deny-by-default
    sshAgent: opt-in-through-egress-proxy
    gitCredentials: egress-proxy-injected
```

Never mount these by default:
- `/var/run/docker.sock`
- Host `/`
- Host `$HOME`
- Cloud metadata credentials
- SSH private keys
- Kubernetes config
- Cloud provider config directories

Policy is enforced by:
1. **Fast backend** — namespaces, cgroups, seccomp, Landlock
2. **Compat backend** — container hardening (no privileged, no host network, caps dropped, seccomp)

Policy is configurable per sandbox via create request flags, but defaults are always applied first.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-17.1: Host home directory not mounted by default
- [x] AC-17.2: Docker socket not mounted by default
- [x] AC-17.3: Cloud metadata credentials not accessible
- [x] AC-17.4: SSH private keys not mounted by default
- [ ] AC-17.5: Git credentials brokered through the egress proxy (not dumped into environment) *(2026-07-15: AC updated per PROP-009)*
- [x] AC-17.6: Exec output limited to 8MiB by default
- [x] AC-17.7: Exec timeout 120s by default
- [x] AC-17.8: Max processes 256 by default
- [x] AC-34.1: No sandbox has a writable bind mount of any host directory by default *(2026-07-15: added per PROP-009; 2026-08-31 verified: compat create/exec derive workspace, artifacts, and cache sources from daemon-managed named volumes (`managedVolumeName`); host paths surface only when the caller explicitly sets `WorkspaceMode: "bind"`. `CreateRequest` exposes no arbitrary-mount field. Regression guard: `pkg/api/mounts_internal_test.go`)*
- [ ] AC-34.3: Secrets are represented as egress-proxy injection rules and are not stored as plaintext files under `~/.pi-box` *(2026-07-15: added per PROP-009)*

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/policy/` | Policy engine |
| F3: Fast Backend | Policy enforced by backend |
| F4: Compat Backend | Policy enforced by backend |
| F12: Secrets & Network | Policy includes network/secrets defaults |

## Security Considerations

- Policy is the security foundation — all backends must enforce it
- Policy defaults are deny-by-default
- Policy overrides require explicit user opt-in
- Policy violations are logged and reported

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F3: Fast Backend | Internal feature | Backend enforces policy |
| F4: Compat Backend | Internal feature | Backend enforces policy |
| F12: Secrets & Network | Internal feature | Policy includes network/secrets |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go policy engine in `pkg/policy/` |
| **Infrastructure** | Backend-specific policy enforcement |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T17.1: Policy engine 🟡 (secrets/egress-proxy items blocked on egress-enforcement ADR)

**Description:** Implement policy engine. Default policy loading, per-sandbox overrides, policy validation. *(2026-07-15: secrets/cache host-decoupling policy updated per PROP-009.)*

**Acceptance criteria:**
- [x] Default policy loaded from `~/.pi-box/config.yaml`
- [x] Filesystem defaults: hostHomeMount=false, workspace=rw, artifacts=rw, caches=scoped, root=readonly
- [x] Process defaults: maxProcesses=256, defaultTimeout=120s, maxOutput=8MiB
- [x] Network defaults: mode=restricted, deny list, allow list
- [ ] Secrets defaults: env=deny-by-default, sshAgent=opt-in-through-egress-proxy, gitCredentials=egress-proxy-injected *(blocked on egress-enforcement ADR — see F11/F30)*
- [x] Per-sandbox overrides merge with defaults (override cannot relax defaults)
- [x] Policy rejects writable host cache bind mounts by default *(2026-08-31: structural — cache sources are always daemon-managed named volumes; no request field can inject a host cache path. Guard test in `pkg/api/mounts_internal_test.go`)*
- [x] Policy validated on sandbox creation

**Verification:**
- [x] `go build ./pkg/policy/...`
- [x] Unit test: default policy loaded correctly
- [x] Unit test: per-sandbox overrides merge correctly

**Files:** `pkg/policy/engine.go`, `pkg/policy/default.go`, `pkg/policy/override.go`
**Size:** M
**Depends on:** None

### T17.2: Backend policy enforcement ✅ (host-mount/limit ACs verified 2026-08-31)

**Description:** Implement backend-specific policy enforcement. Fast backend applies namespaces/cgroups/seccomp. Compat backend applies container hardening.

**Acceptance criteria:**
- [x] Fast backend enforces: namespaces, cgroups, seccomp, Landlock
- [x] Compat backend enforces: no privileged, no host network, caps dropped, seccomp
- [x] Docker socket never mounted
- [x] Host home never mounted
- [x] Writable host cache bind mounts are rejected by default *(2026-08-31: cache mounts are always daemon-managed named volumes; guard test `pkg/api/mounts_internal_test.go`)*
- [x] Cloud metadata never accessible
- [x] Process limits enforced (maxProcesses=256)
- [x] Output limits enforced (maxOutput=8MiB)
- [x] Timeout enforced (defaultTimeout=120s)

**Verification:**
- [x] `go build ./pkg/policy/...`
- [x] Integration test: fast backend enforces policy
- [x] Integration test: compat backend enforces policy
- [x] Security test: sandbox cannot access host home
- [x] Security test: sandbox cannot access Docker socket
- [x] Security test: sandbox cannot access cloud metadata

**Files:** `pkg/policy/fast.go`, `pkg/policy/compat.go`
**Size:** M
**Depends on:** T17.1 (policy engine), F3 (Fast Backend), F4 (Compat Backend)

## Verification Plan

- [x] `go build ./pkg/policy/...` succeeds
- [x] Default policy loaded correctly
- [x] Fast backend enforces all security constraints
- [x] Compat backend enforces all security constraints
- [x] Docker socket never mounted
- [x] Host home never mounted
- [x] Cloud metadata never accessible
- [x] Process limits enforced
- [x] Output limits enforced
- [x] Timeout enforced

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Policy override relaxation not specified | §15 Security defaults | Add: "Overrides cannot relax default deny policies" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Per-user policy profiles (future)
- Policy versioning (future)
- Policy audit logging (future)
- Dynamic policy updates (future)
