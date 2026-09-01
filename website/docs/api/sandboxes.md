---
sidebar_position: 2
---

# Sandbox endpoints

## Create

`POST /v1/sandboxes`

```json
{
  "name": "myapp",
  "template": "node-python",
  "mode": "fast",
  "ttlSeconds": 7200,
  "workspace": { "mode": "copy", "source": "", "maxSize": "5Gi" },
  "network": { "mode": "restricted", "allow": ["internal.corp"] }
}
```

`name` is required. `template` defaults to `base`, `mode` defaults to
`auto` (resolved by the [selection engine](/runtimes/selection-and-fallback)).
`network.mode` is one of `none` / `restricted` / `open` and is fixed for
the sandbox's lifetime; `restricted` is the default.

Response: `201` `{ "id": "<sandbox-id>" }`. Emits `pi.sandbox.created`.

## List / inspect / destroy

```
GET    /v1/sandboxes
GET    /v1/sandboxes/{id}
DELETE /v1/sandboxes/{id}
```

`DELETE` tears down the runtime object and removes daemon-managed volumes,
then emits `pi.sandbox.destroyed`.

## Exec

`POST /v1/sandboxes/{id}/exec`

```json
{ "command": "pnpm install", "cwd": "/workspace", "timeoutMs": 120000, "maxOutputBytes": 8388608, "network": "" }
```

Add `?stream=true` (or `Accept: application/x-ndjson`) for a streamed
NDJSON response. Non-streaming response:

```json
{ "exitCode": 0, "durationMs": 4210, "stdout": "...", "stderr": "", "truncated": false, "timedOut": false }
```

## Clone

`POST /v1/sandboxes/{id}/clone` — `{ "url": "https://github.com/org/repo.git" }`

Runs `git clone` inside the sandbox into `/workspace`.

## Files

| Endpoint | Body |
|----------|------|
| `GET /v1/sandboxes/{id}/files/list?path=<p>` | — |
| `GET /v1/sandboxes/{id}/files/read` | `{ "path": "/workspace/x" }` |
| `POST /v1/sandboxes/{id}/files/write` | `{ "path": "/workspace/x", "content": "..." }` (≤ 1 MiB inline) |
| `POST /v1/sandboxes/{id}/files/pull` | `{ "src": "/workspace/x", "dest": "./x" }` |
| `POST /v1/sandboxes/{id}/files/push` | `{ "src": "./x", "dest": "/workspace/x" }` |

File read/pull is for inspection, not deliverable export.

## Diff / patch

`GET /v1/sandboxes/{id}/diff` and `GET /v1/sandboxes/{id}/patch` return the
workspace diff / patch as text. Read-only views.

## Output

`POST /v1/sandboxes/{id}/output`

```json
{ "action": "list" }
{ "action": "pull", "dest": "./out" }
{ "action": "pack", "output": "./out.tar.gz" }
```

The only deliverable export path. `pull` / `pack` emit
`pi.artifact.delivered` after success.

## Snapshots

```
POST /v1/sandboxes/{id}/snapshot/create    { "name": "before-refactor" }
GET  /v1/sandboxes/{id}/snapshot/list
POST /v1/sandboxes/{id}/snapshot/rollback  { "name": "before-refactor" }
POST /v1/sandboxes/{id}/snapshot/delete    { "name": "before-refactor" }
```

## Logs

```
GET /v1/sandboxes/{id}/logs                      full entries
GET /v1/sandboxes/{id}/logs/history              summary
GET /v1/sandboxes/{id}/logs?action=egress        egress-proxy denials
```
