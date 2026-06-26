# F05: Command Execution

> Source: `SPEC.md` §6 Features F7
> Status: 🟡 Spec written
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F7 | Command Execution | Streaming stdout/stderr exec with timeout, output limits, exit code, and truncation metadata | M1 |

## Expanded Specification

Command execution is the hot path for the coding agent inner loop. It runs a command inside a sandbox session and streams stdout/stderr back to the caller.

The exec endpoint accepts:
```json
{
  "command": "pnpm test",
  "cwd": "/workspace",
  "timeoutMs": 60000,
  "maxOutputBytes": 8388608,
  "network": "restricted"
}
```

And returns:
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

Key behaviors:
1. **Streaming output** — stdout/stderr streamed via SSE or chunked transfer to the caller
2. **Timeout** — command killed after `timeoutMs` (default: 120000ms = 120s). `timedOut` flag set to true.
3. **Output limit** — output truncated at `maxOutputBytes` (default: 8388608 = 8MiB). `truncated` flag set to true.
4. **Exit code** — returned accurately from the sandboxed process
5. **Duration** — measured from command start to process exit
6. **CWD** — working directory inside sandbox (default: /workspace)
7. **Resource limits** — CPU, memory, processes enforced by backend (fast: cgroup, compat: cgroup/container limits)
8. **Network mode** — per-exec network override (none/restricted/open)

The exec is synchronous from the API perspective (HTTP request waits for command completion). For long-running commands, the caller should use a reasonable timeout.

CLI flags:
- `--cwd` — working directory inside sandbox
- `--timeout` — command timeout (e.g., `60s`)
- `--max-output` — max output size (e.g., `8MiB`)
- `--memory` — memory limit (e.g., `1Gi`)
- `--cpu` — CPU limit (e.g., `2`)
- `--network` — network mode override (none/restricted/open)
- `--json` — JSON output mode

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-7.1: `pi box exec <id> -- <cmd>` runs command with streaming stdout/stderr
- [ ] AC-7.2: Exit code returned accurately
- [ ] AC-7.3: Timeout status reported when exceeded
- [ ] AC-7.4: Output truncated when exceeding maxOutput, with `truncated` flag
- [ ] AC-7.5: `--cwd`, `--timeout`, `--max-output`, `--memory`, `--cpu`, `--json` options honored
- [ ] AC-7.6: Exec overhead p50 < 10ms in fast mode (SPEC.md §19)

Each criterion must be:
- **Observable** — you can see it happen or verify its effect
- **Testable** — an automated test can assert it
- **Traceable** — links back to `SPEC.md` acceptance criteria

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/exec/` | Command execution engine |
| `pkg/runtime/fast/` | Fast backend exec implementation |
| `pkg/runtime/compat/` | Compat backend exec implementation |
| F2: Daemon API | Exec endpoint in API |
| F3: Fast Backend | Fast backend exec |
| F4: Compat Backend | Compat backend exec |
| F10: Logs & History | Exec results logged here |

## Security Considerations

- Command strings are not shell-interpolated by the CLI (passed as-is to daemon)
- Shell injection risk is the sandbox's responsibility (seccomp, namespaces, Landlock)
- Output truncation prevents memory exhaustion from runaway output
- Timeout prevents resource starvation from hanging commands
- Network mode per-exec overrides session default (more restrictive only)
- No environment variables injected by default (deny-by-default per SPEC.md §16)

Reference `SPEC.md` §8 (Security Model) for full security constraints.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F3: Fast Backend | Internal feature | ⚠️ Partially — exec needs backend to run in |
| F8: Session Lifecycle | Internal feature | Available |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Go exec engine in `pkg/exec/` |
| **Infrastructure** | Backend-specific exec implementations |

**ADR references:** None yet.
**ADR gaps:** None identified.

## Tasks

### T5.1: Exec engine core

**Description:** Implement the exec engine that starts a process in the sandbox, streams stdout/stderr, enforces timeout and output limits, and returns the result.

**Acceptance criteria:**
- [ ] Command starts in sandbox with specified cwd
- [ ] stdout/stderr streamed to caller
- [ ] Timeout kills command and sets `timedOut: true`
- [ ] Output truncated at maxOutputBytes with `truncated: true`
- [ ] Exit code captured accurately
- [ ] Duration measured from start to exit

**Verification:**
- [ ] `go build ./pkg/exec/...`
- [ ] Unit tests for timeout and truncation logic
- [ ] Integration test: command runs and returns exit code

**Files:** `pkg/exec/engine.go`
**Size:** M
**Depends on:** F8 (Session Lifecycle), F3 (Fast Backend — process execution)

### T5.2: Exec API endpoint

**Description:** Implement `POST /v1/sandboxes/{id}/exec` endpoint. Accepts exec request, delegates to engine, streams response.

**Acceptance criteria:**
- [ ] Endpoint accepts exec request JSON
- [ ] Streaming stdout/stderr via SSE or chunked transfer
- [ ] Response includes exitCode, durationMs, stdout, stderr, truncated, timedOut
- [ ] Timeout in HTTP layer (request doesn't hang indefinitely)
- [ ] Error responses are actionable per SPEC.md §28

**Verification:**
- [ ] `go build ./cmd/pi-sandboxd/...`
- [ ] Integration test: exec endpoint returns correct response

**Files:** `pkg/api/sandbox_exec.go`
**Size:** S
**Depends on:** T5.1 (exec engine)

### T5.3: CLI exec command

**Description:** Implement `pi box exec <name> -- <cmd>` with all flags (cwd, timeout, max-output, memory, cpu, network, json).

**Acceptance criteria:**
- [ ] `pi box exec demo -- pnpm test` runs command
- [ ] `--cwd /workspace` sets working directory
- [ ] `--timeout 60s` sets timeout
- [ ] `--max-output 8MiB` sets output limit
- [ ] `--json` produces JSON output
- [ ] Streaming output displayed in real-time for interactive use

**Verification:**
- [ ] `go build ./cmd/pi/...`
- [ ] Integration test: CLI exec works with daemon

**Files:** `cmd/pi/box/exec.go`
**Size:** S
**Depends on:** T5.2 (exec API endpoint)

## Verification Plan

- [ ] `go build ./pkg/exec/...` succeeds
- [ ] Exec engine handles timeout correctly
- [ ] Exec engine handles output truncation correctly
- [ ] Exec API endpoint streams response
- [ ] CLI exec command works end-to-end
- [ ] Benchmark: warm exec p50 < 10ms (fast mode)

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Exec streaming protocol not specified | §9 Interface Contract | Add: "stdout/stderr streamed via SSE or chunked transfer" |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Interactive shell mode (separate `pi box shell` command)
- Background/daemon command execution (all exec is synchronous)
- Command history within exec (handled by F10: Logs & History)
- Multi-command pipelines (single command per exec call)
