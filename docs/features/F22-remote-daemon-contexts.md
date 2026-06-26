# F22: Remote Daemon Contexts

> Source: `SPEC.md` §6 Features F22
> Status: 🟡 Spec written
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F22 | Remote Daemon Contexts | CLI context management for local and remote daemons, including `pi context create/use/list/inspect/delete` and context-aware `pi box` commands | M6 |

## Expanded Specification

Remote daemon contexts let a user switch the CLI between local and remote daemon targets. Context state is explicit, inspectable, and overrideable per command.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-25.1: `pi context create workstation ssh://gpu-box.local` creates a remote context
- [ ] AC-25.2: `pi context use workstation` switches the active context
- [ ] AC-25.3: `pi context list` shows local and remote contexts
- [ ] AC-25.4: `pi box create` uses the active context
- [ ] AC-25.5: Commands can override the active context explicitly

## Interface Impact

| Component | Impact |
|-----------|--------|
| `cmd/pi/context/` | Context CLI group (new — to be created) |
| `cmd/pi/cli` | Global context override flag |
| `pkg/context/` | Context store (new — to be created) |
| `cmd/pi/box` | Context-aware daemon client |

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

## Tasks

### T22.1: Context store and CLI

**Acceptance criteria:**
- [ ] Create/use/list/inspect/delete commands work
- [ ] Active context persists in user config

**Verification:**
- [ ] Unit tests for context store
- [ ] CLI integration tests for context commands

**Files:** `cmd/pi/context/commands.go (new — to be created)`, `pkg/context/store.go (new — to be created)`
**Size:** M
**Depends on:** F2

### T22.2: Context-aware command routing

**Acceptance criteria:**
- [ ] `pi box` commands use active context
- [ ] Per-command context override works
- [ ] Local context remains the default

**Verification:**
- [ ] Integration test: active context routes API calls
- [ ] Integration test: explicit override wins

**Files:** `cmd/pi/cli/root.go`, `cmd/pi/box/box.go`
**Size:** M
**Depends on:** T22.1, F23

## Verification Plan

- [ ] Context commands work
- [ ] `pi box` uses active context
- [ ] Override behavior is deterministic

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Context config schema is not specified | §31 M6, §34 Config file | PROP-003 |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Context file schema and credential references | F22, F23 | ADR for remote context model |

## Out of Scope

- Implementing remote transport/auth internals
