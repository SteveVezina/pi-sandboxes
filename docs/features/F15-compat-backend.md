# F4: Compat Backend

> Source: `SPEC.md` §6 Features F4
> Status: ✅ Implemented *(2026-08-28: T15.2c lifecycle recovery — container reconciliation implemented and verified)*
> Category: Infrastructure

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F4 | Compat Backend | OCI container backend (runc/containerd/Podman) for maximum compatibility | M1 |

## Expanded Specification

The compat backend provides maximum language/tool compatibility using standard OCI containers through runc/containerd/Podman/Docker-compatible plumbing.

Hardening defaults:
- No privileged containers
- No host network
- No Docker socket
- No hostPath unless explicitly configured
- Drop Linux capabilities by default
- Seccomp profile enabled
- AppArmor/SELinux where available

The compat backend creates containers that:
1. Pulls/uses the template's OCI image
2. Creates container with hardened defaults (no privileged, no host network, caps dropped, seccomp enabled)
3. Mounts workspace, artifacts, caches, /tmp, /home/agent
4. Starts the container with the given command
5. Streams stdout/stderr back to the daemon
6. Collects exit code, duration on completion

Runtime CLI operations used for container lifecycle are bounded to avoid wedging the daemon or test suite when a local OCI runtime stalls while pulling or starting an image.

OCI runtime selection (first available):
1. containerd (preferred)
2. Podman
3. runc
4. Docker (fallback)

The compat backend is cross-platform (Linux, macOS via Colima/Docker, Windows via WSL2/Docker).

Per PROP-008 / ADR-005 (`SPEC.md` §14.7.5):

