# PROP-010: Agent Entrypoint Resolution

## Status
🟡 Proposed

⚠️ **BLOCKING** — F29 Agent Run (AC-31.1) cannot launch the in-sandbox
agent process without this. No other unapproved PROPs exist.

## Block Spec Reference
`SPEC.md` §6 Features (F29), §7 Acceptance Criteria (AC-31), §12 CLI
requirements, §34 Configuration file

## Problem

F29 Agent Run is specified as "`pi-box run <agent>` starts the autonomous
agent loop inside the sandbox". The run state model, the
`/v1/sandboxes/{id}/agent-run` API, the `pi.run.started`/`pi.run.completed`
events, and the `pi-box run` CLI are all implemented (T29.1). But the spec
never says **how an agent name maps to a runnable process inside the
sandbox** — no entrypoint, no image contract, no argument/prompt
convention. So `superviseRun` currently drives the run through its states
without launching anything.

`SPEC.md` §12 shows `pi-box run <agent> [--repo <url>] [--prompt ...]` but
§6/§39 never define what `<agent>` resolves to.

## Proposed Amendment

### 1. Add §12.x "Agent entrypoint resolution" (after the `pi-box run` CLI entry)

> An **agent** is a named, locally-registered runnable. `pi-box run
> <name>` resolves `<name>` in this order:
>
> 1. **Local agent registry** — `~/.pi-box/agents/<name>.yaml`:
>
>    ```yaml
>    name: my-agent
>    entrypoint: ["/opt/agent/run"]     # argv, run inside the sandbox
>    workdir: /workspace                # optional, default /workspace
>    env:                               # optional static env
>      AGENT_MODEL: claude-sonnet
>    promptEnv: PI_AGENT_PROMPT         # optional; --prompt is placed here
>    promptFile: /workspace/.pi/prompt  # optional; --prompt is also written here
>    image: ""                          # optional; overrides the template image for this run
>    ```
>
> 2. **Template convention** — if no registry entry exists and the
>    sandbox's template image provides `/opt/agent/run` (executable), that
>    is the entrypoint with default `workdir`, `promptEnv=PI_AGENT_PROMPT`.
>
> 3. **`--cmd` passthrough** — `pi-box run --cmd "<command>" <name>` runs
>    the given shell command as the agent (name is used only for the run
>    label and events). Bypasses 1 and 2.
>
> If none resolve, `pi-box run` fails with: *"no agent 'X' — add
> ~/.pi-box/agents/X.yaml, use a template image with /opt/agent/run, or
> pass --cmd"*.

### 2. Add to §6 Features F29 description

> The agent process is launched as a single long-lived `exec` inside the
> sandbox (`entrypoint` argv, `workdir`, merged env). `--repo <url>` is
> cloned into `/workspace` before launch. `--prompt` is delivered via
> `promptEnv` and/or `promptFile`. The host supervises via the run state
> API and lifecycle events; it does not drive the loop.

### 3. Add AC-31.4 and AC-31.5 (§7, after AC-31.3)

- [ ] AC-31.4: `pi-box run <name>` resolves the entrypoint from
  `~/.pi-box/agents/<name>.yaml`, else a template `/opt/agent/run`, else
  `--cmd`; an unresolvable name fails with actionable guidance
- [ ] AC-31.5: `--repo` is cloned into `/workspace` and `--prompt` is
  delivered via `promptEnv`/`promptFile` before the agent process starts

### 4. Add to §34 Configuration file

> `~/.pi-box/agents/` holds agent registry YAML files (one per agent).
> Agent definitions are local trusted config, like templates. A registry
> entry never grants a sandbox more than its policy allows (network,
> mounts, resources unchanged).

### Non-goals

- No hosted agent marketplace / remote agent fetch.
- No multi-agent orchestration (still F29 out-of-scope).
- No agent-specific network or filesystem escapes — daemon policy is
  unchanged and authoritative.
- The per-iteration **agent event stream** (SSE/WebSocket of the agent's
  own frames) is a separate concern; this PROP covers only launching the
  process and the coarse `pi.run.*` lifecycle events.

## Rationale

Templates already established the "local, daemon-owned, YAML, trusted"
pattern (F5/F28). Agents are the same shape one level up: a named runnable
the user registers locally. The three-tier resolution (registry →
template convention → `--cmd`) covers the common cases without a registry
service: a power user writes one YAML; a template author bakes
`/opt/agent/run` into an image; a quick experiment uses `--cmd`.

## Impact

Features affected:
- F29 Agent Run: `superviseRun` launches the resolved entrypoint;
  `--repo`/`--prompt` prep; new ACs.
- F1 CLI Entry Point: `pi-box run` gains `--cmd`.
- F5 Template System: documents the optional `/opt/agent/run` convention.
- F17 Policy Enforcement: agent runs inherit sandbox policy unchanged
  (explicit AC that a registry entry can't relax it).

ADRs likely required:
- Agent run supervision + cancellation semantics (graceful-first + forced
  kill timeout across runtimes) — already an open ADR gap in F29.

Implementation blocked:
- Yes for the real in-sandbox agent launch (AC-31.1, AC-31.4, AC-31.5).
- No for the already-shipped run state model, API, events, and CLI
  wiring.

## Assumption While Awaiting Acceptance

`pi-box run` continues to create the sandbox, register the run, and emit
`pi.run.started`/`pi.run.completed`, but does not launch a process. The
`superviseRun` stub stays. No `~/.pi-box/agents/` directory is created or
read.

## Requested By

2026-08-31: surfaced during F29 T29.1 implementation — the run
machinery works end to end but has nothing to run. User goal: "finish all
possible tasks" — this is the block-spec amendment that unblocks the last
F29 work.

## Cascade Required on Acceptance

1. `SPEC.md` §12: add "Agent entrypoint resolution".
2. `SPEC.md` §6 F29: expand the description (process launch, repo/prompt prep).
3. `SPEC.md` §7: add AC-31.4, AC-31.5.
4. `SPEC.md` §34: add `~/.pi-box/agents/`.
5. `docs/features/F29-agent-run.md`: close the "agent entrypoint
   resolution" spec gap; author T29.4 (entrypoint resolution + launch)
   and T29.5 (repo/prompt prep); reset AC-31.1 note.
6. `docs/features/F07-template-system.md`: note the `/opt/agent/run`
   convention.
7. `docs/features/INDEX.md` + `docs/proposals/INDEX.md`.
8. Add an ADR for agent run supervision + cancellation.

## Implementation Blocked?

Yes for the in-sandbox agent launch. Pending human review.
