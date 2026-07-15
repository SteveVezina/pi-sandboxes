# PI Agent Sandbox Runtime - Source of Truth Spec

## 1. Project thesis

Build a local-first, open-source sandbox runtime for PI coding agents.

The runtime must provide tiny, fast, isolated developer workspaces where AI coding agents can clone repositories, read and write files, run Node.js, Python, Go, and Rust commands, build and run apps, collect logs, export artifacts, and produce patches.

The project is not Kubernetes-first, cloud-first, or SaaS-first. It starts as developer local tooling with a daemon, CLI, API, and SDKs. Kubernetes and remote control planes can come later.

Core positioning:

> Local-first sandboxes for coding agents. Fast warm exec, tiny footprint, selectable isolation.

The first version should feel like:

```bash
pi-box box create node-python
pi-box box clone <repo>
pi-box box exec -- pnpm test
pi-box box diff
pi-box box artifacts pull ./out
```

## 2. Primary goals

1. Provide the fastest practical local coding-agent sandbox loop.
2. Keep footprint small enough to run many sandboxes on a developer workstation.
3. Support real coding workloads in Node.js, Python, Go, and Rust.
4. Keep sandbox sessions warm and long-lived.
5. Make command execution inside existing sessions extremely fast.
6. Provide selectable isolation levels instead of forcing one runtime.
7. Provide a clean local CLI and daemon API usable by PI Agent and other coding agents.
8. Support workspace snapshots, rollback, diffs, logs, and artifact export.
9. Avoid mounting the developer home directory or secrets by default.
10. Be benchmark-driven from day one.

## 3. Non-goals for the first release

Do not start with these:

- Kubernetes controller or CRDs.
- Multi-region control plane.
- Full SaaS product.
- Firecracker rewrite.
- Custom VMM implementation.
- Docker-in-Docker as a default capability.
- GPU workloads.
- GUI/browser desktop agents in the first release. The GUI workbench is a later M7 client surface.
- Full Docker Compose compatibility.
- Complex enterprise identity management.
- Background cloud scheduler.

These can be added later after the local developer runtime is useful.

## 4. Core design principle

Do not create and destroy a sandbox for every tool call.

Bad pattern:

```text
agent tool call -> create container or VM -> run command -> destroy sandbox
```

Correct pattern:

```text
create sandbox session once
  -> keep workspace mounted
  -> keep process or guest agent alive
  -> keep dependency caches mounted
  -> run many exec calls through the same session
  -> snapshot or rollback when needed
  -> destroy when the session expires
```

The runtime optimizes the AI coding-agent inner loop:

```text
clone repo
inspect files
edit files
run tests
read output
fix errors
run app
collect logs
export patch/artifacts
repeat
```

## 5. Core Concepts

| Concept | Description |
|---------|-------------|
| **Sandbox session** | An isolated execution environment with a unique ID, workspace, and lifecycle. Created once, kept warm, destroyed on TTL or explicit close. |
| **Template** | A declarative definition of the language/toolchain environment (base OS, installed tools, cache mounts, network policy). |
| **Runtime mode** | The isolation backend used for a session: `fast` (namespaces), `compat` (OCI container), `secure` (gVisor), `isolated` (Kata), `microvm` (Firecracker/CH). |
| **Workspace** | The editable filesystem inside a sandbox: `/workspace` (repo checkout), `/artifacts` (build outputs), `/cache` (dependency caches), `/tmp` (ephemeral). |
| **Snapshot** | A point-in-time copy of a sandbox's filesystem state, enabling rollback. Implemented via overlay upperdir, reflink, or tar/zstd. |
| **Artifact** | Files intentionally exported from a sandbox (build outputs, test reports, patches). |
| **Cache** | Scoped dependency caches (npm, pnpm, pip, uv, go-mod, cargo) mounted into sandboxes to avoid redundant downloads. |
| **Policy** | The security configuration for a sandbox: filesystem mounts, process limits, network mode, secret exposure. |

## 6. Features

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F1 | CLI Entry Point | `pi-box` binary with `box` subcommands for sandbox lifecycle management | M1 |
| F2 | Daemon API | `pi-sandboxd` local daemon exposing Unix socket HTTP API for sandbox operations | M1 |
| F3 | Fast Backend | Native Linux sandbox using namespaces, cgroups, seccomp, Landlock isolation | M1 |
| F4 | Compat Backend | OCI container backend (runc/containerd/Podman) for maximum compatibility | M1 |
| F5 | Template System | Declarative template definition, build, and management (base, node, python, go, rust, node-python, polyglot) | M1 |
| F6 | Workspace & File Operations | Clone, file read/write, diff, patch, pull/push within sandbox sessions | M1 |
| F7 | Command Execution | Streaming stdout/stderr exec with timeout, output limits, exit code, and truncation metadata | M1 |
| F8 | Session Lifecycle | Create, list, inspect, destroy, TTL expiration, warm session reuse | M1 |
| F9 | Artifact Export | List, pull, and pack artifacts from sandbox sessions | M1 |
| F10 | Logs & Command History | Command history, stdout/stderr logs, exit codes, duration, timeout status | M1 |
| F11 | Secrets & Network Model | Configurable network modes (none/restricted/open), domain allowlist, secret broker for Git credentials | M2 |
| F12 | Cache Model | Scoped dependency cache mounts, cache pruning, cache promotion | M2 |
| F13 | Snapshot & Rollback | Filesystem-level snapshot creation and rollback (overlay/reflink) | M2 |
| F14 | Benchmarks | Mandatory benchmark suite measuring warm exec, install, test, snapshot, export, and density | M1 |
| F15 | SDKs | TypeScript and Python SDKs with streaming output support | M3 |
| F16 | System Commands | `pi-box system status/doctor/prune/disk-usage` for local state inspection | M1 |
| F17 | Policy Enforcement | Default security policy: no host home mount, no Docker socket, process limits, output limits | M2 |
| F18 | Secure Backend | gVisor-backed runtime mode for unknown or untrusted repositories, including runsc integration, secure-mode lifecycle, compatibility notes, and benchmark comparison against fast/compat | M4 |
| F19 | Runtime Selection & Fallback | Runtime detection and backend selection across fast, compat, secure, and future microVM modes, including fallback to compat mode when secure mode is unavailable or incompatible | M4 |
| F20 | MicroVM Backend | Firecracker or Cloud Hypervisor backend with `pi-vmm-manager`, tiny guest rootfs, workspace disk, template snapshot restore, artifact export, and reseed-on-restore behavior | M5 |
| F21 | MicroVM Guest Control Plane | Guest-side `pi-init` and `pi-agentd` over virtio-vsock for command execution, lifecycle coordination, file/artifact transfer, and sandbox readiness reporting | M5 |
| F22 | Remote Daemon Contexts | CLI context management for local and remote daemons, including `pi-box context create/use/list/inspect/delete` and context-aware `pi-box box` commands | M6 |
| F23 | Remote Daemon Transport & Auth | SSH/Tailscale/WireGuard-friendly remote daemon access with secure local-to-remote API authentication and remote workstation support | M6 |
| F24 | Cross-Platform GUI Workbench | Desktop application for macOS, Windows, and Linux that connects to local or remote PI daemons, creates and manages sandbox sessions, and exposes common sandbox workflows without replacing the CLI | M7 |
| F25 | GUI Workspace Authorization | Explicit project-folder selection, allowed folder management, and safe bind/copy workspace setup for GUI-launched sessions | M7 |
| F26 | GUI Session Operations | Dashboard and session views for create/list/inspect/exec/logs/diff/patch/artifacts/snapshots/destroy using existing daemon API operations | M7 |
| F27 | GUI Settings and Diagnostics | GUI controls for daemon connection, active context, default template/runtime mode/network policy, engine health, doctor output, and support bundle export | M7 |

