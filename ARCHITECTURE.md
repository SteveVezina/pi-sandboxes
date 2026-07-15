# PI Agent Sandbox Runtime — Architecture

This document describes the architecture of the PI Agent Sandbox Runtime, a local-first sandbox system for AI coding agents.

> **Reference:** See [SPEC.md](./SPEC.md) for requirements, [docs/features/](./docs/features/) for feature specs, and [docs/decisions/](./docs/decisions/) for architecture decision records.

## 1. Design Principles

The architecture is shaped by these invariants:

| Principle | Implication |
|-----------|-------------|
| **Fix the root cause** | Never add compensating code for agent/tool issues; fix at the source |
| **Spec-first development** | Code follows spec, never the other way around |
| **Local-first** | Starts as developer tooling; Kubernetes/remote comes later |
| **Keep sessions warm** | Do not create/destroy per tool call; reuse sessions |
| **Selectable isolation** | Offer multiple runtime modes; don't force one |
| **Security by default** | No host mounts, no Docker socket, no cloud metadata by default |
| **Benchmark-driven** | Measure everything from day one |

## 2. High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                         End User / Agent                            │
│  CLI (pi)  │  GUI Workbench  │  SDK (TypeScript / Python)          │
└──────┬──────────────────────────┬───────────────────────┬───────────┘
       │                          │                       │
       │       HTTP / WebSocket   │       HTTP            │
       │       Unix socket        │       Unix socket     │
       ▼                          ▼                       ▼
┌─────────────────────────────────────────────────────────────────────┐
│                        pi-sandboxd (Daemon)                         │
│                                                                     │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐              │
│  │ Session Mgr │  │ Template Eng │  │  Policy Eng   │              │
│  └──────┬──────┘  └──────┬───────┘  └──────┬────────┘              │
│         │                 │                  │                       │
│  ┌──────┴──────┐  ┌──────┴───────┐  ┌──────┴────────┐              │
│  │  Exec Mgr   │  │ Snapshot Mgr │  │  Secrets Mgr  │              │
│  └──────┬──────┘  └──────────────┘  └───────────────┘              │
│         │                 │                  │                       │
│  ┌──────┴──────┐  ┌──────┴───────┐  ┌──────┴────────┐              │
│  │ Workspace   │  │   Cache Mgr  │  │   Network Mgr │              │
│  └─────────────┘  └──────────────┘  └───────────────┘              │
│                                                                     │
│  ┌─────────────┐  ┌──────────────┐  ┌───────────────┐              │
│  │  Artifact   │  │   Remote     │  │    MCP        │              │
│  │  Exporter   │  │   Contexts   │  │    Protocol   │              │
│  └─────────────┘  └──────────────┘  └───────────────┘              │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐    │
│  │                    Runtime Dispatcher                       │    │
│  │  ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐ ┌────────┐  │    │
│  │  │ Fast   │ │ Compat │ │ Secure │ │ Isolated││ MicroVM │  │    │
│  │  │(ns)    │ │(OCI)   │ │(gVisor)│ │(Kata)   │ │(FC/CH) │  │    │
│  │  └────────┘ └────────┘ └────────┘ └────────┘ └────────┘  │    │
│  └─────────────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────────────┘
```

## 3. Component Details

### 3.1 CLI (`cmd/pi/`)

The command-line interface is the primary entry point for users and agents.

```
cmd/pi/
├── main.go              # Cobra root command
├── box/                 # Sandbox lifecycle subcommands
│   ├── create.go        # pi box create
│   ├── clone.go         # pi box clone
│   ├── exec.go          # pi box exec
│   ├── diff.go          # pi box diff
│   ├── patch.go         # pi box patch
│   ├── artifacts.go     # pi box artifacts (list/pull/pack)
│   ├── logs.go          # pi box logs
│   ├── history.go       # pi box history
│   ├── snapshot.go      # pi box snapshot/rollback
│   └── destroy.go       # pi box destroy
├── system/              # System management subcommands
│   ├── status.go        # pi system status
│   ├── doctor.go        # pi system doctor
│   ├── prune.go         # pi system prune
│   └── disk-usage.go    # pi system disk-usage
├── context/             # Remote context management
│   ├── create.go        # pi context create
│   ├── use.go           # pi context use
│   ├── list.go          # pi context list
│   ├── inspect.go       # pi context inspect
│   └── delete.go        # pi context delete
├── bench/               # Benchmark runner
│   └── run.go           # pi bench run
├── cache/               # Cache management
│   └── list.go          # pi cache list
└── shell/               # Interactive shell
    └── shell.go         # pi shell
