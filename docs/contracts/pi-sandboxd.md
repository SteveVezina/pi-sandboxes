# pi-sandboxd Contract Digest

> Source: `SPEC.md` §9 Interface Contract
> Scope: local daemon HTTP API, GUI-readable system/context endpoints, and remote-client auth constraints.

## Daemon API

| Method | Path | Request | Response | Features |
|--------|------|---------|----------|----------|
| `POST` | `/v1/sandboxes` | `CreateSandboxRequest` | `{ "id": string }` | F1, F2, F8, F24, F25, F26 |
| `GET` | `/v1/sandboxes` | — | `Sandbox[]` | F1, F2, F8, F24, F26 |
| `GET` | `/v1/sandboxes/{id}` | — | `Sandbox` | F1, F2, F8, F26 |
| `DELETE` | `/v1/sandboxes/{id}` | — | `204` or action payload | F1, F2, F8, F26 |
| `POST` | `/v1/sandboxes/{id}/clone` | `{ "url": string }` | action payload | F6, F26 |
| `POST` | `/v1/sandboxes/{id}/exec` | `ExecRequest` | `ExecResult` or NDJSON stream | F7, F10, F26 |
| `POST` | `/v1/sandboxes/{id}/files/write` | `{ "path": string, "content": string }` | action payload | F6 |
| `GET` | `/v1/sandboxes/{id}/files/read` | query `path` | file payload | F6 |
| `GET` | `/v1/sandboxes/{id}/diff` | — | diff payload | F6, F26 |
| `GET` | `/v1/sandboxes/{id}/patch` | — | patch payload | F6, F26 |
| `GET` | `/v1/sandboxes/{id}/artifacts/list` | — | `{ "files": string[] }` | F9, F26 |
| `POST` | `/v1/sandboxes/{id}/artifacts/pull` | `{ "destination": string }` | action payload | F9, F26 |
| `POST` | `/v1/sandboxes/{id}/artifacts/pack` | `{ "output": string }` | action payload | F9, F26 |
| `POST` | `/v1/sandboxes/{id}/snapshot/create` | `{ "name": string }` | action payload | F13, F26 |
| `GET` | `/v1/sandboxes/{id}/snapshot/list` | — | snapshot list | F13, F26 |
| `POST` | `/v1/sandboxes/{id}/snapshot/rollback` | `{ "name": string }` | action payload | F13, F26 |
| `POST` | `/v1/sandboxes/{id}/snapshot/delete` | `{ "name": string }` | action payload | F13 |
| `GET` | `/v1/sandboxes/{id}/logs/list` | — | command logs | F10, F26 |
| `GET` | `/v1/sandboxes/{id}/logs/history` | — | command history | F10, F26 |

## Request/Response Shapes

### CreateSandboxRequest

```json
{
  "name": "demo",
  "template": "node-python",
  "mode": "fast",
  "ttlSeconds": 7200,
  "workspace": {
    "mode": "copy",
    "source": "/path/to/project",
    "maxSize": "5Gi"
  }
}
```

Rules:
- `name` is required.
- Missing `template` defaults to `base`.
- Missing `mode` defaults to `fast`.
- Missing `workspace.mode` defaults to `copy`.
- `workspace.mode=bind` requires an explicit `workspace.source`.
- Unsafe bind sources are rejected: home directory itself, SSH keys, Kubernetes config, cloud credential directories, and Docker sockets.

### Sandbox

```json
{
  "id": "sandbox-id",
  "name": "demo",
  "template": "node-python",
  "mode": "fast",
  "state": "warm",
  "workspace": "/path/to/project",
  "workspace_mode": "copy",
  "created_at": "RFC3339 timestamp",
  "updated_at": "RFC3339 timestamp",
  "ttl_seconds": 7200,
  "last_used": "RFC3339 timestamp"
}
```

### ExecRequest

```json
{
  "command": "pnpm test",
  "cwd": "/workspace",
  "timeoutMs": 120000,
  "maxOutputBytes": 8388608
}
```

`POST /v1/sandboxes/{id}/exec?stream=true` returns newline-delimited JSON events with `type` values `stdout`, `stderr`, and `done`.

### ExecResult

```json
{
  "exitCode": 0,
  "stdout": "ok\n",
  "stderr": "",
  "durationMs": 12,
  "truncated": false,
  "timedOut": false
}
```

## GUI/System Endpoints

| Method | Path | Response | Features |
|--------|------|----------|----------|
| `GET` | `/health` | `{ "status": "ok" }` | F2, F24 |
| `GET` | `/v1/system/status` | daemon state, `pi_home`, config path | F16, F27 |
| `GET` | `/v1/system/doctor` | doctor-equivalent diagnostics | F16, F27 |
| `GET` | `/v1/system/runtimes` | runtime/backend availability | F19, F27 |
| `GET` | `/v1/support-bundle` | redacted diagnostics bundle | F27 |
| `GET` | `/v1/contexts` | active context and configured contexts | F22, F24, F27 |
| `POST` | `/v1/contexts/use` | `{ "active": string }` | F22, F24, F27 |

## Remote Transport/Auth Constraints

- Remote HTTP contexts require bearer-token auth.
- Remote auth failures must never fall back to unauthenticated access.
- Context files may store `auth.type` and `auth.token_env`, but never the raw token value.
- SDKs and GUI clients must not persist bearer tokens into sandbox workspaces.

## Browser Client Constraints

- GUI requests from localhost development origins must pass CORS preflight.
- Allowed CORS headers include `Content-Type`, `Accept`, and `Authorization`.
- The GUI remains a daemon client; it must not implement a separate sandbox lifecycle.
