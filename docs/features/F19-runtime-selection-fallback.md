# F19: Runtime Selection & Fallback

> Source: `SPEC.md` §6 Features F19
> Status: 🟢 Implemented
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F19 | Runtime Selection & Fallback | Runtime detection and backend selection across fast, compat, secure, and future microVM modes, including fallback to compat mode when secure mode is unavailable or incompatible | M4 |

## Expanded Specification

Runtime selection centralizes backend discovery, explicit mode handling, auto-selection, and fallback. It must make backend decisions visible in system doctor output, logs, and command history.

Fallback cannot weaken policy silently. If fallback is allowed, the user-visible result must explain what happened and which backend actually ran.

## Acceptance Criteria

Mapped from `SPEC.md` § Acceptance Criteria:

- [x] AC-22.1: `pi-box system doctor` reports available runtime backends
- [x] AC-22.2: Backend selection honors explicit `--mode` requests
- [x] AC-22.3: Auto-selection prefers an available compatible backend based on trust/config
- [x] AC-22.4: Secure-mode startup failure can fall back to compat mode when policy permits
- [x] AC-22.5: Fallback decisions are visible in logs/history

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/runtime/` | Runtime registry/detection |
| `pkg/system/` | Doctor backend availability |
| `pkg/logs/` | Fallback decisions recorded |
| `cmd/pi-box/box` | Explicit and auto runtime mode flags |
| `pi-sandboxd` API | Create defaults to the detected best available runtime and rejects unavailable explicit runtime modes |

## Security Considerations

- Fallback must not silently relax security policy.
- Explicit `--mode secure` failures require actionable guidance.
- Backend availability checks must not expose secrets or host-specific sensitive details.

## Dependencies

| Dependency | Type | Status |
|-----------|------|--------|
| F3: Fast Backend | Internal feature | Runtime option |
| F15: Compat Backend | Internal feature | Runtime option/fallback |
| F18: Secure Backend | Internal feature | Runtime option |
| F17: Policy Enforcement | Internal feature | Fallback guardrails |

## Implementation Approach

| Category | Meaning |
|----------|---------|
| **Service-layer** | Runtime detection/selection module |

## Tasks

### T19.1: Runtime registry and detection

**Acceptance criteria:**
- [x] Detect fast, compat, secure, and future microVM backend availability
- [x] `pi-box system doctor` reports backend status

**Verification:**
- [x] Unit tests for runtime detection
- [x] `pi-box system doctor` includes runtime backend table

**Files:** `pkg/runtime/detect/detect.go`, `pkg/system/doctor.go`
**Size:** M
**Depends on:** F3, F15, F18

### T19.2: Selection and fallback policy

**Acceptance criteria:**
- [x] Explicit mode selection is honored
- [x] Auto-selection follows config/trust policy
- [x] Secure-to-compat fallback is policy-gated and logged

**Verification:**
- [x] Unit tests for explicit/auto/fallback selection
- [x] Integration test: secure unavailable fallback behavior

**Files:** `pkg/runtime/detect/detect.go`, `pkg/logs/entry.go`
**Size:** M
**Depends on:** T19.1, F17

### T19.3: Daemon-facing supported runtime contract ✅

**Description:** Keep daemon and GUI runtime selection aligned on user-facing runtime mode names and host capability checks.

**Acceptance criteria:**
- [x] Runtime discovery reports supported modes as `secure`, `fast`, `compat`, and `microvm`
- [x] gVisor remains an implementation detail behind the `secure` mode
- [x] Sandbox creation without an explicit mode uses the detected best available mode
- [x] Sandbox creation with an unavailable explicit mode fails before creating session metadata

**Verification:**
- [x] Unit tests verify runtime detection exposes the best mode in the available list
- [x] API tests verify default-mode creation and unavailable explicit-mode rejection

**Files:** `pkg/runtime/detect/detect.go`, `pkg/api/sandbox_create.go`, `tests/runtime/detect/detect_test.go`, `tests/api/sandbox_test.go`
**Size:** S
**Depends on:** T19.1, T19.2

## Verification Plan

- [x] Runtime availability detection works
- [x] Doctor reports backends
- [x] Fallback decisions are observable
- [x] Policy prevents unsafe fallback

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| ~~Auto-selection trust/config policy is high-level~~ | §13 Runtime modes | **Resolved 2026-06-27:** implemented deterministic runtime priority in `pkg/runtime/detect/detect.go`; future policy expansion can be proposed separately. |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Runtime selection precedence | F18, F19, F20 | ADR for runtime registry and fallback |

## Out of Scope

- Implementing the backend internals selected by the registry
