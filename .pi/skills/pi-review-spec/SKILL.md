---
name: pi-review-spec
description: Reviews feature specs for completeness, traceability, and correctness before implementation begins. Use when a feature spec has been written and needs validation against the block spec, contract digests, and ADRs before proceeding to planning/implementation.
---

# Pi Review Spec

> ⛔ **NEVER modify `{spec_path}` directly.**
> If you find spec gaps, write a proposal to `docs/proposals/PROP-{NNN}-{slug}.md`.

> 📂 **Document placement rule:** Review verdict is inline (in the response) only.
> If a gap found → `docs/proposals/PROP-{NNN}.md`. Never write to `docs/reviews/` or `docs/process/`.

## Overview

Validate that a feature spec (`docs/features/F{N}-*.md`) is correct, complete, and ready to drive implementation. This is the quality gate between "spec written" and "planning begins." Catches missing acceptance criteria, untraceable requirements, contract mismatches, and spec drift before expensive implementation work starts.

## When to Use

- After `/skill:pi-feature-spec` produces a new feature spec
- Before moving a feature from 🟡 (spec written) to 🟢 (ready for planning)
- When the block spec has been updated and existing feature specs need revalidation
- When a feature spec references new ADRs that have been accepted
- Periodic audit of feature specs for staleness

**When NOT to use:** Writing a feature spec (use `pi-feature-spec`), reviewing code (use `/skill:pi-execute-plan` Step 8a inline review).

## Process

### Step 1: Load Context

```
1. {spec_path}   → source of truth
2. docs/features/F{N}-*.md                  → the spec under review
3. docs/contracts/*.md                      → upstream interfaces referenced
4. docs/decisions/ADR-*.md                  → architectural decisions referenced
5. docs/features/INDEX.md                   → current status
```

### Step 2: Traceability Check

Every claim in the feature spec MUST trace back to the block spec:

| Check | Pass Criteria |
|-------|---------------|
| Feature definition | Exact match or faithful expansion of block spec § Features F{N} |
| Acceptance criteria | Each AC maps to a specific block spec acceptance criterion |
| No invented requirements | Nothing in the feature spec that isn't sourced from block spec |
| Out of scope | Matches block spec § Out of Scope |
| Security considerations | Derived from block spec § Sandbox Security Model |

**If an AC doesn't trace → flag as "untraceable" (potential invented requirement)**

### Step 3: Completeness Check

| Section | Required? | Check |
|---------|-----------|-------|
| Definition | ✅ | Exact text from block spec included |
| Expanded Specification | ✅ | Sufficient detail to implement without ambiguity |
| Acceptance Criteria | ✅ | ≥1 AC per feature requirement; all testable |
| Interface Impact | ✅ | Identifies which endpoints/events are affected |
| Security Considerations | ✅ | Non-empty; references security model |
| Dependencies | ✅ | All upstream blocks listed with status |
| Implementation Approach | ✅ | References ADRs; category assigned |
| Tasks | ✅ | ≥1 task; all sized S or M; each has verification |
| Verification Plan | ✅ | E2E verification described |
| Spec Gaps | If any | Gaps documented, not worked around |
| Out of Scope | ✅ | Explicit boundaries stated |

**Task `**Files:**` section checks (W2 — run for every task):**

```bash
# 1. No phantom paths — every cited non-test file must exist:
#    (test file citations are checked in Step 8 — inline process, no external skill needed)
for path in $(grep -A1 '\*\*Files:\*\*' docs/features/F{N}-*.md | grep -v 'Files' | grep -v '^--$' | grep -oE '[a-z][^` ,]+\.(py|ts|yaml|json|md|sh)');
  do [ -f "$path" ] || echo "PHANTOM: $path"; done