## 7. Acceptance Criteria

### AC-1: CLI Works (F1)
- [ ] `pi-box box create --name demo --template node-python --mode fast` creates a sandbox
- [ ] `pi-box box list` shows created sandboxes
- [ ] `pi-box box inspect demo` shows sandbox details
- [ ] `pi-box box destroy demo` cleans up sandbox

### AC-2: Daemon API Responds (F2)
- [ ] `pi-sandboxd` listens on `~/.pi-box/sandboxd.sock`
- [ ] `POST /v1/sandboxes` creates a sandbox
- [ ] `GET /v1/sandboxes` lists sandboxes
- [ ] `GET /v1/sandboxes/{id}` returns sandbox state
- [ ] `DELETE /v1/sandboxes/{id}` destroys a sandbox

### AC-3: Fast Backend Isolates (F3)
- [ ] Sandbox runs in isolated namespace/cgroup environment
- [ ] Host filesystem is not mounted by default
- [ ] Process limits enforced (maxProcesses: 256)
- [ ] Command timeout enforced (default: 120s)
- [ ] Output truncation at maxOutput (default: 8MiB)

### AC-4: Compat Backend Works (F4)
- [ ] Sandbox runs as OCI container via runc/containerd/Podman
- [ ] No privileged containers by default
- [ ] No host network by default
- [ ] No Docker socket mount by default
- [ ] Seccomp profile enabled
- [ ] Capabilities dropped by default

### AC-5: Templates Are Usable (F5)
- [ ] `base`, `node`, `python`, `go`, `rust`, `node-python`, `polyglot` templates defined
- [ ] `pi-box template list` shows available templates
- [ ] `pi-box template inspect <name>` shows template details
- [ ] Templates configure correct toolchains and cache mounts

### AC-6: File Operations Work (F6)
- [ ] `pi-box box clone <repo>` clones a repository into sandbox workspace
- [ ] `pi-box box files read <id> <path>` reads a file from sandbox
- [ ] `pi-box box files write <id> <path>` writes a file to sandbox
- [ ] `pi-box box diff <id>` shows workspace diff
- [ ] `pi-box box patch <id>` exports workspace as patch

### AC-7: Exec Streams Output (F7)
- [ ] `pi-box box exec <id> -- <cmd>` runs command with streaming stdout/stderr
- [ ] Exit code returned accurately
- [ ] Timeout status reported when exceeded
- [ ] Output truncated when exceeding maxOutput, with `truncated` flag
- [ ] `--cwd`, `--timeout`, `--max-output`, `--memory`, `--cpu`, `--json` options honored

### AC-8: Session Lifecycle (F8)
- [ ] Sandbox created once and kept warm
- [ ] Multiple exec calls reuse the same session
- [ ] TTL expiration triggers cleanup
- [ ] `pi-box box destroy --all` cleans all sandboxes

### AC-9: Artifacts Export (F9)
- [ ] `pi-box box artifacts list <id>` lists available artifacts
- [ ] `pi-box box artifacts pull <id> <dest>` pulls artifacts to host
- [ ] `pi-box box artifacts pack <id> --output <file>` creates archive

### AC-10: Logs Available (F10)
- [ ] `pi-box box logs <id>` shows command logs
- [ ] `pi-box box history <id>` shows command history
- [ ] Each log entry includes: command, exit code, duration, timeout status, output truncation

### AC-11: Network Modes Work (F11)
- [ ] `none` mode blocks all outbound network
- [ ] `restricted` mode enforces domain allowlist
- [ ] `open` mode allows full outbound access
- [ ] Default deny: metadata endpoint (169.254.169.254), host localhost, private LANs

### AC-12: Caches Are Scoped (F12)
- [ ] `/cache/npm`, `/cache/pnpm`, `/cache/pip`, `/cache/uv`, `/cache/go-mod`, `/cache/go-build`, `/cache/cargo` mounted
- [ ] Caches scoped by template/runtime/user
- [ ] `pi-box system prune` can clean caches

### AC-13: Snapshots Work (F13)
- [ ] `pi-box box snapshot <id> <name>` creates a named snapshot
- [ ] `pi-box box rollback <id> <name>` restores to snapshot
- [ ] Snapshot creation uses overlay upperdir or reflink
- [ ] Snapshot metadata stored under `~/.pi-box/sandboxes/<id>/snapshots/`

### AC-14: Benchmarks Run (F14)
- [ ] `pi-box bench run` executes full benchmark suite
- [ ] All 13 required benchmarks execute: warm_exec_echo, warm_exec_shell, file_scan_rg, git_clone_small, pnpm_install_cached, uv_sync_cached, go_test_cached, cargo_test_cached, snapshot_create, snapshot_rollback, artifact_export_20mb, parallel_10, parallel_100
- [ ] Output includes p50/p95 latency and memory per sandbox
- [ ] Per-mode comparison (fast vs compat)

### AC-15: SDKs Work (F15)
- [ ] TypeScript SDK: `client.sandboxes.create()`, `.clone()`, `.exec()`, `.diff()`
- [ ] Python SDK: `client.sandboxes.create()`, `.clone()`, `.exec()`, `.diff()`
- [ ] Both support streaming output

### AC-16: System Commands Work (F16)
- [ ] `pi-box system status` shows daemon and sandbox status
- [ ] `pi-box system doctor` validates configuration
- [ ] `pi-box system prune` cleans old state
- [ ] `pi-box system disk-usage` shows storage breakdown

### AC-17: Policy Enforced (F17)
- [ ] Host home directory not mounted by default
- [ ] Docker socket not mounted by default
- [ ] Cloud metadata credentials not accessible
- [ ] SSH private keys not mounted by default
- [ ] Git credentials brokered (not dumped into environment)
- [ ] Exec output limited to 8MiB by default
- [ ] Exec timeout 120s by default
- [ ] Max processes 256 by default

### AC-18: End-to-End Agent Loop (F1, F2, F3, F5-F10)
- [ ] `pi-box box create --name demo --template node-python --mode fast`
- [ ] `pi-box box clone demo https://github.com/some/repo`
- [ ] `pi-box box exec demo -- pnpm install`
- [ ] `pi-box box exec demo -- pnpm test`
- [ ] `pi-box box diff demo`
- [ ] `pi-box box artifacts pull demo ./out`
- [ ] `pi-box box destroy demo`
- [ ] All steps succeed without direct host filesystem access

### AC-19: Warm Exec Performance (F3, F4, F14)
- [ ] Fast mode warm exec p50 < 10ms
- [ ] Compat mode warm exec p50 < 100ms
- [ ] New warm session assignment < 100ms
- [ ] Artifact export 20MB < 500ms local
- [ ] Idle fast sandbox memory < 64 MiB

### AC-20: Multi-Language Support (F5, F7)
- [ ] Node.js: `npm install`, `pnpm install`, `pnpm test`, `pnpm build`, `npm run dev`
- [ ] Python: `pip install -r requirements.txt`, `uv sync`, `uv run pytest`, `python script.py`
- [ ] Go: `go mod download`, `go test ./...`, `go build ./...`
- [ ] Rust: `cargo fetch`, `cargo test`, `cargo build`

