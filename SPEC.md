# PI Agent Sandbox Runtime - Source of Truth Spec

## 1. Project thesis

Build a local-first, open-source sandbox runtime for PI coding agents.

The runtime must provide tiny, fast, isolated developer workspaces where AI coding agents can clone repositories, read and write files, run Node.js, Python, Go, and Rust commands, build and run apps, collect logs, export artifacts, and produce patches.

The project is not Kubernetes-first, cloud-first, or SaaS-first. It starts as developer local tooling with a daemon, CLI, API, and SDKs. Kubernetes and remote control planes can come later.

Core positioning:

> Local-first sandboxes for coding agents. Fast warm exec, tiny footprint, selectable isolation.

The first version should feel like:

```bash
pi box create node-python
pi box clone <repo>
pi box exec -- pnpm test
pi box diff
pi box artifacts pull ./out
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
- GUI/browser desktop agents.
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

## 5. High-level architecture

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

The CLI is thin. PI Agent and other tools should talk directly to `pi-sandboxd` over a local Unix socket or localhost API.

## 6. User-facing runtime modes

Expose simple profiles. Do not expose low-level runtime complexity to normal users.

| Mode | Backend | Purpose | First release |
|---|---|---|---|
| `fast` | Native Linux sandbox using namespaces, cgroups, seccomp, Landlock/bubblewrap/nsjail style isolation | Fastest and smallest local trusted mode | Yes |
| `compat` | runc/containerd/Podman style OCI container | Best compatibility for Node.js/Python/Go/Rust projects | Yes |
| `secure` | gVisor/runsc | Better isolation while keeping container-like UX | Later |
| `isolated` | Kata Containers | Stronger VM-grade enterprise isolation | Later |
| `microvm` | Firecracker or Cloud Hypervisor | Snapshot-first microVM mode | Later |
| `desktop` | Full VM/KubeVirt/Apple Virtualization/WSL2 backend | GUI/browser/computer-use agents | Future |
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

## 7. Isolation strategy

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

## 8. Local filesystem layout

Use predictable state under `~/.pi` by default.

```text
~/.pi/
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
pi system status
pi system doctor
pi system prune
pi system disk-usage
```

## 9. Workspace model

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

## 10. Cache model

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

## 11. Templates

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
pi template list
pi template inspect node-python
pi template build node-python
pi template update node-python
pi template prune
```

## 12. CLI requirements

The CLI binary should be named `pi` initially, with `box` as the sandbox subcommand.

### 12.1 Create sandbox

```bash
pi box create --name app1 --template node-python
pi box create node-python
pi box create --mode fast node-python
pi box create --mode compat rust
```

### 12.2 List and inspect

```bash
pi box list
pi box inspect app1
pi box status app1
```

### 12.3 Clone repository

```bash
pi box clone app1 https://github.com/acme/app
pi box clone app1 git@github.com:acme/app.git
```

SSH credentials must not be blindly mounted. Use a controlled Git credential helper or explicit user opt-in.

### 12.4 Execute command

```bash
pi box exec app1 -- pnpm install
pi box exec app1 -- pnpm test
pi box exec app1 -- python scripts/check.py
pi box exec app1 -- go test ./...
pi box exec app1 -- cargo test
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
pi box shell app1
```

Shell is for humans, not the agent hot path.

### 12.6 Files

```bash
pi box files list app1 /workspace
pi box files read app1 /workspace/package.json
pi box files write app1 /workspace/src/index.ts < index.ts
pi box files pull app1 /workspace/dist ./dist
pi box files push app1 ./README.md /workspace/README.md
```

### 12.7 Diff and patch

```bash
pi box diff app1
pi box patch app1 > changes.patch
pi box apply-patch app1 changes.patch
```

### 12.8 Artifacts

```bash
pi box artifacts list app1
pi box artifacts pull app1 ./artifacts
pi box artifacts pack app1 --output artifacts.tar.zst
```

### 12.9 Snapshots

