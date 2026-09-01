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
pi-box template history <name>           # local revision list
pi-box template diff <left> <right>      # compare two templates or revisions
pi-box template rollback <name> <n>      # restore a prior revision
pi-box template export <name> -o <file>  # portable OCI bundle
pi-box template import <bundle.tar>      # install as source: imported
pi-box template promote <name> --default # default for `box create` with no template
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

## history / diff / rollback

Every write to a local template records a revision under
`~/.pi-box/templates/<name>/revisions/`.

```bash
pi-box template history my-node                # rev N, timestamp, content digest
pi-box template diff my-node@1 my-node@3        # or: diff my-node node
pi-box template rollback my-node 1              # restore rev 1 (recorded as a new revision)
```

A ref is a template name, or `name@N` for revision `N`.

## Template metadata (F28)

Built-in templates keep their minimal shape. Extended fields are all
optional: `version`, `summary`, `description`, `tags`, `source`
(`builtin`/`local`/`snapshot`/`imported` + `parent`/`forkedFrom`/
`snapshotOf`), `compatibility.runtimes`, `networkDomains` (seeds a
sandbox's `network.allow`), `resources`, `lineage` (generation + content
digests).

## export / import

```bash
pi-box template export my-node -o my-node.oci.tar
pi-box template import my-node.oci.tar --name my-node-copy
```

The bundle is an **OCI image layout** tar (ADR-008): a config blob with
provenance (`contentDigest`, `lineage`, `source`, exporter version), a
definition layer (the template YAML), `vnd.pi-sandbox.template.*` media
types. It is a valid OCI artifact — `oras` / `skopeo` can also handle it.

`import` verifies every blob digest, re-runs `validate`, and installs the
template with `source.type: imported`. It is **untrusted until you run
`import` explicitly** — nothing is fetched automatically, and sandbox
`create` never pulls a template. A name collision is rejected; use
`--name` to install a copy.

## promote

```bash
pi-box template promote my-node --default
```

Marks `my-node` as the template `pi-box box create` uses when no template
argument is given (recorded in `~/.pi-box/templates/.default`). If the
marked template is later deleted, `create` falls back to `base`.

:::note Still planned (F28)
`snapshot` (from a sandbox), an `oci://<ref>` registry transport for
export/import, and the GUI surface are specified (`SPEC.md` §18.1) but not
yet implemented.
:::