### AC-21: Secure Backend Works (F18)
- [ ] `pi-box box create --mode secure <template>` creates a sandbox using gVisor/runsc when available
- [ ] Secure sandboxes execute commands through the same daemon API as fast/compat sandboxes
- [ ] Secure mode does not mount the host home directory or Docker socket by default
- [ ] Secure mode exposes compatibility errors with actionable guidance
- [ ] Benchmarks compare fast vs compat vs secure modes

### AC-22: Runtime Selection and Fallback Works (F19)
- [ ] `pi-box system doctor` reports available runtime backends
- [ ] Backend selection honors explicit `--mode` requests
- [ ] Auto-selection prefers an available compatible backend based on trust/config
- [ ] Secure-mode startup failure can fall back to compat mode when policy permits
- [ ] Fallback decisions are visible in logs/history

### AC-23: MicroVM Backend Works (F20)
- [ ] `pi-vmm-manager` can start and stop a microVM sandbox
- [ ] Firecracker or Cloud Hypervisor backend boots a tiny guest rootfs
- [ ] Template snapshot restore creates a ready workspace quickly
- [ ] Workspace disk persists sandbox changes for the session
- [ ] Artifact export works from microVM sandboxes
- [ ] Reseed-on-restore hook runs after snapshot restore
- [ ] Benchmarks include microVM mode comparison
- [ ] MicroVM backend reports unavailable when `/dev/kvm` or Firecracker is unavailable
- [ ] Guest rootfs is read-only
- [ ] Workspace disk is writable ext4
- [ ] Artifact export uses the guest control plane

### AC-24: MicroVM Guest Control Plane Works (F21)
- [ ] `pi-init` starts inside the guest and reports readiness
- [ ] `pi-agentd` communicates with the host over virtio-vsock
- [ ] Exec requests stream stdout/stderr over the guest control channel
- [ ] Guest lifecycle events map back to sandbox state
- [ ] File and artifact transfer work without direct host filesystem mounting
- [ ] Host and guest exchange newline-delimited JSON frames over virtio-vsock
- [ ] Exec stdout/stderr stream frames carry base64 payloads
- [ ] Final exec response includes exit code, duration, timeout, and truncation metadata
- [ ] Host marks sandbox warm only after the guest sends `ready`

### AC-25: Remote Daemon Contexts Work (F22)
- [ ] `pi-box context create workstation ssh://gpu-box.local` creates a remote context
- [ ] `pi-box context use workstation` switches the active context
- [ ] `pi-box context list` shows local and remote contexts
- [ ] `pi-box box create` uses the active context
- [ ] Commands can override the active context explicitly
- [ ] Contexts persist in `~/.pi-box/contexts.yaml`
- [ ] Context schema supports `target`, `transport`, and `auth.type`
- [ ] `--context <name>` overrides the active context

### AC-26: Remote Transport and Auth Work (F23)
- [ ] Remote daemon API calls are authenticated
- [ ] Remote access works over SSH-friendly transport
- [ ] Tailscale/WireGuard network paths are supported without API redesign
- [ ] Credentials are not persisted inside sandbox workspaces
- [ ] Remote workstation use case works end-to-end
- [ ] `http` remote contexts require bearer-token auth
- [ ] `ssh` remote contexts use SSH agent transport authentication
- [ ] Remote auth failures never fall back to unauthenticated access

### AC-27: Cross-Platform GUI Workbench Works (F24)
- [ ] GUI app starts on macOS, Windows, and Linux development hosts
- [ ] GUI can connect to a local daemon
- [ ] GUI can connect to a configured remote context
- [ ] GUI shows connected/disconnected state and daemon version
- [ ] GUI can create a sandbox session without shelling out for normal lifecycle operations
- [ ] GUI does not implement a separate sandbox lifecycle outside `pi-sandboxd`

### AC-28: GUI Workspace Authorization Works (F25)
- [ ] User must explicitly select a project folder before GUI-launched local workspace access
- [ ] Selected project folder is displayed before session creation
- [ ] Default workspace mode is `copy`
- [ ] `bind` mode requires explicit opt-in
- [ ] Allowed folders can be listed and removed from GUI settings
- [ ] Host home directory, SSH keys, cloud config, Kubernetes config, and Docker socket are not mounted by default

### AC-29: GUI Session Operations Work (F26)
- [ ] Dashboard lists recent and active sandbox sessions
- [ ] GUI can create, inspect, and destroy sessions
- [ ] GUI can run commands with streaming stdout/stderr
- [ ] GUI displays command history, logs, exit code, duration, timeout status, and truncation status
- [ ] GUI can display workspace diff and export patch
- [ ] GUI can list and pull artifacts
- [ ] GUI can create and rollback snapshots when the daemon reports snapshot support

### AC-30: GUI Settings and Diagnostics Work (F27)
- [ ] GUI can view and change active context
- [ ] GUI can set default template, runtime mode, and network mode preferences
- [ ] GUI displays runtime/backend availability from daemon diagnostics
- [ ] GUI exposes `pi-box system doctor` equivalent results
- [ ] GUI can export a support bundle containing daemon diagnostics, GUI logs, version metadata, and redacted configuration
- [ ] Daemon policy overrides conflicting GUI preferences

## 8. Security Model

### Filesystem Isolation
- Host home directory (`$HOME`) not mounted by default
- `/var/run/docker.sock` not mounted by default
- Host root (`/`) not mounted by default
- Cloud metadata credentials (169.254.169.254) not accessible
- SSH private keys not mounted by default
- Kubernetes/cloud config directories not mounted by default
- Workspace (`/workspace`) and artifacts (`/artifacts`) are read-write
- Root filesystem is read-only where possible
- `/tmp` is process-local

### Process Limits
- Maximum processes: 256 (configurable)
- Default timeout: 120s (configurable)
- Max output: 8MiB (configurable)
- cgroup v2 enforces CPU/memory/disk limits

### Network Isolation
- Default mode: `restricted`
- Domain allowlist enforced (github.com, registry.npmjs.org, pypi.org, files.pythonhosted.org, proxy.golang.org, crates.io, static.crates.io)
- Default deny: metadata endpoint, host localhost, private LANs, cluster-local ranges
- Domain-aware egress (not only IP-based) because registries use dynamic IPs

### Secrets Model
- Environment variables: deny-by-default
- SSH agent: opt-in only
- Git credentials: brokered (not dumped into environment)
- Long-term: per-secret `exposeTo` policy (e.g., github-token → git only, never to shell)

### Runtime Mode Security
- `fast`: Linux namespaces, cgroups, seccomp, Landlock — suitable for trusted local use
- `compat`: OCI containers with hardened defaults (no privileged, no host network, caps dropped, seccomp enabled)
- `secure`: gVisor for unknown/untrusted repos — may have syscall compatibility issues
- `microvm`: VM-grade isolation — snapshot-first, read-only rootfs, virtio-vsock control channel

## 9. Interface Contract

