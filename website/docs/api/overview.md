---
sidebar_position: 1
---

# API overview

`pi-sandboxd` exposes a local HTTP API.

## Transport

| | |
|--|--|
| **Unix socket** (default) | `~/.pi-box/sandboxd.sock` — override with `--socket` or `$PI_SOCKET_PATH` |
| **HTTP** (dev / remote) | `pi-sandboxd --http-port <n>` → `127.0.0.1:<n>`; `--http-addr <host>` / `$PI_HTTP_ADDR` change the bind host, `$PORT` overrides the port |
| **Egress proxy** (opt-in) | `pi-sandboxd --egress-proxy-port <n>` — see [Egress & credentials](/api/credentials) |
| **Events webhook** (opt-in) | `pi-sandboxd --events-webhook <url>` — see [Lifecycle events](/api/events) |

All request and response bodies are JSON. Errors are returned as
`{"error": "<message>"}` with an appropriate status code.

## Auth

The Unix socket and a **loopback** HTTP listener (`127.0.0.1`) have no auth —
filesystem permissions / local-user trust are the boundary.

Any **non-loopback** HTTP bind (`--http-addr 0.0.0.0`, a container deploy, a
private-network address) **requires** a bearer token:

- Set `PI_DAEMON_TOKEN` in the daemon's environment (never a flag — argv leaks
  to `ps` and shell history).
- Every route except `GET /health` and CORS pre-flight then requires
  `Authorization: Bearer <token>`; a missing or wrong token returns `401` and
  the handler never runs.
- If `PI_DAEMON_TOKEN` is unset on a non-loopback bind, the daemon **refuses to
  start** (fail-closed) — a public daemon with no auth is an unauthenticated
  sandbox-exec API.

On the client, point an `http` context's `auth.token_env` at the env var
holding the same token — see [contexts](/cli/contexts). Prefer a private
network (SSH forward, Tailscale, WireGuard) over a public bind.

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
| GET · POST | `/v1/templates` · `/v1/templates/{name}` · `/v1/templates/fork` · `/v1/templates/validate` · `/v1/templates/diff` · `/v1/templates/{name}/history` · `/v1/templates/{name}/rollback` · `/v1/templates/export` · `/v1/templates/import` · `/v1/templates/{name}/promote` |
| GET | `/v1/system/status` · `/doctor` · `/runtimes` |
| GET | `/v1/support-bundle` |
| GET · POST · PUT · DELETE | `/v1/contexts` · `/v1/contexts/{name}` · `/v1/contexts/use` |
| POST | `/v1/sandboxes/{id}/agent-run` |
| GET · POST | `/v1/agent-runs` · `/v1/agent-runs/{id}` · `/v1/agent-runs/{id}/cancel` |
