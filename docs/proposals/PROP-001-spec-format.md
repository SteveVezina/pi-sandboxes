# PROP-001: Add Structured Block Spec Schema to SPEC.md

## Status
✅ Applied to block spec (2026-06-26)

## Block Spec Reference
`SPEC.md` — full document

## Problem

The `pi-feature-spec` skill (and the spec-driven development workflow defined in AGENTS.md) requires SPEC.md to contain a structured schema with these mandatory sections:

- **§ Features table** — F{N} rows, each with a one-line description
- **§ Acceptance Criteria** — testable criteria, each mapped to a feature
- **§ Core Concepts** — domain concepts the block operates on
- **§ Security Model** — sandbox isolation, network, secrets
- **§ Interface Contract** — API inputs/outputs
- **§ Dependencies** — upstream services/blocks needed
- **§ Out of Scope** — explicit boundaries

Currently, SPEC.md is written as a **product spec** (goals, architecture, milestones, task list). The required content exists — it's just scattered across sections 2–33 instead of being in the structured format the tooling expects.

This means the `pi-feature-spec` skill cannot be invoked to generate feature specs. The workflow is blocked from the start.

### Evidence

The `pi-feature-spec` skill (Step 1) requires:
1. Feature row from § Features table → **missing**
2. Related acceptance criteria from § Acceptance Criteria → **missing**
3. Related core concepts from § Core Concepts → **missing**
4. Security implications from § Sandbox Security Model → **missing** (content exists in §15 but not structured)
5. Interface contract from § Interface Contract → **missing** (content exists in §13 but not structured)
6. Dependencies from § Dependencies → **missing**
7. Out of scope from § Out of Scope → **missing** (content exists in §3 but not structured)

## Proposed Amendment

Add the following sections to SPEC.md **after §4 (Core design principle)** and **before §5 (High-level architecture)**:

### 1. New §5: Core Concepts

```markdown
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
```

### 2. New §6: Features (replaces existing §5 numbering)

```markdown
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
```

### 3. New §7: Acceptance Criteria

```markdown
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
```

### 4. New §8: Security Model

```markdown
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
```

### 5. New §9: Interface Contract

```markdown
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
```

### 6. New §10: Dependencies

```markdown
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
```

### 7. New §11: Out of Scope

```markdown
## 11. Out of Scope

The following are explicitly NOT part of the first release:

- Kubernetes controller or CRDs
- Multi-region control plane
- Full SaaS product
- Firecracker rewrite (phase 5 only uses it, doesn't rewrite it)
- Custom VMM implementation
- Docker-in-Docker as a default capability
- GPU workloads
- GUI/browser desktop agents
- Full Docker Compose compatibility
- Complex enterprise identity management
- Background cloud scheduler
- Windows/macOS native backends (initial target: Linux-first; macOS uses Lima/Colima/Docker; Windows uses WSL2)
- Desktop mode (future)
- WASM/WASI tool sandboxing (future)
```

## Rationale

The existing SPEC.md content has all the information needed — it's just organized as a product spec (goals, architecture, milestones) rather than the structured block spec format required by the `pi-feature-spec` skill and the spec-driven development workflow.

This amendment adds the required structural sections without changing any technical requirements, architecture decisions, or behavior. The content is extracted from existing sections:
- Features → derived from §24 (Milestones) and §32 (Task list)
- Acceptance Criteria → derived from §24 (Milestones), §30 (Success criteria), and §33 (Engineering constraints)
- Core Concepts → extracted from §5-§20
- Security Model → extracted from §15, §16, §17
- Interface Contract → extracted from §13, §14
- Dependencies → extracted from §6, §7, §22
- Out of Scope → extracted from §3

## Impact

### Features Affected
All features (F1-F17) — the amendment makes them specifiable.

### ADRs Affected
None initially. Future ADRs will reference this structure.

### Implementation Blocked?
Yes. Without this amendment, the `pi-feature-spec` skill cannot be invoked. No feature specs can be written. No plan can be created. No implementation can begin.

## Assumption (non-blocking)
The amendment preserves all existing technical requirements. No behavior changes. No new features added. No features removed. Only structural reorganization.

## Requested By
2026-06-26 — Bootstrap phase: SPEC.md needs structured format to enable spec-driven development workflow.

## Cascade completed
- SPEC.md: Added §5 (Core Concepts), §6 (Features), §7 (Acceptance Criteria), §8 (Security Model), §9 (Interface Contract), §10 (Dependencies), §11 (Out of Scope). Original §5-§33 renumbered to §12-§40.
- docs/proposals/INDEX.md: Status updated to ✅ Applied to block spec.
- No ADRs affected (none existed).
- No feature specs existed yet (bootstrap phase).
- No plan.md (not yet created).
