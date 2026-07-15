# F4: Compat Backend

> Source: `SPEC.md` §6 Features F4
> Status: 🟢 Implemented
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
| F8: Session Lifecycle | Container lifecycle management |

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
| F8: Session Lifecycle | Internal feature | Container lifecycle |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Infrastructure** | OCI container runtime integration |
| **Service-layer** | Go compat backend in `pkg/runtime/compat/` |

**ADR references:** None yet.
**ADR gaps:** None identified.

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

### T15.2: Container creation

**Description:** Implement container creation with hardened defaults. OCI container spec generation, container start.

**Acceptance criteria:**
- [x] Container created from template OCI image
- [x] Runtime CLI creation fails instead of hanging indefinitely when the OCI runtime stalls
- [x] No privileged mode
- [x] No host network (bridge network)
- [x] No Docker socket mount
- [x] No hostPath unless explicitly configured
- [x] Capabilities dropped by default
- [x] Seccomp profile enabled
- [x] Workspace, artifacts, caches mounted

**Verification:**
- [x] `go build ./pkg/runtime/compat/...`
- [x] Integration test: container created with hardened defaults
- [x] Unit test: stalled runtime CLI creation times out
- [x] Integration test: container cannot access host filesystem

**Files:** `pkg/runtime/compat/create.go`
**Size:** M
**Depends on:** T15.1 (runtime detection), F5 (Template System — OCI images)

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
| — | — | — |

## Out of Scope

- Docker-in-Docker (explicitly out of scope per SPEC.md §3)
- GPU support (explicitly out of scope per SPEC.md §3)
- Custom VMM (explicitly out of scope per SPEC.md §3)
- Firecracker/Cloud Hypervisor (Milestone 5)
