# F01: CLI Entry Point

> Source: `SPEC.md` §6 Features F1
> Status: 🟢 Reviewed
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F1 | CLI Entry Point | `pi-box` binary with `box` subcommands for sandbox lifecycle management | M1 |

## Expanded Specification

The `pi-box` binary is the user-facing entry point for all sandbox operations. It uses a cobra-based CLI structure with a `box` subcommand namespace. The CLI is thin — it parses flags, validates input, and delegates to `pi-sandboxd` via the local Unix socket API.

The CLI supports the following subcommand groups:
- `pi-box box` — sandbox lifecycle (create, list, inspect, clone, exec, shell, files, diff, patch, artifacts, snapshot, logs, destroy)
- `pi-box system` — daemon and state management (status, doctor, prune, disk-usage)
- `pi-box bench` — benchmark suite (run)
- `pi-box template` — template management (list, inspect, build, update, prune)

Each subcommand maps to one or more API calls. The CLI handles:
- Flag parsing with sensible defaults from `~/.pi-box/config.yaml`
- JSON output mode (`--json`) for machine consumption
- Error formatting that is actionable (see SPEC.md §28)
- Context switching for future remote daemon support (Milestone 6)

The CLI must not implement business logic — all operations go through the daemon API. This keeps the CLI testable and allows IDE/SDK integrations to bypass it entirely.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-1.1: `pi-box box create --name demo --template node-python --mode fast` creates a sandbox via daemon API
- [x] AC-1.2: `pi-box box list` lists all sandboxes by querying the daemon
- [x] AC-1.3: `pi-box box inspect demo` displays sandbox details from daemon
- [x] AC-1.4: `pi-box box destroy demo` sends destroy request to daemon
- [x] AC-1.5: `--json` flag produces valid JSON output for all box commands
- [x] AC-1.6: `pi-box system status` shows daemon connection status and sandbox summary
- [x] AC-1.7: `pi-box system doctor` validates configuration and reports issues
- [x] AC-1.8: `pi-box system prune` removes old sandbox state
- [x] AC-1.9: `pi-box system disk-usage` shows storage breakdown by category
- [x] AC-1.10: `pi-box bench run` invokes benchmark suite
- [x] AC-1.11: `pi-box template list/inspect/build/update/prune` commands work
- [x] AC-1.12: Error messages are actionable (not just "failed")

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-box/` | New Go module with cobra CLI |
| `~/.pi-box/config.yaml` | Default config file parsed by CLI |
| `pi-sandboxd` | CLI delegates to daemon; no direct backend calls |
| `--json` flag | Present on all box commands for machine consumption |

Reference `SPEC.md` §9 (Interface Contract) for API shapes the CLI consumes.

## Security Considerations

- CLI reads config from `~/.pi-box/config.yaml` — no elevated privileges
- `pi-box system doctor` may inspect filesystem permissions
- `pi-box system prune` modifies `~/.pi-box/` state — requires explicit confirmation
- No secrets passed as CLI arguments (use flags with stdin for sensitive input)
- Git credentials handled by daemon, never exposed to CLI process

Reference `SPEC.md` §8 (Security Model) for sandbox security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F2: Daemon API | Internal feature | ✅ Implemented |
| Cobra (Go CLI library) | External dependency | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go CLI in `cmd/pi-box/` using cobra |
| **Configuration** | Reads `~/.pi-box/config.yaml` for defaults |

**ADR references:** None yet.
**ADR gaps:** None identified.

### Surfacing an ADR need

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How should the CLI handle connection failures to the daemon? | F1, F2 | ADR-NNN: CLI error handling and retry strategy |

## Tasks

### T1.1: CLI binary skeleton with cobra

**Description:** Create the `cmd/pi-box/` Go module with cobra-based CLI structure. Root command with `box`, `system`, `bench`, `template` subcommand groups. No functionality yet — just the command tree.

**Acceptance criteria:**
- [x] `pi-box --help` shows root command with subcommand groups
- [x] `pi-box box --help` shows all box subcommands
- [x] `pi-box system --help` shows all system subcommands
- [x] `pi-box bench --help` shows bench subcommands
- [x] `pi-box template --help` shows template subcommands

**Verification:**
- [x] `go build ./cmd/pi-box/...`
- [x] `./bin/pi-box --help`

**Files:** `cmd/pi-box/main.go`, `cmd/pi-box/root.go`, `cmd/pi-box/box/`, `cmd/pi-box/system/`, `cmd/pi-box/bench/`, `cmd/pi-box/template/`
**Size:** S
**Depends on:** None

### T1.2: Box subcommands stub with daemon connection

**Description:** Implement `box create/list/inspect/destroy/clone/exec/shell/files/diff/patch/artifacts/snapshot/logs` subcommand stubs that connect to `pi-sandboxd` via Unix socket and print "stub: not yet implemented".

**Acceptance criteria:**
- [x] All box subcommands connect to daemon socket
- [x] Connection failure shows actionable error (SPEC.md §28)
- [x] `--json` flag produces `{"error": "stub", "command": "..."}` for stub commands

**Verification:**
- [x] `go build ./cmd/pi-box/...`
- [x] `./bin/pi-box box create --help` works
- [x] `./bin/pi-box box create mybox` shows connection error (no daemon running)

**Files:** `cmd/pi-box/box/create.go`, `cmd/pi-box/box/list.go`, `cmd/pi-box/box/inspect.go`, `cmd/pi-box/box/destroy.go`, `cmd/pi-box/box/clone.go`, `cmd/pi-box/box/exec.go`, `cmd/pi-box/box/shell.go`, `cmd/pi-box/box/files.go`, `cmd/pi-box/box/diff.go`, `cmd/pi-box/box/artifacts.go`, `cmd/pi-box/box/snapshot.go`, `cmd/pi-box/box/logs.go`
**Size:** M
**Depends on:** None

### T1.3: System subcommands

**Description:** Implement `system status/doctor/prune/disk-usage` subcommands. These query the daemon for local state information.

**Acceptance criteria:**
- [x] `pi-box system status` shows daemon connection status
- [x] `pi-box system doctor` validates config file and reports issues
- [x] `pi-box system prune` asks for confirmation before removing state
- [x] `pi-box system disk-usage` shows breakdown by category

**Verification:**
- [x] `go build ./cmd/pi-box/...`
- [x] `./bin/pi-box system status` works with and without daemon

**Files:** `cmd/pi-box/system/status.go`, `cmd/pi-box/system/doctor.go`, `cmd/pi-box/system/prune.go`, `cmd/pi-box/system/disk-usage.go`
**Size:** S
**Depends on:** F2 (daemon must be running for full functionality)

### T1.4: Config file parsing

**Description:** Implement `~/.pi-box/config.yaml` parsing with defaults for runtime mode, network mode, exec timeout, max output, cache settings.

**Acceptance criteria:**
- [x] Default config created on first `pi-box system doctor` run
- [x] Config values override CLI flags with lower precedence
- [x] Invalid config produces actionable error

**Verification:**
- [x] `go build ./cmd/pi-box/...`
- [x] Unit tests for config parsing

**Files:** `pkg/cli/config/config.go`, `~/.pi-box/config.yaml` (created by doctor)
**Size:** S
**Depends on:** None

## Verification Plan

- [x] `go build ./cmd/pi-box/...` succeeds
- [x] `./bin/pi-box --help` shows all subcommand groups
- [x] All box commands connect to daemon (or show connection error)
- [x] `--json` flag produces valid JSON
- [x] Error messages are actionable per SPEC.md §28

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| CLI error format not fully specified | §28 Error handling | Add CLI-specific error format examples |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How should the CLI handle connection failures to the daemon? | F1, F2 | ADR-NNN: CLI error handling and retry strategy |

Note: ADRs are block-level. Flag the need here; author the ADR file as a separate commit.

## Out of Scope

- Remote daemon support (Milestone 6) — CLI connects to local socket only
- Interactive shell within CLI (delegated to `exec` with `--interactive` flag)
- Plugin system for CLI extensions
- Tab completion (future improvement)