### Daemon API (Unix socket: `~/.pi-box/sandboxd.sock`, optional HTTP: `127.0.0.1:7777`)

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/sandboxes` | Create sandbox session |
| GET | `/v1/sandboxes` | List sandbox sessions |
| GET | `/v1/sandboxes/{id}` | Inspect sandbox state |
| DELETE | `/v1/sandboxes/{id}` | Destroy sandbox session |
| POST | `/v1/sandboxes/{id}/clone` | Clone repository into workspace |
| POST | `/v1/sandboxes/{id}/exec` | Execute command (streaming) |
| POST | `/v1/sandboxes/{id}/files/write` | Write file to workspace |
| GET | `/v1/sandboxes/{id}/files/read` | Read file from workspace |
| GET | `/v1/sandboxes/{id}/diff` | Get workspace diff |
| GET | `/v1/sandboxes/{id}/patch` | Get workspace patch |
| POST | `/v1/sandboxes/{id}/artifacts/export` | Export artifacts |
| POST | `/v1/sandboxes/{id}/snapshot` | Create snapshot |
| POST | `/v1/sandboxes/{id}/rollback` | Rollback to snapshot |
| GET | `/v1/sandboxes/{id}/logs` | Get command logs |

### Create Request
```json
{
  "template": "node-python",
  "mode": "fast",
  "ttlSeconds": 7200,
  "workspace": { "mode": "copy", "maxSize": "5Gi" },
  "resources": { "cpu": "2", "memory": "2Gi", "processes": 256 },
  "network": { "mode": "restricted", "allowDomains": ["github.com", "..."] }
}
```

### Exec Request
```json
{
  "command": "pnpm test",
  "cwd": "/workspace",
  "timeoutMs": 60000,
  "maxOutputBytes": 8388608,
  "network": "restricted"
}
```

### Exec Response
```json
{
  "exitCode": 0,
  "durationMs": 1842,
  "stdout": "...",
  "stderr": "...",
  "truncated": false,
  "timedOut": false
}
```

### CLI Commands (all map to API calls)
```
pi-box box create <name> [template] [flags]
pi-box box list
pi-box box inspect <name>
pi-box box clone <name> <url>
pi-box box exec <name> -- <cmd> [flags]
pi-box box shell <name>
pi-box box files list|read|write|pull|push <name> [args]
pi-box box diff <name>
pi-box box patch <name>
pi-box box artifacts list|pull|pack <name> [flags]
pi-box box snapshot <name> <action> [name]
pi-box box logs <name>
pi-box box destroy <name> [--all]
pi-box system status|doctor|prune|disk-usage
pi-box bench run [flags]
pi-box template list|inspect|build|update|prune [flags]
```

## 10. Dependencies

| Dependency | Type | Required For | Notes |
|-----------|------|-------------|-------|
| Linux kernel namespaces (user, mount, PID) | OS feature | F3 (Fast backend) | Linux-only; macOS/Windows require workaround |
| cgroup v2 | OS feature | F3, F17 | Linux-only |
| seccomp | OS feature | F3, F17 | Linux-only |
| Landlock | OS feature | F3 | Kernel ≥ 5.13, optional |
| runc / containerd / Podman | External tool | F4 (Compat backend) | Any OCI runtime |
| gVisor (runsc) | External tool | F16 (Secure backend) | Optional, later milestone |
| Firecracker / Cloud Hypervisor | External tool | F17 (MicroVM backend) | Optional, later milestone |
| Git | External tool | F6 (Clone) | Required for all backends |
| Template base images (Debian slim, etc.) | OCI images | F5 (Templates) | Pull-on-demand |

## 11. Out of Scope

The following are explicitly NOT part of the first release:

- Kubernetes controller or CRDs
- Multi-region control plane
- Full SaaS product
- Firecracker rewrite (phase 5 only uses it, doesn't rewrite it)
- Custom VMM implementation
- Docker-in-Docker as a default capability
- GPU workloads
- Browser/computer-use agents
- Full Docker Compose compatibility
- Complex enterprise identity management
- Background cloud scheduler
- Windows/macOS native backends (initial target: Linux-first; macOS uses Lima/Colima/Docker; Windows uses WSL2)
- Desktop mode (future)
- WASM/WASI tool sandboxing (future)

## 12. High-level architecture

```text
PI Agent / CLI / IDE / SDK
        |
        v
  pi-sandboxd local daemon
        |
        +-- session manager
        +-- runtime manager
        +-- template manager
        +-- workspace manager
        +-- cache manager
        +-- artifact manager
        +-- snapshot manager
        +-- policy manager
        +-- egress manager
        +-- benchmark/telemetry hooks
        |
        v
  selectable sandbox backends
        +-- fast: native process sandbox
        +-- compat: runc/container backend
        +-- secure: gVisor backend
        +-- isolated: Kata backend, later
        +-- microvm: Firecracker or Cloud Hypervisor backend, later
```

The CLI is thin. PI Agent, SDKs, IDE integrations, and the GUI workbench should talk directly to `pi-sandboxd` over a local Unix socket, localhost API, or the remote context transport.

The GUI workbench is a client surface, not a second runtime. It must not run sandbox workloads in the UI process or implement a separate sandbox lifecycle. Normal GUI operations use the daemon API or SDK. Shelling out to the `pi-box` CLI is reserved for diagnostics or compatibility gaps.

## 13. User-facing runtime modes

Expose simple profiles. Do not expose low-level runtime complexity to normal users.

| Mode | Backend | Purpose | First release |
|---|---|---|---|
| `fast` | Native Linux sandbox using namespaces, cgroups, seccomp, Landlock/bubblewrap/nsjail style isolation | Fastest and smallest local trusted mode | Yes |
| `compat` | runc/containerd/Podman style OCI container | Best compatibility for Node.js/Python/Go/Rust projects | Yes |
| `secure` | gVisor/runsc | Better isolation while keeping container-like UX | Later |
| `isolated` | Kata Containers | Stronger VM-grade enterprise isolation | Later |
| `microvm` | Firecracker or Cloud Hypervisor | Snapshot-first microVM mode | Later |
| `desktop` | Full VM/KubeVirt/Apple Virtualization/WSL2 backend | Later browser/computer-use agents; not required for the M7 GUI workbench | Future |
| `tool` | WASM/WASI | MCP/plugin tool sandboxing | Future |

Default mode should be `auto`:

```text
Linux:
  prefer fast or compat depending trust/config
  secure if gVisor installed and requested
  microvm if /dev/kvm and backend installed

macOS:
  use a Linux helper VM, Lima/Colima/Docker backend, or remote Linux daemon

Windows:
  use WSL2 backend first
```

The initial target is Linux-first.

## 14. Isolation strategy

### 7.1 Fast mode

Fast mode is for trusted local developer use.

Use Linux primitives:

- user namespaces
- mount namespaces
- PID namespaces
- cgroup v2
- seccomp
- Landlock where available
- restricted `/proc`
- read-only root where possible
- writable `/workspace`, `/tmp`, `/home/agent`, `/cache`, and `/artifacts`

This mode should have the lowest warm exec overhead.

### 7.2 Compat mode

Compat mode uses normal OCI containers through runc/containerd/Podman/Docker-compatible plumbing.

Use for:

- maximum language/tool compatibility
- repos that expect normal Linux behavior
- app builds and tests
- first portable backend

Hardening defaults:

- no privileged containers
- no host network
- no Docker socket
- no hostPath unless explicitly configured
- no default Kubernetes service account token when eventually used in Kubernetes
- drop Linux capabilities by default
- seccomp profile enabled
- AppArmor/SELinux where available

### 7.3 Secure mode

Secure mode uses gVisor.

Use for:

- unknown repos
- semi-untrusted generated code
- better-than-container isolation
- reasonable local and future multi-tenant defaults

Expected limitations:

- possible syscall compatibility issues
- slower syscall-heavy and filesystem-heavy workloads
- Docker-in-Docker and Testcontainers may fail or require fallback
- FUSE, ptrace, eBPF, kernel modules, privileged networking, and low-level debug tools are not reliable
- browser automation may need a special template

### 7.4 MicroVM mode

MicroVM mode should not be the MVP. It should come after the local daemon, CLI, and benchmarks are proven.

When implemented, build a PI-specific microVM platform layer rather than rewriting Firecracker first.

Preferred shape:

```text
pi-sandboxd
  -> pi-vmm-manager
      -> Firecracker or Cloud Hypervisor
          -> tiny Linux guest
              -> pi-init
              -> pi-agentd over virtio-vsock
