---
sidebar_position: 2
---

# Sandboxes

## create

```bash
pi-box box create [template] [--name <n>] [--mode <mode>]
```

| Flag | Default | Notes |
|------|---------|-------|
| positional `template` | `base` | one of `base`, `node`, `python`, `go`, `rust`, `node-python`, `polyglot` |
| `-t, --template` | — | alternative to the positional arg |
| `-n, --name` | template name | sandbox name used by later commands |
| `-m, --mode` | `fast` | `fast`, `compat`, or `secure` |

```bash
pi-box box create node-python --name myapp --mode compat
```

## list / inspect

```bash
pi-box box list [--json]
pi-box box inspect <name> [--json]
```

## clone

```bash
pi-box box clone <name> <git-url>
```

Clones a public repository into `/workspace`.

## exec

```bash
pi-box box exec <name> [-c <cwd>] [-t <seconds>] [-j] -- <command...>
```

Everything after `--` runs inside the sandbox.

| Flag | Default |
|------|---------|
| `-c, --cwd` | `/workspace` |
| `-t, --timeout` | `120` (seconds) |
| `-j, --json` | off |

Output is capped at 8 MiB.

```bash
pi-box box exec myapp -- pnpm install
pi-box box exec myapp -c /workspace/api -- go test ./...
```

## shell

```bash
pi-box box shell <name>
```

See [Interactive shell](/getting-started/interactive-shell).

## diff / patch

```bash
pi-box box diff <name>     # unified diff of the workspace
pi-box box patch <name>    # workspace patch (read-only view)
```

These are inspection views. To deliver a patch to the host, use the
[output channel](/cli/artifacts-and-output).

## files

```bash
pi-box box files list  <name> [path]
pi-box box files read  <name> <path>
pi-box box files write <name> <path>
pi-box box files pull  <name> <src> <dest>
pi-box box files push  <name> <src> <dest>
```

File read/pull is a debugging and inspection API, not a deliverable export
path — use the [output channel](/cli/artifacts-and-output) for that.

## destroy

```bash
pi-box box destroy <name>
pi-box box destroy --all
```
