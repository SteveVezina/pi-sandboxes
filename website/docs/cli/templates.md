---
sidebar_position: 7
---

# Templates

Templates define the language/runtime environment for a sandbox.

Built-in templates: `base`, `node`, `python`, `go`, `rust`, `node-python`,
`polyglot`.

```bash
pi-box template list                     # installed templates
pi-box template inspect <name>           # metadata, source/lineage, digest, compatibility, validation
pi-box template fork <source> <new-name> # editable local copy, with lineage
pi-box template validate <path-or-name>  # schema + policy-relevant checks
pi-box template build <name>             # build local artifacts for compatible runtimes
pi-box template update <name>            # refresh a built-in from product-managed definitions
pi-box template prune                    # remove unused local artifacts
```

## fork

```bash
pi-box template fork node my-node
```

Loads `node`, records `source.forkedFrom` and a `lineage` entry
(generation, parent content digest), and writes `my-node` to the local
library. Fails if `my-node` already exists.

## validate

```bash
pi-box template validate my-node          # an installed template
pi-box template validate ./template.yaml  # a file
pi-box template validate my-node --json
```

Checks: name present, base/image resolvable, cache destinations absolute
and under `/cache/`, mount destinations absolute, tool version pins
well-formed, `network` is `none`/`restricted`/`open`,
`compatibility.runtimes` uses known modes and `supported`/`planned`/
`unsupported`, `source.type` valid. Exit non-zero when there are problems.

## Template metadata (F28)

Built-in templates keep their minimal shape. Extended fields are all
optional: `version`, `summary`, `description`, `tags`, `source`
(`builtin`/`local`/`snapshot`/`imported` + `parent`/`forkedFrom`/
`snapshotOf`), `compatibility.runtimes`, `networkDomains` (seeds a
sandbox's `network.allow`), `resources`, `lineage` (generation + content
digests).

:::note Still planned (F28)
`snapshot`, `diff`, `history`, `rollback`, `promote`, `import`, `export`
are specified (`SPEC.md` §18.1) but not yet implemented.
:::
