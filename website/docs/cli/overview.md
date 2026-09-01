---
sidebar_position: 1
---

# CLI overview

`pi-box` is the command-line client for `pi-sandboxd`. It talks to the
daemon over `~/.pi-box/sandboxd.sock`, or to a remote daemon selected by
the active [context](/cli/contexts).

## Command tree

```text
pi-box
  box        sandbox lifecycle
    create | list | inspect | destroy | clone | exec | shell
    diff | patch
    artifacts   list | pull | pack
    snapshot    create | list | rollback | delete
    files       list | read | write | pull | push
    logs | history | egress
  run        start an autonomous agent loop in a sandbox
  output     list | pull | pack   (same channel as `box artifacts`)
  template   list | inspect | fork | validate | history | diff | rollback | build | update | prune
  context    create | use | list | delete
  system     status | doctor | prune | disk-usage
  bench      run
```

## Global flags

| Flag | Meaning |
|------|---------|
| `--context <name>` | Use a specific daemon context for this command (overrides the active one). |
| `--json` | Machine-readable output. Available on `box` subcommands. |

## Pages

- [Sandboxes](/cli/sandboxes) — create, exec, clone, files, diff/patch
- [Artifacts & output](/cli/artifacts-and-output)
- [Snapshots](/cli/snapshots)
- [Logs & history](/cli/logs-and-history)
- [System](/cli/system)
- [Templates](/cli/templates)
- [Contexts](/cli/contexts)
- [Agent run](/cli/agent-run)
