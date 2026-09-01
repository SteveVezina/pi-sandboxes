---
sidebar_position: 2
---

# Compatibility

## Backend requirements

| Backend | Requires |
|---------|----------|
| `fast` | Linux kernel with user namespaces, cgroups v2, seccomp; Landlock optional (falls back to mount-namespace only) |
| `compat` | Docker or Podman (containerd support planned) |
| `secure` | Linux + `runsc` on `PATH` |
| `microvm` | Linux + `/dev/kvm`; a microVM manager (`pi-vmm-manager`) |

## Container engine notes (compat / secure)

- Podman runs rootless by default — the daemon keeps the invoking user's
  identity so bind-free volumes stay writable.
- Docker Desktop on macOS/Windows does not support extra mount options;
  the OCI engine adjusts automatically.
- The seccomp profile is written under `~/.pi-box` and passed to the
  engine; on non-Linux hosts Docker sends the profile content client-side.

## Known gaps

| Area | Status |
|------|--------|
| `secure` (gVisor) driver | Present but not currently building against the shared OCI engine on Linux — being reworked. |
| `microvm` L3 network isolation, `compat` egress firewall | Pending — needs a Linux + Docker host. |
| Template `fork` / `snapshot` / `import` / `export` | Specified (F28), not implemented. |
| In-sandbox agent process launch | Run state + events wired; process launch pending an "agent entrypoint" spec. |
