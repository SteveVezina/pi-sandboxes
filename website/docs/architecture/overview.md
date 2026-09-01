---
sidebar_position: 1
---

# Architecture overview

## Design invariants

| Principle | Implication |
|-----------|-------------|
| Fix the root cause | No compensating code for agent/tool bugs — fix at the source |
| Spec-first | Code follows the spec, never the reverse |
| Local-first | Developer tooling first; remote / cluster comes later |
| Keep sandboxes warm | Reuse sandboxes; don't create/destroy per tool call |
| Selectable isolation | Multiple runtime modes; don't force one |
| Security by default | No host mounts, no Docker socket, no cloud metadata |
| Benchmark-driven | Measure from day one |

## Layers

```text
  CLI (pi-box) · GUI Workbench · SDK (TS / Python)
        │  Unix socket / HTTP / WebSocket
        ▼
  pi-sandboxd  (daemon)
    ├── Sandbox Manager · Template Engine · Policy Engine
    ├── Exec Manager · Snapshot Manager · Secrets / Credential Store
    ├── Workspace · Cache · Network (egress proxy)
    ├── Output Delivery · Remote Contexts · Lifecycle Events
    └── Runtime driver contract  ──►  fast · compat · secure · microvm
```

Everything above the **runtime driver contract** is runtime-neutral —
files, output delivery, logs, metadata, policy evaluation, API semantics.
A driver owns only isolation, process creation, mounts, network
attachment, resource control, and termination. It exposes `Probe`,
`Create`, `Start`, `Exec`, `Inspect`, `Stop`, `Destroy`, `Stats`.

`compat` and `secure` share one OCI engine layer (Docker / Podman);
`secure` is the same lifecycle with a `runsc` runtime handler, not a
parallel implementation.

## Sandbox identity

A sandbox handle carries the stable **sandbox ID** (user-facing) and the
driver-owned **runtime object ID** (container / VM ID) as distinct fields.
The sandbox ID is never mutated.

## API surface

See the [API reference](/api/overview) — that page is generated from the
actual routes in `pkg/daemon/router.go`, not from this overview.