```bash
pi box snapshot app1 before-refactor
pi box snapshots app1
pi box rollback app1 before-refactor
pi box snapshot delete app1 before-refactor
```

### 12.10 Run/serve apps

```bash
pi box exec app1 -- pnpm dev --host 0.0.0.0
pi box port-forward app1 3000:3000
pi box logs app1
```

### 12.11 Destroy

```bash
pi box destroy app1
pi box destroy --all
```

## 13. Daemon API

`pi-sandboxd` exposes a local API over Unix socket by default.

Default socket:

```text
~/.pi/sandboxd.sock
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

## 14. Agent SDK requirements

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

## 15. Security defaults

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

## 16. Secrets model

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

## 17. Network model

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

## 18. Snapshot and rollback

Initial implementation can use filesystem-level snapshots:

- overlay upperdir copy
- reflink copy when supported
- btrfs/zfs later
- tar/zstd fallback

Commands:

```bash
pi box snapshot app1 before-change
pi box rollback app1 before-change
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

## 19. Artifact model

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
pi box artifacts list app1
pi box artifacts pull app1 ./artifacts
pi box artifacts pack app1 --output artifacts.tar.zst
```

Artifact export should avoid copying the whole workspace unless requested.

## 20. Logs and telemetry

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
pi box logs app1
pi box history app1
pi box metrics app1
```

Telemetry must be local-only by default.

## 21. Benchmarks

Benchmark suite is mandatory.

Command:

```bash
pi bench run
pi bench run --mode fast
pi bench run --mode compat
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

## 22. Implementation language

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

## 23. Repository structure

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

## 24. Milestones

### Milestone 1: Local Linux MVP

Deliver:

- `pi` CLI
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
pi box create --name demo --template node-python --mode fast
pi box clone demo https://github.com/some/repo
pi box exec demo -- pnpm install
pi box exec demo -- pnpm test
pi box diff demo
pi box artifacts pull demo ./out
pi box destroy demo
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
- `pi system doctor`
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
pi context create workstation ssh://gpu-box.local
pi context use workstation
pi box create node-python
```

## 25. Compatibility expectations

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

## 26. Service dependencies later

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

## 27. Configuration file

Default config:

```yaml
runtime:
  defaultMode: auto
  allowModes:
    - fast
    - compat

storage:
  root: ~/.pi
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
```

## 28. Error handling requirements

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
Last output saved to: ~/.pi/sandboxes/app1/logs/exec-42.log
Try: pi box logs app1 --exec 42
```

Runtime fallback errors must explain next steps:

```text
gVisor backend is not installed.
Available modes: fast, compat
Install docs: pi docs gvisor
Try: pi box create --mode compat node-python
```

## 29. Documentation requirements

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

## 30. Success criteria

The project is successful when:

1. A developer can install it locally and run a real coding-agent loop in under 10 minutes.
2. PI Agent can create a sandbox, clone a repo, run tests, modify files, and export a patch without direct host filesystem access.
3. Warm exec is fast enough to feel interactive.
4. Node.js, Python, Go, and Rust templates work with common projects.
5. The benchmark suite clearly compares fast, compat, secure, and later microVM modes.
6. Developers understand where files, caches, logs, artifacts, and snapshots live.
7. Isolation is selectable per sandbox session.
8. The project can later grow into remote workstation, microVM, and Kubernetes-backed modes without redesigning the API.

## 31. Product north star

The north star is not simply running containers or microVMs.

The north star is:

> A coding agent can get a warm, isolated, language-ready workspace almost instantly, run hundreds of commands with low overhead, safely manage code changes, and export only the useful diff and artifacts.

## 32. First coding-agent task list

A coding agent implementing this project should start with the following tasks in order:

1. Create the repository skeleton.
2. Implement `pi` CLI with `box` subcommands using stubbed daemon calls.
3. Implement `pi-sandboxd` local daemon with Unix socket API.
4. Implement sandbox metadata store under `~/.pi/sandboxes`.
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

## 33. Important engineering constraints

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
