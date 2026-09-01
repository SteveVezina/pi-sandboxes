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

:::note
`snapshot`, `diff`, `history`, `rollback`, `promote`, `import`, `export`
endpoints are specified (`SPEC.md` §9) but not yet implemented.
:::