- Compat is a thin `Driver` over a shared `pkg/runtime/oci` engine (`PodmanEngine`, `DockerEngine`, later containerd); Docker/Podman CLI construction is no longer duplicated per backend.
- Mount policy: `/workspace`, `/artifacts`, `/cache` are `rw,nosuid,nodev` (exec allowed — `./gradlew`, `node_modules/.bin/*`, `.venv/bin/python` must work); `noexec` applies to `/tmp` and secret mounts only.
- Containers are not created with `--rm`; explicit destroy plus daemon startup reconciliation and orphan garbage collection replace auto-remove.
- The stable sandbox ID and the runtime container ID are distinct fields; `spec.ID` is never overwritten after creation.
- Resource limits (memory, swap, CPUs, PIDs, ulimits) come from the shared `ResourceLimits` model and are passed at creation.
- Containers run as an explicit unprivileged user (Podman `--userns=keep-id --user uid:gid`; Docker explicit `--user` mapping).
- The project's versioned seccomp profile is passed explicitly via `--security-opt seccomp=` on both engines.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-4.1: Sandbox runs as OCI container via runc/containerd/Podman
- [x] AC-4.2: No privileged containers by default
- [x] AC-4.3: No host network by default
- [x] AC-4.4: No Docker socket mount by default
- [x] AC-4.5: Seccomp profile enabled
- [x] AC-4.6: Capabilities dropped by default
- [x] AC-4.7: Warm exec p50 < 100ms (SPEC.md §19)

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/runtime/compat/` | Compat backend implementation |
| `deploy/security/seccomp-profile.json` | Seccomp profile (shared with fast backend) |
| F2: Daemon API | Compat backend dispatch |
| F5: Template System | Templates define OCI images |
| F8: Sandbox Lifecycle | Container lifecycle management |

## Security Considerations

- No privileged containers
- No host network
- No Docker socket mount
- No hostPath unless explicitly configured
- Capabilities dropped by default
- Seccomp profile enabled
- AppArmor/SELinux where available

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F5: Template System | Internal feature | Templates define OCI images |
| F8: Sandbox Lifecycle | Internal feature | Container lifecycle |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Infrastructure** | OCI container runtime integration |
| **Service-layer** | Go compat backend in `pkg/runtime/compat/` |

**ADR references:** None yet.
**ADR gaps:**

| Question | Affects | Notes |
|----------|---------|-------|
| Additional OCI engines for the macOS/Windows story — Apple `container` (macOS 15+, per-container lightweight VM), `colima`, `podman machine`, `lima` | F15, F19 | The `oci.Engine` interface already abstracts this; on non-Linux the engine still runs Linux-in-a-VM. `fast`, `secure` (gVisor), and `microvm` (KVM) are inherently Linux — `compat` is the only portable isolation tier, and remote-daemon mode (F22/F23) is the alternative for a native macOS/Windows CLI against a Linux host. Not urgent; capture if a PROP touches engine selection. |

## Tasks

### T15.1: OCI runtime detection

**Description:** Implement OCI runtime detection. Try containerd, Podman, runc, Docker in order.

**Acceptance criteria:**
- [x] Detects containerd (preferred)
- [x] Detects Podman
- [x] Detects runc
- [x] Detects Docker
- [x] Uses first available runtime
- [x] Reports error if no OCI runtime found

**Verification:**
- [x] `go build ./pkg/runtime/compat/...`
- [x] Unit test: runtime detection with mock binaries

**Files:** `pkg/runtime/compat/runtime.go`
**Size:** S
**Depends on:** None

### T15.2a: Shared OCI engine extraction ✅ *(2026-07-14: done per PROP-008 — Docker/Podman share one CLIEngine)*

**Description:** Extract the duplicated Docker/Podman CLI construction into a shared `pkg/runtime/oci` engine. Behavior-preserving refactor; compat delegates to the engine.

**Acceptance criteria:**
- [x] `oci.Engine` interface with Docker/Podman implementations (`oci.CLIEngine` — the two CLIs are argument-compatible; runtime-specific flags hang off `extraCreateArgs` for T15.2b)
- [x] Container created from template OCI image via `oci.Engine`
- [x] Runtime CLI creation fails instead of hanging indefinitely when the OCI runtime stalls
- [x] Docker/Podman argument construction exists in exactly one place per operation (`oci/cli.go`)
- [x] Sandbox ID never overwritten with runtime container ID (`Container.RuntimeObjectID` carries the container ID)

**Verification:**
- [x] `go build ./pkg/runtime/...` (darwin + GOOS=linux)
- [x] Unit test: stalled runtime CLI creation times out (`tests/runtime/oci/engine_test.go`)
- [x] Unit test: sandbox ID preserved after creation (`tests/runtime/compat/identity_test.go`)
- [x] Existing compat integration tests pass unchanged

**Files:** `pkg/runtime/oci/engine.go`, `pkg/runtime/oci/cli.go`, `pkg/runtime/compat/create.go`, `pkg/runtime/compat/exec.go`, `pkg/runtime/compat/lifecycle.go`
**Size:** M
**Depends on:** T15.1, T19.1 (driver contract)

### T15.2b: Compat hardening corrections ✅ *(2026-07-14: done per PROP-008)*

**Acceptance criteria:**
- [x] Workspace, artifacts, caches mounted `rw,nosuid,nodev` on Linux (exec allowed); `noexec` kept on `/tmp` tmpfs only; `/home/agent` allows exec for user-installed tools
- [x] Explicit unprivileged user (Podman `--userns=keep-id --user uid:gid` from invoking user; Docker fixed `--user 1000:1000`)
- [x] Project-versioned seccomp profile (`pkg/runtime/oci/seccomp-profile.json`, v1 deny-list) written to `~/.pi-box/security/` and passed via `--security-opt seccomp=` (Docker: all platforms — CLI sends content; Podman: native Linux only — remote machines resolve paths server-side)
- [x] `ResourceLimits` (memory, swap, cpus, pids, nofile ulimit) passed at creation; SPEC §20/§22 defaults applied when unset (2 CPUs, 2 GiB, 256 PIDs)
- [x] No privileged mode, no host network, no Docker socket, no hostPath unless configured, capabilities dropped (unchanged)

**Verification:**
- [x] Unit tests: workspace-class mounts never `noexec`; `/tmp` stays `noexec` (`tests/runtime/oci/engine_test.go`)
- [x] Unit tests: Docker `--user 1000:1000`; Podman `--userns=keep-id --user`
- [x] Unit test: seccomp profile materialized and passed by path
- [x] Unit test: resource-limit flags rendered from `ResourceLimits`
- [x] Existing integration test: container cannot access host filesystem

**Files:** `pkg/runtime/oci/cli.go`, `pkg/runtime/oci/seccomp.go`, `pkg/runtime/oci/seccomp-profile.json`, `pkg/runtime/compat/create.go`
**Size:** M
**Depends on:** T15.2a

### T15.2c: Lifecycle recovery — no --rm + reconciliation ✅ *(2026-08-28: reconciliation implemented — compat.Reconcile wired into daemon startup)*

**Acceptance criteria:**
- [x] No `--rm`; container survives daemon crash for post-mortem inspection
- [x] Daemon startup reconciles sandbox store against `Inspect` results
- [x] Orphaned containers (no sandbox) are garbage-collected

**Verification:**
- [x] Integration test: daemon restart reconciles existing containers (`TestReconcile_RemovesOrphanAndReportsMissing`, `tests/runtime/compat/reconcile_test.go`)
- [x] Integration test: orphan container is garbage-collected (`TestReconcile_RemovesOrphanAndReportsMissing`, `tests/runtime/compat/reconcile_test.go`)

**Files:** `pkg/runtime/oci/*.go`, `pkg/runtime/compat/lifecycle.go`, `pkg/runtime/compat/reconcile.go`, `pkg/daemon/daemon.go`
**Size:** M
**Depends on:** T15.2a

### T15.3: Container exec and lifecycle

**Description:** Implement container exec, stdout/stderr streaming, exit code collection, container cleanup.

**Acceptance criteria:**
- [x] Command executes in container
- [x] stdout/stderr streamed to caller
- [x] Exit code captured accurately
- [x] Container cleaned up on destroy
- [x] Warm exec p50 < 100ms (SPEC.md §19)

**Verification:**
- [x] `go build ./pkg/runtime/compat/...`
- [x] Integration test: exec works in container
- [x] Benchmark: warm exec p50 < 100ms

**Files:** `pkg/runtime/compat/exec.go`, `pkg/runtime/compat/lifecycle.go`
**Size:** M
**Depends on:** T15.2 (container creation)

## Verification Plan

- [x] `go build ./pkg/runtime/compat/...` succeeds
- [x] OCI runtime detection works
- [x] Container created with hardened defaults
- [x] Exec works in container
- [x] Container cleaned up on destroy
- [x] Benchmark: warm exec p50 < 100ms (compat mode)
- [x] Benchmark: idle memory < 64 MiB (compat mode, SPEC.md §19)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| OCI runtime priority not specified | §7.2 Compat mode | Add: "Prefer containerd > Podman > runc > Docker" |

### Resolved gaps

| Gap | Block Spec Section | Resolution |
|-----|-------------------|------------|
| Template `base` field not mapped to OCI image name | §18 Templates, §20 Daemon API | PROP-007 — added `ResolveImage()` and `ResolveTemplateImage()` functions, image resolution in sandbox creation flow, state verification before WARM transition |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| ~~Shared OCI engine layering~~ | F4, F18 | **Resolved 2026-07-14:** ADR-005 (per PROP-008) |

## Out of Scope

- Docker-in-Docker (explicitly out of scope per SPEC.md §3)
- GPU support (explicitly out of scope per SPEC.md §3)
- Custom VMM (explicitly out of scope per SPEC.md §3)
- Firecracker/Cloud Hypervisor (Milestone 5)
