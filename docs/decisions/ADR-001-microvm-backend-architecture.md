# ADR-001: MicroVM Backend Architecture

## Status
Accepted

## Context

F20 requires a microVM backend with a VMM manager, tiny guest rootfs, template snapshot restore, workspace disk, output delivery, and reseed-on-restore behavior. The block spec previously allowed either Firecracker or Cloud Hypervisor without choosing the first implementation target.

## Decision

The first microVM backend targets Firecracker. Cloud Hypervisor remains a later backend behind the same runtime interface.

MicroVM mode requires Linux with `/dev/kvm` and a supported host kernel. When `/dev/kvm` or Firecracker is unavailable, runtime selection reports microVM as unavailable. It may only fall back when policy explicitly permits fallback.

The guest rootfs is read-only. Each sandbox receives a writable ext4 workspace disk. Template restore starts from a read-only template snapshot plus a fresh writable workspace disk. The reseed-on-restore hook runs after the workspace disk is attached and before guest readiness is reported.

Artifacts are exported through the guest control plane, not by directly mounting host paths inside the guest.

## Consequences

- The initial implementation can focus on one VMM and one disk format.
- Host capability detection is part of runtime selection, not ad hoc backend startup.
- MicroVM artifact and file operations depend on the guest control plane.

## References

- `SPEC.md` §14.7.4 MicroVM mode
- `docs/features/F20-microvm-backend.md`
- `docs/features/F21-microvm-guest-control-plane.md`
- PROP-003

