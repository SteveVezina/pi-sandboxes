# F29: Agent Run

> Source: `SPEC.md` §6 Features F29
> Status: 🔵 In progress — run state model + API + lifecycle events + CLI wiring done (2026-08-31); real in-sandbox agent process launch blocked on the "agent entrypoint resolution" spec gap
> Category: Service-layer / CLI

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F29 | Agent Run | `pi-box run <agent>` starts the autonomous agent loop inside the sandbox and streams lifecycle and agent events to the host supervisor | M8 |

## Expanded Specification

Agent Run makes the sandbox the agent's computer. The autonomous agent loop runs inside the sandbox; the host supervises over the daemon API and receives lifecycle and agent events. The host does not drive the loop exec-by-exec.

`exec` remains available for setup, debugging, and non-agent use. `pi-box run <agent> [--repo <url>] [--prompt ...]` creates or selects a sandbox, prepares the workspace, launches the configured agent entrypoint inside the sandbox, and streams events until completion or cancellation.

The daemon emits service-level lifecycle events already defined in `.pi/block.yaml` via `pkg/events` (ADR-007 — slog sink always on, `--events-webhook` opt-in). F29 emits `pi.run.started` / `pi.run.completed` through `events.Emit`; the emitter and envelope already exist.
- `pi.run.started`
- `pi.run.completed`
- `pi.artifact.delivered` after output-channel delivery

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-31.1: `pi-box run <agent> [--repo <url>] [--prompt ...]` starts the autonomous agent loop inside the sandbox *(2026-08-31: `pi-box run` command + `POST /v1/sandboxes/{id}/agent-run` + run state model wired; the actual in-sandbox agent process is not launched — needs agent entrypoint resolution, see Spec Gaps. `--repo` prep also not wired to clone yet.)*
- [x] AC-31.2: The host receives `pi.run.started` and `pi.run.completed` lifecycle events without driving the loop exec-by-exec *(2026-08-31: `events.Emit` in `StartAgentRun` / `finishRun`; `AgentRunStore.UpdateState` rejects a second terminal transition so `pi.run.completed` fires exactly once. The CLI polls run state — it never drives the loop. Verified: `pkg/api/agent_run_internal_test.go`.)*
- [x] AC-31.3: `exec` remains available for setup, debugging, and non-agent use *(unchanged — `POST /v1/sandboxes/{id}/exec` untouched)*
- [x] AC-33.1: Artifacts and patches leave the sandbox only through `POST /v1/sandboxes/{id}/output` *(F9; `pi.artifact.delivered` already fires after delivery)*

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

### T29.1: Agent run API and state model ✅ *(2026-08-31)*

**Description:** Daemon state for starting, inspecting, and cancelling an agent run inside a sandbox.

**Acceptance criteria:**
- [x] API can start an agent run in a WARM sandbox — `StartAgentRun`
- [x] API rejects agent run start when sandbox is not runnable — `409` when not `WARM`; `409` when the sandbox already has an active run
- [x] Run state records `run_id`, `sandbox_id`, agent name, status, timestamps, and exit metadata — `sandbox.AgentRun`
- [x] `pi.run.started` and `pi.run.completed` emitted exactly once per run — `AgentRunStore.UpdateState` rejects transitions out of a terminal state; events emitted in `StartAgentRun` / `finishRun`

**Verification:**
- [x] Unit: terminal-transition rejection, exactly-once completed event, cancel path — `pkg/api/agent_run_internal_test.go`
- [x] API tests: start / reject-not-warm / reject-missing-agent / cancel-then-conflict

**Files:** `pkg/api/agent_run.go`, `pkg/sandbox/agent_run.go`
**Size:** M
**Depends on:** F8

**Note:** the `r.PathValue` calls in the previous scaffolding were dead (gorilla/mux routes) — switched to `mux.Vars`.

### T29.2: CLI command 🟡 *(2026-08-31 — command wired; repo prep + real agent launch pending)*

**Description:** `pi-box run <agent> [--repo <url>] [--prompt ...]`.

**Acceptance criteria:**
- [x] `pi-box run <agent>` is a top-level command (`cli.AddCommand(runCmd)`); creates a sandbox, starts the run, polls state to a terminal state, cancels on SIGINT
- [x] `--template` / `--mode` flags (default `python` / `fast`); `--prompt` forwarded in the start request
- [x] Non-zero / non-COMPLETED run propagates to CLI exit status; SIGINT exits 130
- [ ] `--repo` prepares the workspace before agent start — not wired to `POST /v1/sandboxes/{id}/clone` yet
- [ ] Event streaming (currently 1s polling; WebSocket stream is a later refinement)

**Verification:**
- [ ] CLI integration test with a mock agent — blocked on real agent launch

**Files:** `cmd/pi-box/box/run.go`
**Size:** M
**Depends on:** T29.1

### T29.3: Output channel integration ✅ *(2026-08-31 — satisfied by F9)*

**Description:** Route agent-produced patches/artifacts through F9 Output Delivery.

**Acceptance criteria:**
- [x] Agent patch/artifact delivery uses `POST /v1/sandboxes/{id}/output` — F9 is the only export path; nothing in the agent-run code adds a second one
- [x] `pi.artifact.delivered` fires only after successful output delivery — F9 T8.2/T8.3 (`events.Emit` after the copy succeeds, ADR-007)

**Verification:**
- [x] `pkg/api/events_internal_test.go` + F9 tests cover the delivery-event path; a mock-agent end-to-end variant belongs with the real agent launch

**Files:** `pkg/api/sandbox_output.go`
**Size:** S (folded into F9)
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
| **Agent entrypoint resolution is undefined.** `pi-box run <agent>` takes an agent *name* but the spec never says how a name maps to a runnable command / image / config inside the sandbox. Blocks the actual in-sandbox agent process launch (AC-31.1). | §6 F29, §12 | Specify: an agent registry (`~/.pi-box/agents/<name>.yaml` with `entrypoint`, `image?`, `env?`), or a convention (`/opt/agent/run` in the template image), or a `--cmd` passthrough. |
| How `--prompt` reaches the in-sandbox agent (env var? file? argv?) is unspecified. | §6 F29 | Tie to the entrypoint resolution above — e.g. `PI_AGENT_PROMPT` env + `PI_AGENT_PROMPT_FILE`. |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Should agent run cancellation be graceful-first with forced kill timeout across all runtimes? | F29, F8, F20 | ADR-NNN: Agent run supervision and cancellation |
| How does the host stream per-iteration agent events (not just the 5 lifecycle events)? SSE topic per run? WebSocket? The current CLI polls run state. | F29 | ADR-NNN: agent event stream transport (separate from ADR-007 lifecycle events) |

## Out of Scope

- Multi-agent orchestration
- Browser/computer-use agents
- Cloud scheduler
