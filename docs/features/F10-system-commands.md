# F10: System Commands

> Source: `SPEC.md` §6 Features F16
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F16 | System Commands | `pi system status/doctor/prune/disk-usage` for local state inspection | M1 |

## Expanded Specification

System commands provide local state inspection and maintenance for the pi-sandbox runtime. They operate on the `~/.pi/` directory structure.

Commands:
1. **status** — Shows daemon connection status, number of active sandboxes, total state size
2. **doctor** — Validates configuration file, checks for common issues (missing directories, permission problems, disk space), reports recommendations
3. **prune** — Removes old sandbox state (destroyed sessions, orphaned data, old logs). Asks for confirmation before destructive operations.
4. **disk-usage** — Shows storage breakdown by category (sandboxes, templates, caches, images, logs)

State directories under `~/.pi/`:
```
~/.pi/
  config.yaml
  sandboxd.sock
  templates/     — template definitions
  sandboxes/     — sandbox metadata and state
  caches/        — dependency caches
  images/        — container images, rootfs
  logs/          — daemon logs
```

CLI commands:
```bash
pi system status
pi system doctor
pi system prune
pi system disk-usage
```

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-16.1: `pi system status` shows daemon and sandbox status
- [x] AC-16.2: `pi system doctor` validates configuration
- [x] AC-16.3: `pi system prune` cleans old state
- [x] AC-16.4: `pi system disk-usage` shows storage breakdown

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/system/` | System command handlers |
| `cmd/pi/system/` | CLI system subcommands |
| `~/.pi/` | State directory inspected by system commands |

## Security Considerations

- `pi system prune` is destructive — requires explicit confirmation
- `pi system doctor` reads filesystem permissions but doesn't modify
- No elevated privileges required
- All operations scoped to `~/.pi/` directory

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F4: Session Lifecycle | Internal feature | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go system commands in `pkg/system/` |
| **Configuration** | Reads `~/.pi/` directory structure |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T10.1: System status

**Description:** Implement `pi system status` showing daemon connection and sandbox summary.

**Acceptance criteria:**
- [x] Shows daemon connection status (connected/disconnected)
- [x] Shows number of active sandboxes (WARM + EXECUTING states)
- [x] Shows total sandbox count
- [x] Shows `~/.pi/` directory existence

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] Integration test: status with and without daemon

**Files:** `pkg/system/status.go`, `cmd/pi/system/status.go`
**Size:** S
**Depends on:** F4 (Session Lifecycle — sandbox state)

### T10.2: System doctor

**Description:** Implement `pi system doctor` validating configuration and reporting issues.

**Acceptance criteria:**
- [x] Checks `~/.pi/config.yaml` exists and is valid YAML
- [x] Checks required directories exist (`sandboxes/`, `templates/`)
- [x] Checks disk space (warns if < 1GiB free)
- [x] Checks filesystem permissions on `~/.pi/`
- [x] Reports issues with actionable recommendations
- [x] Creates default config if missing (non-destructive)

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] Integration test: doctor with valid and invalid config

**Files:** `pkg/system/doctor.go`, `cmd/pi/system/doctor.go`
**Size:** S
**Depends on:** None

### T10.3: System prune

**Description:** Implement `pi system prune` removing old sandbox state.

**Acceptance criteria:**
- [x] Removes destroyed sandbox metadata
- [x] Removes orphaned sandbox data (no corresponding metadata)
- [x] Removes old log files (> 30 days)
- [x] Asks for confirmation before destructive operations
- [x] `--yes` flag skips confirmation

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] Integration test: prune removes old state

**Files:** `pkg/system/prune.go`, `cmd/pi/system/prune.go`
**Size:** S
**Depends on:** F4 (Session Lifecycle)

### T10.4: System disk-usage

**Description:** Implement `pi system disk-usage` showing storage breakdown.

**Acceptance criteria:**
- [x] Shows size of `sandboxes/` directory
- [x] Shows size of `templates/` directory
- [x] Shows size of `caches/` directory
- [x] Shows size of `images/` directory
- [x] Shows size of `logs/` directory
- [x] Shows total size
- [x] Output is human-readable (MiB/GiB)

**Verification:**
- [x] `go build ./cmd/pi/...`
- [x] Integration test: disk-usage shows correct sizes

**Files:** `pkg/system/disk-usage.go`, `cmd/pi/system/disk-usage.go`
**Size:** S
**Depends on:** None

## Verification Plan

- [x] `go build ./cmd/pi/...` succeeds
- [x] All 4 system commands work
- [x] `pi system doctor` detects and reports issues
- [x] `pi system prune` removes old state safely
- [x] `pi system disk-usage` shows correct breakdown

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Prune retention period not specified | §8 Local filesystem layout | Add: "Old logs (> 30 days) cleaned by prune" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Remote system commands (Milestone 6)
- System metrics/telemetry (future)
- System auto-repair (future)
