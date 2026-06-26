# F19: Runtime Selection & Fallback

> Source: `SPEC.md` §6 Features F19
> Status: 🟡 Spec written
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

- [ ] AC-22.1: `pi system doctor` reports available runtime backends
- [ ] AC-22.2: Backend selection honors explicit `--mode` requests
- [ ] AC-22.3: Auto-selection prefers an available compatible backend based on trust/config
- [ ] AC-22.4: Secure-mode startup failure can fall back to compat mode when policy permits
- [ ] AC-22.5: Fallback decisions are visible in logs/history

## Interface Impact

| Component | Impact |
|-----------|--------|
| `pkg/runtime/` | Runtime registry/detection (new — to be created) |
| `pkg/system/` | Doctor backend availability |
| `pkg/logs/` | Fallback decisions recorded |
| `cmd/pi/box` | Explicit and auto runtime mode flags |

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
- [ ] Detect fast, compat, secure, and future microVM backend availability
- [ ] `pi system doctor` reports backend status

**Verification:**
- [ ] Unit tests for runtime detection
- [ ] `pi system doctor` includes runtime backend table

**Files:** `pkg/runtime/registry.go (new — to be created)`, `pkg/system/doctor.go`
**Size:** M
**Depends on:** F3, F15, F18

### T19.2: Selection and fallback policy

**Acceptance criteria:**
- [ ] Explicit mode selection is honored
- [ ] Auto-selection follows config/trust policy
- [ ] Secure-to-compat fallback is policy-gated and logged

**Verification:**
- [ ] Unit tests for explicit/auto/fallback selection
- [ ] Integration test: secure unavailable fallback behavior

**Files:** `pkg/runtime/selector.go (new — to be created)`, `pkg/logs/entry.go`
**Size:** M
**Depends on:** T19.1, F17

## Verification Plan

- [ ] Runtime availability detection works
- [ ] Doctor reports backends
- [ ] Fallback decisions are observable
- [ ] Policy prevents unsafe fallback

## Spec Gaps

### Block spec gaps (propose amendment — NEVER edit the block spec directly)

| Gap | Block Spec Section | Proposed Amendment |
|-----|-------------------|--------------------|
| Auto-selection trust/config policy is high-level | §13 Runtime modes | Add exact precedence rules if implementation needs more detail |

### ADR gaps (needs architectural decision)

| Question | Affects Features | Proposed ADR |
|----------|-----------------|--------------|
| Runtime selection precedence | F18, F19, F20 | ADR for runtime registry and fallback |

## Out of Scope

- Implementing the backend internals selected by the registry