```

MicroVM design goals:

- no SSH in the hot path
- no systemd in default templates
- control channel over virtio-vsock
- static guest agent
- read-only rootfs
- separate workspace, cache, and artifacts disks
- snapshot-first template restore
- explicit reseed-on-restore hook

First microVM implementation contract:

- Firecracker is the primary M5 backend.
- Cloud Hypervisor remains a later compatible backend behind the same runtime interface.
- MicroVM mode requires Linux with `/dev/kvm` and a supported host kernel.
- If `/dev/kvm` or Firecracker is unavailable, runtime selection reports microVM as unavailable; it does not silently fall back unless policy permits fallback.
- The guest rootfs is read-only.
- Each sandbox receives a writable ext4 workspace disk.
- Template restore starts from a read-only template snapshot plus a fresh writable workspace disk.
- The reseed-on-restore hook runs after the workspace disk is attached and before the guest reports ready.
- Artifact export copies data through the guest control plane, not by directly mounting host paths inside the guest.

MicroVM guest control protocol:

- The host and guest communicate over virtio-vsock using newline-delimited JSON control frames.
- Each frame has `type`, `id`, `session_id`, `method`, `payload`, and optional `error` fields.
- `type` is one of `request`, `response`, `event`, or `stream`.
- `method` is one of `hello`, `ready`, `exec`, `file.read`, `file.write`, `artifact.list`, `artifact.pull`, or `shutdown`.
- Exec streaming uses `stream` frames whose payload includes `stream: stdout|stderr` and `data: base64-bytes`.
- The final exec response includes `exit_code`, `duration_ms`, `timed_out`, and `truncated`.
- Guest readiness is explicit: `pi-init` starts `pi-agentd`, `pi-agentd` sends a `ready` event, and only then may the host mark the sandbox warm.

### 7.5 Runtime driver contract

*(Added per PROP-008, 2026-07-14)*

All isolation backends implement one internal lifecycle driver contract owned by `pkg/runtime`:

- A driver exposes: `Probe`, `Create`, `Start`, `Exec`, `Inspect`, `Stop`, `Destroy`, and `Stats`.
- Files, artifacts, logs, metadata, policy evaluation, and API semantics live **above** the driver layer. Drivers own only isolation, process creation, mounts, network attachment, resource controls, and termination.
- Workspace snapshots are runtime-independent; drivers never gate snapshot semantics.
- A sandbox handle carries the stable session ID and the driver-owned runtime object ID as **distinct** fields. The session ID is never mutated after creation.
- `Probe` returns a structured capability report (availability, missing prerequisites, per-capability flags, isolation tier, compatibility tier). Availability probes must actually execute; a probe that always succeeds is a defect. A runtime is never summarized by a single security integer.
- Compat and secure modes share one OCI engine layer (image ensure, create, start, exec, inspect, stop, remove) with pluggable engine implementations (Podman, Docker, later containerd). Secure mode is the same OCI lifecycle with a `runsc` runtime handler, not a parallel implementation.
- Compat/secure mount policy: `/workspace`, `/artifacts`, and `/cache` are `rw,nosuid,nodev` (exec allowed); `noexec` applies to `/tmp` and secret mounts only.
- Containers are not created with auto-remove; the daemon reconciles runtime state at startup and garbage-collects orphans.
- The project ships and versions its own seccomp profile and passes it explicitly to every OCI engine.
- All drivers honor a shared resource-limit model (memory, swap, CPUs, PIDs, ulimits).

Runtime selection separates four concepts that must never be collapsed into one ordered list:

1. requested mode (`fast|compat|secure|isolated|microvm|auto`)
2. workload trust (`trusted|reviewed|untrusted`)
3. discovered host capabilities
4. explicit fallback policy (allow/deny lists)

Selection never silently downgrades isolation below the requested mode. A denied fallback fails with actionable guidance derived from the capability report, and fallback decisions remain visible in logs and history.

## 15. Local filesystem layout

Use predictable state under `~/.pi-box` by default for host-side Pi sandbox runtime state.

Legacy `~/.pi` data is not automatically migrated or pruned by default; Pi Box leaves that directory untouched unless a future migration command is explicitly specified.

```text
~/.pi-box/
  config.yaml
  sandboxd.sock

  templates/
    base/
    node/
    python/
    go/
    rust/
    node-python/
    polyglot/

  sandboxes/
    <sandbox-id>/
      meta.json
      workspace/
      artifacts/
      logs/
      snapshots/
      upper/
      work/

  caches/
    npm/
    pnpm/
    pip/
    uv/
    go-build/
    go-mod/
    cargo/
    sccache/

  images/
    rootfs/
    kernels/
    initrds/
    microvm/
```

Users must be able to inspect and clean local state.

Required commands:

```bash
pi-box system status
pi-box system doctor
pi-box system prune
pi-box system disk-usage
```

## 16. Workspace model

Each sandbox session has:

```text
/workspace   repo checkout and editable files
/artifacts   build outputs, logs, reports, exported files
/cache       dependency caches
/tmp         temporary files
/home/agent  minimal home directory
```

The default workspace must be isolated from the host home directory.

Do not mount `$HOME` by default.

Support three workspace modes:

| Mode | Description |
|---|---|
| `copy` | Copy repo/files into sandbox workspace. Safest default. |
| `bind` | Bind mount explicit host directory. User must opt in. |
| `overlay` | Read-only base plus writable upperdir. Good for snapshots and rollback. |

GUI-launched sessions must preserve this model. The GUI must require explicit project-folder selection before local workspace access, display the selected folder before session creation, default to `copy`, and make `bind` an explicit per-session or per-folder opt-in. Folder picker grants are advisory UI state only; daemon policy remains authoritative.

## 17. Cache model

Dependency caches are first-class. Coding-agent latency is often dominated by package installation and builds.

Provide scoped cache mounts:

```text
/cache/npm
/cache/pnpm
/cache/yarn
/cache/pip
/cache/uv
/cache/go-mod
/cache/go-build
/cache/cargo
/cache/sccache
```

Policy:

- caches are not secrets
- caches should be scoped by template/runtime/user
- shared read-only cache plus per-session writable overlay is preferred later
- cache promotion must be explicit or validated
- cache pruning must be available

Recommended fast tools:

- Node.js: pnpm and corepack
- Python: uv and pip
- Go: GOMODCACHE and GOCACHE
- Rust: cargo cache and optional sccache

## 18. Templates

Templates define language/runtime environments.

Initial templates:

```text
base
node
python
go
rust
node-python
polyglot
```

Base template contents:

```text
bash or dash
busybox/coreutils
git
curl
ca-certificates
openssh-client
tar
gzip
zstd
unzip
jq
ripgrep
pi-agentd when needed
```

Node template:

```text
Node.js LTS
npm
pnpm
corepack
```

Python template:

```text
Python 3.x
uv
pip
venv support
```

Go template:

```text
Go stable toolchain
GOMODCACHE configured
GOCACHE configured
```

Rust template:

```text
rustc
cargo
rustup optional
sccache optional
```

Example template file:

```yaml
name: node-python
runtime: auto
base: debian-slim
tools:
  - git
  - curl
  - ripgrep
  - jq
  - node:22
  - pnpm
  - python:3.13
  - uv