# 2. No duplicate basenames in a single task's Files list:
#    A basename appearing twice = copy-paste error; second entry should be a different file
#    (manual scan: read each Files: line and check for repeated filenames)
```

Flag any phantom non-test path as 🔴. Flag any duplicate basename as 🟡 (likely copy-paste error — identify the intended second file).

### Step 4: Contract Consistency Check

This is the most failure-prone step. Work through **every contract file** systematically. Do not skim — read the exact field names and compare them literally against the feature spec.

#### 4a. Load all contracts and map them to the feature

For the feature under review, identify which contracts are relevant:

| Contract file | Relevant when feature... |
|--------------|-------------------------|
| `docs/contracts/orchestrator.md` | ...receives dispatches, pushes events, reports completion, handles cancel, exposes /health |
| `docs/contracts/session-manager.md` | ...reads message history, appends messages, persists/reads state |
| `docs/contracts/gateway.md` | ...configures Pi to route LLM calls through Gateway |
| `docs/contracts/secret-manager.md` | ...resolves credentials for Git push |
| `docs/contracts/workspaces.md` | ...works with workspace files or Git push |

Read **every** relevant contract file in full before proceeding.

#### 4b. For each contract: check all directions

Each contract has multiple directions. Check ALL of them:

| Direction | Example | Check |
|-----------|---------|-------|
| Pi receives FROM upstream | `POST /v1/run` payload | Field names, types, required vs optional match spec |
| Pi sends TO upstream | `POST /internal/events/{run_id}` | Schema matches contract exactly |
| Pi reads FROM upstream | `GET /v1/sessions/{id}/messages` | All read operations covered by tasks |
| Pi writes TO upstream | `POST /v1/sessions/{id}/messages` | All write operations covered by tasks |
| Negative constraints | e.g. "This block does NOT call Block-X directly" | Spec must not reference a direct call when contract forbids it |

#### 4c. Field-level comparison — produce a table

For each request/response payload in the spec, produce a side-by-side comparison:

```markdown
| Field | Contract value | Spec value | Match? |
|-------|---------------|------------|--------|
| `resource_limits.cpu_*` | `cpu_seconds` | `cpu_cores` | ❌ MISMATCH |
| `references` in Secret Mgr | array `[]` | singular `reference` | ❌ MISMATCH |
| `events[].type` | `tool_call` | `tool_call` | ✅ |
```

**Flag every mismatch as a contract violation**, even minor ones (singular vs plural, underscore vs camelCase, string vs number).

#### 4d. Contract completeness check

For each contract, verify that **every operation** the contract defines is either:
- Covered by a spec task, OR
- Explicitly noted as out of scope with justification

Example: `docs/contracts/session-manager.md` defines four operations. Are all four covered?

```markdown
| Contract operation | Covered by task? | Notes |
|-------------------|-----------------|-------|
| POST /v1/sessions/{id}/messages | ✅ T5.3 | |
| PUT /v1/sessions/{id}/state | ❌ NO TASK | Missing — crash recovery not specced |
| GET /v1/sessions/{id}/messages | ✅ T5.0 | |
| GET /v1/sessions/{id}/state | ❌ NO TASK | Missing — resume after crash not specced |
```

#### 4e. Event schema two-layer check

The event system has two distinct naming layers. Both must be consistent:

| Layer | Field | Valid values | Source |
|-------|-------|-------------|--------|
| Streaming type | `type` | `thinking`, `tool_call`, `tool_result`, `text_delta`, `file_change`, `command_output`, `status`, `done`, `error` | `docs/contracts/orchestrator.md` |
| Audit action type | `action_type` | `pi.run.started`, `pi.step.started`, `pi.tool.called`, `pi.tool.completed`, `pi.file.changed`, `pi.command.executed`, `pi.run.completed`, `pi.sandbox.created`, `pi.sandbox.destroyed` | `AGENTS.md § Event Types` |

Check:
- [ ] Spec does not use `pi.xxx` names as the `type` field value
- [ ] Spec does not use `tool_call`-style names as the `action_type` field value
- [ ] Spec does not conflate the two layers into a single "Pi event type" column
- [ ] Event object schema is `{ type, data, timestamp, workspace_id, actor_id, action_type, resource_id }` — traceability fields inline (contract schema extended; see PROP-002)
- [ ] Pure streaming events (`text_delta`, `thinking`) have `action_type: null`

#### 4f. Negative constraint check

Some contracts explicitly state what this block must NOT do. Verify each:

| Constraint | Source | Check |
|-----------|--------|-------|
| This block does NOT call {Other Block} directly (example) | `docs/contracts/{block}.md` | Spec must not reference a `GET/POST /...` call to that block or a "metadata call" if forbidden |
| Git credentials must not be persisted to container filesystem | `docs/contracts/secret-manager.md` | Spec must not write credentials to non-tmpfs paths |
| Credentials used in memory only, discarded after operation | `docs/contracts/secret-manager.md` | Spec must not cache credentials between runs |

#### 4g. AGENTS.md cross-cutting requirements check

Verify the spec respects the cross-cutting requirements defined in `AGENTS.md`:

- [ ] Every emitted event includes the full traceability envelope (`workspace_id`, `actor_id`, `timestamp`, `action_type`, `resource_id`)
- [ ] No event in the spec omits the envelope without explicit justification
- [ ] Upstream source events from `.pi/block.yaml` § `lifecycle_events` (and any block-specific raw event stream listed in the block spec) are all mapped (none missed)

**Reverse direction check (W5 — added to catch omissions in AGENTS.md itself):**

For every upstream raw event **named in the spec's mapping table** (e.g., `text_delta`, `message_update`, `tool_execution_start` for an LLM-driven block):
- [ ] Verify it appears in `AGENTS.md § Event Types` — even if its `action_type` is `null`
- [ ] If an event is handled by the spec but absent from AGENTS.md → flag as **AGENTS.md gap** and create/reference a proposal

> **Why:** The review check was previously unidirectional (spec → AGENTS.md only). Pi-internal streaming events like `text_delta` and `message_update` were handled correctly in the spec but never appeared in AGENTS.md, making the table an incomplete reference for new contributors.

**Produce a verdict for Step 4:**
- ✅ Consistent — all fields match, all operations covered, no negative constraints violated
- ⚠️ Drift Found — list each specific mismatch with contract reference
- 🔴 Contract Violation — field name/type mismatch or negative constraint violated

**If contracts have drifted since spec was written → flag for update**

#### 4h. Numeric field semantic check (W6)

Some contract fields have non-obvious semantics: they are a *budget* or *total*
that the implementation *derives* into a different unit (e.g., a rate or a count).
Field-name comparison alone won't catch an AC that describes the derived unit
using the wrong terminology.

For every numeric field in the spec's ACs and implementation notes, verify:

| Field | Contract semantic | Wrong AC wording | Correct AC wording |
|-------|------------------|-----------------|--------------------|
| `cpu_seconds` | Total CPU-time budget (seconds) | "cannot consume more than N vCPUs" | "CPU rate capped at `cpu_seconds / max_duration_seconds` cores" |
| `memory_mb` | Hard memory limit in MiB | "memory budget" | "container OOM-killed at {N} MiB" |
| `max_duration_seconds` | Wall-clock timeout | "CPU timeout" | "wall-clock execution timeout" |
| `disk_mb` | Ephemeral storage cap | "workspace-only quota" | "total container writable storage cap" |

Flag any AC whose prose description uses a colloquial shorthand that misrepresents
the contract field's actual unit or enforcement semantics as 🟡 NEEDS_REVISION.

- [ ] All referenced ADRs exist and are "Accepted" status
- [ ] Implementation approach is consistent with ADR decisions
- [ ] No contradictions between feature spec and ADR consequences
- [ ] If ADR gaps are noted, they're specific enough that an ADR can be authored using the template in `/skill:pi-feature-spec` § Surfacing an ADR need

#### 5a. Topology assumption check

For every ADR referenced and every networking/communication detail in the spec:

| Check | Pass criteria |
|-------|---------------|
| Does the spec assume same-cluster pod IP routing? | Either (a) explicitly scoped to single-cluster only, OR (b) resolved by an ADR that covers split-environment |
| Does the spec use `podSelector` in a NetworkPolicy? | Same as above — `podSelector` is same-cluster only |
| Does the spec use cluster-local DNS (`*.svc.cluster.local`)? | Same — cluster-local DNS does not cross environment boundaries |
| Does the spec assume a single `KUBECONFIG`? | If split-environment topology is in scope, secondary kubeconfig handling must be referenced |

**If any check fails without an ADR covering it → flag as topology gap; author an ADR using the template in `/skill:pi-feature-spec` § Surfacing an ADR need.**

This check exists because ADR-004's split-environment gap went undetected through F5/F6/F12/F13 spec authoring — the assumption was implicit in "direct Pod IP" and `podSelector` wording without being marked as single-cluster only. See ADR-008 § Gap detection retrospective.

### Step 6: Testability Check

For each acceptance criterion:
- [ ] Can be verified by an automated test
- [ ] Test boundary is clear (unit? integration? e2e?)
- [ ] Negative cases covered (what should NOT happen)
- [ ] Mock services sufficient to test this (or new mock needed)

### Step 7: Produce Review Verdict

```markdown
## Spec Review: F{N} — {Feature Name}

