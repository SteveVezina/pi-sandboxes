# F03: Fast Backend

> Source: `SPEC.md` §6 Features F3
> Status: 🟢 Implemented
> Category: Infrastructure

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F3 | Fast Backend | Native Linux sandbox using namespaces, cgroups, seccomp, Landlock isolation | M1 |

## Expanded Specification

The fast backend provides the lowest-overhead sandbox isolation using Linux kernel primitives. It is designed for trusted local developer use where the code being executed is not malicious.

Isolation layers:
1. **User namespace** — maps root inside sandbox to unprivileged user on host
2. **Mount namespace** — isolated filesystem view; only declared mounts are accessible
3. **PID namespace** — sandbox processes are isolated from host PID space
4. **cgroup v2** — CPU, memory, disk I/O, and PID limits enforced by kernel
5. **seccomp** — syscall whitelist blocks dangerous syscalls (mount, ptrace, reboot, kexec, bpf)
6. **Landlock** (where available) — additional filesystem access control
7. **Restricted /proc** — limited process visibility inside sandbox
8. **Read-only root** — root filesystem is read-only where possible; writable dirs: `/workspace`, `/tmp`, `/home/agent`, `/cache`, `/artifacts`

The fast backend creates a process tree that:
1. Sets up namespaces (user, mount, PID)
2. Creates cgroup v2 hierarchy with resource limits
3. Applies seccomp profile
4. Mounts workspace, artifacts, caches, and read-only rootfs
5. Starts the sandboxed process with the given command
6. Streams stdout/stderr back to the daemon
7. Collects exit code, duration, and resource usage on completion

