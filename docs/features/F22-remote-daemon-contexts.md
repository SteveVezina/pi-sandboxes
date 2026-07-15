# F22: Remote Daemon Contexts

> Source: `SPEC.md` §6 Features F22
> Status: ⚠️ Needs re-verify
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F22 | Remote Daemon Contexts | CLI context management for local and remote daemons, including `pi context create/use/list/inspect/delete` and context-aware `pi box` commands | M6 |

## Expanded Specification

Remote daemon contexts let a user switch the CLI between local and remote daemon targets. Context state is explicit, inspectable, and overrideable per command.

Per ADR-003, context state is stored in `~/.pi-box/contexts.yaml`. Each context has `target`, `transport`, and `auth.type`. The active context may be overridden per command with `--context <name>`.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-25.1: `pi context create workstation ssh://gpu-box.local` creates a remote context
- [x] AC-25.2: `pi context use workstation` switches the active context
- [x] AC-25.3: `pi context list` shows local and remote contexts
- [x] AC-25.4: `pi box create` uses the active context
- [x] AC-25.5: Commands can override the active context explicitly
- [x] AC-25.6: Contexts persist in `~/.pi-box/contexts.yaml`
- [x] AC-25.7: Context schema supports `target`, `transport`, and `auth.type`
- [x] AC-25.8: `--context <name>` overrides the active context

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi/context/` | Context CLI group |
| `cmd/pi/cli` | Global context override flag |
| `pkg/context/` | Context store |
| `cmd/pi/box` | Context-aware daemon client |
| `docs/decisions/ADR-003-remote-context-and-auth-model.md` | Context/auth model decision |

## Security Considerations

- Contexts must not store raw credentials in plaintext.
- Active context changes must be explicit and inspectable.
- Remote context configuration must not leak into sandbox workspaces.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F2: Daemon API | Internal feature | Target API |
| F23: Remote Daemon Transport & Auth | Internal feature | Remote access/auth |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | CLI context store and daemon client routing |

**ADR references:** ADR-003 (Remote Context and Auth Model).
**ADR gaps:** None.

## Tasks

### T22.1: Context store and CLI ✅

**Acceptance criteria:**
- [x] Create/use/list/inspect/delete commands work
- [x] Active context persists in user config
- [x] Contexts persist in `~/.pi-box/contexts.yaml`
- [x] Context schema supports `target`, `transport`, and `auth.type`

**Verification:**
- [x] Unit tests for context store (`tests/context/store_test.go`)
- [x] CLI integration tests for context commands (`tests/context/cli_test.go`)

**Files:** `cmd/pi/context/commands.go`, `pkg/context/store.go`, `tests/context/store_test.go`, `tests/context/cli_test.go`
**Size:** M
**Depends on:** F2

### T22.2: Context-aware command routing ✅

**Acceptance criteria:**
- [x] `pi box` commands use active context
- [x] Per-command context override works
- [x] Local context remains the default
- [x] `--context <name>` overrides active context

**Verification:**
- [x] Integration test: active context routes API calls (`tests/context/resolve_test.go`)
- [x] Integration test: explicit override wins (`tests/context/cli_test.go`)

**Files:** `cmd/pi/cli/root.go`, `cmd/pi/box/box.go`
**Size:** M
**Depends on:** T22.1, F23

## Verification Plan

- [x] Context commands work
- [x] `pi box` uses active context
- [x] Override behavior is deterministic
- [x] Context schema validation works

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| — | — | — |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| — | — | — |

## Out of Scope

- Implementing remote transport/auth internals
