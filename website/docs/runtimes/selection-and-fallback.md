---
sidebar_position: 3
---

# Selection & fallback

When `mode` is `auto` (or omitted), the daemon resolves a concrete mode
from four separate inputs:

1. **Requested mode** — what the caller asked for.
2. **Workload trust** — trusted vs untrusted.
3. **Discovered capabilities** — each backend's `Probe` returns a
   structured capability report (availability, missing prerequisites,
   per-capability booleans, isolation tier, compatibility tier).
4. **Fallback policy** — an explicit allow/deny list of substitutions.

## Rules

- `auto` for a **trusted** workload resolves in performance order; for an
  **untrusted** workload it resolves in isolation order.
- Isolation is **never silently downgraded** below the requested mode. If
  you ask for `secure` and it is unavailable, the request fails with
  actionable guidance rather than dropping to `fast`.
- A denied fallback fails; every fallback decision is logged.

## Inspecting capabilities

```bash
pi-box system doctor          # human summary
curl --unix-socket ~/.pi-box/sandboxd.sock http://d/v1/system/runtimes
```

The `runsc` handler is the same OCI lifecycle as `compat` with a
different runtime handler — `secure` is not a parallel implementation.
