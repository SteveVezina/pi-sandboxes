# F27: GUI Settings and Diagnostics

> Source: `SPEC.md` §6 Features F27
> Status: 🟡 Spec written
> Category: Client / Operations

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F27 | GUI Settings and Diagnostics | GUI controls for daemon connection, active context, default template/runtime mode/network policy, engine health, doctor output, and support bundle export | M7 |

## Expanded Specification

The GUI settings surface manages client preferences and operational diagnostics. Users can view and change the active context, set GUI defaults for template/runtime/network mode, inspect daemon/runtime availability, view doctor-style diagnostics, manage allowed folders, and export a redacted support bundle.

Settings are client preferences. Daemon policy remains authoritative.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [ ] AC-30.1: GUI can view and change active context
- [ ] AC-30.2: GUI can set default template, runtime mode, and network mode preferences
- [ ] AC-30.3: GUI displays runtime/backend availability from daemon diagnostics
- [ ] AC-30.4: GUI exposes `pi system doctor` equivalent results
- [ ] AC-30.5: GUI can export a support bundle containing daemon diagnostics, GUI logs, version metadata, and redacted configuration
- [ ] AC-30.6: Daemon policy overrides conflicting GUI preferences

## Interface Impact

| Component | Impact |
|-----------|--------|
| `apps/gui/` | Settings, diagnostics, support bundle export (new — to be created) |
| `~/.pi/config.yaml` | Stores GUI defaults and allowed folders |
| `pi-sandboxd` API | Source of health, runtime availability, and policy errors |
| `pi system doctor` behavior | Baseline for doctor-equivalent diagnostics |

## Security Considerations

- Support bundles must redact credentials, bearer tokens, environment secrets, and private paths where appropriate.
- GUI preferences must not override daemon policy.
- Remote auth failures must stay actionable and must not fall back to unauthenticated access.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F10: System Commands | Internal feature | Implemented |
| F17: Policy Enforcement | Internal feature | Implemented |
| F19: Runtime Selection & Fallback | Internal feature | Implemented |
| F22: Remote Daemon Contexts | Internal feature | Implemented |
| F24: Cross-Platform GUI Workbench | Internal feature | Spec written |
| F25: GUI Workspace Authorization | Internal feature | Spec written |
| ADR-004 | Architecture decision | Accepted |

## Implementation Approach

Implement settings as GUI preference management plus read-only daemon diagnostics. Doctor-equivalent output may use daemon API diagnostics directly or shell out to `pi system doctor` only where no structured API exists yet. Support bundle generation collects GUI logs, daemon diagnostics, version metadata, and redacted configuration.

**ADR references:** ADR-003 (Remote Context and Auth Model), ADR-004 (GUI Workbench Architecture and Trust Boundaries).
**ADR gaps:** None.

## Tasks

### T27.1: Context and defaults settings ⚠️

**Description:** Add settings for active context and GUI default template/runtime/network preferences.

**Acceptance criteria:**
- [ ] GUI can view and change active context
- [ ] GUI can set default template preference
- [ ] GUI can set default runtime mode preference
- [ ] GUI can set default network mode preference

**Verification:**
- [ ] Unit tests for settings persistence
- [ ] Component tests for settings controls

**Files:** `apps/gui/ (new — to be created)`
**Size:** M
**Depends on:** F24

### T27.2: Diagnostics and runtime availability ⚠️

**Description:** Display daemon health, runtime/backend availability, and doctor-equivalent diagnostics.

**Acceptance criteria:**
- [ ] Runtime/backend availability is visible
- [ ] Doctor-equivalent diagnostic results are visible
- [ ] Policy errors override conflicting GUI preferences

**Verification:**
- [ ] Mock daemon diagnostic tests
- [ ] Manual smoke against local daemon doctor output

**Files:** `apps/gui/ (new — to be created)`
**Size:** M
**Depends on:** T27.1, F10, F19

### T27.3: Support bundle export ⚠️

**Description:** Export a redacted support bundle for debugging GUI and daemon issues.

**Acceptance criteria:**
- [ ] Bundle includes daemon diagnostics
- [ ] Bundle includes GUI logs
- [ ] Bundle includes version metadata
- [ ] Bundle includes redacted configuration
- [ ] Secrets and bearer tokens are redacted

**Verification:**
- [ ] Unit tests for redaction
- [ ] Support bundle smoke test

**Files:** `apps/gui/ (new — to be created)`
**Size:** M
**Depends on:** T27.2

## Verification Plan

- [ ] Settings persistence tests cover context/defaults updates
- [ ] Diagnostic tests cover runtime availability and policy conflict display
- [ ] Support bundle tests verify expected files and redaction

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

- Enterprise identity management
- Hosted support upload service
