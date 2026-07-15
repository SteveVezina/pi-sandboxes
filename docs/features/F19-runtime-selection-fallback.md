# F19: Runtime Selection & Fallback

> Source: `SPEC.md` §6 Features F19
> Status: 🟢 Implemented *(2026-07-14: PROP-008 cascade complete — T19.1/T19.2 re-verified on driver contract)*
> Category: Service-layer

## Definition (from block spec)

| Feature ID | Name | Description | Milestone |
|------------|------|-------------|-----------|
| F19 | Runtime Selection & Fallback | Runtime detection and backend selection across fast, compat, secure, and future microVM modes, including fallback to compat mode when secure mode is unavailable or incompatible | M4 |

## Expanded Specification

Runtime selection centralizes backend discovery, explicit mode handling, auto-selection, and fallback. It must make backend decisions visible in system doctor output, logs, and command history.

Fallback cannot weaken policy silently. If fallback is allowed, the user-visible result must explain what happened and which backend actually ran.

Per PROP-008 / ADR-005 (`SPEC.md` §14.7.5):

- Backends implement the lifecycle `Driver` contract (`Probe/Create/Start/Exec/Inspect/Stop/Destroy/Stats`); detection is no longer a metadata-only interface.
- `Probe` returns a structured `CapabilityReport` (availability, missing prerequisites, capability flags, isolation tier, compatibility tier). The one-integer `GetSecurityLevel()` is removed. Probes must actually execute.
- Selection separates four inputs — requested mode, workload trust, host capabilities, explicit fallback allow/deny policy — replacing the single global priority list. `auto` is trust-dependent: trusted work prefers performance ordering, untrusted work prefers isolation ordering.
- Isolation is never silently downgraded below the requested mode; denied fallbacks fail with actionable guidance from the capability report.

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

### T19.1: Runtime registry and detection ✅ *(2026-07-14: re-verified per PROP-008 — capability reports + registry landed)*

**Acceptance criteria:**
- [x] Detect fast, compat, secure, and microVM backend availability via registered probers (`pkg/runtime.Registry`)
- [x] `Probe` returns a structured `CapabilityReport` (availability, reason, missing prerequisites, capability flags, isolation/compat tiers)
- [x] Probes actually execute their checks (no always-true validation)
- [x] Doctor and `GET /v1/system/runtimes` render capability reports instead of `security_level` integers (GUI updated to match)

**Verification:**
- [x] Unit tests for registry ordering, duplicate rejection, probe execution (`tests/runtime/registry_test.go`)
- [x] Unit tests for per-mode reports (`tests/runtime/detect/detect_test.go`)
- [x] API test: `/v1/system/runtimes` backends carry `isolation_tier`/`compat_tier`, never `security_level` (`tests/api/system_test.go`)
- [x] Doctor surfaces unavailable-mode reasons and missing prerequisites

**Files:** `pkg/runtime/capabilities.go`, `pkg/runtime/driver.go`, `pkg/runtime/registry.go`, `pkg/runtime/detect/detect.go`, `pkg/system/doctor.go`, `pkg/system/system.go`, `pkg/api/system.go`, `apps/gui/src/api.ts`, `apps/gui/src/main.tsx`
**Size:** M
**Depends on:** F3, F15, F18

### T19.2: Selection and fallback policy ✅ *(2026-07-14: re-verified per PROP-008 — four-input selection engine landed)*

**Acceptance criteria:**
- [x] Explicit mode selection is honored
- [x] Selection takes requested mode, workload trust, host capabilities, and explicit fallback allow/deny policy as separate inputs (`pkg/runtime/selector.go`)
- [x] `auto` resolution is trust-dependent (trusted → performance ordering fast→compat→secure; untrusted → isolation ordering secure→isolated→microvm, never shared-kernel modes)
- [x] Isolation never silently downgrades below the requested mode (fallback candidates below the requested isolation tier are rejected even when allow-listed); denied fallback fails with actionable guidance including missing prerequisites
- [x] Fallback decisions are persisted with requested-vs-resolved mode fields (`requested_mode`, `fallback_reason` in session metadata)

**Verification:**
- [x] Unit tests for explicit/auto/fallback selection across trust levels (`tests/runtime/selector_test.go` — 8 cases: explicit honored, guidance on failure, upward-only fallback, no-downgrade, deny-wins, trusted/untrusted auto, untrusted never shared-kernel)
- [x] API path: sandbox creation resolves mode through the selection engine; unavailable explicit mode fails with capability-report guidance

**Files:** `pkg/runtime/selector.go`, `pkg/api/sandbox_create.go`, `pkg/session/meta.go`, `pkg/session/store.go`
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
| ~~Runtime selection precedence~~ | F18, F19, F20 | **Resolved 2026-07-14:** ADR-005 (runtime driver contract and selection engine, per PROP-008) |

## Out of Scope

- Implementing the backend internals selected by the registry
