---
sidebar_position: 3
---

# Interactive shell

```bash
pi-box box shell myapp
```

Opens an interactive session that reads commands from your terminal, runs
them inside the sandbox, and streams stdout/stderr back. It is backed by a
WebSocket PTY (`GET /v1/sandboxes/{id}/shell`).

```text
myapp> ls
myapp> cat package.json
myapp> exit
```

Type `exit` or `quit` (or press Ctrl-D) to close. The shell is for
diagnostics and exploration; scripted work should use `pi-box box exec`.