```

**Key design:** The CLI talks to the daemon via HTTP over a Unix socket (`~/.pi/sandboxd.sock`). All business logic lives in the daemon; the CLI is a thin client.

### 3.2 Daemon (`cmd/pi-sandboxd/`)

The sandbox daemon is the core service. It manages session lifecycle, runtime dispatch, and all sandbox operations.

```
cmd/pi-sandboxd/
├── main.go              # Daemon entry point
└── server.go            # HTTP server setup
```

**Startup flow:**
1. Create `~/.pi/sandboxes/` and `~/.pi/templates/` directories
2. Load templates from `~/.pi/templates/`
3. Start HTTP server on Unix socket
4. Listen for API requests

### 3.3 Session Manager (`pkg/session/`)

Manages the lifecycle of sandbox sessions: create, list, inspect, destroy, TTL expiration.

```
pkg/session/
├── manager.go           # Session CRUD, warm reuse
├── ttl.go               # TTL expiration handling
├── store.go             # Persistent session state
└── types.go             # Session, State, Config types
```

**Key invariant:** Sessions are created once and kept warm. The runtime dispatcher maintains the session handle across exec calls.

### 3.4 Runtime Dispatcher (`pkg/runtime/`)

Routes session operations to the appropriate backend based on the selected mode.

```
pkg/runtime/
├── dispatcher.go        # Mode-based routing
├── fast/                # Linux namespace backend
│   ├── backend.go       # Namespace/cgroup/seccomp setup
│   ├── exec.go          # Command execution
│   └── cleanup.go       # Namespace teardown
├── compat/              # OCI container backend
│   ├── backend.go       # runc/containerd/Podman setup
│   ├── exec.go          # Container exec
│   └── cleanup.go       # Container teardown
├── secure/              # gVisor backend
│   ├── backend.go       # runsc setup
│   ├── exec.go          # Secure exec
│   └── cleanup.go       # Cleanup
├── isolated/            # Kata Containers backend
│   ├── backend.go       # Kata setup
│   ├── exec.go          # VM exec
│   └── cleanup.go       # Cleanup
├── microvm/             # Firecracker / Cloud-Hypervisor backend
│   ├── backend.go       # VMM setup
│   ├── exec.go          # Guest exec via vsock
│   └── cleanup.go       # VM teardown
└── fallback.go          # Runtime detection & fallback logic
```

**Fallback chain:** `fast → compat → secure → isolated → microvm`. If the preferred mode is unavailable, the dispatcher tries the next mode.

### 3.5 Command Executor (`pkg/exec/`)

Handles streaming command execution with timeout, output limits, and metadata collection.

```
pkg/exec/
├── executor.go          # Core exec engine
├── stream.go            # Stdout/stderr streaming
├── timeout.go           # Timeout handling
├── limits.go            # Output size limits (maxOutput)
└── types.go             # ExecRequest, ExecResult, ExitCode
```

**Key features:**
- Streaming stdout/stderr via WebSocket or HTTP
- Configurable timeout (default: 120s)
- Output truncation at maxOutput (default: 8MiB)
- Exit code tracking
- Duration measurement

### 3.6 Workspace Manager (`pkg/workspace/`)

Handles file operations inside sandbox sessions.

```
pkg/workspace/
├── workspace.go         # Clone, read, write, diff, patch
├── git.go               # Git operations
└── types.go             # FileOp, Diff, Patch types
```

**Key operations:**
- `Clone(repo)` — Clone repository into workspace
- `ReadFile(id, path)` — Read file from sandbox
- `WriteFile(id, path, content)` — Write file to sandbox
- `Diff(id)` — Show workspace diff
- `Patch(id)` — Export workspace as patch

### 3.7 Template System (`pkg/template/`)

Declarative template definition for language/toolchain environments.

```
pkg/template/
├── template.go          # Template CRUD
├── builder.go           # Template building from OCI images
├── list.go              # Template listing
└── types.go             # Template, Toolchain, CacheMount types
```

**Built-in templates:** `base`, `node`, `python`, `go`, `rust`, `node-python`, `polyglot`

### 3.8 Policy Engine (`pkg/policy/`)

Enforces security policies on sandbox creation and execution.

```
pkg/policy/
├── policy.go            # Default security policy
├── network.go           # Network mode enforcement
├── mounts.go            # Mount policy enforcement
├── limits.go            # Process/output limits
└── types.go             # Policy, NetworkMode, MountRule types
```

**Default deny policy:**
- No host home directory mount
- No Docker socket mount
- No cloud metadata access (169.254.169.254)
- No SSH private key mount
- Git credentials brokered (not dumped into environment)

### 3.9 Snapshot Manager (`pkg/snapshot/`)

Filesystem-level snapshot creation and rollback.

```
pkg/snapshot/
├── snapshot.go          # Create, list, rollback
├── overlay.go           # Overlay upperdir implementation
├── reflink.go           # Copy-on-write reflink implementation
└── types.go             # Snapshot, SnapshotMeta types
```

**Storage:** `~/.pi/sandboxes/<id>/snapshots/`

### 3.10 Cache Manager (`pkg/cache/`)

Scoped dependency cache mounts to avoid redundant downloads.

```
pkg/cache/
├── cache.go             # Cache mount management
├── prune.go             # Cache pruning
└── types.go             # CacheEntry, CacheScope types
```

**Scoped caches:** npm, pnpm, pip, uv, go-mod, go-build, cargo

### 3.11 Secrets Manager (`pkg/secrets/`)

Secure secret broker for Git credentials and other sensitive data.

```
pkg/secrets/
├── secrets.go           # Secret store & retrieval
├── broker.go            # Git credential broker
└── types.go             # Secret, SecretStore types
```

### 3.12 Network Manager (`pkg/network/`)

Configurable network modes: none, restricted, open.

```
pkg/network/
├── network.go           # Network mode enforcement
├── allowlist.go         # Domain allowlist
└── types.go             # NetworkMode, DomainRule types
```

### 3.13 Artifact Exporter (`pkg/artifacts/`)

List, pull, and pack artifacts from sandbox sessions.

```
pkg/artifacts/
├── artifacts.go         # List, pull, pack
└── types.go             # Artifact, ArtifactPack types
```

### 3.14 Remote Daemon (`pkg/remote/`)

Remote daemon contexts and transport/authentication.

```
pkg/remote/
├── context.go           # Context management (local/remote)
├── transport.go         # SSH/Tailscale/WireGuard transport
├── auth.go              # Secure local-to-remote authentication
└── types.go             # Context, RemoteConfig types
```

### 3.15 MCP Protocol (`pkg/mcp/`)

Model Context Protocol support for agent integration.

```
pkg/mcp/
├── mcp.go               # MCP server implementation
└── types.go             # MCP message types
```

### 3.16 GUI Workbench (`pkg/gui/`)

Cross-platform desktop application for macOS, Windows, and Linux.

```
pkg/gui/
├── gui.go               # GUI workbench orchestration
├── workspace_auth.go    # Explicit project-folder selection
├── session_ops.go       # Dashboard and session views
└── settings.go          # Settings and diagnostics
```

## 4. Data Flow

### 4.1 Sandbox Creation

```
CLI/SDK/GUI
    │
    │ POST /v1/sandboxes {template, mode, config}
    ▼
