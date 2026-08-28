# F10: Logs & Command History

> Source: `SPEC.md` §6 Features F10
> Status: 🟢 Reviewed *(2026-08-28: re-verified — `history` CLI command and full-content `logs` output were undocumented gaps, now implemented; also fixed a pre-existing test bug that leaked log directories into the real user's ~/.pi-box)*
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F10 | Logs & Command History | Command history, stdout/stderr logs, exit codes, duration, timeout status | M1 |

## Expanded Specification

Logs and command history track all activity within sandboxes. Each exec command produces a log entry with:
- Command string
- Exit code
- Duration (milliseconds)
- Timeout status (boolean)
- Output truncation status (boolean)
- stdout log file path
- stderr log file path
- Resource usage (when available)

Log entries are stored under `~/.pi-box/sandboxes/<id>/logs/` with naming convention `exec-{sequence}.log`.

Each log entry is a JSON file containing:
```json
{
  "sequence": 42,
  "timestamp": "2026-06-26T10:00:00Z",
  "command": "pnpm test",
  "exitCode": 0,
  "durationMs": 1842,
  "timedOut": false,
  "truncated": false,
  "stdoutPath": "~/.pi-box/sandboxes/app1/logs/exec-42.stdout",
  "stderrPath": "~/.pi-box/sandboxes/app1/logs/exec-42.stderr"
}
```

Full stdout/stderr content is stored in separate files (not inline in the JSON entry) to avoid large JSON files.

CLI commands:
```bash
pi-box box logs <name>
pi-box box history <name>
pi-box box metrics <name>
```

- `logs` — Shows full log entries with stdout/stderr content
- `history` — Shows command history summary (command, exit code, duration, status)
- `metrics` — Shows resource usage summary (when available)

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-10.1: `pi-box box logs <id>` shows command logs *(2026-08-28: was only printing a one-line summary, not full stdout/stderr content per the Expanded Specification — fixed)*
- [x] AC-10.2: `pi-box box history <id>` shows command history *(2026-08-28: command didn't exist at all — implemented, backed by the existing `/logs/history` endpoint)*
- [x] AC-10.3: Each log entry includes: command, exit code, duration, timeout status, output truncation

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/logs/` | Log management |
| `~/.pi-box/sandboxes/<id>/logs/` | Log storage |
| F5: Command Execution | Produces log entries |
| F2: Daemon API | Logs endpoint |

## Security Considerations

- Logs stored under `~/.pi-box/` with restricted permissions
- No secrets in logs (command arguments may contain secrets — warn users)
- Log rotation not implemented in MVP (future hardening)
- Log files are plain text (no encryption)

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F5: Command Execution | Internal feature | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go logs package |
| **Configuration** | Log directory under `~/.pi-box/sandboxes/<id>/` |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T9.1: Log entry storage

**Description:** Implement log entry storage. Each exec command creates a JSON log entry and stdout/stderr files.

**Acceptance criteria:**
- [x] Log entries stored as JSON under `~/.pi-box/sandboxes/<id>/logs/exec-{seq}.json`
- [x] stdout/stderr content stored in separate files
- [x] Sequence number auto-incremented per sandbox
- [x] Timestamp recorded at command completion

**Verification:**
- [x] `go build ./pkg/logs/...`
- [x] Unit test: log entry created with correct fields

**Files:** `pkg/logs/entry.go`
**Size:** S
**Depends on:** F5 (Command Execution — produces log data)

### T9.2: Log listing and history ✅ *(2026-08-28: implemented — `history` command and full-content `logs` output didn't exist; added both plus a real test)*

**Description:** Implement log listing and command history retrieval.

**Acceptance criteria:**
- [x] `pi-box box logs <id>` shows full log entries with stdout/stderr
- [x] `pi-box box history <id>` shows summary (command, exit code, duration)
- [x] Logs ordered by sequence number (newest first) (`Manager.List`/`History` sort descending by sequence)
- [x] Empty sandbox shows "no commands executed" message (both `logsCmd` and `historyCmd`)

**Verification:**
- [x] `go build ./pkg/api/... ./cmd/pi-box/...`
- [x] Integration test: `TestLogsAndHistory`, `tests/api/logs_test.go` — execs a command, asserts `/logs`, `/logs/history`, and stdout/stderr content routes all return correct data

**Files:** `pkg/logs/list.go`, `pkg/api/sandbox_logs.go`, `cmd/pi-box/box/box.go`
**Size:** S
**Depends on:** T9.1 (log entry storage)

## Verification Plan

- [x] `go build ./pkg/api/... ./cmd/pi-box/...` succeeds
- [x] Log entries created for each exec command
- [x] Logs and history commands display correct data
- [x] stdout/stderr content accessible from log entries

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Log file naming convention not specified | §20 Logs and telemetry | Add: "exec-{sequence}.json with separate .stdout/.stderr files" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Log rotation/cleanup (future)
- Structured log aggregation (future)
- Remote log shipping (future)
- Log search/filtering (future)
- Resource usage metrics (placeholder, not implemented in MVP)
