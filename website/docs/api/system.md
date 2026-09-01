---
sidebar_position: 3
---

# System endpoints

## Health

`GET /health` → `{ "status": "ok" }` — unauthenticated liveness check.

## Status

`GET /v1/system/status` — daemon connection state plus active and total
sandbox counts.

## Doctor

`GET /v1/system/doctor` — config validation, directory checks, disk space,
permissions, and per-issue recommendations. Backs `pi-box system doctor`.

## Runtimes

`GET /v1/system/runtimes` — the capability report for each runtime backend
on this host: availability, missing prerequisites, per-capability booleans,
isolation tier, and compatibility tier. Backs `pi-box system doctor`'s
"available backends" line and the GUI diagnostics view.

## Support bundle

`GET /v1/support-bundle` — a diagnostics archive (daemon diagnostics,
version metadata, redacted configuration) for bug reports.
