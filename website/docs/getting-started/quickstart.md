---
sidebar_position: 2
---

# Quickstart

First sandbox, first agent command, in about five minutes. Assumes
[installation](/getting-started/installation) is done.

## 1. Start the daemon

```bash
pi-sandboxd --socket ~/.pi-box/sandboxd.sock &
pi-box system status
```

The daemon listens on `~/.pi-box/sandboxd.sock`. Pass `--http-port <n>` to
also expose `127.0.0.1:<n>` for local development, and
`--egress-proxy-port <n>` to enable the egress proxy (see
[Egress & credentials](/api/credentials)).

## 2. Check the host

```bash
pi-box system doctor
```

Reports available runtime backends and the mode `auto` would resolve to.

## 3. Create a sandbox

```bash
pi-box box create node-python --name myapp
```

The positional argument is the **template** (`base`, `node`, `python`,
`go`, `rust`, `node-python`, `polyglot`). `--name` defaults to the template
name; `--mode` defaults to `fast`.

```bash
pi-box box create node-python --name myapp --mode compat
```

## 4. Clone a repository

```bash
pi-box box clone myapp https://github.com/nodejs/node.git
```

## 5. Run commands

```bash
pi-box box exec myapp -- echo "hello from the sandbox"
pi-box box exec myapp -- pnpm install
```

Everything after `--` runs inside the sandbox. Add `--json` for
machine-readable output.

## 6. Inspect the work

```bash
pi-box box diff myapp        # workspace diff
pi-box box patch myapp       # workspace patch (read-only view)
pi-box box logs myapp        # command output
pi-box box history myapp     # command history summary
```

## 7. Export deliverables

Artifacts and patches leave the sandbox through the single output channel:

```bash
pi-box box artifacts list myapp
pi-box box artifacts pull myapp ./out
pi-box box artifacts pack myapp --output ./myapp.tar.gz
```

## 8. Destroy

```bash
pi-box box destroy myapp
pi-box box destroy --all
```

## Next

- [CLI reference](/cli/overview)
- [Interactive shell](/getting-started/interactive-shell)
- [SDKs](/getting-started/sdk)
