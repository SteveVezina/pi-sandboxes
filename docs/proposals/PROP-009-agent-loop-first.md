# PROP-009: Agent-Loop-First Re-Aim — Rename Sessions to Sandboxes, One Output Channel, Host Decoupling

## Status
🟡 Proposed (awaiting acceptance)

> Note: PROP-006 is still 🟡 Proposed. This PROP was explicitly requested by the human
> after a platform-landscape review (Cloudflare Sandboxes GA, Vercel Sandbox GA, E2B,
> Modal, Daytona) and is independent of PROP-006 (templates); treated as a permitted
> exception to the "no new PROP while unapproved proposals exist" rule.

## Block Spec Reference
`SPEC.md` §1 Project thesis, §2 Primary goals, §5 Core Concepts, §6 Features (F6, F8, F9, F26), §7 Acceptance Criteria (AC-8, AC-28, AC-29, performance ACs), §12 High-level architecture, §15 Local filesystem layout, §16 Workspace model, §17 Cache model, §20 Daemon API, §23 Secrets model, §24 Network model, §25 Snapshot and rollback, §36 GUI requirements

## Sources

2026-07 platform landscape review of sandbox runtimes for AI agents:

- Cloudflare Sandboxes GA (2026-04): "agents have their own computers" — agent loop runs
  inside the sandbox; core API shrunk to exec / streaming files / processes / tunnels /
  snapshots; code interpreter, git, and terminal demoted to helpers (2026-06 deprecation
  guide). Egress proxy injects credentials at the network layer; the agent never sees
  the token.
- Vercel Sandbox GA: Firecracker microVMs, persistence by default, runtime-updatable
  egress network policies, credentials brokering, single deliverable path (git push or
  preview URL).
- E2B / Modal / Daytona: same convergence — small exec/files/process core, snapshot as
  warm-start mechanism, no host filesystem coupling by construction.

## Problem

Three related drifts from where the industry landed:

1. **Positioning drift.** The block spec models a "sandbox session" as a workspace
   toolbox that a host-side caller drives exec-by-exec. The converged industry model is
   the inverse: the sandbox is the agent's computer — the autonomous agent loop runs
   *inside* it, and the host is only a supervisor over HTTP. Our own lifecycle events
   (`pi.run.started` from `agent_start` in `.pi/block.yaml`) already assume the inside
   model, but no feature makes it first-class.

2. **Too many output channels.** F6/F9 give five overlapping ways to move data out:
   `files` read, `artifacts pull`, `diff`, `patch`, and snapshot export. Platforms have
   two: a files API for debugging and one deliverable channel. Every extra channel is
   exfiltration surface that the policy engine must reason about.

3. **Host filesystem coupling.** Sandboxes bind-mount host directories
   (`~/.pi-box/caches/...` per §17), snapshots live as loose per-sandbox host
   directories (§25), and secrets sit on host disk (§23). A poisoned cache written by
   one sandbox flows into the next through the host. Cloud platforms have zero host
   coupling by definition; a local-first runtime should get as close as the local
   substrate allows.

Separately, the term **"session"** is used for the sandbox lifecycle concept itself
("sandbox session", F8 "Session Lifecycle", GUI "Sessions" nav) while the daemon API
already says `/v1/sandboxes`. Two names for one concept, and "session" now collides
with three legitimate connection-scoped uses (vsock exec session, interactive shell
session, MCP protocol session).

## Requested By

