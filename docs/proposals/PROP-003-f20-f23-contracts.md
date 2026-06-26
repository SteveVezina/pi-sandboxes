# PROP-003: Specify F20-F23 MicroVM and Remote Contracts

## Status
✅ Applied to block spec (2026-06-26)

## Block Spec Reference
`SPEC.md` §6 Features F20-F23, §7 AC-23 through AC-26, §15 MicroVM mode, §31 Milestones 5-6, §34 Configuration file

## Problem

F20-F23 are now extracted into the structured feature table, but their feature specs still contain blocking implementation gaps:

- F20 does not decide Firecracker vs Cloud Hypervisor or host requirements.
- F20 does not define the workspace disk/snapshot format.
- F21 does not define the guest control protocol over vsock.
- F22 does not define the context configuration schema.
- F23 does not define the remote daemon authentication and transport contract.

These are public or cross-component contracts. Coding them first would violate the spec-first rule because the implementation would silently invent API shapes, protocol messages, config fields, and security behavior.

## Proposed Amendment

Add the following contract details to `SPEC.md`.

### MicroVM backend contract

For the first microVM implementation:

- Firecracker is the primary M5 backend.
- Cloud Hypervisor remains a later compatible backend behind the same runtime interface.
- MicroVM mode requires Linux with `/dev/kvm` and a supported host kernel.
- If `/dev/kvm` or Firecracker is unavailable, runtime selection reports microVM as unavailable; it does not silently fall back unless policy permits fallback.
- The guest rootfs is read-only.
- Each sandbox receives a writable ext4 workspace disk.
- Template restore starts from a read-only template snapshot plus a fresh writable workspace disk.
- The reseed-on-restore hook runs after the workspace disk is attached and before the guest reports ready.
- Artifact export copies data through the guest control plane, not by directly mounting host paths inside the guest.

### MicroVM guest control protocol

The host and guest communicate over virtio-vsock using newline-delimited JSON control frames.

Each frame has:

```json
{
  "type": "request|response|event|stream",
  "id": "request-id",
  "session_id": "sandbox-id",
  "method": "hello|ready|exec|file.read|file.write|artifact.list|artifact.pull|shutdown",
  "payload": {},
  "error": {
    "code": "string",
    "message": "string"
  }
}
```

Exec streaming uses `stream` frames with:

```json
{
  "type": "stream",
  "id": "request-id",
  "session_id": "sandbox-id",
  "payload": {
    "stream": "stdout|stderr",
    "data": "base64-bytes"
  }
}
```

The final exec response includes:

```json
{
  "exit_code": 0,
  "duration_ms": 0,
  "timed_out": false,
  "truncated": false
}
```

Guest readiness is explicit: `pi-init` starts `pi-agentd`, `pi-agentd` sends a `ready` event, and only then may the host mark the sandbox warm.

### Remote context configuration

Context state is stored in `~/.pi/contexts.yaml`.

```yaml
active: local
contexts:
  local:
    target: unix://~/.pi/sandboxd.sock
    transport: unix
    auth:
      type: none
  workstation:
    target: ssh://gpu-box.local
    transport: ssh
    auth:
      type: ssh-agent
```

Required context fields:

- `target`: daemon endpoint URI
- `transport`: `unix`, `http`, or `ssh`
- `auth.type`: `none`, `bearer-token`, or `ssh-agent`

The active context may be overridden per command with `--context <name>`.

### Remote daemon transport and auth

Remote daemon access keeps the existing daemon HTTP API unchanged.

Supported transports:

- `unix`: local Unix socket
- `http`: direct HTTP endpoint, intended for private networks such as Tailscale or WireGuard
- `ssh`: SSH-forwarded daemon access

Authentication rules:

- `unix` contexts may use `auth.type: none`.
- `http` contexts require bearer-token auth.
- `ssh` contexts use SSH agent authentication for the transport.
- Bearer tokens are stored outside sandbox workspaces and are never injected into sandbox environment variables.
- Remote auth failures return actionable errors and do not fall back to unauthenticated access.

## Acceptance Criteria Additions

Amend the existing AC-23 through AC-26 sections with the contract details above.

### AC-23 additions
- [ ] MicroVM backend reports unavailable when `/dev/kvm` or Firecracker is unavailable
- [ ] Guest rootfs is read-only
- [ ] Workspace disk is writable ext4
- [ ] Artifact export uses the guest control plane

### AC-24 additions
- [ ] Host and guest exchange newline-delimited JSON frames over virtio-vsock
- [ ] Exec stdout/stderr stream frames carry base64 payloads
- [ ] Final exec response includes exit code, duration, timeout, and truncation metadata
- [ ] Host marks sandbox warm only after the guest sends `ready`

### AC-25 additions
- [ ] Contexts persist in `~/.pi/contexts.yaml`
- [ ] Context schema supports `target`, `transport`, and `auth.type`
- [ ] `--context <name>` overrides the active context

### AC-26 additions
- [ ] `http` remote contexts require bearer-token auth
- [ ] `ssh` remote contexts use SSH agent transport authentication
- [ ] Remote auth failures never fall back to unauthenticated access

## Cascade Required on Acceptance

When this PROP is accepted:

1. Update `SPEC.md` with the proposed contract details.
2. Update F20-F23 feature specs to remove the blocking block-spec gaps.
3. Add or update ADR references for:
   - MicroVM backend architecture
   - Guest vsock protocol
   - Remote context/auth model
4. Split L-sized tasks in F20, F21, and F23 into M-sized implementation tasks.
5. Update `docs/features/INDEX.md` and `docs/plan.md`.
6. Mark this proposal applied in `docs/proposals/INDEX.md`.

## Implementation Blocked?

Resolved. F20-F23 implementation may proceed against the accepted contracts and ADRs.

## Cascade completed

Completed on 2026-06-26:

- Updated `SPEC.md` with the F20-F23 microVM, guest protocol, context, and remote auth contracts.
- Added ADR-001 (MicroVM Backend Architecture).
- Added ADR-002 (MicroVM Guest Control Protocol).
- Added ADR-003 (Remote Context and Auth Model).
- Updated F20-F23 feature specs with new acceptance criteria and ADR references.
- Split L-sized F20, F21, and F23 tasks into M-sized implementation tasks.
- Updated `docs/features/INDEX.md` and `docs/plan.md` to unblock F20-F23.