The fast backend is Linux-only. macOS and Windows require a Linux helper VM or remote daemon (future).

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-3.1: Sandbox runs in isolated namespace/cgroup environment
- [ ] AC-3.2: Host filesystem is not mounted by default
- [ ] AC-3.3: Process limits enforced (maxProcesses: 256)
- [ ] AC-3.4: Command timeout enforced (default: 120s)
- [ ] AC-3.5: Output truncation at maxOutput (default: 8MiB)
- [ ] AC-3.6: Exec overhead p50 < 10ms (SPEC.md §19)

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/runtime/fast/` | Fast backend implementation |
| `deploy/security/seccomp-profile.json` | Syscall whitelist |
| `~/.pi/` | cgroup hierarchy for sandbox processes |
| F7: Command Execution | Fast backend is the execution target |
| F8: Session Lifecycle | Fast backend manages process lifecycle |

## Security Considerations

- **No host filesystem access** — only workspace, artifacts, caches, /tmp are mounted
- **No privileged operations** — seccomp blocks mount, ptrace, reboot, kexec, bpf
- **No capability escalation** — all capabilities dropped except minimal set
- **Landlock as defense-in-depth** — additional filesystem control beyond mount namespace
- **cgroup v2 hard limits** — CPU throttling, memory OOM kill, disk I/O cap
- **Process limit** — maxProcesses: 256 enforced by cgroup

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F8: Session Lifecycle | Internal feature | ⚠️ Partially — backend needs session context |
| Linux kernel namespaces | OS feature | Linux-only |
| cgroup v2 | OS feature | Linux-only |
| seccomp | OS feature | Linux-only |
| Landlock | OS feature | Kernel ≥ 5.13, optional |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Infrastructure** | Linux kernel primitives, cgroup v2, seccomp profile |
| **Service-layer** | Go backend in `pkg/runtime/fast/` |

**ADR references:** None yet.
**ADR gaps:** None identified.

### Surfacing an ADR need

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How to handle missing Landlock on older kernels? | F3 | ADR-NNN: Landlock fallback strategy |

## Tasks

### T3.1: Namespace setup

**Description:** Implement Linux namespace setup (user, mount, PID) for sandbox processes. Use `syscall.Unshare` or Go's `os/exec` with namespace flags.

**Acceptance criteria:**
- [ ] User namespace maps sandbox root to unprivileged host user (uid 1000)
- [ ] Mount namespace isolates filesystem view
- [ ] PID namespace isolates process tree
- [ ] Namespace setup completes in < 5ms

**Verification:**
- [ ] `go build ./pkg/runtime/fast/...`
- [ ] Integration test: sandboxed process sees isolated PID 1

**Files:** `pkg/runtime/fast/namespace.go`
**Size:** M
**Depends on:** F8 (Session Lifecycle — needs sandbox ID context)

### T3.2: cgroup v2 resource limits

**Description:** Implement cgroup v2 hierarchy with CPU, memory, disk I/O, and PID limits. Create cgroup per sandbox session.

**Acceptance criteria:**
- [ ] cgroup v2 hierarchy created under `~/.pi/cgroups/<sandbox-id>/`
- [ ] CPU limit enforced (cgroup CPU weight)
- [ ] Memory limit enforced (cgroup memory.max, OOM kill on breach)
- [ ] Disk I/O limit enforced (cgroup io.max)
- [ ] PID limit enforced (cgroup pids.max = 256)
- [ ] cgroup cleaned up on sandbox destroy

**Verification:**
- [ ] `go build ./pkg/runtime/fast/...`
- [ ] Integration test: process exceeding memory limit is OOM killed
- [ ] Integration test: process exceeding PID limit is blocked

**Files:** `pkg/runtime/fast/cgroup.go`
**Size:** M
**Depends on:** T3.1 (namespace setup)

### T3.3: Seccomp profile

**Description:** Implement seccomp syscall whitelist. Create seccomp profile JSON and apply via `prctl(PR_SET_SECCOMP)`.

**Acceptance criteria:**
- [ ] Seccomp profile blocks: mount, ptrace, reboot, kexec, bpf, bootctl, swapon, swapoff
- [ ] Seccomp profile allows: read, write, open, close, stat, fstat, mmap, mprotect, brk, execve, clone, fork, vfork, kill, exit, wait4, kill, gettid, set_tid, set_tid, clock_gettime, clock_nanosleep, nanosleep, getuid, getgid, geteuid, getegid, access, pipe, select, sched_yield, mremap, msync, mincore, madvise, dup, dup2, close, waitpid, sigreturn, io_setup, io_destroy, io_submit, io_cancel, io_getevents, poll, ppoll, pselect6, signalfd, timerfd, eventfd, futex, set_robust_list, get_robust_list, splice, tee, readv, writev, fcntl, flock, fsync, fdatasync, truncate, ftruncate, getdents, getcwd, chdir, mkdir, rmdir, rename, link, symlink, readlink, chmod, chown, lchown, umask, gettimeofday, getrlimit, rusage, sysinfo, times, getuid, getgid, geteuid, getegid, setuid, setgid, getgroups, setgroups, setresuid, setresgid, getresuid, getresgid, prctl, arch_prctl, exit_group, epoll_create, epoll_ctl, epoll_wait, eventfd2, epoll_create1, epoll_ctl, epoll_wait, epoll_pwait, clone3, set_tid, set_robust_list, get_robust_list, rseq
- [ ] Seccomp applied before process exec

**Verification:**
- [ ] `go build ./pkg/runtime/fast/...`
- [ ] Integration test: sandboxed process cannot call mount syscall
- [ ] Integration test: sandboxed process cannot ptrace another process

**Files:** `deploy/security/seccomp-profile.json`, `pkg/runtime/fast/seccomp.go`
**Size:** M
**Depends on:** T3.1 (namespace setup)

### T3.4: Filesystem mounts and rootfs

**Description:** Implement read-only rootfs mount with writable overlays for workspace, artifacts, caches, /tmp, /home/agent.

**Acceptance criteria:**
- [ ] Root filesystem is read-only (overlayfs or bind mount)
- [ ] `/workspace` is writable and contains the repo checkout
- [ ] `/artifacts` is writable for build outputs
- [ ] `/cache` directories are writable (npm, pnpm, pip, uv, go-mod, go-build, cargo)
- [ ] `/tmp` is process-local (tmpfs or bind mount)
- [ ] `/home/agent` is writable with minimal user config
- [ ] Host directories are NOT mounted by default

**Verification:**
- [ ] `go build ./pkg/runtime/fast/...`
- [ ] Integration test: sandboxed process cannot access host filesystem

**Files:** `pkg/runtime/fast/mounts.go`
**Size:** M
**Depends on:** T3.1 (namespace setup), T3.2 (cgroup setup)

### T3.5: Landlock (optional enhancement)

**Description:** Implement Landlock filesystem access control where kernel supports it (≥ 5.13). Provides additional defense-in-depth beyond mount namespace.

**Acceptance criteria:**
- [ ] Landlock policy applied if kernel ≥ 5.13
- [ ] Landlock falls back gracefully if kernel < 5.13 (no error, just mount namespace)
- [ ] Landlock restricts filesystem access to workspace, artifacts, caches, /tmp, /home/agent

**Verification:**
- [ ] `go build ./pkg/runtime/fast/...`
- [ ] Integration test: Landlock active on kernel ≥ 5.13
- [ ] Integration test: no error on kernel < 5.13

**Files:** `pkg/runtime/fast/landlock.go`
**Size:** S
**Depends on:** T3.4 (filesystem mounts)

## Verification Plan

- [ ] `go build ./pkg/runtime/fast/...` succeeds
- [ ] Integration tests verify namespace isolation
- [ ] Integration tests verify cgroup limits
- [ ] Integration tests verify seccomp blocks dangerous syscalls
- [ ] Benchmark: warm exec p50 < 10ms (SPEC.md §19)
- [ ] Benchmark: idle memory < 64 MiB (SPEC.md §19)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Landlock fallback behavior not specified | §7.1 Fast mode | Add: "If Landlock unavailable (kernel < 5.13), continue with mount namespace only" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How to handle missing Landlock on older kernels? | F3 | ADR-NNN: Landlock fallback strategy |

Note: ADRs are block-level. Flag the need here; author the ADR file as a separate commit.

## Out of Scope

- macOS/Windows support (requires Linux helper VM — future)
- gVisor backend (F16 — secure mode, Milestone 4)
- MicroVM backend (F17 — Milestone 5)
- Docker-in-Docker (explicitly out of scope per SPEC.md §3)
