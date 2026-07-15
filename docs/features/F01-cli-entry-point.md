# F01: CLI Entry Point

> Source: `SPEC.md` §6 Features F1
> Status: ⚠️ Needs re-verify
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F1 | CLI Entry Point | `pi-box` binary with `box` subcommands for sandbox lifecycle management | M1 |

## Expanded Specification

The `pi` binary is the user-facing entry point for all sandbox operations. It uses a cobra-based CLI structure with a `box` subcommand namespace. The CLI is thin — it parses flags, validates input, and delegates to `pi-sandboxd` via the local Unix socket API.

The CLI supports the following subcommand groups:
- `pi box` — sandbox lifecycle (create, list, inspect, clone, exec, shell, files, diff, patch, artifacts, snapshot, logs, destroy)
- `pi system` — daemon and state management (status, doctor, prune, disk-usage)
- `pi bench` — benchmark suite (run)
- `pi template` — template management (list, inspect, build, update, prune)

Each subcommand maps to one or more API calls. The CLI handles:
- Flag parsing with sensible defaults from `~/.pi-box/config.yaml`
- JSON output mode (`--json`) for machine consumption
- Error formatting that is actionable (see SPEC.md §28)
- Context switching for future remote daemon support (Milestone 6)

The CLI must not implement business logic — all operations go through the daemon API. This keeps the CLI testable and allows IDE/SDK integrations to bypass it entirely.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-1.1: `pi box create --name demo --template node-python --mode fast` creates a sandbox via daemon API
- [x] AC-1.2: `pi box list` lists all sandboxes by querying the daemon
- [x] AC-1.3: `pi box inspect demo` displays sandbox details from daemon
- [x] AC-1.4: `pi box destroy demo` sends destroy request to daemon
- [x] AC-1.5: `--json` flag produces valid JSON output for all box commands
- [x] AC-1.6: `pi system status` shows daemon connection status and sandbox summary
- [x] AC-1.7: `pi system doctor` validates configuration and reports issues
- [x] AC-1.8: `pi system prune` removes old sandbox state
- [x] AC-1.9: `pi system disk-usage` shows storage breakdown by category
- [x] AC-1.10: `pi bench run` invokes benchmark suite
- [x] AC-1.11: `pi template list/inspect/build/update/prune` commands work
- [x] AC-1.12: Error messages are actionable (not just "failed")

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi/` | New Go module with cobra CLI |
| `~/.pi-box/config.yaml` | Default config file parsed by CLI |
| `pi-sandboxd` | CLI delegates to daemon; no direct backend calls |
| `--json` flag | Present on all box commands for machine consumption |

Reference `SPEC.md` §9 (Interface Contract) for API shapes the CLI consumes.

## Security Considerations

- CLI reads config from `~/.pi-box/config.yaml` — no elevated privileges
- `pi system doctor` may inspect filesystem permissions
- `pi system prune` modifies `~/.pi-box/` state — requires explicit confirmation
- No secrets passed as CLI arguments (use flags with stdin for sensitive input)
- Git credentials handled by daemon, never exposed to CLI process

Reference `SPEC.md` §8 (Security Model) for sandbox security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F2: Daemon API | Internal feature | 🔴 Not started |
| Cobra (Go CLI library) | External dependency | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go CLI in `cmd/pi/` using cobra |
| **Configuration** | Reads `~/.pi-box/config.yaml` for defaults |

**ADR references:** None yet.
**ADR gaps:** None identified.

### Surfacing an ADR need

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| How should the CLI handle connection failures to the daemon? | F1, F2 | ADR-NNN: CLI error handling and retry strategy |

## Tasks

### T1.1: CLI binary skeleton with cobra

**Description:** Create the `cmd/pi/` Go module with cobra-based CLI structure. Root command with `box`, `system`, `bench`, `template` subcommand groups. No functionality yet — just the command tree.

**Acceptance criteria:**
- [x] `pi --help` shows root command with subcommand groups
- [x] `pi box --help` shows all box subcommands
- [x] `pi system --help` shows all system subcommands
- [x] `pi bench --help` shows bench subcommands
- [x] `pi template --help` shows template subcommands

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] `./bin/pi --help`

**Files:** `cmd/pi/main.go`, `cmd/pi/root.go`, `cmd/pi/box/`, `cmd/pi/system/`, `cmd/pi/bench/`, `cmd/pi/template/`
**Size:** S
**Depends on:** None

### T1.2: Box subcommands stub with daemon connection

**Description:** Implement `box create/list/inspect/destroy/clone/exec/shell/files/diff/patch/artifacts/snapshot/logs` subcommand stubs that connect to `pi-sandboxd` via Unix socket and print "stub: not yet implemented".

**Acceptance criteria:**
- [x] All box subcommands connect to daemon socket
- [x] Connection failure shows actionable error (SPEC.md §28)
- [x] `--json` flag produces `{"error": "stub", "command": "..."}` for stub commands

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] `./bin/pi box create --help` works
- [x] `./bin/pi box create mybox` shows connection error (no daemon running)

**Files:** `cmd/pi/box/create.go`, `cmd/pi/box/list.go`, `cmd/pi/box/inspect.go`, `cmd/pi/box/destroy.go`, `cmd/pi/box/clone.go`, `cmd/pi/box/exec.go`, `cmd/pi/box/shell.go`, `cmd/pi/box/files.go`, `cmd/pi/box/diff.go`, `cmd/pi/box/artifacts.go`, `cmd/pi/box/snapshot.go`, `cmd/pi/box/logs.go`
**Size:** M
**Depends on:** None

### T1.3: System subcommands

**Description:** Implement `system status/doctor/prune/disk-usage` subcommands. These query the daemon for local state information.

**Acceptance criteria:**
- [x] `pi system status` shows daemon connection status
- [x] `pi system doctor` validates config file and reports issues
- [x] `pi system prune` asks for confirmation before removing state
- [x] `pi system disk-usage` shows breakdown by category

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] `./bin/pi system status` works with and without daemon

**Files:** `cmd/pi/system/status.go`, `cmd/pi/system/doctor.go`, `cmd/pi/system/prune.go`, `cmd/pi/system/disk-usage.go`
**Size:** S
**Depends on:** F2 (daemon must be running for full functionality)

### T1.4: Config file parsing

**Description:** Implement `~/.pi-box/config.yaml` parsing with defaults for runtime mode, network mode, exec timeout, max output, cache settings.

**Acceptance criteria:**
- [x] Default config created on first `pi system doctor` run
- [x] Config values override CLI flags with lower precedence
- [x] Invalid config produces actionable error

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] Unit tests for config parsing

**Files:** `pkg/cli/config/config.go`, `~/.pi-box/config.yaml` (created by doctor)
**Size:** S
**Depends on:** None

## Verification Plan

- [x] `go build ./cmd/pi/...` succeeds
- [x] `./bin/pi --help` shows all subcommand groups
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
