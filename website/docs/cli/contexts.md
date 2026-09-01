---
sidebar_position: 8
---

# Contexts

A context tells `pi-box` which daemon to talk to — the local one, or a
remote daemon over SSH or HTTP.

```bash
pi-box context create <name> <target> [--transport <t>] [--auth <a>] [--token-env <VAR>]
pi-box context use <name>
pi-box context list [--json]
pi-box context inspect <name> [--json]
pi-box context delete <name>
```

| Field | Values |
|-------|--------|
| `target` | `ssh://gpu-box.local`, `https://daemon.host:7777`, or a local socket path |
| `--transport` | `unix`, `http`, `ssh` (auto-detected from the target if omitted) |
| `--auth` | `none`, `bearer-token`, `ssh-agent` (auto-detected from transport) |
| `--token-env` | env var holding the bearer token (required for `http`) |

```bash
pi-box context create workstation ssh://gpu-box.local
pi-box context use workstation
pi-box box create node-python --name build   # runs on gpu-box.local
```

Any command accepts `--context <name>` to override the active context for
that one invocation. A local context is always the default.
