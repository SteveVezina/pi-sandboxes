---
sidebar_position: 3
---

# Persistence layout

All host-side state lives under `~/.pi-box` (override the socket path with
`pi-sandboxd --socket`). Legacy `~/.pi` data is never touched.

```text
~/.pi-box/
  config.yaml
  sandboxd.sock

  templates/
    base/ node/ python/ go/ rust/ node-python/ polyglot/

  sandboxes/
    <sandbox-id>/
      meta.json           # sandbox metadata (name, template, mode, network, TTL, state)
      workspace/          # (fast mode) working tree
      artifacts/
      logs/
        exec-<n>.json     # per-command entry
        exec-<n>.stdout
        exec-<n>.stderr
        egress.jsonl      # egress-proxy denials (restricted mode)
      snapshots/
      upper/ work/        # overlay dirs

  snapshots/
    content-addressed-store/

  runtime/
    caches/               # daemon-managed dependency cache volumes, scoped
    volumes/              # daemon-managed workspace/artifact volumes (compat/secure)

  images/
    rootfs/ kernels/ initrds/ microvm/
```

Inspect and clean it with `pi-box system status | doctor | prune |
disk-usage`.

:::note
`~/.pi-box` holds control-plane state, templates, sandbox metadata,
daemon-managed snapshots, and runtime-managed cache/volume state **only** —
never plaintext secrets or writable host bind targets.
:::