**Reviewer:** Agent
**Date:** YYYY-MM-DD
**Spec version reviewed:** {git commit or date}

### Traceability

| AC | Block Spec Source | Verdict |
|----|-------------------|---------|
| AC-{N}.1 | § {section} | ✅ Traceable / ⚠️ Loose / 🔴 Untraceable |
| AC-{N}.2 | ... | ... |

### Completeness: {Complete / Gaps Found}

- [ ] {missing section or detail}

### Contract Consistency: {Consistent / Drift Found}

- [ ] {inconsistency details}

### ADR Alignment: {Aligned / Conflicts Found}

- [ ] {conflict details}

### Testability: {All Testable / Issues Found}

- [ ] {untestable criterion}

### Plan-AC Sync: {Synced / Drift Found}

- [ ] {uncovered AC — no plan row}

### Verdict: APPROVED | NEEDS_REVISION | BLOCKED

**Summary:** {one paragraph}

**Required fixes before planning can begin:**
1. {specific fix}
2. ...

**Recommended improvements (non-blocking):**
1. {suggestion}
2. ...
```

### Step 8: Test Coverage and Gap Resync

> A spec review that only flags gaps but doesn't resync the spec and plan
> leaves silent debt. This step finds gaps **and** immediately records them
> in the feature spec and plan so they are visible and tracked.

#### 8a. Inventory and map

```bash
# All existing test files
find {code.tests_root} -name "*_test.go" | sort

