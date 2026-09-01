---
sidebar_position: 7
---

# Templates

Templates define the language/runtime environment for a sandbox.

Built-in templates: `base`, `node`, `python`, `go`, `rust`, `node-python`,
`polyglot`.

```bash
pi-box template list                # installed templates
pi-box template inspect <name>      # template detail
pi-box template build <name>        # build local artifacts for compatible runtimes
pi-box template update <name>       # refresh a built-in from product-managed definitions
pi-box template prune               # remove unused local artifacts
```

:::note Planned (F28)
Local template lifecycle — `fork`, `snapshot`, `validate`, `diff`,
`history`, `rollback`, `promote`, `import`, `export` — is specified
(`SPEC.md` §18.1) but not yet implemented.
:::
