# F27: GUI Settings and Diagnostics

> Source: `SPEC.md` §6 Features F27
> Status: 🟢 Implemented
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

- [x] AC-30.1: GUI can view and change active context
- [x] AC-30.2: GUI can set default template, runtime mode, and network mode preferences
- [x] AC-30.3: GUI displays runtime/backend availability from daemon diagnostics
- [x] AC-30.4: GUI exposes `pi-box system doctor` equivalent results
- [x] AC-30.5: GUI can export a support bundle containing daemon diagnostics, GUI logs, version metadata, and redacted configuration
- [x] AC-30.6: Daemon policy overrides conflicting GUI preferences

## Interface Impact

| Component | Impact |
|-----------|--------|
| `apps/gui/` | Settings, diagnostics, support bundle export |
| `~/.pi-box/config.yaml` | Stores GUI defaults and allowed folders |
| `GET /v1/system/status` | GUI-readable daemon/system status |
| `GET /v1/system/doctor` | GUI-readable doctor-equivalent diagnostics |
| `GET /v1/system/runtimes` | GUI-readable runtime/backend availability |
| `GET /v1/support-bundle` | Redacted GUI support bundle payload |
| `GET /v1/contexts` | GUI-readable active context and configured contexts |
| `POST /v1/contexts/use` | GUI active context switch |
| `pi-box system doctor` behavior | Baseline for doctor-equivalent diagnostics |

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
| F24: Cross-Platform GUI Workbench | Internal feature | Implemented |
| F25: GUI Workspace Authorization | Internal feature | Implemented |
| ADR-004 | Architecture decision | Accepted |

## Implementation Approach

Implement settings as GUI preference management plus read-only daemon diagnostics. Doctor-equivalent output may use daemon API diagnostics directly or shell out to `pi-box system doctor` only where no structured API exists yet. Support bundle generation collects GUI logs, daemon diagnostics, version metadata, and redacted configuration.

**ADR references:** ADR-003 (Remote Context and Auth Model), ADR-004 (GUI Workbench Architecture and Trust Boundaries).
**ADR gaps:** None.

## Tasks

### T27.1: Context and defaults settings ✅

**Description:** Add settings for active context and GUI default template/runtime/network preferences.

**Acceptance criteria:**
- [x] GUI can view and change active context
- [x] GUI can set default template preference
- [x] GUI can set default runtime mode preference
- [x] GUI can set default network mode preference
- [x] GUI uses the default network mode for sandbox command execution unless changed in the sandbox view

**Verification:**
- [x] `npm run build` passes in `apps/gui`
- [x] API tests cover context list/use endpoints
- [x] Live smoke verifies GUI reads active context from daemon context store
- [x] Build smoke verifies settings persistence and controls
- [x] Component smoke verifies settings controls
- [x] Build smoke verifies network preference is passed into GUI exec requests

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`, `apps/gui/src/styles.css`, `pkg/api/contexts.go`, `tests/api/contexts_test.go`
**Size:** M
**Depends on:** F24

### T27.2: Diagnostics and runtime availability ✅

**Description:** Display daemon health, runtime/backend availability, and doctor-equivalent diagnostics.

**Acceptance criteria:**
- [x] Runtime/backend availability is visible
- [x] Doctor-equivalent diagnostic results are available through the GUI support bundle and daemon route
- [x] Policy errors override conflicting GUI preferences

**Verification:**
- [x] API tests cover status, doctor, runtimes, and support bundle routes
- [x] Manual smoke against local daemon diagnostic output

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`, `pkg/api/system.go`, `pkg/daemon/router.go`, `tests/api/system_test.go`
**Size:** M
**Depends on:** T27.1, F10, F19

### T27.3: Support bundle export ✅

**Description:** Export a redacted support bundle for debugging GUI and daemon issues.

**Acceptance criteria:**
- [x] Bundle includes daemon diagnostics
- [x] Bundle includes GUI-visible diagnostic payload
- [x] Bundle includes GUI renderer logs
- [x] Bundle includes version metadata
- [x] Bundle includes redacted configuration
- [x] Secrets and bearer tokens are redacted

**Verification:**
- [x] API tests verify home-path redaction
- [x] `npm run build` verifies GUI support bundle export includes client log entries
- [x] Support bundle live smoke verifies redacted payload

**Files:** `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`, `pkg/api/system.go`, `tests/api/system_test.go`
**Size:** M
**Depends on:** T27.2

## Verification Plan

- [x] Settings persistence is implemented in GUI local preferences and build-verified
- [x] Diagnostic tests cover runtime availability route and support bundle redaction
- [x] Support bundle tests verify expected payload and redaction

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

- Template source/lineage/validation state in the default-template picker — F28 (PROP-006) extends the picker once the richer template metadata exists.
- Enterprise identity management
- Hosted support upload service
