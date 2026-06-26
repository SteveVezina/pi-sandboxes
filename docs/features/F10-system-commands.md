# F10: System Commands

> Source: `SPEC.md` §6 Features F16
> Status: 🟡 Spec written
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

- [ ] AC-16.1: `pi system status` shows daemon and sandbox status
- [ ] AC-16.2: `pi system doctor` validates configuration
- [ ] AC-16.3: `pi system prune` cleans old state
- [ ] AC-16.4: `pi system disk-usage` shows storage breakdown

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
- [ ] Shows daemon connection status (connected/disconnected)
- [ ] Shows number of active sandboxes (WARM + EXECUTING states)
- [ ] Shows total sandbox count
- [ ] Shows `~/.pi/` directory existence

**Verification:**
- [ ] `go build ./cmd/pi/...`
- [ ] Integration test: status with and without daemon

**Files:** `pkg/system/status.go`, `cmd/pi/system/status.go`
**Size:** S
**Depends on:** F4 (Session Lifecycle — sandbox state)

### T10.2: System doctor

**Description:** Implement `pi system doctor` validating configuration and reporting issues.

**Acceptance criteria:**
- [ ] Checks `~/.pi/config.yaml` exists and is valid YAML
- [ ] Checks required directories exist (`sandboxes/`, `templates/`)
- [ ] Checks disk space (warns if < 1GiB free)
- [ ] Checks filesystem permissions on `~/.pi/`
- [ ] Reports issues with actionable recommendations
- [ ] Creates default config if missing (non-destructive)

**Verification:**
- [ ] `go build ./cmd/pi/...`
- [ ] Integration test: doctor with valid and invalid config

**Files:** `pkg/system/doctor.go`, `cmd/pi/system/doctor.go`
**Size:** S
**Depends on:** None

### T10.3: System prune

**Description:** Implement `pi system prune` removing old sandbox state.

**Acceptance criteria:**
- [ ] Removes destroyed sandbox metadata
- [ ] Removes orphaned sandbox data (no corresponding metadata)
- [ ] Removes old log files (> 30 days)
- [ ] Asks for confirmation before destructive operations
- [ ] `--yes` flag skips confirmation

**Verification:**
- [ ] `go build ./cmd/pi/...`
- [ ] Integration test: prune removes old state

**Files:** `pkg/system/prune.go`, `cmd/pi/system/prune.go`
**Size:** S
**Depends on:** F4 (Session Lifecycle)

### T10.4: System disk-usage

**Description:** Implement `pi system disk-usage` showing storage breakdown.

**Acceptance criteria:**
- [ ] Shows size of `sandboxes/` directory
- [ ] Shows size of `templates/` directory
- [ ] Shows size of `caches/` directory
- [ ] Shows size of `images/` directory
- [ ] Shows size of `logs/` directory
- [ ] Shows total size
- [ ] Output is human-readable (MiB/GiB)

**Verification:**
- [ ] `go build ./cmd/pi/...`
- [ ] Integration test: disk-usage shows correct sizes

**Files:** `pkg/system/disk-usage.go`, `cmd/pi/system/disk-usage.go`
**Size:** S
**Depends on:** None

## Verification Plan

- [ ] `go build ./cmd/pi/...` succeeds
- [ ] All 4 system commands work
- [ ] `pi system doctor` detects and reports issues
- [ ] `pi system prune` removes old state safely
- [ ] `pi system disk-usage` shows correct breakdown

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