mounts:
  workspace: /workspace
  artifacts: /artifacts
  caches:
    npm: /cache/npm
    pnpm: /cache/pnpm
    uv: /cache/uv
    pip: /cache/pip
network:
  default: restricted
```

Template commands:

```bash
pi-box template list
pi-box template inspect node-python
pi-box template build node-python
pi-box template update node-python
pi-box template prune
```

## 19. CLI requirements

The CLI binary should be named `pi-box`, with `box` as the sandbox subcommand.

### 12.1 Create sandbox

```bash
pi-box box create --name app1 --template node-python
pi-box box create node-python
pi-box box create --mode fast node-python
pi-box box create --mode compat rust
```

### 12.2 List and inspect

```bash
pi-box box list
pi-box box inspect app1
pi-box box status app1
```

### 12.3 Clone repository

```bash
pi-box box clone app1 https://github.com/acme/app
pi-box box clone app1 git@github.com:acme/app.git
```

SSH credentials must not be blindly mounted. Use a controlled Git credential helper or explicit user opt-in.

### 12.4 Execute command

```bash
pi-box box exec app1 -- pnpm install
pi-box box exec app1 -- pnpm test
pi-box box exec app1 -- python scripts/check.py
pi-box box exec app1 -- go test ./...
pi-box box exec app1 -- cargo test
```

Options:

```bash
--cwd /workspace
--timeout 60s
--max-output 8MiB
--memory 1Gi
--cpu 2
--network restricted
--json
```

### 12.5 Shell

```bash
pi-box box shell app1
```

Shell is for humans, not the agent hot path.

### 12.6 Files

```bash
pi-box box files list app1 /workspace
pi-box box files read app1 /workspace/package.json
pi-box box files write app1 /workspace/src/index.ts < index.ts
pi-box box files pull app1 /workspace/dist ./dist
pi-box box files push app1 ./README.md /workspace/README.md
```

### 12.7 Diff and patch

```bash
pi-box box diff app1
pi-box box patch app1 > changes.patch
pi-box box apply-patch app1 changes.patch
```

### 12.8 Artifacts

```bash
pi-box box artifacts list app1
pi-box box artifacts pull app1 ./artifacts
pi-box box artifacts pack app1 --output artifacts.tar.zst
```

### 12.9 Snapshots

```bash
pi-box box snapshot app1 before-refactor
pi-box box snapshots app1
pi-box box rollback app1 before-refactor
pi-box box snapshot delete app1 before-refactor
```

### 12.10 Run/serve apps

```bash
pi-box box exec app1 -- pnpm dev --host 0.0.0.0
pi-box box port-forward app1 3000:3000
pi-box box logs app1
```

### 12.11 Destroy

```bash
pi-box box destroy app1
pi-box box destroy --all
```

## 20. Daemon API

`pi-sandboxd` exposes a local API over Unix socket by default.

Default socket:

```text
~/.pi-box/sandboxd.sock
```

Optional localhost HTTP for development:

```text
127.0.0.1:7777
```

Required API resources:

```http
POST   /v1/sandboxes
GET    /v1/sandboxes
GET    /v1/sandboxes/{id}
DELETE /v1/sandboxes/{id}

POST   /v1/sandboxes/{id}/clone
POST   /v1/sandboxes/{id}/exec
POST   /v1/sandboxes/{id}/files/write
GET    /v1/sandboxes/{id}/files/read
GET    /v1/sandboxes/{id}/diff
GET    /v1/sandboxes/{id}/patch
POST   /v1/sandboxes/{id}/artifacts/export
POST   /v1/sandboxes/{id}/snapshot
POST   /v1/sandboxes/{id}/rollback
GET    /v1/sandboxes/{id}/logs
```

Exec endpoint must support streaming stdout, stderr, exit code, timeout status, and truncated output metadata.

Example create request:

```json
{
  "template": "node-python",
  "mode": "fast",
  "ttlSeconds": 7200,
  "workspace": {
    "mode": "copy",
    "maxSize": "5Gi"
  },
  "resources": {
    "cpu": "2",
    "memory": "2Gi",
    "processes": 256
  },
  "network": {
    "mode": "restricted",
    "allowDomains": [
      "github.com",
      "registry.npmjs.org",
      "pypi.org",
      "files.pythonhosted.org",
      "proxy.golang.org",
      "crates.io",
      "static.crates.io"
    ]
  }
}
```

Example exec request:

```json
{
  "command": "pnpm test",
  "cwd": "/workspace",
  "timeoutMs": 60000,
  "maxOutputBytes": 8388608,
  "network": "restricted"
}
```

Example exec response:

```json
{
  "exitCode": 0,
  "durationMs": 1842,
  "stdout": "...",
  "stderr": "...",
  "truncated": false,
  "timedOut": false
}
```

## 21. Agent SDK requirements

Provide TypeScript and Python SDKs.

TypeScript example:

```ts
const box = await client.sandboxes.create({
  template: "node-python",
  mode: "fast"
});

await box.clone("https://github.com/acme/app");
const result = await box.exec("pnpm test", { timeoutMs: 60000 });
const diff = await box.diff();
```

Python example:

```python
box = client.sandboxes.create(template="python", mode="fast")
box.clone("https://github.com/acme/app")
result = box.exec("uv run pytest -q", timeout_ms=60000)
diff = box.diff()
```

SDKs must support streaming output.

## 22. Security defaults

Default policy:

```yaml
defaults:
  filesystem:
    hostHomeMount: false
    workspace: read-write
    artifacts: read-write
    caches: scoped
    root: read-only-where-possible

  process:
    maxProcesses: 256
    defaultTimeout: 120s
    maxOutput: 8MiB

  network:
    mode: restricted
    deny:
      - 169.254.169.254
      - host-localhost
      - private-lans
      - cluster-local
    allow:
      - github.com
      - registry.npmjs.org
      - pypi.org
      - files.pythonhosted.org
      - proxy.golang.org
      - crates.io
      - static.crates.io

  secrets:
    env: deny-by-default
    sshAgent: opt-in
    gitCredentials: brokered
```

Never mount these by default:

- `/var/run/docker.sock`
- host `/`
- host `$HOME`
- cloud metadata credentials
- SSH private keys
- Kubernetes config
- cloud provider config directories

## 23. Secrets model

Do not dump secrets into sandbox environment by default.

Use a secret broker later.

Initial Git support options:

1. Public HTTPS clone.
2. User-approved SSH agent forwarding for Git only.
3. User-approved token credential helper scoped to Git operations.

Long-term model:

```yaml
secrets:
  github-token:
    exposeTo:
      - git
      - gh
    neverExposeToShell: true