Daemon API Handler
    │
    │ 1. Validate policy
    │ 2. Resolve template
    │ 3. Select runtime backend
    ▼
Runtime Dispatcher
    │
    │ Backend-specific setup
    │ (namespaces, OCI container, gVisor, Kata, MicroVM)
    ▼
Session Manager
    │
    │ Store session state
    │ Start TTL timer
    ▼
Sandbox session created
    │
    │ Session ID returned to caller
```

### 4.2 Command Execution

```
CLI/SDK/GUI
    │
    │ POST /v1/sandboxes/{id}/exec {command, cwd, timeout}
    ▼
Daemon API Handler
    │
    │ 1. Validate session exists
    │ 2. Validate policy (limits, network)
    ▼
Command Executor
    │
    │ Stream stdout/stderr
    │ Enforce timeout
    │ Enforce output limits
    ▼
Runtime Backend
    │
    │ Execute in sandbox context
    ▼
Session logs stored
    │
    │ ExecResult {exitCode, duration, truncated, stdout, stderr}
```

### 4.3 Snapshot & Rollback

```
CLI/SDK/GUI
    │
    │ POST /v1/sandboxes/{id}/snapshots
    ▼
Snapshot Manager
    │
    │ Backend-specific snapshot
    │ (overlay upperdir / reflink / VM snapshot)
    ▼
