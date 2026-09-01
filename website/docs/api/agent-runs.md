---
sidebar_position: 4
---

# Agent run endpoints

## Start

`POST /v1/sandboxes/{id}/agent-run`

```json
{ "agent": "my-agent", "repo_url": "https://github.com/org/repo.git", "prompt": "..." }
```

`agent` is required. The sandbox must be `WARM`. Returns `201` with
`{ run_id, sandbox_id, agent, state }` and emits `pi.run.started`.
`409` if the sandbox isn't runnable or already has an active run.

## Inspect

`GET /v1/agent-runs/{id}` → the full run record:

```json
{
  "run_id": "...",
  "sandbox_id": "...",
  "agent_name": "my-agent",
  "state": "RUNNING",
  "exit_code": 0,
  "started_at": "2026-08-31T12:00:00Z",
  "completed_at": "0001-01-01T00:00:00Z",
  "error": ""
}
```

States: `PENDING` → `STARTING` → `RUNNING` → `COMPLETED` / `FAILED` /
`CANCELLED`. Terminal states are final.

## Cancel

`POST /v1/agent-runs/{id}/cancel` — moves a non-terminal run to
`CANCELLED`. `409` if already terminal. Emits `pi.run.completed`
(`data.status = "CANCELLED"`).

## List

`GET /v1/agent-runs` — all runs.

:::warning
`pi.run.completed` fires exactly once per run. The in-sandbox agent
process launch is not implemented yet — see the
[CLI page](/cli/agent-run).
:::
