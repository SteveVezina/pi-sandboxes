---
sidebar_position: 2
---

# Security model

## Default deny

Never mounted or reachable by default, in any runtime mode:

| Resource | Default |
|----------|---------|
| Host home directory | not mounted |
| Docker / Podman socket | not mounted |
| Cloud metadata (`169.254.169.254`), host gateway, host localhost | blocked at the network layer |
| SSH private keys | not mounted |
| Kubernetes config, cloud-provider config dirs | not mounted |
| Git credentials | brokered through the egress proxy, never placed in env / argv / files |

`/workspace`, `/artifacts`, and the caches are **daemon-managed volumes**,
not writable host bind mounts. The create request exposes no field for an
arbitrary host mount.

## Process limits

| Limit | Default | Configurable |
|-------|---------|--------------|
| Max processes | 256 | yes |
| Exec timeout | 120 s | yes (`timeoutMs`) |
| Max output | 8 MiB | yes (`maxOutputBytes`) |
| CPU / memory | backend default (2 CPU / 2 GiB) | yes |

## Runtime hardening

- Unprivileged user, all capabilities dropped, `no-new-privileges`.
- Read-only root filesystem where the backend supports it.
- `fast`: user/mount/PID/network namespaces, cgroups v2, a
  project-versioned seccomp-bpf profile, Landlock (mount-namespace-only
  fallback when unavailable).
- `compat` / `secure`: no privileged mode, no host network, caps dropped,
  the same seccomp profile passed explicitly to the OCI engine, no
  container auto-remove (the daemon reconciles and GCs orphans on start).

## Network model

| Mode | Behavior |
|------|----------|
| `none` | no outbound route |
| `restricted` (default) | domain allowlist through the daemon egress proxy |
| `open` | full outbound (opt-in) — the default-deny set still applies |

Network mode is fixed at sandbox create time. See
[Egress & credentials](/api/credentials) for the proxy and credential
injection.

## Secrets

No plaintext secrets are stored on disk under `~/.pi-box`. Credentials are
registered as injection *rules*; the resolved value lives only in the
daemon's memory and is injected into approved outbound requests by the
egress proxy.