Snapshot stored at ~/.pi/sandboxes/{id}/snapshots/{name}/
```

## 5. API Surface

### 5.1 Session Endpoints

| Method | Path | Description |
|--------|------|-------------|
| POST | `/v1/sandboxes` | Create sandbox |
| GET | `/v1/sandboxes` | List sandboxes |
| GET | `/v1/sandboxes/{id}` | Inspect sandbox |
| DELETE | `/v1/sandboxes/{id}` | Destroy sandbox |
| POST | `/v1/sandboxes/{id}/clone` | Clone repository |
| POST | `/v1/sandboxes/{id}/exec` | Execute command |
| GET | `/v1/sandboxes/{id}/logs` | Get command logs |
| GET | `/v1/sandboxes/{id}/history` | Get command history |
| POST | `/v1/sandboxes/{id}/diff` | Get workspace diff |
| POST | `/v1/sandboxes/{id}/patch` | Export workspace patch |
| GET | `/v1/sandboxes/{id}/artifacts` | List artifacts |
| POST | `/v1/sandboxes/{id}/artifacts/pull` | Pull artifacts |
| POST | `/v1/sandboxes/{id}/artifacts/pack` | Pack artifacts |
| POST | `/v1/sandboxes/{id}/snapshots` | Create snapshot |
| POST | `/v1/sandboxes/{id}/rollback` | Rollback to snapshot |

### 5.2 System Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/system/status` | Daemon & sandbox status |
| GET | `/v1/system/doctor` | Configuration validation |
| POST | `/v1/system/prune` | Clean old state |
| GET | `/v1/system/disk-usage` | Storage breakdown |

### 5.3 Template Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/v1/templates` | List templates |
| GET | `/v1/templates/{name}` | Inspect template |

### 5.4 WebSocket Endpoints

| Path | Description |
|------|-------------|
| `/v1/sandboxes/{id}/exec/ws` | Streaming exec output |
| `/v1/sandboxes/{id}/logs/ws` | Live log streaming |

## 6. Security Model

### 6.1 Default Deny

| Resource | Default | Rationale |
|----------|---------|-----------|
| Host home directory | Not mounted | Prevent agent from reading user files |
| Docker socket | Not mounted | Prevent container escape |
| Cloud metadata (169.254.169.254) | Blocked | Prevent credential theft |
| SSH private keys | Not mounted | Prevent key exposure |
| Git credentials | Brokered | Credentials injected at exec time, not in env |

