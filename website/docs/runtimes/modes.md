---
sidebar_position: 1
---

# Runtime modes

The daemon offers **selectable isolation**. Pick a mode per sandbox
(`--mode` / `mode` in the create request); `auto` lets the daemon choose.

| Mode | Boundary | Package | Host requirement |
|------|----------|---------|------------------|
| `fast` | Linux namespaces + cgroups v2 + seccomp-bpf + Landlock | `pkg/runtime/fast` | Linux kernel; no extra packages |
| `compat` | OCI container (Docker / Podman), hardened | `pkg/runtime/compat` + `pkg/runtime/oci` | a container engine |
| `secure` | gVisor (`runsc`) — user-space kernel, full syscall filtering | `pkg/runtime/gvisor` | Linux + `runsc` |
| `microvm` | Hardware-virtualized microVM | `pkg/runtime/microvm` | Linux + KVM |

## Hardened defaults (every mode)

- No host home mount, no Docker socket, no cloud-metadata access.
- Unprivileged user, capabilities dropped, `no-new-privileges`.
- Read-only root where possible; `/workspace`, `/artifacts`, caches are
  daemon-managed volumes, not host bind mounts.
- Process limit 256, output cap 8 MiB, exec timeout 120 s.
- `restricted` network by default.

## When to use which

| Situation | Mode |
|-----------|------|
| Trusted repo, Linux host, lowest overhead | `fast` |
| Trusted repo, any host with Docker/Podman | `compat` |
| Unknown or untrusted repository | `secure` |
| Strongest boundary / hostile code | `microvm` |

## Platform support

| Platform | fast | compat | secure | microvm |
|----------|------|--------|--------|---------|
| Linux (native) | ✅ | ✅ | ✅ (with `runsc`) | ✅ (with KVM) |
| macOS | — | ✅ (Docker Desktop) | — | — |
| Windows | — | ✅ (WSL2) | — | — |

`pi-box system doctor` reports which modes are usable on the current host.