# Near-duplicates (same module prefix, two files)
find {code.tests_root} -name "*_test.go" -exec basename {} \; | sort | \
  awk -F_ 'NF>2{print $1"_"$2}' | sort | uniq -d
```

For each task in the spec, for each AC:

| AC | Existing test file | Covers AC? | Gap? |
|----|-------------------|------------|------|
| AC-{N}.1 | `{code.tests_root}events/translator_test.go` | ✅ real test | — |
| AC-{N}.2 | none found | — | 🔴 NO TEST |
| AC-{N}.3 | `{code.tests_root}runtime/run_handler_test.go` | skip only | 🟡 STUB |

#### 8b. Verify cited paths exist and are not stubs

```bash
# No phantom test citations:
grep -rh 'Files:' docs/features/F*.md | grep -o '{code.tests_root}[^` ,)]*\.go' | sort -u | \
  while read f; do [ -f "$f" ] || echo "MISSING: $f"; done

# Detect skip-only files (no real assertion):
for f in $(grep -rh 'Files:' docs/features/F*.md | grep -o '{code.tests_root}[^` ,)]*\.go' | sort -u); do
  [ -f "$f" ] && grep -q 't\.Skip\|t\.Skipf\|t\.SkipNow' "$f" && echo "STUB: $f"
done
```

**Markdown backtick balance check:**
```bash
grep -n '\*\*Files:\*\*' docs/features/F{N}-*.md -A1 | \
  awk '/Files:/{getline; gsub(/[^`]/,""); if(length%2!=0) print NR": UNBALANCED: "$0}'
```

#### 8c. Resync spec and plan for every gap found

For each 🔴 NO TEST or 🟡 STUB gap found:

**In the feature spec** (`docs/features/F{N}-*.md`):
- Update the task's `**Files:**` to cite the correct real test (or stub)
- If the task was ✅ but only has a stub covering its AC → reset task status to ⚠️
- Add a note: `*(Test gap detected {date}: {AC} has no real test — stub only)*`

**In the feature spec** (`docs/features/F{N}-*.md`):
- For every task reset to ⚠️: update the `## Tasks` table row from ✅ to ⚠️
- Add a dated note: `*(2026-MM-DD: reset — AC-{N}.X has stub-only coverage; needs real test)*`

**Rule: a spec review that finds a test gap but does not reset the task status
in the feature spec is incomplete. The gap must be visible before the review verdict is produced.**

#### 8d. Feature Spec AC-to-Task Sync Check

> **Rule:** Every AC in a feature spec MUST have a corresponding task row in `## Tasks`
> (either a dedicated row or noted as covered by another task).

For the feature under review:

1. **Extract all AC-{N}.{X} references** from the spec file.
2. **Read `docs/features/F{N}-*.md` `## Tasks` table** — collect all task rows.
3. **Check coverage:** for each AC, verify at least one task row references it.
4. **If an AC has no task row:**
   - Add a new task row to `## Tasks` covering the AC.
   - If the feature has NO task rows at all → flag as 🔴 BLOCKED — spec has ACs but no tasks.

**Rule: a spec review that adds an AC without updating the plan is
incomplete. The plan must reflect every AC before the review verdict
is produced.**

#### 8e. Verdict input

#### 8e. Verdict input

- All ACs have real tests → ✅ pass Step 8, proceed to Step 9
- Some ACs have stubs only → 🟡 NEEDS_REVISION (tasks reset to ⚠️, plan updated)
- Some ACs have no test at all and no stub → 🔴 NEEDS_REVISION (same, plus stub must be created)

### Step 9: Update Feature Index

If approved:
- Update `docs/features/INDEX.md`: 🟡 → 🟢 (ready for planning)

If needs revision:
- Keep at 🟡; note revision needed
- Create specific issues in the spec's "Spec Gaps" section

## Review Severity Levels

| Level | Meaning | Action |
|-------|---------|--------|
| 🔴 BLOCKED | Untraceable requirement, security gap, or contract violation | Cannot proceed. Fix spec first. |
| 🟡 NEEDS_REVISION | Missing sections, weak ACs, or stale references | Fix before planning, but not fundamentally broken. |
| 🟢 APPROVED | Complete, traceable, testable, consistent | Proceed to planning. |
