---
name: pi-runtime-sre
description: Infrastructure and deployment for a Pi block. Use when creating Dockerfiles, K8s manifests, security profiles (seccomp, network policies), resource limits, monitoring config, or any deployment/ops work.
---

# Pi Runtime SRE

> ⛔ **NEVER modify the block spec (`{spec_path}` per `.pi/block.yaml`) directly.**
> If you find spec gaps, write a proposal to `docs/proposals/PROP-{NNN}-{slug}.md`.
>
> 🔍 **Block config:** Per-block values (egress allowlist, lifecycle events, verify commands) live in `.pi/block.yaml`.

## Overview

Build and maintain the infrastructure that runs this block securely in production. Blocks that give code, agents, or untrusted input direct access to filesystem / shell / network are security-sensitive — the SRE layer is what prevents containers from becoming a liability.

## When to Use

- Creating or modifying Dockerfiles (service image + any sandbox image)
- Writing Kubernetes manifests (Deployment, Service, NetworkPolicy, RBAC)
- Defining seccomp / AppArmor profiles
- Configuring resource limits (cgroups)
- Setting up monitoring, alerting, or observability
- Designing container lifecycle (create / destroy / pool / cleanup)
- Health check implementation
- Capacity planning and scaling strategy
- Incident response runbooks

**When NOT to use:** Application logic (use `/skill:pi-execute-plan` Step 3 inline implementation), architecture decisions (surface as feature spec gap or PROP).

## Process

### Step 0: Verify Container Commands

Before deploying ANY container image, **always verify the actual commands/binaries** it provides:

```bash
# 1. Check what binaries exist in the image
docker run --rm <image> ls /usr/bin/
docker run --rm <image> ls /usr/local/bin/

# 2. Verify the specific command you plan to use
docker run --rm <image> <command> --help
docker run --rm <image> <command> --version

# 3. Check the image's entrypoint (may override your command)
docker inspect <image> --format '{{.Config.Entrypoint}}'
docker inspect <image> --format '{{.Config.Cmd}}'

# 4. Test the exact command you'll deploy in a dry-run
docker run --rm <image> <command> <args> 2>&1 | head -20
```

**Common pitfalls:**
- `git daemon` ❌ → `git-daemon` ✅ (standalone binary, not a git subcommand)
- `python -m http.server` vs `python3 -m http.server` (Python 2 vs 3)
- Entrypoint overrides your command — always check `docker inspect`
- Some images use `exec` wrappers that change available binaries
- Alpine images may strip man pages/help text — test, don't guess

**Rule: If you're not 100% sure the command exists, test it. Never assume.**

### Step 1: Load Context

```
1. docs/design-principles.md              → platform invariants (DP-NNN)
2. {spec_path}                      → block spec, especially § Security Model and § Deployment Topology
3. .pi/block.yaml                         → per-block egress allowlist, verify commands
4. docs/contracts/*.md                    → upstream interfaces this block must reach
5. docs/decisions/ADR-*.md                → architecture decisions in force
6. deploy/ existing files                 → current state
```

**Topology parity check (mandatory before any manifest change):**

For every manifest you write or modify, answer:
1. Does this introduce a layer that exists in production but not in local dev (or vice versa)? If yes → **stop**. Add the missing layer first, or write a PROP documenting the deviation and get explicit approval.
2. Does any service URL, DNS name, or address assume a topology that contradicts an accepted design principle (e.g. ECS vs K8s split, single-cluster vs split-environment)? If yes → **fix before proceeding**.
3. Verify per-environment exposure matches the block spec's § Deployment Topology section.

### Step 2: Security Profile Design

Most Pi blocks layer their security model like this (adapt per block — not every block needs every layer):

