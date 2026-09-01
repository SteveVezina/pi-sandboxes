# PI Agent Sandbox Runtime

A local-first, open-source sandbox runtime for PI coding agents.

Provides tiny, fast, isolated developer workspaces where AI coding agents can clone repositories, read and write files, run Node.js / Python / Go / Rust commands, build and run applications, collect logs, export artifacts, and produce patches.

> **Core positioning:** Local-first sandboxes for coding agents. Fast warm exec, tiny footprint, selectable isolation.

## Quick Start

```bash
# Build
make build

# Start the sandbox daemon
./pi-sandboxd --socket ~/.pi-box/sandboxd.sock

# Use the CLI
./pi-box box create node-python
./pi-box box clone <repo>
./pi-box box exec -- pnpm test
./pi-box box diff
./pi-box box artifacts pull ./out
```

## Installation

### From source

```bash
git clone https://gitlab.com/pi-sandbox/pi-sandbox-runtime.git
cd pi-sandbox-runtime
make install
```

### Docker

```bash
docker build -t pi-sandbox .
docker run -d \
  --name pi-sandbox \
  -v ~/.pi-box:/home/pi/.pi-box \
  -p 9001:9001 \
  pi-sandbox
```

### Pre-built binaries

Binaries are available on the [releases page](https://gitlab.com/pi-sandbox/pi-sandbox-runtime/-/releases).

## Architecture at a glance

```
┌─────────────────────────────────────────────────────────┐
│                        CLI (pi)                         │
│  box create │ box clone │ box exec │ box diff │ box pull│
└─────────────────────────┬───────────────────────────────┘
                          │ HTTP / Unix socket
┌─────────────────────────▼───────────────────────────────┐
│                    Daemon (pi-sandboxd)                  │
│  Sandbox Manager │ Template Engine │ Policy Engine      │
│  Command Executor │ Snapshot Manager │ Output Delivery │
│  Secrets Manager │ Cache Manager │ Workspace Manager    │
└─────────────────────────┬───────────────────────────────┘
                          │ Runtime selection
┌─────────────────────────▼───────────────────────────────┐
│                  Runtime Backends                        │
│  Fast (namespaces) │ Compat (OCI) │ Secure (gVisor)     │
│  Isolated (Kata) │ MicroVM (Firecracker/CH)             │
└─────────────────────────────────────────────────────────┘
```

See [ARCHITECTURE.md](./ARCHITECTURE.md) for full details.

## Features

All 27 features across 7 milestones are implemented and reviewed.

| Milestone | Scope | Features |
|-----------|-------|----------|
| **M1** | Local Linux MVP | CLI, Daemon API, Fast/Compat Backends, Template System, Workspace Ops, Command Execution, Sandbox Lifecycle, Output Delivery, Logs, System Commands, Benchmarks |
| **M2** | Hardening & Cache | Secrets & Network Model, Cache Model, Snapshot & Rollback, Policy Enforcement |
| **M3** | Agent Integrations | SDKs (Python, TypeScript) |
| **M4** | Secure Backend | Secure Backend (gVisor), Runtime Selection & Fallback |
| **M5** | MicroVM Backend | MicroVM Backend, MicroVM Guest Control Plane |
| **M6** | Remote Daemon Mode | Remote Daemon Contexts, Remote Transport & Auth |
| **M7** | Cross-Platform GUI | GUI Workbench, Workspace Authorization, Sandbox Operations, Settings & Diagnostics |

See [docs/features/INDEX.md](./docs/features/INDEX.md) for the full feature dashboard.

## Runtime Modes

Select the isolation level that matches your threat model:

| Mode | Backend | Isolation | Speed |
|------|---------|-----------|-------|
| `fast` | Linux namespaces | Process-level | Fastest |
| `compat` | OCI container | Container-level | Fast |
| `secure` | gVisor | syscall-reuser | Moderate |
| `isolated` | Kata Containers | VM-level | Slower |
| `microvm` | Firecracker / Cloud-Hypervisor | MicroVM-level | Slowest |

## Development

```bash
# Run tests
make test

# Run with coverage
make test-coverage

# Lint
make lint

# Format
make format

# Start mock services
make mock-up
make mock-down
```

### Project structure

```
├── cmd/              # Entry points (pi, pi-sandboxd, pi-agentd, pi-init)
├── pkg/              # Core library packages
│   ├── api/          # REST / WebSocket API handlers
│   ├── daemon/       # Daemon lifecycle & management
│   ├── exec/         # Command execution engine
│   ├── runtime/      # Runtime backend implementations
│   ├── sandbox/      # Sandbox lifecycle management
│   ├── workspace/    # File system operations
│   ├── template/     # Template system
│   ├── policy/       # Policy enforcement
│   ├── snapshot/     # Snapshot & rollback
│   ├── cache/        # Dependency cache management
│   ├── secrets/      # Secrets management
│   ├── network/      # Network policy
│   ├── logs/         # Log collection & history
│   ├── artifacts/    # Artifact export
│   ├── git/          # Git operations
│   ├── context/      # Execution context
│   ├── system/       # System commands
│   ├── terminal/     # Terminal emulation
│   ├── mcp/          # MCP protocol support
│   ├── remote/       # Remote daemon support
│   ├── gui/          # GUI workbench
│   └── types/        # Shared types
├── tests/            # Integration & unit tests
├── docs/             # Specs, features, contracts, decisions
├── examples/         # Usage examples
├── specs/            # Block spec & design docs
└── Makefile          # Build & test automation
```

## Documentation

User- and API-facing docs (install, quickstart, CLI, API, runtime modes,
architecture) live in [`website/`](./website/), a Docusaurus site in the
pnpm + turbo workspace:

```bash
pnpm install
pnpm run docs:dev     # http://localhost:3000
```

Run `pnpm run docs:build` before any user-visible change —
`onBrokenLinks: 'throw'` makes drift a build failure. See
`website/docs/contributing-docs.md`.

## Contributing

1. Read [SPEC.md](./SPEC.md) — the master source of truth
2. Read [docs/design-principles.md](./docs/design-principles.md) — platform invariants
3. Check [docs/features/INDEX.md](./docs/features/INDEX.md) for active work
4. Propose changes via [PROP process](./docs/proposals/)

## License

MIT

## Links

- [SPEC.md](./SPEC.md) — Master spec
- [ARCHITECTURE.md](./ARCHITECTURE.md) — Architecture overview
- [docs/features/](./docs/features/) — Feature specs
- [docs/decisions/](./docs/decisions/) — Architecture decision records
- [docs/contracts/](./docs/contracts/) — Upstream API contracts
- [docs/proposals/](./docs/proposals/) — Spec change proposals
