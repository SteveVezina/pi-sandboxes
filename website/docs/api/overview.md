---
sidebar_position: 1
---

# API overview

`pi-sandboxd` exposes a local HTTP API.

## Transport

| | |
|--|--|
| **Unix socket** (default) | `~/.pi-box/sandboxd.sock` — override with `--socket` |
| **HTTP** (dev / remote) | `pi-sandboxd --http-port <n>` → `127.0.0.1:<n>` |
| **Egress proxy** (opt-in) | `pi-sandboxd --egress-proxy-port <n>` — see [Egress & credentials](/api/credentials) |
| **Events webhook** (opt-in) | `pi-sandboxd --events-webhook <url>` — see [Lifecycle events](/api/events) |

All request and response bodies are JSON. Errors are returned as
`{"error": "<message>"}` with an appropriate status code.

## Auth

The local socket has no auth (filesystem permissions are the boundary). A
remote daemon reached over HTTP requires a bearer token — see
[contexts](/cli/contexts).

## Endpoint groups

- [Sandboxes](/api/sandboxes) — create / exec / clone / files / diff / patch / output / snapshot
- [System](/api/system) — health, runtimes
- [Agent runs](/api/agent-runs)
- [Templates](/api/templates)
- [Credentials](/api/credentials)
- [Lifecycle events](/api/events)

## Full route table

| Method | Path |
|--------|------|
| GET | `/health` |
| POST · GET | `/v1/credentials` |
| POST · GET | `/v1/sandboxes` |
| GET · DELETE | `/v1/sandboxes/{id}` |
| POST | `/v1/sandboxes/{id}/exec` |
| GET | `/v1/sandboxes/{id}/shell` (WebSocket) |
| POST | `/v1/sandboxes/{id}/clone` |
| GET · POST | `/v1/sandboxes/{id}/files/list` · `/files/read` · `/files/write` · `/files/pull` · `/files/push` |
| GET | `/v1/sandboxes/{id}/diff` · `/patch` |
| POST | `/v1/sandboxes/{id}/output` |
| POST | `/v1/sandboxes/{id}/snapshot` · `/snapshot/create` · `/snapshot/list` · `/snapshot/rollback` · `/snapshot/delete` · `/rollback` |
| GET | `/v1/sandboxes/{id}/logs` · `/logs/list` · `/logs/history` |
| GET · POST | `/v1/templates` · `/v1/templates/{name}` · `/v1/templates/fork` · `/v1/templates/validate` · `/v1/templates/diff` · `/v1/templates/{name}/history` · `/v1/templates/{name}/rollback` · `/v1/templates/export` · `/v1/templates/import` |
| GET | `/v1/system/status` · `/doctor` · `/runtimes` |
| GET | `/v1/support-bundle` |
| GET · POST · PUT · DELETE | `/v1/contexts` · `/v1/contexts/{name}` · `/v1/contexts/use` |
| POST | `/v1/sandboxes/{id}/agent-run` |
| GET · POST | `/v1/agent-runs` · `/v1/agent-runs/{id}` · `/v1/agent-runs/{id}/cancel` |
