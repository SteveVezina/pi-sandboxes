---
sidebar_position: 7
---

# Template endpoints

The local template library (F28). Templates are daemon-owned assets under
`~/.pi-box/templates`; built-ins are installed on first use.

## List

`GET /v1/templates`

```json
{
  "count": 7,
  "templates": [
    { "name": "node", "version": "", "summary": "", "source": "builtin", "tags": null }
  ]
}
```

## Inspect

`GET /v1/templates/{name}`

```json
{
  "template": { "name": "node", "base": "debian-slim", "...": "..." },
  "image": "docker.io/library/node:22-bookworm",
  "contentDigest": "sha256:…",
  "problems": []
}
```

`contentDigest` is a stable sha256 over the definition, excluding lineage
and timestamps. `problems` is the validation result.

## Fork

`POST /v1/templates/fork` — `{ "source": "node", "name": "my-node" }`

Creates an editable local template derived from `source`. Records
`source.forkedFrom` and a `lineage` entry (generation, parent digest).
`201` on success, `409` if the target name exists or the source is
missing.

## Validate

`POST /v1/templates/validate`

```json
{ "name": "my-node" }
{ "template": { "name": "x", "base": "debian-slim", "network": "restricted" } }
```

Provide either an installed template `name` or an inline `template`.
Response: `{ "valid": bool, "problems": [string] }`.

## History

`GET /v1/templates/{name}/history` → `{ name, count, revisions: [{ n, time, digest }] }`,
newest first. Every write to a local template records a revision.

## Diff

`POST /v1/templates/diff` — `{ "left": "node", "right": "my-node@2" }`

Each ref is a template name or `name@N`. Response:
`{ left, right, diff: "<text>" }`.

## Rollback

`POST /v1/templates/{name}/rollback` — `{ "revision": 1 }`

Restores the template to revision `N`. The rollback is itself recorded as
a new revision and bumps `lineage.generation`. `409` if the revision or
template is missing.

## Export

`POST /v1/templates/export` — `{ "name": "my-node" }`

Returns the **OCI image layout bundle tar** (`application/octet-stream`,
`Content-Disposition: attachment`). See
[the CLI page](/cli/templates#export--import) for the bundle format
(ADR-008).

## Import

`POST /v1/templates/import` — raw bundle tar as the request body,
optional `?name=<n>` to rename. Body cap: 32 MiB.

Verifies blob digests, re-runs `Validate`, installs with
`source.type: imported`. `201` on success; `409` on a name collision.

## Promote

`POST /v1/templates/{name}/promote` — `{ "default": true }`

Marks the template as the default for sandbox creation when no template is
given. `404` if the template is missing.

:::note
`snapshot` (from a sandbox) and an `oci://<ref>` transport are specified
(`SPEC.md` §9 / §18.1) but not yet implemented.
:::