### 6.2 Process Limits

| Limit | Default | Configurable |
|-------|---------|--------------|
| Max processes | 256 | Yes |
| Exec timeout | 120s | Yes |
| Max output | 8MiB | Yes |
| CPU limit | No default | Yes |
| Memory limit | No default | Yes |

### 6.3 Network Modes

| Mode | Description | Use case |
|------|-------------|----------|
| `none` | All outbound blocked | Unknown repos, strict isolation |
| `restricted` | Domain allowlist | Production workloads |
| `open` | Full outbound access | Trusted repos, development |

## 7. Runtime Modes Comparison

| Mode | Backend | Isolation | Speed | Use case |
|------|---------|-----------|-------|----------|
| `fast` | Linux namespaces | Process-level | Fastest | Trusted repos, local dev |
| `compat` | OCI container | Container-level | Fast | Maximum compatibility |
| `secure` | gVisor | Syscall re-implementation | Moderate | Unknown/untrusted repos |
| `isolated` | Kata Containers | VM-level | Slower | High isolation needed |
| `microvm` | Firecracker / CH | MicroVM-level | Slowest | Maximum isolation, multi-tenant |

## 8. Persistence Layout

```
~/.pi/
├── sandboxd.sock          # Daemon socket
├── sandboxes/
│   └── {session-id}/
│       ├── session.json   # Session state
│       ├── workspace/     # Repo checkout
│       ├── artifacts/     # Build outputs
│       ├── cache/         # Dependency caches
│       ├── snapshots/     # Point-in-time copies
│       └── logs/          # Command history
├── templates/
│   └── {template-name}/   # Template definitions
├── caches/
│   └── {scope}/           # Scoped dependency caches
└── contexts.json          # Remote context config
```

## 9. Cross-Cutting Concerns

### 9.1 Traces

Every operation carries tracing context: `workspace_id`, `actor_id`, `run_id`, `session_id`.

### 9.2 Lifecycle Events

The daemon emits service-level lifecycle events:

| Event | When |
|-------|------|
| `pi.sandbox.created` | On pod/session creation |
| `pi.run.started` | From Pi `agent_start` event |
| `pi.run.completed` | From Pi `agent_end` event |
| `pi.sandbox.destroyed` | On pod/session destruction (TTL or explicit) |
| `pi.artifact.delivered` | After successful Workspaces POST /output |

### 9.3 Benchmark Suite

Mandatory benchmark suite measures:
- Warm exec latency (echo, shell)
- File system performance (ripgrep scan)
- Git clone speed
- Package install speed (pnpm, uv, Go, Cargo)
- Snapshot creation/rollback
- Artifact export
- Parallel density (10, 100 sandboxes)

## 10. Milestones

| Milestone | Status | Scope |
|-----------|--------|-------|
| M1 | ✅ | Local Linux MVP (CLI, daemon, backends, templates, workspace, exec, session, artifacts, logs, system, benchmarks) |
| M2 | ✅ | Hardening & Cache (secrets, network, cache model, snapshots, policy) |
| M3 | ✅ | Agent Integrations (SDKs) |
| M4 | ✅ | Secure Backend (gVisor, runtime selection) |
| M5 | ✅ | MicroVM Backend (Firecracker/CH, guest control plane) |
| M6 | ✅ | Remote Daemon Mode (contexts, transport, auth) |
| M7 | ✅ | Cross-Platform GUI (workbench, auth, session ops, settings) |

## 11. References

- [SPEC.md](./SPEC.md) — Master spec with requirements and acceptance criteria
- [docs/features/](./docs/features/) — Detailed feature specs
- [docs/decisions/](./docs/decisions/) — Architecture decision records
- [docs/contracts/](./docs/contracts/) — Upstream API contracts
- [docs/design-principles.md](./docs/design-principles.md) — Platform invariants
- [.pi/block.yaml](./.pi/block.yaml) — Project configuration
