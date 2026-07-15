# F29: Agent Run

> Source: `SPEC.md` §6 Features F29
> Status: 🟡 Spec written
> Category: Service-layer / CLI

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F29 | Agent Run | `pi-box run <agent>` starts the autonomous agent loop inside the sandbox and streams lifecycle and agent events to the host supervisor | M8 |

## Expanded Specification

Agent Run makes the sandbox the agent's computer. The autonomous agent loop runs inside the sandbox; the host supervises over the daemon API and receives lifecycle and agent events. The host does not drive the loop exec-by-exec.

`exec` remains available for setup, debugging, and non-agent use. `pi-box run <agent> [--repo <url>] [--prompt ...]` creates or selects a sandbox, prepares the workspace, launches the configured agent entrypoint inside the sandbox, and streams events until completion or cancellation.

The daemon emits service-level lifecycle events already defined in `.pi/block.yaml`:
- `pi.run.started`
- `pi.run.completed`
- `pi.artifact.delivered` after output-channel delivery

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-31.1: `pi-box run <agent> [--repo <url>] [--prompt ...]` starts the autonomous agent loop inside the sandbox
- [ ] AC-31.2: The host receives `pi.run.started` and `pi.run.completed` lifecycle events without driving the loop exec-by-exec
- [ ] AC-31.3: `exec` remains available for setup, debugging, and non-agent use
- [ ] AC-33.1: Artifacts and patches leave the sandbox only through `POST /v1/sandboxes/{id}/output`

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi-box/` | Adds `pi-box run <agent>` command |
| `pkg/api/` | Agent run start/stream/cancel endpoints |
| `pkg/runtime/` | Launches long-lived agent process inside sandbox |
| F9: Output Delivery | Agent deliverables use output channel |
| F30: Egress Proxy | Agent network credentials flow through proxy |

## Security Considerations

- Agent process inherits sandbox policy; no host filesystem coupling is added.
- Secrets are available only through F30 egress-proxy injection.
- Output delivery remains the only deliverable export path.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F8: Sandbox Lifecycle | Internal feature | Needs re-verify |
| F9: Output Delivery | Internal feature | Needs re-verify |
| F30: Egress Proxy | Internal feature | Spec written |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **CLI** | `pi-box run` command and flags |
| **Service-layer** | Daemon agent-run lifecycle and event streaming |
| **Runtime** | Start/monitor/cancel agent process inside sandbox |

**ADR references:** None yet.
**ADR gaps:** Agent run supervision and cancellation semantics may need an ADR if they affect multiple runtimes.

## Tasks

### T29.1: Agent run API and state model

**Description:** Specify and implement daemon state for starting, inspecting, streaming, and cancelling an agent run inside a sandbox.

**Acceptance criteria:**
- [ ] API can start an agent run in a WARM sandbox
- [ ] API rejects agent run start when sandbox is not runnable
- [ ] Run state records `run_id`, `sandbox_id`, agent name, status, timestamps, and exit metadata
- [ ] `pi.run.started` and `pi.run.completed` events are emitted exactly once per run

**Verification:**
- [ ] Unit tests for run state transitions
- [ ] API tests for start/reject/inspect/cancel

**Files:** `pkg/api/agent_run.go`, `pkg/daemon/agent_run.go`
**Size:** M
**Depends on:** F8

### T29.2: CLI command

**Description:** Add `pi-box run <agent> [--repo <url>] [--prompt ...]`.

**Acceptance criteria:**
- [ ] CLI starts an agent run and streams events
- [ ] `--repo` prepares the workspace before agent start
- [ ] `--prompt` passes initial prompt to the in-sandbox agent
- [ ] Non-zero agent exit propagates to CLI exit status

**Verification:**
- [ ] CLI integration test with mock agent

**Files:** `cmd/pi-box/run.go`
**Size:** M
**Depends on:** T29.1

### T29.3: Output channel integration

**Description:** Route agent-produced patches/artifacts through F9 Output Delivery.

**Acceptance criteria:**
- [ ] Agent patch delivery uses `POST /v1/sandboxes/{id}/output`
- [ ] Artifact delivery uses the same endpoint
- [ ] `pi.artifact.delivered` fires only after successful output delivery

**Verification:**
- [ ] Integration test: mock agent produces patch and artifact through output endpoint

**Files:** `pkg/api/sandbox_output.go`, `pkg/daemon/agent_run.go`
**Size:** M
**Depends on:** F9, T29.1

## Verification Plan

- [ ] `go test ./pkg/api ./pkg/daemon`
- [ ] CLI integration test for `pi-box run`
- [ ] Event stream test for `pi.run.started` and `pi.run.completed`
- [ ] Output-channel test for patch/artifact delivery

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| — | — | — |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Should agent run cancellation be graceful-first with forced kill timeout across all runtimes? | F29, F8, F20 | ADR-NNN: Agent run supervision and cancellation |

## Out of Scope

- Multi-agent orchestration
- Browser/computer-use agents
- Cloud scheduler