```
┌─────────────────────────────────────────────────────┐
│ Layer 1: Filesystem Isolation                        │
│   Only declared mount paths are read-write           │
│   /tmp is process-local                              │
│   No access to host filesystem or other sessions     │
├─────────────────────────────────────────────────────┤
│ Layer 2: Network Isolation                           │
│   Default: deny all egress                           │
│   Allowlist: see `.pi/block.yaml` § security.egress_allowlist │
│   Implementation: NetworkPolicy + network namespace  │
├─────────────────────────────────────────────────────┤
│ Layer 3: Syscall Restriction (seccomp)              │
│   Whitelist approach: only safe syscalls allowed     │
│   Block: mount, ptrace, reboot, kexec, bpf, etc.    │
├─────────────────────────────────────────────────────┤
│ Layer 4: Resource Limits (cgroups v2)               │
│   CPU / memory / disk I/O / PIDs / time              │
│   Hard kill on memory or duration breach             │
├─────────────────────────────────────────────────────┤
│ Layer 5: User Namespace                             │
│   Run as unprivileged user (uid 1000)               │
│   No capabilities, no setuid binaries in image       │
└─────────────────────────────────────────────────────┘
```

### Step 3: Deliverables

Typical artifacts for a block (add or omit per block needs):

| File | Purpose |
|------|---------|
| `deploy/Dockerfile.{service}` | Production image for the block's service |
| `deploy/Dockerfile.{worker}` | Image for any worker / sandbox the block manages |
| `deploy/docker-compose.yml` | Local dev with mock services |
| `deploy/k8s/deployment.yaml` | Service deployment |
| `deploy/k8s/service.yaml` | ClusterIP service |
| `deploy/k8s/network-policy.yaml` | Egress control (matches `.pi/block.yaml` allowlist) |
| `deploy/k8s/rbac.yaml` | ServiceAccount + minimal permissions |
| `deploy/k8s/configmap.yaml` | Runtime configuration |
| `deploy/security/seccomp-profile.json` | Syscall whitelist |
| `deploy/security/apparmor-profile` | AppArmor policy (optional) |

### Step 4: Testing Security

Every security control must have a verification test. Examples:

```bash
# Filesystem escape
kubectl exec {pod} -- cat /etc/shadow             # → DENIED
kubectl exec {pod} -- ls /host                    # → NOT FOUND

# Network escape (every destination NOT in egress allowlist must fail)
kubectl exec {pod} -- curl https://example.com    # → TIMEOUT/DENIED
kubectl exec {pod} -- ping 8.8.8.8                # → DENIED

# Privilege escalation
kubectl exec {pod} -- sudo su                     # → DENIED
kubectl exec {pod} -- mount /dev/sda1 /mnt        # → DENIED (seccomp)

# Resource limits
kubectl exec {pod} -- stress --cpu 8              # → THROTTLED
kubectl exec {pod} -- dd if=/dev/zero of=big      # → DISK LIMIT HIT
```

### Step 5: Monitoring

Per-block metrics. Use `{block.slug}` as the metric prefix (from `.pi/block.yaml`):

| Metric | Type | Alert threshold |
|--------|------|-----------------|
| `{slug}_active_count` | Gauge | > capacity limit |
| `{slug}_request_duration_seconds` | Histogram | p99 > SLO |
| `{slug}_security_violations_total` | Counter | Any > 0 |
| `{slug}_container_memory_usage_bytes` | Gauge | > 90% of limit |
| `{slug}_container_cpu_usage_seconds` | Counter | sustained > limit |

For each lifecycle event in `.pi/block.yaml` § `lifecycle_events`, ensure a corresponding metric or trace exists.

## Documentation Responsibilities

When your infra or deploy work changes operations, update docs in the same commit.

**Runbooks** (`docs/runbooks/`) — create or update when:
- A new operational procedure is needed (incident response, scaling, recovery)
- An existing procedure changes due to infra changes

Runbook template:
```markdown
# Runbook: {Title}

## Symptoms
What does the operator see that triggers this runbook?

## Diagnosis
Step-by-step commands to identify root cause.

## Resolution
Step-by-step fix for each possible cause.

## Prevention
What monitoring/alerting should catch this earlier?
```

**AGENTS.md** — update when:
- A new lifecycle event is added or mapping changes
- Security constraints change (seccomp rules, network policies, cgroup limits)
- Quality gates are modified

Rule: AGENTS.md is the block's contract with future agents. Changes must be minimal and precise. Never add prose that belongs in a runbook or ADR.
