# PROP-008: Formal Runtime Driver Contract and Shared OCI Engine

## Status
✅ Applied to block spec (2026-07-14)

> Note: PROP-006 is still 🟡 Proposed. This PROP was explicitly requested by the human
> and is independent of PROP-006 (templates); treated as a permitted exception to the
> "no new PROP while unapproved proposals exist" rule.

## Block Spec Reference
`SPEC.md` §6 Features (F3, F4, F18, F19), §9 Interface Contract, §13 User-facing runtime modes, §14 Isolation strategy, §22 Security defaults

## Sources

Two external runtime architecture reviews of this project (2026-07):

- [Pi-sandboxes runtime review — driver contract, capability probing, selection engine](https://chatgpt.com/share/6a56f1a7-3178-83ea-803a-25d5f64a034d)
- [Pi-sandboxes runtime review — repo cross-check at commit 310a8cd](https://chatgpt.com/share/6a56f1fe-55a4-83ea-9fdc-053be432323d)

Both reviews converge on the same top recommendation: **do not add more runtime modes;
build one sandbox lifecycle and capability model, then plug execution drivers into it.**
The five public modes (`fast`, `compat`, `secure`, `isolated`, `microvm`) stay exactly as
specified in `SPEC.md` §13.

## Problem

The declared architecture, the runtime detection interface, and the actual backend
implementations have drifted apart. `ARCHITECTURE.md` describes a runtime dispatcher with
complete lifecycle backends, but the shared runtime abstraction is metadata-only:

```go
// pkg/runtime/detect/detect.go:16
type Runtime interface {
    Name() string
    IsAvailable() bool
    IsGvisor() bool
    GetMode() string
    GetSecurityLevel() int
}
```

It cannot create, start, exec, inspect, stop, snapshot, or destroy a sandbox. Every
backend therefore wires its lifecycle into the daemon differently, and each new backend
(microvm, kata, remote) increases the drift.

### Verified defects (current `main`)

| # | Defect | Location | Consequence |
|---|--------|----------|-------------|
| D1 | Runtime interface is metadata-only | `pkg/runtime/detect/detect.go:16` | Backends bypass any shared lifecycle; daemon dispatch is per-backend special cases |
| D2 | One-dimensional `GetSecurityLevel() int` | `pkg/runtime/detect/detect.go:21` | Cannot express "fast has Landlock but no egress policy" vs "microvm has kernel boundary but host mounts"; doctor output shows `security_level: 0` for unavailable runtimes, conflating availability with security |
| D3 | Single global `priority` list | `pkg/runtime/detect/detect.go:25` | Conflates security preference, performance preference, and availability fallback into one ordering; `auto` cannot mean "prefer performance for trusted work, isolation for untrusted work" (SPEC §13 requires trust-dependent auto-selection) |
| D4 | `fast.Validate()` never executes its probe | `pkg/runtime/fast/namespace.go:70` | Builds an `exec.Command("true")` with clone flags but returns `nil` without running it — false-positive availability on Linux hosts where unprivileged user namespaces are disabled |
| D5 | `syscall.SysProcIDRange` does not exist | `pkg/runtime/fast/namespace.go:46-55` | Linux build of the fast backend fails on current Go; correct type is `syscall.SysProcIDMap` (or `golang.org/x/sys/unix`) |
| D6 | Workspace mounted `noexec` on Linux | `pkg/runtime/compat/create.go:99,156` | Breaks core coding-agent flows: `./gradlew`, `node_modules/.bin/*`, `.venv/bin/python`, `go build -o ./bin/app && ./bin/app` |
| D7 | Containers created with `--rm` | `pkg/runtime/compat/create.go:105,162` | Daemon crash → container state vanishes on stop; no post-mortem inspection, no startup reconciliation possible |
| D8 | `spec.ID` overwritten with runtime container ID | `pkg/runtime/compat/create.go:147,204` | Conflates Pi session ID with runtime object ID; stable identity mutated after creation |
| D9 | No resource limits passed to container creation | `pkg/runtime/compat/create.go` | No `--memory`, `--cpus`, `--pids-limit`, `--ulimit` despite SPEC §22 resource-control requirements |
| D10 | No explicit `--user` mapping in CLI creation path | `pkg/runtime/compat/create.go` | Container may run as image default (often root) even though the generated OCI config specifies UID/GID 1000 |
| D11 | Seccomp assumed, not controlled | `pkg/runtime/compat/` | Relies on engine default profiles; Docker and Podman defaults differ — AC-4.5 requires a project-versioned profile |
| D12 | gVisor backend hand-builds OCI bundles | `pkg/runtime/gvisor/runtime.go` | Empty rootfs (image never unpacked), `runsc create` without `start`, root user in config, no `/workspace` mount — duplicates everything the compat backend already does, badly |
| D13 | Docker/Podman CLI construction duplicated | `pkg/runtime/compat/create.go` (parallel Docker/Podman blocks) | Every fix must be applied twice; containerd can never be added cleanly |

### Root cause

**F19 (Runtime Selection & Fallback)** specified detection and selection but never
specified a driver *lifecycle* contract, so each backend (F3 fast, F4 compat, F18
secure) invented its own integration. **§9 Interface Contract** covers CLI/API/SDK
surfaces but is silent on the internal runtime boundary.

## Proposed Amendment

### 1. Lifecycle-based runtime driver contract

Replace the metadata-only interface with a driver contract owned by `pkg/runtime`:

```go
type Driver interface {
    Name() string                 // implementation name: "podman", "runsc", "firecracker"
    Mode() Mode                   // public mode it serves: fast|compat|secure|isolated|microvm
    Probe(ctx context.Context) CapabilityReport

    Create(ctx context.Context, spec SandboxSpec) (Handle, error)
    Start(ctx context.Context, h Handle) error
    Exec(ctx context.Context, h Handle, req ExecRequest) (ExecSession, error)
    Inspect(ctx context.Context, h Handle) (RuntimeState, error)
    Stop(ctx context.Context, h Handle, grace time.Duration) error
    Destroy(ctx context.Context, h Handle) error
    Stats(ctx context.Context, h Handle) (RuntimeStats, error)
}
```

Rules:

- Files, artifacts, logs, metadata, policy evaluation, and API semantics stay **above**
  this layer. Drivers own only isolation, process creation, mounts, network attachment,
  resource controls, and termination.
- Workspace snapshots stay runtime-independent (`SnapshotProvider` remains separate —
  overlay/reflink/tar+zstd per SPEC §25); drivers never gate snapshot semantics.
- `Handle` carries `SessionID` (stable, user-facing) and `RuntimeObjectID`
  (driver-owned) as distinct fields. Fixes D8.
- Every state transition is journaled and idempotent (daemon crash → startup
  reconciliation re-derives state from the driver via `Inspect`).

### 2. Structured capability reports

Replace `GetSecurityLevel() int` with:

```go
type CapabilityReport struct {
    Available        bool
    Reason           string   // human-readable when unavailable
    Missing          []string // e.g. ["linux", "/dev/kvm", "firecracker"]
    KernelBoundary   bool
    Rootless         bool
    UserNamespace    bool
    Seccomp          bool
    Landlock         bool
    NetworkNamespace bool
    EgressPolicy     bool
    Snapshot         bool
    WarmExec         bool
    OCIImages        bool
    HardwareVirt     bool
    IsolationTier    int // 1=contained 2=sandboxed 3=virtualized 4=hardened
    CompatTier       int // separate axis — a runtime is not one integer
}
```

`pi-box system doctor` and `GET /v1/runtimes` render this report: availability, reason,
missing prerequisites, and per-capability flags instead of `security_level: 0`.
Fixes D2. Probes must actually execute (run the namespace probe command, run
`runsc --platform=... do true`), fixing D4.

### 3. Shared OCI engine

Introduce `pkg/runtime/oci` used by compat and secure (and later isolated/Kata):

```go
type Engine interface {
    Probe(ctx context.Context) CapabilityReport
    EnsureImage(ctx context.Context, ref ImageRef) (ImageID, error)
    Create(ctx context.Context, spec ContainerSpec) (ObjectID, error)
    Start(ctx context.Context, id ObjectID) error
    Exec(ctx context.Context, id ObjectID, req ExecRequest) (ExecSession, error)
    Inspect(ctx context.Context, id ObjectID) (ContainerState, error)
    Stop(ctx context.Context, id ObjectID, grace time.Duration) error
    Remove(ctx context.Context, id ObjectID) error
}
```

Implementations: `PodmanEngine`, `DockerEngine` (normalizing today's duplicated CLI
blocks — fixes D13), later `ContainerdEngine`.

The secure mode becomes **the same OCI lifecycle with a different runtime handler**
(`runsc` via `--runtime=runsc` or containerd handler) instead of hand-built bundles.
The current gVisor bundle builder in `pkg/runtime/gvisor/` is deleted. Fixes D12 —
secure inherits image pull/unpack, mounts, exec, logs, limits, and cleanup for free.

### 4. Selection engine: separate the four concepts

Replace the single `priority` list with explicit inputs:

1. **Requested mode** — `fast|compat|secure|isolated|microvm|auto`
2. **Workload trust** — `trusted|reviewed|untrusted` (from repo config/policy)
3. **Host capabilities** — discovered `CapabilityReport`s
4. **Fallback policy** — explicit allow/deny, configured per request or config

```yaml
runtime:
  mode: secure
  fallback:
    allow: [isolated, microvm]   # may go UP in isolation
    deny:  [fast, compat]        # never silently DOWN
```

`auto` resolution becomes trust-dependent (as SPEC §13 already requires):

```text
trusted   + preference performance → fast → compat → secure
untrusted + preference isolation   → secure → isolated → microvm
```

Hard rule: **never silently downgrade isolation below the requested mode.** A denied
fallback fails with actionable guidance (which prerequisite is missing, per the
capability report). Fallback decisions stay visible in logs/history (AC-22.5 unchanged).
Fixes D3.

### 5. Compat hardening corrections

| Fix | Change |
|-----|--------|
| D6 | `/workspace`, `/artifacts`, `/cache`: `rw,nosuid,nodev` (exec allowed). Keep `noexec` on `/tmp` and secret mounts only |
| D7 | Drop `--rm`; add explicit destroy + daemon startup reconciliation + orphan GC |
| D9 | Add shared `ResourceLimits` (memory, swap, cpus, pids, ulimits) to `SandboxSpec`, mapped by every engine |
| D10 | Podman: `--userns=keep-id --user uid:gid`; Docker: explicit `--user 1000:1000` with documented mapping strategy |
| D11 | Ship and version `deploy/security/seccomp-profile.json`; pass explicitly via `--security-opt seccomp=` on both engines |

### 6. Fast backend correctness (P0, blocking)

- Replace `syscall.SysProcIDRange` with `syscall.SysProcIDMap` (or migrate to
  `golang.org/x/sys/unix`). Fixes D5 — restores Linux build.
- `fast.Validate()` must run its probe command and report real errors. Fixes D4.
- Deeper fast-mode work (dedicated `pi-sandbox-init` helper, pivot_root, seccomp/Landlock
  application, PID-1 reaping) is acknowledged but **deferred** — tracked as F3 tasks,
  not part of this PROP.

### 7. Implementation order

```text
P0 (blocking):   D5 build fix, D4 real validation
P1 (contract):   Driver interface + registry + CapabilityReport + Handle ID separation
                 Selection engine with explicit fallback policy
P2 (compat):     Shared OCI engine (Podman/Docker), D6-D11 fixes, reconciliation
P3 (secure):     Rebuild gVisor on shared OCI engine, delete bundle builder
```

Target package layout:

```text
pkg/runtime/
├── driver.go          # Driver, Handle, SandboxSpec, ResourceLimits
├── capabilities.go    # CapabilityReport
├── registry.go        # driver registration
├── selector.go        # mode/trust/capability/fallback resolution
├── oci/               # shared Engine: podman.go, docker.go, spec.go
├── fast/              # implements Driver
├── compat/            # thin Driver over oci.Engine
├── gvisor/            # thin Driver over oci.Engine + runsc handler
└── microvm/           # implements Driver (existing vsock/agent code unchanged)
```

## Impact Analysis

| Component | Change |
|-----------|--------|
| `pkg/runtime/detect/detect.go` | Replaced by `registry.go` + `selector.go` + `capabilities.go`; `AllRuntimes`/doctor output switches to capability reports |
| `pkg/runtime/oci/` (new) | Shared engine, Podman/Docker implementations |
| `pkg/runtime/compat/` | Becomes thin driver over OCI engine; mount/user/limits/`--rm`/ID fixes |
| `pkg/runtime/gvisor/` | Handcrafted bundle code deleted; rebuilt as OCI runtime handler |
| `pkg/runtime/fast/namespace.go` | `SysProcIDMap` fix, real `Validate()` |
| `pkg/runtime/microvm/` | Adapts to Driver interface (no behavior change) |
| `pkg/api/sandbox_create.go` | Dispatches through registry/selector instead of per-backend branches |
| `pkg/system/doctor.go` | Renders capability reports |
| `pkg/logs/` | Fallback decision entries gain requested-vs-resolved mode fields |
| `docs/features/F03-fast-backend.md` | Validation + build-fix tasks |
| `docs/features/F15-compat-backend.md` | Mount/user/limits/lifecycle AC updates |
| `docs/features/F18-secure-backend.md` | Rebuild-on-OCI-engine approach |
| `docs/features/F19-runtime-selection-fallback.md` | Selection engine replaces priority list; resolves its open ADR gap ("runtime registry and fallback") |
| `SPEC.md` §14 | Amendment: driver contract + shared OCI engine as the isolation-strategy implementation rule |
| New ADR | ADR-005: Runtime driver contract and selection engine (resolves the ADR gap recorded in F19) |

## Dependencies

- F3 Fast Backend, F4 Compat Backend, F18 Secure Backend — refactored onto the contract
- F19 Runtime Selection & Fallback — superseded selection logic
- F17 Policy Enforcement — supplies trust input to the selector
- PROP-007 (applied) — image resolution feeds `oci.Engine.EnsureImage`

## Out of Scope

Deferred deliberately (both reviews agree these come after the contract is stable):

- New runtime modes or drivers: `process`, `wasm`, libkrun, Kata, Cloud Hypervisor,
  QEMU, Apple VF, Kubernetes/Nomad/Slurm providers, remote driver
- Network egress proxy / domain allowlisting (separate network subsystem PROP)
- Template build system, image digest locking, warm snapshot pools
- Scheduler/placement interface
- Exec/PTY protocol unification (WebSocket subprotocol) — separate PROP
- Lifecycle state-machine renaming (WARM vs READY split) — separate PROP
- Constraint-based scoring beyond mode/trust/fallback (latency budgets, GPU, placement)

## Cascade completed

Applied on 2026-07-14 (accepted by human same day):

- **Block spec:** `SPEC.md` §14 — added §14.7.5 "Runtime driver contract" (driver lifecycle, capability reports, shared OCI engine, mount/lifecycle/seccomp/limits rules, four-input selection, no-silent-downgrade)
- **ADRs:** `docs/decisions/ADR-005-runtime-driver-contract.md` created (resolves the runtime registry/fallback ADR gap recorded in F15, F18, F19)
- **Feature specs cascaded:**
  - `docs/features/F03-fast-backend.md` — T3.1 reset ⚠️ (SysProcIDMap build fix, real `Validate()`)
  - `docs/features/F15-compat-backend.md` — status ⚠️; T15.2 reset ⚠️ (OCI engine, mount exec policy, no `--rm`, limits, user mapping, versioned seccomp, ID separation); ADR gap resolved
  - `docs/features/F18-secure-backend.md` — status ⚠️; T18.1 reset ⚠️ (rebuilt on shared OCI engine); ADR gap resolved
  - `docs/features/F19-runtime-selection-fallback.md` — status ⚠️; T19.1/T19.2 reset ⚠️ (capability reports, four-input selection); ADR gap resolved
- **INDEX files:** `docs/features/INDEX.md` (F4/F18/F19 → ⚠️, M4 → ⚠️, summary note), `docs/proposals/INDEX.md` (PROP-008 → ✅ Applied)
- **Plan:** `docs/plan.md` Active Cursor points at PROP-008 P0-P3 implementation order