```

## 24. Network model

The network must be configurable per sandbox and per exec.

Modes:

| Mode | Description |
|---|---|
| `none` | No outbound network. |
| `restricted` | Domain allowlist through egress proxy. Default. |
| `open` | Full outbound access. User must opt in. |

Default deny targets:

```text
169.254.169.254
host gateway IP
host localhost
private LANs unless allowed
local Kubernetes ranges when applicable
```

Domain-aware egress is preferred over only IP-based filtering because package registries and Git hosts use dynamic IPs.

## 25. Snapshot and rollback

Initial implementation can use filesystem-level snapshots:

- overlay upperdir copy
- reflink copy when supported
- btrfs/zfs later
- tar/zstd fallback

Commands:

```bash
pi-box box snapshot app1 before-change
pi-box box rollback app1 before-change
```

MicroVM mode later should support template snapshots.

Important microVM restore hook:

```text
on_restore:
  reseed entropy
  regenerate machine-id if present
  reset network identity
  clear temporary secrets
  notify pi-agentd
```

## 26. Artifact model

Artifacts are files intentionally exported from the sandbox.

Default artifact locations:

```text
/artifacts
/workspace/dist
/workspace/build
/workspace/coverage
/workspace/test-results
/workspace/target/release
```

Artifact commands:

```bash
pi-box box artifacts list app1
pi-box box artifacts pull app1 ./artifacts
pi-box box artifacts pack app1 --output artifacts.tar.zst
```

Artifact export should avoid copying the whole workspace unless requested.

## 27. Logs and telemetry

Each sandbox should produce:

- command history
- stdout/stderr logs
- exit codes
- duration
- timeout status
- output truncation status
- resource usage when available
- artifact manifest
- snapshot history

CLI:

```bash
pi-box box logs app1
pi-box box history app1
pi-box box metrics app1
```

Telemetry must be local-only by default.

## 28. Benchmarks

Benchmark suite is mandatory.

Command:

```bash
pi-box bench run
pi-box bench run --mode fast
pi-box bench run --mode compat
```

Required benchmarks:

| Benchmark | Purpose |
|---|---|
| `warm_exec_echo` | Measures hot exec overhead. |
| `warm_exec_shell` | Measures shell command startup overhead. |
| `file_scan_rg` | Measures filesystem scan overhead. |
| `git_clone_small` | Measures network and Git path. |
| `pnpm_install_cached` | Measures Node dependency cache path. |
| `uv_sync_cached` | Measures Python dependency cache path. |
| `go_test_cached` | Measures Go toolchain and cache. |
| `cargo_test_cached` | Measures Rust toolchain and cache. |
| `snapshot_create` | Measures snapshot creation. |
| `snapshot_rollback` | Measures rollback. |
| `artifact_export_20mb` | Measures artifact packing/export. |
| `parallel_10` | Measures 10 concurrent sandboxes. |
| `parallel_100` | Measures high-density behavior where hardware allows. |

Example output:

```text
mode=fast template=node-python
warm_exec_echo_p50=5.2ms
warm_exec_echo_p95=9.8ms
pnpm_install_cached_p50=1.8s
artifact_export_20mb_p50=240ms
idle_memory_per_sandbox=38MiB

mode=compat template=node-python
warm_exec_echo_p50=24.1ms
warm_exec_echo_p95=51.6ms
pnpm_install_cached_p50=2.1s
idle_memory_per_sandbox=72MiB
```

Initial target metrics:

| Metric | Target |
|---|---:|
| Fast mode warm exec overhead | single-digit ms p50 |
| Compat mode warm exec overhead | tens of ms p50 |
| New warm session assignment | under 100 ms |
| Artifact export 20 MB | under 500 ms local |
| Idle fast sandbox memory | as small as practical, target under 64 MiB where possible |

## 29. Implementation language

Recommended implementation:

- Go for initial daemon, CLI, runtime integrations, and APIs.
- Rust for guest agent, microVM manager, hot-path supervisor, or security-critical low-level pieces later.

Alternative:

- Rust end-to-end is acceptable if the team prefers it.

Practical recommendation:

```text
Phase 1:
  Go CLI + daemon + native/runc backends

Phase 2:
  Rust pi-agentd or exec-supervisor if benchmarks justify it

Phase 3:
  Rust pi-vmm-manager for microVM backend
```

Do not rewrite Firecracker in phase 1.

## 30. Repository structure

Preferred repo layout:

```text
pibox/
  cmd/
    pi/
    pi-sandboxd/
    pi-agentd/

  pkg/ or crates/
    api/
    cli/
    daemon/
    runtime/
      native/
      runc/
      gvisor/
      kata/
      microvm/
    workspace/
    snapshot/
    cache/
    artifacts/
    policy/
    net/
    exec/
    telemetry/

  templates/
    base/
    node/
    python/
    node-python/
    go/
    rust/
    polyglot/

  sdk/
    typescript/
    python/

  integrations/
    mcp/
    pi-agent/
    openhands/
    aider/

  benchmarks/
    warm-exec/
    git-clone/
    pnpm-install/
    uv-sync/
    go-test/
    cargo-test/
    artifact-export/

  docs/
    architecture.md
    isolation.md
    templates.md
    security.md
    benchmarks.md
