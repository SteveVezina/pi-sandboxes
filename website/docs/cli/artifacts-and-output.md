---
sidebar_position: 3
---

# Artifacts & output

Build artifacts and coding-agent patches leave a sandbox through **one
channel** — `POST /v1/sandboxes/{id}/output`. `pi-box box artifacts` and
the top-level `pi-box output` are two front ends for the same channel.

Default output sources inside the sandbox:

```text
/artifacts
/workspace/dist
/workspace/build
/workspace/coverage
/workspace/test-results
/workspace/target/release
+ the workspace patch (when there are changes)
```

## list

```bash
pi-box box artifacts list <name>
```

Lists deliverable files (path, size, mtime) and patch metadata.

## pull

```bash
pi-box box artifacts pull <name> <dest-dir>
```

Copies deliverables (and `workspace.patch` when present) to a host
directory, preserving structure. Emits `pi.artifact.delivered` after the
copy succeeds.

## pack

```bash
pi-box box artifacts pack <name> [-o <file>]
```

Creates a compressed archive (`tar.gz`) of the deliverables.
`-o, --output` defaults to `/tmp/artifacts.tar.gz`.

:::note
An archive size cap is not yet enforced (needs a block-spec default). The
`pi.artifact.delivered` event reports the resulting byte count.
:::
