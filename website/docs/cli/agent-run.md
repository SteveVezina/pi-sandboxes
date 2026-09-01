---
sidebar_position: 9
---

# Agent run

```bash
pi-box run <agent> [--repo <url>] [--prompt <text>] [--template <t>] [--mode <m>]
```

Creates a sandbox, starts an autonomous agent run inside it, and polls run
state until it reaches a terminal state (or you press Ctrl-C to cancel).
The host supervises — it does **not** drive the loop command by command.

| Flag | Default |
|------|---------|
| `-t, --template` | `python` |
| `-m, --mode` | `fast` |
| `-r, --repo` | — |
| `-p, --prompt` | — |

Exit status: the sandbox agent's exit code; a non-`COMPLETED` run exits
non-zero; Ctrl-C exits `130`.

The daemon emits `pi.run.started` and `pi.run.completed`
[lifecycle events](/api/events) for each run.

:::warning Work in progress
The run state model, API, and CLI are wired, but the actual in-sandbox
agent process is not launched yet — "agent entrypoint resolution" (how an
agent name maps to a runnable command) is an open spec gap. `--repo`
workspace prep is also not wired yet.
:::