```

## 31. Milestones

### Milestone 1: Local Linux MVP

Deliver:

- `pi-box` CLI
- `pi-sandboxd` daemon
- local Unix socket API
- `fast` backend prototype
- `compat` backend prototype
- `base`, `node-python`, `go`, and `rust` templates
- create/list/inspect/destroy
- clone
- exec with streaming output
- file read/write
- diff/patch
- artifacts export
- basic logs
- basic benchmark suite

Definition of done:

```bash
pi-box box create --name demo --template node-python --mode fast
pi-box box clone demo https://github.com/some/repo
pi-box box exec demo -- pnpm install
pi-box box exec demo -- pnpm test
pi-box box diff demo
pi-box box artifacts pull demo ./out
pi-box box destroy demo
```

### Milestone 2: Hardening and cache performance

Deliver:

- scoped dependency caches
- cgroup v2 limits
- output limits
- timeout enforcement
- process limits
- network modes: none/restricted/open
- egress allowlist prototype
- snapshot/rollback filesystem implementation
- `pi-box system doctor`
- benchmark dashboard or JSON output

### Milestone 3: Agent integrations

Deliver:

- TypeScript SDK
- Python SDK
- MCP server
- PI Agent adapter
- example coding-agent loop
- JSON exec mode
- structured command history

### Milestone 4: Secure backend

Deliver:

- gVisor backend support
- runtime detection
- fallback to compat mode
- compatibility documentation
- benchmark comparison fast vs compat vs secure

### Milestone 5: MicroVM backend

Deliver:

- `pi-vmm-manager`
- Firecracker or Cloud Hypervisor backend
- tiny guest rootfs
- `pi-init`
- `pi-agentd` over vsock
- template snapshot restore
- workspace disk
- artifact export
- reseed-on-restore hook
- benchmark comparison

### Milestone 6: Remote daemon mode

Deliver:

- CLI contexts
- SSH/Tailscale/WireGuard-friendly remote daemon access
- secure local-to-remote API auth
- remote workstation use case

Example:

```bash
pi-box context create workstation ssh://gpu-box.local
pi-box context use workstation
pi-box box create node-python
```

Remote daemon context contract:

- Context state is stored in `~/.pi-box/contexts.yaml`.
- Required context fields are `target`, `transport`, and `auth.type`.
- `target` is the daemon endpoint URI.
- `transport` is one of `unix`, `http`, or `ssh`.
- `auth.type` is one of `none`, `bearer-token`, or `ssh-agent`.
- The active context may be overridden per command with `--context <name>`.

Supported remote transports:

- `unix`: local Unix socket.
- `http`: direct HTTP endpoint, intended for private networks such as Tailscale or WireGuard.
- `ssh`: SSH-forwarded daemon access.

Authentication rules:

- `unix` contexts may use `auth.type: none`.
- `http` contexts require bearer-token auth.
- `ssh` contexts use SSH agent authentication for the transport.
- Bearer tokens are stored outside sandbox workspaces and are never injected into sandbox environment variables.
- Remote auth failures return actionable errors and do not fall back to unauthenticated access.

### Milestone 7: Cross-platform GUI workbench

Deliver:

- desktop GUI app for macOS, Windows, and Linux
- onboarding for local daemon, remote context, or connect later
- explicit workspace authorization and allowed folder management
- dashboard for recent and active sandbox sessions
- session detail with command runner, streaming logs, history, diff, patch export, artifacts, snapshots, rollback, and destroy
- settings for active context, daemon connection, default template, runtime mode, network mode, allowed folders, and diagnostics
- support bundle export with redacted configuration

The GUI workbench should use a restrained desktop-tool interface:

- left navigation: Dashboard, Sessions, Templates, Contexts, Policies, Settings
- status area: active daemon/context, connection state, runtime availability, and current version
- primary action: create a sandbox session from a project folder, repository URL, or template
- centered onboarding and authorization flows inspired by the accepted PROP-004 screenshots, with PI-specific text and workflows

Recommended first implementation stack:

- Tauri or another small cross-platform desktop shell
- TypeScript frontend
- Rust or native host bridge only for OS integration such as file picking, tray/menu integration, and local process supervision

The GUI should consume existing daemon API operations wherever possible. If these endpoints are missing when F24-F27 begin, they must be specified before implementation:

- list templates and inspect template metadata
- report runtime/backend availability
- report daemon version and health
- return session list with lifecycle state, template, mode, workspace source, created time, TTL, and last command summary
- stream exec output for a selected session
- return command history and logs
- return diff, patch, artifact, snapshot, and storage usage metadata

## 32. Compatibility expectations

The platform must support common commands:

Node.js:

```bash
npm install
pnpm install
pnpm test
pnpm build
npm run dev
```

Python:

```bash
pip install -r requirements.txt
uv sync
uv run pytest
python script.py
```

Go:

```bash
go mod download
go test ./...
go build ./...
```

Rust:

```bash
cargo fetch
cargo test
cargo build
```

Known difficult workflows:

- Docker Compose
- Testcontainers
- Docker-in-Docker
- kind/minikube inside sandbox
- FUSE
- eBPF
- kernel modules
- privileged networking
- VPNs
- packet capture
- advanced debuggers/profilers
- Playwright/Chrome edge cases

Provide fallback guidance:

```text
If secure mode fails, try compat mode.
If app needs services, use platform-managed side services later instead of Docker-in-Docker.
If browser automation fails, use a browser-specific template or future desktop mode.
```

## 33. Service dependencies later

Do not make Docker Compose inside the sandbox the default answer.

Future platform-managed services:

```yaml
services:
  postgres:
    image: postgres:17
  redis:
    image: redis:8
```

The platform should run side services next to the sandbox, expose them on a private network, and inject connection strings into the sandbox through the broker.

## 34. Configuration file

Default config:

```yaml
contexts:
  active: local
  entries:
    local:
      target: unix://~/.pi-box/sandboxd.sock
      transport: unix
      auth:
        type: none

runtime:
  defaultMode: auto
  allowModes:
    - fast
    - compat

storage:
  root: ~/.pi-box
  maxTotalSize: 100Gi

network:
  defaultMode: restricted

security:
  mountHomeByDefault: false
  allowDockerSocket: false
  allowPrivileged: false

exec:
  defaultTimeout: 120s
  maxOutputBytes: 8388608
  maxProcesses: 256

cache:
  enabled: true
  maxSize: 50Gi

gui:
  rememberLastConnection: true
  activeContext: local
  allowedFolders:
    - path: /Users/example/project
      defaultWorkspaceMode: copy
  defaults:
    template: node-python
    mode: auto
    network: restricted
```

GUI preferences are user-level client preferences, separate from daemon policy. If GUI preferences conflict with daemon policy, daemon policy wins and the GUI displays the policy error.

## 35. Error handling requirements

Errors must be actionable.

Bad:

```text
exec failed
```

Good:

```text
Command timed out after 60s.
Sandbox: app1
Command: pnpm test
Last output saved to: ~/.pi-box/sandboxes/app1/logs/exec-42.log
Try: pi-box box logs app1 --exec 42
```

Runtime fallback errors must explain next steps:

```text
gVisor backend is not installed.
Available modes: fast, compat
Install docs: pi-box docs gvisor
Try: pi-box box create --mode compat node-python
```

## 36. Documentation requirements

Must include:

- Quickstart
- Architecture
- CLI reference
- Template authoring
- Runtime modes and isolation tradeoffs
- Security model
- Cache model
- Artifact model
- Agent API
- SDK docs
- Benchmarks
- Troubleshooting
- Known incompatibilities

## 37. Success criteria

The project is successful when:

1. A developer can install it locally and run a real coding-agent loop in under 10 minutes.
2. PI Agent can create a sandbox, clone a repo, run tests, modify files, and export a patch without direct host filesystem access.
3. Warm exec is fast enough to feel interactive.
4. Node.js, Python, Go, and Rust templates work with common projects.
5. The benchmark suite clearly compares fast, compat, secure, and later microVM modes.
6. Developers understand where files, caches, logs, artifacts, and snapshots live.
7. Isolation is selectable per sandbox session.
8. The project can later grow into remote workstation, microVM, and Kubernetes-backed modes without redesigning the API.

## 38. Product north star

The north star is not simply running containers or microVMs.

The north star is:

> A coding agent can get a warm, isolated, language-ready workspace almost instantly, run hundreds of commands with low overhead, safely manage code changes, and export only the useful diff and artifacts.

## 39. First coding-agent task list

A coding agent implementing this project should start with the following tasks in order:

1. Create the repository skeleton.
2. Implement `pi-box` CLI with `box` subcommands using stubbed daemon calls.
3. Implement `pi-sandboxd` local daemon with Unix socket API.
4. Implement sandbox metadata store under `~/.pi-box/sandboxes`.
5. Implement `box create/list/inspect/destroy` for a simple local directory-backed sandbox.
6. Implement `box exec` in dev mode.
7. Add fast backend isolation incrementally: cwd isolation, env isolation, process limits, then namespaces/cgroups/seccomp.
8. Implement workspace directory and artifact directory.
9. Implement clone command.
10. Implement diff and patch using Git when available.
11. Implement logs and command history.
12. Implement basic templates as metadata.
13. Implement compat backend using a local container runtime.
14. Implement cache mount plumbing.
15. Implement snapshot/rollback with filesystem copy or reflink.
16. Implement benchmark suite.
17. Add TypeScript SDK.
18. Add Python SDK.
19. Add MCP server integration.
20. Add gVisor backend after MVP.

## 40. Important engineering constraints

- Keep the hot path simple.
- Avoid SSH for agent exec.
- Avoid Docker socket mounts.
- Avoid host home mounts.
- Avoid Kubernetes assumptions.
- Avoid cloud assumptions.
- Prefer local Unix socket API.
- Prefer explicit user opt-in for risky features.
- Measure before optimizing.
- Do not rewrite Firecracker until benchmarks prove the need.
