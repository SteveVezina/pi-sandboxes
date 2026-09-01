---
slug: /
sidebar_position: 1
title: Introduction
---

# PI Agent Sandbox Runtime

A local-first, open-source sandbox runtime for AI coding agents. It gives
agents tiny, fast, isolated developer workspaces where they can clone
repositories, read and write files, run Node.js / Python / Go / Rust
commands, build and run applications, collect logs, export artifacts, and
produce patches.

> **Positioning:** local-first sandboxes for coding agents. Fast warm
> exec, tiny footprint, selectable isolation.

## Two binaries

| Binary | Role |
|--------|------|
| `pi-sandboxd` | The daemon. Owns sandbox lifecycle, runtime backends, and all state under `~/.pi-box`. Listens on a Unix socket. |
| `pi-box` | The CLI. Talks to the daemon over the socket (or a remote context). |

There is also a TypeScript SDK and a Python SDK, and a cross-platform GUI
workbench — all daemon clients.

## Runtime modes

The daemon offers **selectable isolation**. Pick per sandbox:

| Mode | Isolation | Use it for |
|------|-----------|-----------|
| `fast` | Linux namespaces + cgroups + seccomp + Landlock | Trusted repos, lowest overhead (Linux only) |
| `compat` | OCI container (Docker / Podman), hardened | Portable; trusted repos on any host with a container engine |
| `secure` | gVisor / `runsc` | Unknown or untrusted repositories |
| `microvm` | Hardware-virtualized microVM | Strongest boundary |

`auto` resolution picks a mode from the workload's trust level and the
host's capabilities, and never silently downgrades below what you asked
for. See [Runtime modes](/runtimes/modes).

## Security defaults

Every sandbox, every mode: no host home mount, no Docker socket, no cloud
metadata access, process/output/timeout limits, and `restricted` network
by default. See [Security model](/architecture/security-model).

## Next

- [Installation](/getting-started/installation)
- [Quickstart](/getting-started/quickstart) — first sandbox in ~5 minutes
- [CLI reference](/cli/overview)
- [Daemon API reference](/api/overview)