2026-07-14: User asked whether the block targets the right end goal ("we want the
sandbox to run autonomous agent loop, not sure we need that much way to exfiltrate
data... we are still too much attached to the host filesystem"), requested a comparison
with Cloudflare/Vercel/etc., and requested renaming sessions to sandboxes.

## Proposed Amendment

Five separable amendments. Each can be accepted or rejected independently; A has no
dependency on B–E.

### A. Rename "session" → "sandbox" (terminology)

The lifecycle concept is a **sandbox**, full stop. "Session" is reserved for
connection-scoped state only.

Copy-paste-ready block-spec edits:

- §5 Core Concepts: rename the "Sandbox session" term to "Sandbox"; reword "Runtime
  mode: the isolation backend used for a session" to "... used for a sandbox".
- §2 goals 4–5: "Keep sandboxes warm and long-lived." / "Make command execution inside
  existing sandboxes extremely fast."
- §4 core-design flow: "create sandbox once → run many exec calls through the same
  sandbox → destroy when the sandbox expires".
- §6: F8 "Session Lifecycle" → "Sandbox Lifecycle"; F9 "...from sandbox sessions" →
  "...from sandboxes"; F26 "GUI Session Operations" → "GUI Sandbox Operations".
- §7: AC-8 heading and body, AC-28/AC-29 ("create a sandbox", "Dashboard lists recent
  and active sandboxes"), performance ACs ("New warm sandbox assignment < 100ms").
- §12: "session manager" component → "sandbox manager".
- §20 API table wording: "Create sandbox session" → "Create sandbox", etc. (paths
  already `/v1/sandboxes`, unchanged).
- Stream frame field `session_id` → `sandbox_id` (protocol change; see Impact).
- §16/§17: "per-session writable overlay" → "per-sandbox writable overlay";
  "GUI-launched sessions" → "GUI-launched sandboxes".
- §36 GUI: left navigation "Sessions" → "Sandboxes"; "session detail" → "sandbox
  detail"; "return session list" → "return sandbox list".

Explicitly **kept as "session"** (connection-scoped, correct usage):

| Usage | Where | Why kept |
|-------|-------|----------|
| vsock exec protocol session | `pkg/runtime/microvm/protocol.go`, `vsock.go` | Guest transport connection, not lifecycle |
| Interactive shell session | `pkg/api/sandbox_shell.go`, `box shell` | A shell attachment to a sandbox |
| MCP protocol session | `pkg/mcp/server.go` | MCP spec terminology |

### B. Agent-in-sandbox becomes first-class (new feature)

Add feature **F29 Agent Run** (F27 is GUI Settings; F28 is reserved by pending
PROP-006): `box run <agent> [--repo <url>] [--prompt ...]` starts
the autonomous agent loop *inside* the sandbox; the daemon streams lifecycle and agent
events out (`pi.run.started`, `pi.run.completed` already specified in `.pi/block.yaml`).
The host never drives the loop exec-by-exec; `exec` remains for setup, debugging, and
non-agent use.

### C. One deliverable channel (amend F6/F9, §20)

- Deliverables (artifacts, patch) leave through a single output endpoint
  (`POST /v1/sandboxes/{id}/output` locally; maps to Workspaces `POST /output` when
  attached to the Pi platform). `pi.artifact.delivered` fires only on this channel.
- `diff` and `patch` remain — the patch is the coding-agent deliverable — but as
  read-only views over the workspace, routed through the output channel for export.
- `files read` is reclassified as a debug/inspection API, not an export path.
- Snapshot is a warm-start/rollback mechanism only (§25); remove snapshot-as-export.
- Git push happens from inside the sandbox through the egress proxy (E) with a scoped
  token the sandbox never holds.

### D. Host filesystem decoupling (amend §15, §17, §23, §25)

- **Caches (§17):** no host bind mounts into sandboxes. Caches become daemon-managed
  volumes (per runtime backend) or snapshot layers baked into templates. The
  "shared read-only cache plus per-sandbox writable overlay" note in §17 becomes the
  required model, not a later preference.
- **Snapshots (§25):** content-addressed store under `~/.pi-box/snapshots/` owned by
  the daemon, not loose per-sandbox directories writable by runtime code.
- **Secrets (§23):** no plaintext secrets on host disk; secrets exist only as
  egress-proxy injection rules (E). Remove `~/.pi-box/secrets/` from §15.
- `~/.pi-box` remains for control-plane state only: socket, config, contexts,
  templates, sandbox metadata, content-addressed snapshots.

### E. Egress proxy with credential injection (new feature, amend §23/§24)

Add feature **F30 Egress Proxy**: outbound traffic from a sandbox routes through a
daemon-owned proxy that (a) enforces the network policy/allowlist at runtime and
(b) injects credentials (git tokens, registry auth) into approved outbound requests.
The agent never sees the token. This is the pattern Cloudflare and Vercel both shipped
and directly serves the §8 security model ("no secrets in sandbox").

## Rationale

- Matches the converged industry shape: small core (exec, streaming files, processes,
  logs, snapshot, network) + agent-runs-inside + one deliverable channel.
- The real exfiltration risk is network egress, not `box diff`; E addresses it
  structurally while C shrinks the surface the policy engine must reason about.
- D removes the only path by which one sandbox can affect another (shared writable
  host cache) and the largest host-trust assumption.
- A costs little now (SDKs already clean, API paths already `/v1/sandboxes`) and gets
  more expensive every release the split terminology ships.

## Impact

| Area | Impact |
|------|--------|
| `SPEC.md` | Edits across §2, §4–§7, §12, §15–§17, §20, §23–§25, §36; new F29, F30 |
| Wire protocol | `session_id` → `sandbox_id` in stream frames — breaking; acceptable pre-1.0, single release note |
| Daemon API | New `POST /v1/sandboxes/{id}/output`; snapshot export endpoint removed |
| Code | `pkg/session/` → `pkg/sandbox/` (mechanical; ~46 identifier references in `pkg/api`, `pkg/daemon`, `cmd/`); microvm/shell/MCP "session" untouched |
| CLI | Help text only ("Create a new sandbox session" → "Create a new sandbox"); command names unchanged |
| GUI | `apps/gui/src/main.tsx` (~161 mentions), `styles.css` (~89), `api.ts` (1); nav label "Sessions" → "Sandboxes" |
| SDKs | No rename impact (already session-free); new `run()` and output-channel methods for F29/C |
| Docs | `README.md`, `ARCHITECTURE.md`, `AGENTS.md` sweeps |
| Existing features | F6/F9 reshaped (C); §17 cache implementation replaced (D); F8/F26 renamed only |

Amendment A is separable and mechanical; B–E change behavior and should cascade into
feature specs before implementation.

## Assumption While Awaiting Acceptance

Code continues to use current terminology and channels. No new features are built on
snapshot-as-export, host cache bind mounts, or host-disk secrets in the interim.

## Acceptance Criteria Changes

- AC-8 renamed "Sandbox Lifecycle"; wording "Multiple exec calls reuse the same
  sandbox".
- New AC for F29: agent loop runs inside the sandbox; host receives `pi.run.started` /
  `pi.run.completed` without driving the loop.
- New AC for F30: outbound request to an allowlisted host succeeds with injected
  credential; the credential is not readable from inside the sandbox; non-allowlisted
  egress is denied.
- AC for C: artifact and patch delivery emit `pi.artifact.delivered` only via the
  output channel; snapshot export path absent.
- AC for D: no sandbox has a writable bind mount of any host directory; cache reuse
  works via read-only shared layer + per-sandbox overlay.

## Cascade Required on Acceptance

1. Apply spec edits (A first — mechanical sweep of `SPEC.md`, `README.md`,
   `ARCHITECTURE.md`, `AGENTS.md`).
2. Rename `pkg/session/` → `pkg/sandbox/`; update `session_id` frame field; GUI sweep.
3. Author feature specs for F29 (agent run) and F30 (egress proxy).
4. Rework feature spec for F6/F9 output-channel consolidation and §17 cache model.
5. Reset task statuses for any open F6/F9/F17-adjacent tasks.

## Implementation Blocked?

No. Amendment A can be implemented immediately on acceptance. B–E require feature
specs (step 3–4 above) before implementation.
