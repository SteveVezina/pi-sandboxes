# F17: Policy Enforcement

> Source: `SPEC.md` §6 Features F17
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F17 | Policy Enforcement | Default security policy: no host home mount, no Docker socket, process limits, output limits | M2 |

## Expanded Specification

Policy enforcement is the security foundation that applies default constraints to all sandbox sessions. It enforces the security model defined in SPEC.md §8 across all backends.

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
    sshAgent: opt-in
    gitCredentials: brokered
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

Policy is configurable per session via create request flags, but defaults are always applied first.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-17.1: Host home directory not mounted by default
- [ ] AC-17.2: Docker socket not mounted by default
- [ ] AC-17.3: Cloud metadata credentials not accessible
- [ ] AC-17.4: SSH private keys not mounted by default
- [ ] AC-17.5: Git credentials brokered (not dumped into environment)
- [ ] AC-17.6: Exec output limited to 8MiB by default
- [ ] AC-17.7: Exec timeout 120s by default
- [ ] AC-17.8: Max processes 256 by default

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

### T17.1: Policy engine

**Description:** Implement policy engine. Default policy loading, per-session overrides, policy validation.

**Acceptance criteria:**
- [ ] Default policy loaded from `~/.pi/config.yaml`
- [ ] Filesystem defaults: hostHomeMount=false, workspace=rw, artifacts=rw, caches=scoped, root=readonly
- [ ] Process defaults: maxProcesses=256, defaultTimeout=120s, maxOutput=8MiB
- [ ] Network defaults: mode=restricted, deny list, allow list
- [ ] Secrets defaults: env=deny-by-default, sshAgent=opt-in, gitCredentials=brokered
- [ ] Per-session overrides merge with defaults (override cannot relax defaults)
- [ ] Policy validated on sandbox creation

**Verification:**
- [ ] `go build ./pkg/policy/...`
- [ ] Unit test: default policy loaded correctly
- [ ] Unit test: per-session overrides merge correctly

**Files:** `pkg/policy/engine.go`, `pkg/policy/default.go`, `pkg/policy/override.go`
**Size:** M
**Depends on:** None

### T17.2: Backend policy enforcement

**Description:** Implement backend-specific policy enforcement. Fast backend applies namespaces/cgroups/seccomp. Compat backend applies container hardening.

**Acceptance criteria:**
- [ ] Fast backend enforces: namespaces, cgroups, seccomp, Landlock
- [ ] Compat backend enforces: no privileged, no host network, caps dropped, seccomp
- [ ] Docker socket never mounted
- [ ] Host home never mounted
- [ ] Cloud metadata never accessible
- [ ] Process limits enforced (maxProcesses=256)
- [ ] Output limits enforced (maxOutput=8MiB)
- [ ] Timeout enforced (defaultTimeout=120s)

**Verification:**
- [ ] `go build ./pkg/policy/...`
- [ ] Integration test: fast backend enforces policy
- [ ] Integration test: compat backend enforces policy
- [ ] Security test: sandbox cannot access host home
- [ ] Security test: sandbox cannot access Docker socket
- [ ] Security test: sandbox cannot access cloud metadata

**Files:** `pkg/policy/fast.go`, `pkg/policy/compat.go`
**Size:** M
**Depends on:** T17.1 (policy engine), F3 (Fast Backend), F4 (Compat Backend)

## Verification Plan

- [ ] `go build ./pkg/policy/...` succeeds
- [ ] Default policy loaded correctly
- [ ] Fast backend enforces all security constraints
- [ ] Compat backend enforces all security constraints
- [ ] Docker socket never mounted
- [ ] Host home never mounted
- [ ] Cloud metadata never accessible
- [ ] Process limits enforced
- [ ] Output limits enforced
- [ ] Timeout enforced

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
