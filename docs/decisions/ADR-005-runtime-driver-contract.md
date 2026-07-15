# ADR-005: Runtime Driver Contract and Selection Engine

## Status
Accepted (2026-07-14, per PROP-008)

## Context

F19 specified runtime detection and selection but never specified a driver lifecycle contract. Each backend (F3 fast, F4 compat, F18 secure, F20 microvm) invented its own daemon integration. The shared abstraction in `pkg/runtime/detect` was metadata-only (`Name/IsAvailable/IsGvisor/GetMode/GetSecurityLevel`): it could not create, exec, inspect, or destroy a sandbox. A single global priority list conflated security preference, performance preference, and availability fallback, and a one-integer security level could not express capability differences between backends. This ADR resolves the open ADR gap recorded in F15, F18, and F19 ("runtime registry and fallback policy").

## Decision

1. **Lifecycle driver contract.** All backends implement one `Driver` interface in `pkg/runtime`: `Name`, `Mode`, `Probe`, `Create`, `Start`, `Exec`, `Inspect`, `Stop`, `Destroy`, `Stats`. Everything above the driver (files, artifacts, logs, metadata, policy, API semantics) is runtime-neutral. Workspace snapshots stay outside the driver contract.

2. **Identity separation.** A `Handle` carries `SessionID` (stable, user-facing) and `RuntimeObjectID` (driver-owned) as distinct fields. Session IDs are never overwritten with runtime object IDs.

3. **Structured capability reports.** `Probe` returns a `CapabilityReport` (availability, reason, missing prerequisites, per-capability booleans, isolation tier, compatibility tier). Probes must actually execute their checks. `GetSecurityLevel() int` is removed; doctor and `GET /v1/runtimes` render the report.

4. **Shared OCI engine.** `pkg/runtime/oci` defines an `Engine` interface (EnsureImage/Create/Start/Exec/Inspect/Stop/Remove) with `PodmanEngine` and `DockerEngine` implementations (later containerd). Compat and secure are thin drivers over this engine; secure differs only by the `runsc` runtime handler. The handcrafted gVisor OCI bundle builder is deleted.

5. **Selection engine.** Selection takes four separate inputs: requested mode, workload trust, discovered capabilities, and explicit fallback policy (allow/deny). `auto` resolution is trust-dependent (trusted → performance ordering; untrusted → isolation ordering). Isolation is never silently downgraded below the requested mode; denied fallbacks fail with actionable guidance and all fallback decisions are logged (AC-22.5 preserved).

## Consequences

- New backends (Kata, remote, libkrun) plug into one contract instead of adding daemon special cases.
- Compat hardening fixes (workspace exec, no `--rm`, resource limits, explicit user mapping, versioned seccomp profile) apply once in the OCI engine and benefit secure mode automatically.
- Daemon startup reconciliation becomes possible because containers persist after exit and drivers expose `Inspect`.
- F3, F4, F18, F19 task statuses were reset where acceptance criteria changed; re-verification is required.
- The fast backend's Linux build fix (`syscall.SysProcIDMap`) and real availability validation are P0 prerequisites.

## References

- `SPEC.md` §14.7.5 Runtime driver contract
- `docs/proposals/PROP-008-runtime-driver-contract.md`
- `docs/features/F03-fast-backend.md`, `docs/features/F15-compat-backend.md`, `docs/features/F18-secure-backend.md`, `docs/features/F19-runtime-selection-fallback.md`
- ADR-001 (microVM backend sits behind the same runtime interface)
